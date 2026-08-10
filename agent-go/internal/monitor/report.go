package monitor

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"v2ray-agent/internal/model"
	"v2ray-agent/pkg/util"
)

// Status is the aggregate health of a protocol or the whole report.
type Status string

const (
	StatusHealthy  Status = "healthy"
	StatusDegraded Status = "degraded"
	StatusDown     Status = "down"
	StatusUnknown  Status = "unknown"
)

func (s Status) Icon() string {
	switch s {
	case StatusHealthy:
		return "🟢"
	case StatusDegraded:
		return "🟡"
	case StatusDown:
		return "🔴"
	default:
		return "⚪"
	}
}

func (s Status) Color() string {
	switch s {
	case StatusHealthy:
		return util.ColorGreen
	case StatusDegraded:
		return util.ColorYellow
	case StatusDown:
		return util.ColorRed
	default:
		return util.ColorCyan
	}
}

// ProbeResult is the outcome of a single atomic check.
type ProbeResult struct {
	Name      string `json:"name"`
	OK        bool   `json:"ok"`
	Detail    string `json:"detail,omitempty"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
}

// ProtocolReport is the full health picture of one protocol inbound.
type ProtocolReport struct {
	Name          string              `json:"name"`
	Protocol      model.ProtocolType `json:"protocol"`
	Core          string              `json:"core"`
	Transport     string              `json:"transport"`
	Port          int                 `json:"port"`
	ListenAddr    string              `json:"listen_addr"`
	SNI           string              `json:"sni,omitempty"`
	Status        Status              `json:"status"`
	Probes        []ProbeResult       `json:"probes"`
	CertDaysLeft  int                 `json:"cert_days_left,omitempty"`
	CertIssuer    string              `json:"cert_issuer,omitempty"`
	CertSubject   string              `json:"cert_subject,omitempty"`
	ConfigValid   bool                `json:"config_valid"`
	LogErrors     int                 `json:"log_errors"`
	CheckedAt     time.Time           `json:"checked_at"`
	DurationMs    int64               `json:"duration_ms"`
}

// IsDown reports a hard failure.
func (p *ProtocolReport) IsDown() bool { return p.Status == StatusDown }

// Report aggregates all protocol checks plus host-level facts.
type Report struct {
	Overall       Status             `json:"overall"`
	Hostname      string             `json:"hostname"`
	PublicIP      string             `json:"public_ip"`
	Kernel        string             `json:"kernel,omitempty"`
	XrayActive    bool               `json:"xray_active"`
	SingBoxActive bool               `json:"sing_box_active"`
	Protocols     []ProtocolReport   `json:"protocols"`
	CheckedAt     time.Time          `json:"checked_at"`
	DurationMs    int64              `json:"duration_ms"`
}

// ExitCode maps overall status to a process exit code for scripting.
//
//	0 = healthy, 1 = degraded, 2 = down, 3 = unknown.
func (r *Report) ExitCode() int {
	switch r.Overall {
	case StatusHealthy:
		return 0
	case StatusDegraded:
		return 1
	case StatusDown:
		return 2
	default:
		return 3
	}
}

func worst(s, t Status) Status {
	rank := map[Status]int{StatusHealthy: 0, StatusUnknown: 1, StatusDegraded: 2, StatusDown: 3}
	if rank[s] >= rank[t] {
		return s
	}
	return t
}

// Aggregate computes the overall status from the protocol list.
func (r *Report) Aggregate() Status {
	if len(r.Protocols) == 0 {
		return StatusUnknown
	}
	overall := StatusHealthy
	for i := range r.Protocols {
		overall = worst(overall, r.Protocols[i].Status)
	}
	return overall
}

// JSON serializes the report to compact JSON.
func (r *Report) JSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

const divider = "=============================================================="

// Render produces a colored terminal report.
func (r *Report) Render() string {
	var sb strings.Builder
	sb.WriteString("\n")
	writeDivider(&sb)
	fmt.Fprintf(&sb, "%s%s  xraycli 协议健康监控报告%s\n", util.ColorCyan, util.ColorBold, util.ColorReset)
	writeDivider(&sb)
	fmt.Fprintf(&sb, "  主机: %s  公网 IP: %s\n", r.Hostname, r.PublicIP)
	fmt.Fprintf(&sb, "  Xray: %s  Sing-box: %s  检测时间: %s\n",
		serviceBadge(r.XrayActive), serviceBadge(r.SingBoxActive),
		r.CheckedAt.Format("2006-01-02 15:04:05"))
	writeDivider(&sb)

	for i := range r.Protocols {
		p := &r.Protocols[i]
		renderProtocol(&sb, p)
	}

	writeDivider(&sb)
	overall := r.Overall
	fmt.Fprintf(&sb, "  总体状态: %s%s%s  耗时: %dms\n",
		overall.Color(), overall.Icon()+" "+string(overall), util.ColorReset, r.DurationMs)
	writeDivider(&sb)
	return sb.String()
}

func writeDivider(sb *strings.Builder) {
	fmt.Fprintf(sb, "%s%s%s\n", util.ColorYellow, divider, util.ColorReset)
}

func renderProtocol(sb *strings.Builder, p *ProtocolReport) {
	fmt.Fprintf(sb, "\n  %s %s%s%s  [%s/%s] %s:%d\n",
		p.Status.Icon(), p.Status.Color(), p.Name, util.ColorReset,
		p.Core, p.Transport, coalesce(p.ListenAddr, "*"), p.Port)
	if p.SNI != "" {
		fmt.Fprintf(sb, "      SNI: %s\n", p.SNI)
	}
	for _, pr := range p.Probes {
		mark := util.ColorGreen + "✓" + util.ColorReset
		if !pr.OK {
			mark = util.ColorRed + "✗" + util.ColorReset
		}
		lat := ""
		if pr.LatencyMs > 0 {
			lat = fmt.Sprintf(" (%dms)", pr.LatencyMs)
		}
		fmt.Fprintf(sb, "      %s %-14s %s%s\n", mark, pr.Name, pr.Detail, lat)
	}
	if p.CertDaysLeft > 0 || p.CertIssuer != "" {
		warn := ""
		if p.CertDaysLeft <= 14 && p.CertDaysLeft > 0 {
			warn = " " + util.ColorYellow + "(即将到期!)" + util.ColorReset
		} else if p.CertDaysLeft <= 0 {
			warn = " " + util.ColorRed + "(已到期!)" + util.ColorReset
		}
		fmt.Fprintf(sb, "      证书: %s  剩余 %d 天%s\n", coalesce(p.CertIssuer, "N/A"), p.CertDaysLeft, warn)
	}
	if p.LogErrors > 0 {
		fmt.Fprintf(sb, "      %s 近 1h 日志错误: %d%s\n", util.ColorYellow, p.LogErrors, util.ColorReset)
	}
}

func serviceBadge(active bool) string {
	if active {
		return util.ColorGreen + "🟢 active" + util.ColorReset
	}
	return util.ColorRed + "🔴 inactive" + util.ColorReset
}

func coalesce(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
