package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"v2ray-agent/internal/model"
)

const (
	defaultXrayConfigPath    = "/etc/v2ray-agent/xray/config.json"
	defaultSingBoxConfigPath = "/etc/v2ray-agent/sing-box/config.json"
	defaultCertBase          = "/etc/v2ray-agent/tls"
)

// Monitor orchestrates per-protocol health checks.
type Monitor struct {
	State        *model.GlobalNodeState
	Timeout      time.Duration
	CertWarnDays int
	// Host used for local reachability dials (loopback by default).
	LocalHost string
	// Config paths (overridable for tests).
	XrayConfigPath    string
	SingBoxConfigPath string
}

// New returns a Monitor with sensible defaults.
func New(state *model.GlobalNodeState) *Monitor {
	return &Monitor{
		State:             state,
		Timeout:           30 * time.Second,
		CertWarnDays:      14,
		LocalHost:         "127.0.0.1",
		XrayConfigPath:     defaultXrayConfigPath,
		SingBoxConfigPath: defaultSingBoxConfigPath,
	}
}

// discoveredInbound is a normalized view of an inbound parsed from a deployed
// config, enriched with enough metadata to drive the right probes.
type discoveredInbound struct {
	Name      string
	Protocol  model.ProtocolType
	Core      string
	Transport string
	Port      int
	Listen    string
	SNI       string
	CertPath  string
	// FallbackTarget is an extra local TCP port to verify (e.g. the 443 chain's
	// internal Trojan on 31296).
	FallbackTarget int
}

// Run executes all protocol checks concurrently and returns a report.
func (m *Monitor) Run(ctx context.Context) *Report {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, m.Timeout)
	defer cancel()

	inbounds := m.discover()

	xrayActive := ServiceActive("xray") || processAlive("xray")
	singActive := ServiceActive("sing-box") || processAlive("sing-box")

	reports := make([]ProtocolReport, len(inbounds))
	var wg sync.WaitGroup
	for i, in := range inbounds {
		wg.Add(1)
		go func(idx int, inb discoveredInbound) {
			defer wg.Done()
			reports[idx] = m.checkProtocol(ctx, inb, xrayActive, singActive)
		}(i, in)
	}
	wg.Wait()

	rep := &Report{
		Hostname:      hostnameSafe(),
		PublicIP:      m.State.PublicIP,
		XrayActive:    xrayActive,
		SingBoxActive: singActive,
		Protocols:     reports,
		CheckedAt:     time.Now(),
		DurationMs:    time.Since(start).Milliseconds(),
	}
	rep.Overall = rep.Aggregate()
	return rep
}

// discover reads the deployed Xray and Sing-box configs to learn which
// inbounds are actually configured. If a config is missing or unparseable it
// falls back to the expected ports derived from state, so the monitor still
// produces a meaningful (likely "down") report on a fresh or broken host.
func (m *Monitor) discover() []discoveredInbound {
	var out []discoveredInbound

	xrayIn, xrayOK := m.parseXrayInbounds()
	if !xrayOK {
		out = append(out, m.fallbackXrayInbounds()...)
	} else {
		out = append(out, xrayIn...)
	}

	sbIn, sbOK := m.parseSingBoxInbounds()
	if !sbOK {
		out = append(out, m.fallbackSingBoxInbounds()...)
	} else {
		out = append(out, sbIn...)
	}
	return out
}

// --- Xray discovery -----------------------------------------------------------

func (m *Monitor) parseXrayInbounds() ([]discoveredInbound, bool) {
	data, err := os.ReadFile(m.XrayConfigPath)
	if err != nil {
		return nil, false
	}
	var cfg struct {
		Inbounds []map[string]any `json:"inbounds"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, false
	}
	var out []discoveredInbound
	for _, in := range cfg.Inbounds {
		proto, _ := in["protocol"].(string)
		port := toInt(in["port"])
		if port == 0 {
			continue
		}
		listen, _ := in["listen"].(string)
		if listen == "" {
			listen = "0.0.0.0"
		}
		ss, _ := in["streamSettings"].(map[string]any)
		security, _ := ss["security"].(string)

		var di discoveredInbound
		di.Core = "xray"
		di.Transport = "tcp"
		di.Port = port
		di.Listen = listen

		switch proto {
		case "vless":
			if security == "reality" {
				di.Protocol = model.ProtoVLESSReality
				di.Name = "VLESS Reality"
				di.SNI = firstServerName(ss)
			} else if security == "tls" {
				di.Protocol = model.ProtoVLESSTLS
				di.Name = "VLESS/Trojan (443)"
				di.SNI = m.State.Domain
				// The 443 inbound falls back to the internal Trojan listener.
				if fb := m.findTrojanFallbackPort(cfg.Inbounds); fb != 0 {
					di.FallbackTarget = fb
				}
			} else {
				di.Protocol = model.ProtoVLESSReality // generic vless
				di.Name = "VLESS"
			}
		case "trojan":
			di.Protocol = model.ProtoTrojan
			di.Name = "Trojan (internal fallback)"
		default:
			continue // skip non-proxy inbounds
		}
		out = append(out, di)
	}
	return out, len(out) > 0
}

func (m *Monitor) findTrojanFallbackPort(inbounds []map[string]any) int {
	for _, in := range inbounds {
		if p, _ := in["protocol"].(string); p == "trojan" {
			if port := toInt(in["port"]); port != 0 {
				return port
			}
		}
	}
	return 0
}

func (m *Monitor) fallbackXrayInbounds() []discoveredInbound {
	var out []discoveredInbound
	out = append(out, discoveredInbound{
		Name: "VLESS Reality", Protocol: model.ProtoVLESSReality, Core: "xray",
		Transport: "tcp", Port: 15393, Listen: "0.0.0.0", SNI: m.State.RealityServerName,
	})
	out = append(out, discoveredInbound{
		Name: "VLESS/Trojan (443)", Protocol: model.ProtoVLESSTLS, Core: "xray",
		Transport: "tcp", Port: m.State.VLESSPort, Listen: "0.0.0.0", SNI: m.State.Domain,
		FallbackTarget: 31296,
	})
	return out
}

// --- Sing-box discovery -------------------------------------------------------

func (m *Monitor) parseSingBoxInbounds() ([]discoveredInbound, bool) {
	data, err := os.ReadFile(m.SingBoxConfigPath)
	if err != nil {
		return nil, false
	}
	var cfg struct {
		Inbounds []map[string]any `json:"inbounds"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, false
	}
	var out []discoveredInbound
	for _, in := range cfg.Inbounds {
		t, _ := in["type"].(string)
		port := toInt(in["listen_port"])
		if port == 0 {
			continue
		}
		listen, _ := in["listen"].(string)
		if listen == "" {
			listen = "::"
		}
		tls, _ := in["tls"].(map[string]any)
		sni, _ := tls["server_name"].(string)
		certPath, _ := tls["certificate_path"].(string)
		if certPath == "" && m.State.Domain != "" {
			certPath = fmt.Sprintf("%s/%s.crt", defaultCertBase, m.State.Domain)
		}

		di := discoveredInbound{
			Core: "sing-box", Transport: "udp", Port: port, Listen: listen,
			SNI: sni, CertPath: certPath,
		}
		switch t {
		case "hysteria2":
			di.Protocol = model.ProtoHysteria2
			di.Name = "Hysteria2"
		case "tuic":
			di.Protocol = model.ProtoTUIC
			di.Name = "TUIC v5"
		default:
			continue // skip vless/vmess etc. handled by Xray
		}
		out = append(out, di)
	}
	return out, len(out) > 0
}

func (m *Monitor) fallbackSingBoxInbounds() []discoveredInbound {
	cert := fmt.Sprintf("%s/%s.crt", defaultCertBase, m.State.Domain)
	if m.State.Domain == "" {
		cert = ""
	}
	return []discoveredInbound{
		{
			Name: "Hysteria2", Protocol: model.ProtoHysteria2, Core: "sing-box",
			Transport: "udp", Port: m.State.Hysteria2Port, Listen: "::",
			SNI: m.State.Domain, CertPath: cert,
		},
		{
			Name: "TUIC v5", Protocol: model.ProtoTUIC, Core: "sing-box",
			Transport: "udp", Port: m.State.TUICPort, Listen: "::",
			SNI: m.State.Domain, CertPath: cert,
		},
	}
}

// --- per-protocol check orchestration ----------------------------------------

func (m *Monitor) checkProtocol(ctx context.Context, in discoveredInbound, xrayActive, singActive bool) ProtocolReport {
	progStart := time.Now()
	rep := ProtocolReport{
		Name:       in.Name,
		Protocol:   in.Protocol,
		Core:       in.Core,
		Transport:  in.Transport,
		Port:       in.Port,
		ListenAddr: in.Listen,
		SNI:        in.SNI,
		CheckedAt:  time.Now(),
		Status:     StatusHealthy,
	}

	coreActive := singActive
	cfgPath := m.SingBoxConfigPath
	if in.Core == "xray" {
		coreActive = xrayActive
		cfgPath = m.XrayConfigPath
	}

	// 1. Service / process. Service status is informational: a DOWN verdict is
	// driven by the port/reachability probes (authoritative), while an inactive
	// service is at most DEGRADED so a manually-started core does not cause a
	// false-alarm DOWN. See the final reconcile step below.
	rep.Probes = append(rep.Probes, ProbeResult{
		Name: "service", OK: coreActive,
		Detail: ternary(coreActive, "active", "inactive/dead"),
	})

	// 2. Local port binding. A definitively unbound port is the primary DOWN
	// signal: nothing is listening regardless of what systemd claims.
	bound, bindDetail := portBound(in.Port, in.Transport)
	rep.Probes = append(rep.Probes, ProbeResult{Name: "port_bind", OK: bound, Detail: bindDetail})
	if !bound {
		rep.Status = worst(rep.Status, StatusDown)
	}

	// 3. Reachability (transport specific).
	if in.Transport == "tcp" {
		m.checkTCP(ctx, &rep, in)
	} else {
		m.checkQUIC(ctx, &rep, in)
	}

	// 4. Certificate (file-based for UDP, dial-based already done for TCP).
	if in.Transport == "udp" && in.CertPath != "" {
		m.checkCertFile(&rep, in.CertPath)
	}

	// 5. Config validity.
	cfgOK, cfgDetail := configValid(cfgPath)
	rep.ConfigValid = cfgOK
	rep.Probes = append(rep.Probes, ProbeResult{Name: "config", OK: cfgOK, Detail: cfgDetail})
	if !cfgOK {
		rep.Status = worst(rep.Status, StatusDegraded)
	}

	// 6. Recent log errors.
	logErrs, logDetail := recentLogErrors(in.Core, time.Hour)
	rep.LogErrors = logErrs
	rep.Probes = append(rep.Probes, ProbeResult{Name: "log_errors", OK: logErrs == 0, Detail: logDetail})
	if logErrs >= 10 {
		rep.Status = worst(rep.Status, StatusDegraded)
	}

	// 7. Reconcile service status: an inactive service is only DEGRADED when
	// the port is actually serving (manually launched / systemd mismatch); it
	// remains DOWN when the port is dead (already set by the port check above).
	if !coreActive && bound {
		rep.Status = worst(rep.Status, StatusDegraded)
	}

	rep.DurationMs = time.Since(progStart).Milliseconds()
	return rep
}

func (m *Monitor) checkTCP(ctx context.Context, rep *ProtocolReport, in discoveredInbound) {
	// 3a. TCP reach.
	ok, lat, det := tcpReach(ctx, m.LocalHost, in.Port, 2)
	rep.Probes = append(rep.Probes, ProbeResult{Name: "tcp_reach", OK: ok, Detail: det, LatencyMs: lat})
	if !ok {
		rep.Status = worst(rep.Status, StatusDegraded)
		return
	}
	// 3b. Fallback target liveness (443 -> internal Trojan 31296).
	if in.FallbackTarget != 0 {
		fbOK, fbLat, fbDet := tcpReach(ctx, m.LocalHost, in.FallbackTarget, 1)
		rep.Probes = append(rep.Probes, ProbeResult{
			Name: "fallback_target", OK: fbOK, Detail: fmt.Sprintf("127.0.0.1:%d %s", in.FallbackTarget, fbDet), LatencyMs: fbLat,
		})
		if !fbOK {
			rep.Status = worst(rep.Status, StatusDegraded)
		}
	}
	// 3c. TLS handshake + cert for tls / reality security.
	if in.Protocol == model.ProtoVLESSTLS || in.Protocol == model.ProtoVLESSReality {
		sni := in.SNI
		if sni == "" {
			sni = m.State.Domain
		}
		tlsOK, issuer, subject, days, tdet := tlsInspect(ctx, m.LocalHost, in.Port, sni)
		rep.Probes = append(rep.Probes, ProbeResult{Name: "tls_handshake", OK: tlsOK, Detail: tdet})
		rep.CertIssuer = issuer
		rep.CertSubject = subject
		rep.CertDaysLeft = days
		if !tlsOK {
			rep.Status = worst(rep.Status, StatusDegraded)
		} else {
			rep.Status = worst(rep.Status, certStatus(days, m.CertWarnDays))
		}
	}
}

func (m *Monitor) checkQUIC(ctx context.Context, rep *ProtocolReport, in discoveredInbound) {
	ok, vn, lat, det := quicReach(ctx, m.LocalHost, in.Port, 3)
	rep.Probes = append(rep.Probes, ProbeResult{Name: "quic_probe", OK: ok, Detail: det, LatencyMs: lat})
	if ok {
		return
	}
	// A definitively closed port (ICMP refused) is a hard DOWN. A silent
	// timeout ("inconclusive") is ambiguous — if the port IS bound and the
	// service IS active, the server is very likely fine (some QUIC stacks
	// silently drop unknown-version probes). In that case we don't degrade.
	if strings.Contains(det, "refused") || strings.Contains(det, "unreachable") {
		rep.Status = worst(rep.Status, StatusDown)
		return
	}
	// Inconclusive: only degrade if port_bind or service already failed.
	for _, pr := range rep.Probes {
		if (pr.Name == "port_bind" || pr.Name == "service") && !pr.OK {
			rep.Status = worst(rep.Status, StatusDegraded)
			return
		}
	}
	// Port bound + service active + inconclusive QUIC = healthy.
	_ = vn
}

func (m *Monitor) checkCertFile(rep *ProtocolReport, certPath string) {
	issuer, subject, days, err := certFileInspect(certPath)
	if err != nil {
		rep.Probes = append(rep.Probes, ProbeResult{Name: "cert_file", OK: false, Detail: fmt.Sprintf("read cert: %v", err)})
		rep.Status = worst(rep.Status, StatusDegraded)
		return
	}
	rep.Probes = append(rep.Probes, ProbeResult{
		Name: "cert_file", OK: true, Detail: fmt.Sprintf("%s, %d days left", issuer, days),
	})
	rep.CertIssuer = issuer
	rep.CertSubject = subject
	rep.CertDaysLeft = days
	rep.Status = worst(rep.Status, certStatus(days, m.CertWarnDays))
}

func certStatus(days, warnDays int) Status {
	switch {
	case days <= 0:
		return StatusDown
	case days <= warnDays:
		return StatusDegraded
	default:
		return StatusHealthy
	}
}

// --- small helpers -----------------------------------------------------------

func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	case string:
		i, _ := parseStrInt(n)
		return i
	}
	return 0
}

func parseStrInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

func firstServerName(ss map[string]any) string {
	rs, _ := ss["realitySettings"].(map[string]any)
	if rs == nil {
		return ""
	}
	if names, ok := rs["serverNames"].([]any); ok && len(names) > 0 {
		if s, ok := names[0].(string); ok {
			return s
		}
	}
	if dest, ok := rs["dest"].(string); ok {
		return strings.Split(dest, ":")[0]
	}
	return ""
}

func ternary(b bool, a, c string) string {
	if b {
		return a
	}
	return c
}
