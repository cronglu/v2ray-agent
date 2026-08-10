package monitor

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// QUIC v1 wire constants (RFC 9000).
const (
	quicVersion1   = 0x00000001
	quicVersionVNE = 0x0a0a0a0a // an unsupported version that provokes Version Negotiation
	dcidLen        = 8
	scidLen        = 8
)

// quicReach performs a lightweight QUIC liveness probe against host:port.
//
// Strategy: send a long-header packet carrying an *unsupported* QUIC version.
// A conformant QUIC server (quic-go, used by sing-box for both Hysteria2 and
// TUIC) MUST reply with a Version Negotiation packet, which is unencrypted.
// Receiving any datagram within the deadline proves the server is alive and
// processing QUIC; the VN shape is verified for an extra confidence signal.
//
// Because UDP is unreliable, the probe is sent a few times with backoff. A
// timeout is reported as "inconclusive" (port bound but no reply) rather than a
// hard failure, to keep the false-positive rate low.
func quicReach(ctx context.Context, host string, port int, attempts int) (bool, bool, int64, string) {
	addr := &net.UDPAddr{IP: net.ParseIP(host), Port: port}
	if addr.IP == nil {
		return false, false, 0, "invalid host"
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return false, false, 0, fmt.Sprintf("dial udp: %v", err)
	}
	defer conn.Close()

	pkt := buildVNProbePacket()
	deadline := 1500 * time.Millisecond

	for i := 0; i < attempts; i++ {
		select {
		case <-ctx.Done():
			return false, false, 0, "context cancelled"
		default:
		}
		start := time.Now()
		if _, err := conn.Write(pkt); err != nil {
			// A "connection refused" on UDP write means the port is not
			// actually open (ICMP port unreachable surfaced as ECONNREFUSED).
			if isRefused(err) {
				return false, false, 0, "port unreachable (ICMP refused)"
			}
			continue
		}
		_ = conn.SetReadDeadline(time.Now().Add(deadline))
		buf := make([]byte, 1500)
		n, _, rerr := conn.ReadFromUDP(buf)
		lat := time.Since(start).Milliseconds()
		if rerr == nil && n > 0 {
			vn := isVersionNegotiation(buf[:n])
			return true, vn, lat, fmt.Sprintf("QUIC reply %d bytes (vn=%v) in %dms", n, vn, lat)
		}
		// refused on read also indicates the port is closed.
		if rerr != nil && isRefused(rerr) {
			return false, false, 0, "port unreachable (ICMP refused)"
		}
	}
	return false, false, 0, "no QUIC reply (inconclusive)"
}

// buildVNProbePacket constructs a long-header QUIC packet with an unsupported
// version so the server responds with a Version Negotiation packet.
//
// Layout: [first=0xC0][version=VNE][dcid-len][dcid][scid-len][scid][token-len=0]
//         [packet-len=1][pn=0][padding...]
func buildVNProbePacket() []byte {
	pkt := make([]byte, 0, 64)
	// First byte: long header (form=1, fixed=1), type=Initial(00), reserved=00,
	// packet-number length = 4 bits 00 (1 byte).
	pkt = append(pkt, 0xC0)
	// Unsupported version to trigger Version Negotiation.
	var ver [4]byte
	binary.BigEndian.PutUint32(ver[:], quicVersionVNE)
	pkt = append(pkt, ver[:]...)
	// Destination Connection ID (seeds server's initial keys; random).
	dcid := make([]byte, dcidLen)
	_, _ = rand.Read(dcid)
	pkt = append(pkt, byte(dcidLen))
	pkt = append(pkt, dcid...)
	// Source Connection ID.
	scid := make([]byte, scidLen)
	_, _ = rand.Read(scid)
	pkt = append(pkt, byte(scidLen))
	pkt = append(pkt, scid...)
	// Token length (varint 0).
	pkt = append(pkt, 0x00)
	// Packet length (varint) = pn(1) + payload.
	pkt = append(pkt, 0x01) // length = 1
	// Packet number.
	pkt = append(pkt, 0x00)
	// A little padding keeps the packet above the minimum datagram size that
	// some implementations require before responding.
	pkt = append(pkt, make([]byte, 20)...)
	return pkt
}

// isVersionNegotiation reports whether a datagram looks like a QUIC Version
// Negotiation packet (form bit set, fixed bit clear, version == 0).
func isVersionNegotiation(p []byte) bool {
	if len(p) < 7 {
		return false
	}
	if p[0]&0x80 == 0 { // must be long header
		return false
	}
	if p[0]&0x40 != 0 { // VN has fixed bit = 0
		return false
	}
	return binary.BigEndian.Uint32(p[1:5]) == 0x00000000
}

func isRefused(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return contains(s, "refused") || contains(s, "connection reset")
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return len(sub) == 0
}
