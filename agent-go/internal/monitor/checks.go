package monitor

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// --- service / process checks -------------------------------------------------

// ServiceActive reports whether a systemd unit is in the "active" state.
func ServiceActive(name string) bool {
	out, err := exec.CommandContext(context.Background(), "systemctl", "is-active", name).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "active"
}

// processAlive reports whether at least one process with the given binary name
// is running. Falls back gracefully when pgrep is unavailable.
func processAlive(name string) bool {
	out, err := exec.CommandContext(context.Background(), "pgrep", "-x", name).Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

// --- port binding (local /proc) checks ----------------------------------------

// portBound reports whether the given port is bound as a listening socket.
// On Linux this is authoritative by parsing /proc; elsewhere it returns true
// (unknown) so that reachability probes remain the source of truth.
func portBound(port int, transport string) (bool, string) {
	if runtime.GOOS != "linux" {
		return true, "non-linux, skipped"
	}
	var files []string
	switch transport {
	case "tcp":
		files = []string{"/proc/net/tcp", "/proc/net/tcp6"}
	case "udp":
		files = []string{"/proc/net/udp", "/proc/net/udp6"}
	default:
		return false, "unknown transport"
	}
	for _, path := range files {
		ok, addr, err := scanProcSocket(path, port, transport)
		if err != nil {
			continue
		}
		if ok {
			return true, fmt.Sprintf("bound on %s", addr)
		}
	}
	return false, "port not found in local socket table"
}

// scanProcSocket parses a /proc/net/{tcp,udp,tcp6,udp6} file looking for a
// listening socket on the given hex port. For TCP only LISTEN state matches.
func scanProcSocket(path string, wantPort int, transport string) (bool, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first { // header row
			first = false
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		local := fields[1]
		state := fields[3]
		addr, port, ok := splitAddrPort(local)
		if !ok {
			continue
		}
		// Compare by integer to avoid leading-zero mismatch (/proc uses
		// zero-padded hex like "01BB" for port 443).
		if parseHex(port) != wantPort {
			continue
		}
		if transport == "tcp" && !strings.EqualFold(state, "0A") {
			continue // only LISTEN
		}
		return true, fmt.Sprintf("%s:%d", addr, wantPort), nil
	}
	return false, "", nil
}

func splitAddrPort(local string) (string, string, bool) {
	parts := strings.SplitN(local, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func parseHex(s string) int {
	n, _ := strconv.ParseInt(s, 16, 64)
	return int(n)
}

// --- TCP reachability ---------------------------------------------------------

// tcpReach attempts a TCP dial against host:port with a bounded timeout and
// optional retries to avoid false negatives from transient packet loss.
func tcpReach(ctx context.Context, host string, port int, attempts int) (bool, int64, string) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	var lastErr string
	for i := 0; i < attempts; i++ {
		start := time.Now()
		d := &net.Dialer{Timeout: 3 * time.Second}
		conn, err := d.DialContext(ctx, "tcp", addr)
		lat := time.Since(start).Milliseconds()
		if err == nil {
			_ = conn.Close()
			return true, lat, fmt.Sprintf("connected in %dms", lat)
		}
		lastErr = truncateErr(err)
		select {
		case <-ctx.Done():
			return false, lat, "context cancelled"
		case <-time.After(200 * time.Millisecond):
		}
	}
	return false, 0, lastErr
}

// --- TLS handshake + certificate inspection -----------------------------------

// tlsInspect performs a TLS handshake against host:port using serverName and
// returns the leaf certificate issuer/subject, days until expiry, and whether
// the handshake succeeded. insecureSkipVerify is set so we can still report
// certificate details of self-signed or misconfigured endpoints.
func tlsInspect(ctx context.Context, host string, port int, serverName string) (bool, string, string, int, string) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	d := &net.Dialer{Timeout: 3 * time.Second}
	tlsConf := &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: true,
		NextProtos:         []string{"h2", "http/1.1"},
	}
	start := time.Now()
	conn, err := tls.DialWithDialer(d, "tcp", addr, tlsConf)
	if err != nil {
		return false, "", "", 0, truncateErr(err)
	}
	defer conn.Close()
	state := conn.ConnectionState()
	lat := time.Since(start).Milliseconds()
	if len(state.PeerCertificates) == 0 {
		return true, "", "", 0, fmt.Sprintf("handshake ok, no cert (%dms)", lat)
	}
	leaf := state.PeerCertificates[0]
	days := int(time.Until(leaf.NotAfter).Hours() / 24)
	issuer := leaf.Issuer.CommonName
	if issuer == "" {
		issuer = strings.Join(leaf.Issuer.Organization, " ")
	}
	if len(issuer) > 0 && len(issuer) > 40 {
		issuer = issuer[:40] + "..."
	}
	subject := leaf.Subject.CommonName
	return true, issuer, subject, days, fmt.Sprintf("handshake ok (%dms)", lat)
}

// certFileInspect reads a PEM cert file from disk and returns expiry info
// without opening a network connection (used for QUIC protocols whose cert is
// served over UDP and harder to inspect via a TLS dial).
func certFileInspect(certPath string) (string, string, int, error) {
	data, err := os.ReadFile(certPath)
	if err != nil {
		return "", "", 0, err
	}
	// Parse the first PEM block as a certificate.
	cert, err := loadPEMCert(data)
	if err != nil {
		return "", "", 0, err
	}
	days := int(time.Until(cert.NotAfter).Hours() / 24)
	issuer := cert.Issuer.CommonName
	if issuer == "" {
		issuer = strings.Join(cert.Issuer.Organization, " ")
	}
	return issuer, cert.Subject.CommonName, days, nil
}

// --- config validity ----------------------------------------------------------

// configValid parses the JSON config at path and reports whether it is valid.
func configValid(path string) (bool, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Sprintf("read %s: %v", path, err)
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return false, fmt.Sprintf("invalid JSON: %v", err)
	}
	return true, fmt.Sprintf("valid JSON (%d bytes)", len(data))
}

// --- log error scanning -------------------------------------------------------

// recentLogErrors counts error-level lines from a service journal over the
// given lookback window. Returns 0 when journalctl is unavailable (non-Linux).
func recentLogErrors(service string, lookback time.Duration) (int, string) {
	if runtime.GOOS != "linux" {
		return 0, "skipped (non-linux)"
	}
	since := fmt.Sprintf("--since=%d minutes ago", int(lookback.Minutes()))
	out, err := exec.CommandContext(context.Background(), "journalctl",
		"-u", service, since, "-p", "err", "--no-pager", "-q").Output()
	if err != nil {
		// Not necessarily fatal: journal may be empty or journald absent.
		return 0, "journalctl unavailable"
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0, "no errors"
	}
	return len(lines), fmt.Sprintf("%d error lines", len(lines))
}

// --- helpers ------------------------------------------------------------------

func truncateErr(err error) string {
	msg := err.Error()
	if i := strings.Index(msg, ": write:"); i > 0 {
		msg = msg[:i]
	}
	if len(msg) > 90 {
		msg = msg[:90] + "..."
	}
	return msg
}

// dialDenied reports whether a dial failure looks like the server refusing the
// connection (reset/refused) rather than a timeout or unreachable host.
func dialDenied(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "refused") || strings.Contains(s, "reset") ||
		strings.Contains(s, "EOF")
}

// contextError reports whether err is a context deadline / cancellation.
func contextError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) ||
		strings.Contains(err.Error(), "deadline") || strings.Contains(err.Error(), "context")
}

// fileExists is a small helper.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// hostnameSafe returns the system hostname or "unknown".
func hostnameSafe() string {
	h, err := os.Hostname()
	if err != nil || h == "" {
		return "unknown"
	}
	return h
}

// loadPEMCert parses the first CERTIFICATE PEM block from data.
func loadPEMCert(data []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("no PEM block found")
	}
	return x509.ParseCertificate(block.Bytes)
}
