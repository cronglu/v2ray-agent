package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"v2ray-agent/internal/model"
)

// GenerateStandardSingBoxConfig generates official standard Sing-box config.json
func GenerateStandardSingBoxConfig(state *model.GlobalNodeState) ([]byte, error) {
	certFile := fmt.Sprintf("/etc/v2ray-agent/tls/%s.crt", state.Domain)
	keyFile := fmt.Sprintf("/etc/v2ray-agent/tls/%s.key", state.Domain)

	hy2Users := make([]map[string]any, 0)
	tuicUsers := make([]map[string]any, 0)
	for _, u := range state.Users {
		hy2Users = append(hy2Users, map[string]any{
			"password": u.Password,
			"name":     u.Email,
		})
		tuicUsers = append(tuicUsers, map[string]any{
			"uuid":     u.UUID,
			"password": u.Password,
			"name":     u.Email,
		})
	}

	inbounds := []map[string]any{
		// 1. Hysteria2 Inbound (UDP)
		{
			"type":        "hysteria2",
			"tag":         "hy2-in",
			"listen":      "::",
			"listen_port": state.Hysteria2Port,
			"up_mbps":     state.Hysteria2UpMbps,
			"down_mbps":   state.Hysteria2DownMbps,
			"users":       hy2Users,
			"tls": map[string]any{
				"enabled":          true,
				"server_name":      state.Domain,
				"certificate_path": certFile,
				"key_path":         keyFile,
				"alpn":             []string{"h3"},
			},
		},
		// 2. TUIC v5 Inbound (UDP with Proper ServerName)
		{
			"type":               "tuic",
			"tag":                "tuic-in",
			"listen":             "::",
			"listen_port":        state.TUICPort,
			"users":              tuicUsers,
			"congestion_control": "bbr",
			"tls": map[string]any{
				"enabled":          true,
				"server_name":      state.Domain,
				"certificate_path": certFile,
				"key_path":         keyFile,
				"alpn":             []string{"h3"},
			},
		},
	}

	outbounds := []map[string]any{
		{
			"type": "direct",
			"tag":  "direct",
		},
		{
			"type": "block",
			"tag":  "block",
		},
	}

	// WARP WireGuard is configured as an *endpoint* (sing-box 1.12+ removed
	// the legacy WireGuard outbound and replaced it with the endpoint form).
	var endpoints []map[string]any
	if state.WARPEnabled && state.WARPPrivateKey != "" {
		peer := map[string]any{
			"address":      "162.159.192.1:2408",
			"public_key":  "bmXOC+F1FxEMF9dyiK2H5/1SUtzH0JuVo51h2wPfgyo=",
			"allowed_ips": []string{"0.0.0.0/0"},
		}
		if reserved := parseReserved(state.WARPReserved); reserved != nil {
			peer["reserved"] = reserved
		}
		endpoints = append(endpoints, map[string]any{
			"type":        "wireguard",
			"tag":         "warp-ep",
			"address":     []string{state.WARPAddress},
			"private_key": state.WARPPrivateKey,
			"peers":       []map[string]any{peer},
		})
	}

	// Build WARP routing rule only when the warp-out outbound actually exists
	// (requires a valid private key). This prevents referencing a missing
	// outbound that would crash sing-box on startup.
	warpConfigured := state.WARPEnabled && state.WARPPrivateKey != ""

	// Route rules — sing-box 1.12+ removed geosite/geoip databases, so we use
	// plain domain_suffix rules (no external .db dependency) for WARP routing.
	var routeRules []map[string]any

	if warpConfigured {
		routeRules = append(routeRules, map[string]any{
			"domain_suffix": []string{
				"google.com",
				"google.ai",
				"googleai.dev",
				"googleapis.com",
				"googleusercontent.com",
				"googlevideo.com",
				"gstatic.com",
				"openai.com",
				"chatgpt.com",
				"oaistatic.com",
				"oaiusercontent.com",
			},
			"domain_keyword": []string{"gemini", "bard", "openai"},
			"outbound":      "warp-ep",
		})
	}

	// Private/local network direct (no geoip database needed)
	routeRules = append(routeRules, map[string]any{
		"ip_cidr": []string{
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
			"127.0.0.0/8",
			"100.64.0.0/10",
		},
		"outbound": "direct",
	})

	// Block BitTorrent
	routeRules = append(routeRules, map[string]any{
		"protocol": []string{"bittorrent"},
		"outbound": "block",
	})

	configMap := map[string]any{
		"log": map[string]any{
			"level":     "warn",
			"timestamp": true,
		},
		"inbounds":  inbounds,
		"outbounds": outbounds,
		"route": map[string]any{
			"rules":                 routeRules,
			"auto_detect_interface": true,
		},
	}
	if len(endpoints) > 0 {
		configMap["endpoints"] = endpoints
	}

	return json.MarshalIndent(configMap, "", "  ")
}

// SaveSingBoxConfig writes standard sing-box config.json to disk
func SaveSingBoxConfig(state *model.GlobalNodeState) error {
	data, err := GenerateStandardSingBoxConfig(state)
	if err != nil {
		return err
	}
	targetDir := "/etc/v2ray-agent/sing-box"
	_ = os.MkdirAll(targetDir, 0755)
	return os.WriteFile(filepath.Join(targetDir, "config.json"), data, 0644)
}

// parseReserved parses a comma-separated reserved bytes string (e.g. "127,159,60")
// into a []uint8 for sing-box/xray WireGuard outbound. Returns nil if empty or invalid.
func parseReserved(s string) []int {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []int
	for _, p := range parts {
		p = strings.TrimSpace(p)
		var n int
		if _, err := fmt.Sscanf(p, "%d", &n); err != nil || n < 0 || n > 255 {
			return nil
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}