// mocksrv is a test-only server that simulates the xraycli proxy ports so the
// monitor's healthy/degraded detection can be validated on a host without the
// real Xray/Sing-box cores.
//
// It provides:
//   - TCP listeners on 15393, 443, 31296 (the xray inbound + fallback chain)
//   - A TLS listener on 15393 with a self-signed cert (reality proxy)
//   - UDP listeners on 20505, 20185 that answer QUIC Version Negotiation
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cert := genSelfSigned("itunes.apple.com")

	// TCP+TLS on 15393 (VLESS Reality)
	go serveTLS(":15393", cert)
	// Plain TCP on 443 (VLESS/Trojan entry) and 31296 (internal fallback)
	go serveTCP(":443")
	go serveTCP(":31296")
	// UDP QUIC VN responders
	go serveQUICVN(":20505")
	go serveQUICVN(":20185")

	fmt.Println("mocksrv listening: tcp/15393(tls) tcp/443 tcp/31296 udp/20505 udp/20185")

	// Wait for SIGINT/SIGTERM.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	fmt.Println("mocksrv stopping")
}

func serveTCP(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Printf("tcp %s: %v\n", addr, err)
		return
	}
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		_ = c.Close()
	}
}

func serveTLS(addr string, cert tls.Certificate) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Printf("tls %s: %v\n", addr, err)
		return
	}
	cfg := &tls.Config{Certificates: []tls.Certificate{cert}}
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		go func(conn net.Conn) {
			tlsConn := tls.Server(conn, cfg)
			_ = tlsConn.Handshake()
			_ = tlsConn.Close()
		}(c)
	}
}

// serveQUICVN listens for UDP datagrams and, if one looks like a long-header
// QUIC packet with an unsupported version, replies with a minimal Version
// Negotiation packet so the monitor's quicReach probe succeeds.
func serveQUICVN(addr string) {
	conn, err := net.ListenUDP("udp", mustUDPAddr(addr))
	if err != nil {
		fmt.Printf("udp %s: %v\n", addr, err)
		return
	}
	buf := make([]byte, 1500)
	for {
		n, raddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		if n < 6 || buf[0]&0x80 == 0 {
			continue // not a long header
		}
		vn := buildVNPDU(buf[:n])
		_, _ = conn.WriteToUDP(vn, raddr)
	}
}

// buildVNPDU constructs a Version Negotiation packet: first byte 0x80..0xBF
// range (form=1, fixed=0), version = 0, then dcid/scid echoed from the probe,
// followed by a single supported version (QUIC v1).
func buildVNPDU(req []byte) []byte {
	if len(req) < 7 {
		return nil
	}
	off := 1
	off += 4  // skip request version
	dcidLen := int(req[off])
	off++
	if off+dcidLen > len(req) {
		return nil
	}
	dcid := req[off : off+dcidLen]
	off += dcidLen
	scidLen := int(req[off])
	off++
	if off+scidLen > len(req) {
		return nil
	}
	scid := req[off : off+scidLen]

	out := []byte{0x80}            // form=1, fixed=0 (VN marker)
	out = append(out, 0, 0, 0, 0) // version = 0 (VN marker)
	out = append(out, byte(len(dcid)))
	out = append(out, dcid...)
	out = append(out, byte(len(scid)))
	out = append(out, scid...)
	var v [4]byte
	binary.BigEndian.PutUint32(v[:], 0x00000001) // QUIC v1 supported
	out = append(out, v[:]...)
	return out
}

func mustUDPAddr(addr string) *net.UDPAddr {
	a, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		panic(err)
	}
	return a
}

func genSelfSigned(cn string) tls.Certificate {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: cn, Organization: []string{"MockTest"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{cn},
	}
	der, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}
