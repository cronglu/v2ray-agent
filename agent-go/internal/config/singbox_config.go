package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

	// Add WARP WireGuard outbound if configured
	if state.WARPEnabled && state.WARPPrivateKey != "" {
		outbounds = append(outbounds, map[string]any{
			"type":             "wireguard",
			"tag":              "warp-out",
			"server":           "162.159.192.1",
			"server_port":      2408,
			"system_interface": false,
			"interface_name":   "warp",
			"local_address":    []string{state.WARPAddress},
			"private_key":      state.WARPPrivateKey,
			"peer_public_key":  "bmXOC+F1FxEMF9dyiK2H5/1SUtzH0JuVo51h2wPfgyo=",
		})
	}

	routeRules := []map[string]any{
		{
			"geoip":    []string{"cn", "private"},
			"outbound": "direct",
		},
	}

	if state.WARPEnabled {
		routeRules = append([]map[string]any{
			{
				"geosite":       []string{"google", "openai"},
				"domain_suffix": []string{"googleai.dev", "googleusercontent.com", "googleapis.com"},
				"outbound":      "warp-out",
			},
		}, routeRules...)
	}

	configMap := map[string]any{
		"log": map[string]any{
			"level":     "warn",
			"timestamp": true,
		},
		"inbounds":  inbounds,
		"outbounds": outbounds,
		"route": map[string]any{
			"rules":                  routeRules,
			"auto_detect_interface": true,
		},
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
