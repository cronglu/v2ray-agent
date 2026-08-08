package subscription

import (
	"fmt"
	"strings"
	"v2ray-agent/internal/model"
)

// GenerateClashMetaConfig exports standard Clash.Meta / Mihomo YAML subscription
func GenerateClashMetaConfig(state *model.GlobalNodeState) string {
	var sb strings.Builder

	sb.WriteString("port: 7890\n")
	sb.WriteString("socks-port: 7891\n")
	sb.WriteString("allow-lan: true\n")
	sb.WriteString("mode: rule\n")
	sb.WriteString("log-level: info\n")
	sb.WriteString("proxies:\n")

	for _, user := range state.Users {
		// 1. VLESS Reality Vision
		sb.WriteString(fmt.Sprintf("  - name: \"%s-VLESS_Reality\"\n", user.Email))
		sb.WriteString("    type: vless\n")
		sb.WriteString(fmt.Sprintf("    server: %s\n", state.PublicIP))
		sb.WriteString("    port: 15393\n")
		sb.WriteString(fmt.Sprintf("    uuid: %s\n", user.UUID))
		sb.WriteString("    network: tcp\n")
		sb.WriteString("    tls: true\n")
		sb.WriteString("    udp: true\n")
		sb.WriteString("    flow: xtls-rprx-vision\n")
		sb.WriteString(fmt.Sprintf("    servername: %s\n", state.RealityServerName))
		sb.WriteString("    reality-opts:\n")
		sb.WriteString(fmt.Sprintf("      public-key: %s\n", state.RealityPublicKey))
		sb.WriteString(fmt.Sprintf("      short-id: %s\n", state.RealityShortID))
		sb.WriteString("    client-fingerprint: chrome\n\n")

		// 2. Hysteria2 (with Port Hopping support)
		sb.WriteString(fmt.Sprintf("  - name: \"%s-Hysteria2\"\n", user.Email))
		sb.WriteString("    type: hysteria2\n")
		sb.WriteString(fmt.Sprintf("    server: %s\n", state.Domain))
		if state.Hysteria2PortHop != "" {
			sb.WriteString(fmt.Sprintf("    ports: %s\n", state.Hysteria2PortHop))
		} else {
			sb.WriteString(fmt.Sprintf("    port: %d\n", state.Hysteria2Port))
		}
		sb.WriteString(fmt.Sprintf("    password: %s\n", user.Password))
		sb.WriteString("    alpn:\n      - h3\n")
		sb.WriteString(fmt.Sprintf("    sni: %s\n", state.Domain))
		sb.WriteString(fmt.Sprintf("    up: \"%d Mbps\"\n", state.Hysteria2UpMbps))
		sb.WriteString(fmt.Sprintf("    down: \"%d Mbps\"\n\n", state.Hysteria2DownMbps))

		// 3. TUIC v5 (CRITICAL FIX: Valid SNI and NO disable-sni)
		sb.WriteString(fmt.Sprintf("  - name: \"%s-TUIC_v5\"\n", user.Email))
		sb.WriteString("    type: tuic\n")
		sb.WriteString(fmt.Sprintf("    server: %s\n", state.Domain))
		sb.WriteString(fmt.Sprintf("    port: %d\n", state.TUICPort))
		sb.WriteString(fmt.Sprintf("    uuid: %s\n", user.UUID))
		sb.WriteString(fmt.Sprintf("    password: %s\n", user.Password))
		sb.WriteString("    alpn:\n      - h3\n")
		sb.WriteString("    congestion-controller: bbr\n")
		sb.WriteString("    reduce-rtt: true\n")
		sb.WriteString(fmt.Sprintf("    sni: %s\n\n", state.Domain))

		// 4. Trojan+TLS (443 port)
		sb.WriteString(fmt.Sprintf("  - name: \"%s-Trojan_TLS\"\n", user.Email))
		sb.WriteString("    type: trojan\n")
		sb.WriteString(fmt.Sprintf("    server: %s\n", state.Domain))
		sb.WriteString("    port: 443\n")
		sb.WriteString(fmt.Sprintf("    password: %s\n", user.Password))
		sb.WriteString("    udp: true\n")
		sb.WriteString(fmt.Sprintf("    sni: %s\n", state.Domain))
		sb.WriteString("    client-fingerprint: chrome\n\n")
	}

	// Standard Proxy Groups & Rules
	sb.WriteString("proxy-groups:\n")
	sb.WriteString("  - name: 🚀 节点选择\n    type: select\n    proxies:\n      - ♻️ 自动选择\n")
	for _, user := range state.Users {
		sb.WriteString(fmt.Sprintf("      - \"%s-VLESS_Reality\"\n", user.Email))
		sb.WriteString(fmt.Sprintf("      - \"%s-Hysteria2\"\n", user.Email))
		sb.WriteString(fmt.Sprintf("      - \"%s-TUIC_v5\"\n", user.Email))
		sb.WriteString(fmt.Sprintf("      - \"%s-Trojan_TLS\"\n", user.Email))
	}
	sb.WriteString("  - name: ♻️ 自动选择\n    type: url-test\n    url: http://www.gstatic.com/generate_204\n    interval: 300\n    proxies:\n")
	for _, user := range state.Users {
		sb.WriteString(fmt.Sprintf("      - \"%s-VLESS_Reality\"\n", user.Email))
		sb.WriteString(fmt.Sprintf("      - \"%s-Hysteria2\"\n", user.Email))
		sb.WriteString(fmt.Sprintf("      - \"%s-TUIC_v5\"\n", user.Email))
	}

	sb.WriteString("\nrules:\n")
	sb.WriteString("  - GEOIP,CN,DIRECT\n")
	sb.WriteString("  - MATCH,🚀 节点选择\n")

	return sb.String()
}
