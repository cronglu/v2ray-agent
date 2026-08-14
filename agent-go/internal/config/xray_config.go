package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"v2ray-agent/internal/model"
)

// GenerateStandardXrayConfig generates official standard Xray-core config.json
func GenerateStandardXrayConfig(state *model.GlobalNodeState) ([]byte, error) {
	certFile := fmt.Sprintf("/etc/v2ray-agent/tls/%s.crt", state.Domain)
	keyFile := fmt.Sprintf("/etc/v2ray-agent/tls/%s.key", state.Domain)

	// Construct clients
	vlessClients := make([]map[string]any, 0)
	trojanClients := make([]map[string]any, 0)
	for _, u := range state.Users {
		vlessClients = append(vlessClients, map[string]any{
			"id":   u.UUID,
			"flow": "xtls-rprx-vision",
		})
		trojanClients = append(trojanClients, map[string]any{
			"password": u.Password,
		})
	}

	inbounds := []map[string]any{
		// 1. 443 Port VLESS TCP Vision with Fallback to Trojan 31296
		{
			"port":     443,
			"protocol": "vless",
			"tag":      "VLESS_TCP_TLS",
			"settings": map[string]any{
				"clients":    vlessClients,
				"decryption": "none",
				// CRITICAL FIX: Clean fallback to Trojan without ALPN:h2 conflict
				"fallbacks": []map[string]any{
					{
						"dest": 31296,
						"xver": 1,
					},
				},
			},
			"streamSettings": map[string]any{
				"network":  "tcp",
				"security": "tls",
				"tlsSettings": map[string]any{
					"certificates": []map[string]any{
						{
							"certificateFile": certFile,
							"keyFile":         keyFile,
						},
					},
				},
			},
		},
		// 2. 31296 Port Trojan (Fallback from VLESS)
		{
			"port":     31296,
			"listen":   "127.0.0.1",
			"protocol": "trojan",
			"tag":      "trojanTCP",
			"settings": map[string]any{
				"clients": trojanClients,
				"fallbacks": []map[string]any{
					{
						"dest": "31300", // Falls back to Go Web Camouflage Server
					},
				},
			},
			"streamSettings": map[string]any{
				"network":  "tcp",
				"security": "none",
			},
		},
		// 3. VLESS Reality Inbound
		{
			"port":     15393,
			"protocol": "vless",
			"tag":      "VLESS_REALITY",
			"settings": map[string]any{
				"clients":    vlessClients,
				"decryption": "none",
			},
			"streamSettings": map[string]any{
				"network":  "tcp",
				"security": "reality",
				"realitySettings": map[string]any{
					"show":        false,
					"dest":        fmt.Sprintf("%s:443", state.RealityServerName),
					"xver":        0,
					"serverNames": []string{state.RealityServerName},
					"privateKey":  state.RealityPrivateKey,
					"shortIds":    []string{state.RealityShortID},
				},
			},
		},
	}

	outbounds := []map[string]any{
		{
			"protocol": "freedom",
			"tag":      "direct",
		},
		{
			"protocol": "blackhole",
			"tag":      "block",
		},
	}

	// Add WARP WireGuard outbound if enabled
	if state.WARPEnabled && state.WARPPrivateKey != "" {
		peer := map[string]any{
			"publicKey": "bmXOC+F1FxEMF9dyiK2H5/1SUtzH0JuVo51h2wPfgyo=",
			"endpoint":  "162.159.192.1:2408",
		}
		if reserved := parseReserved(state.WARPReserved); reserved != nil {
			peer["reserved"] = reserved
		}
		outbounds = append(outbounds, map[string]any{
			"protocol": "wireguard",
			"tag":      "warp_out",
			"settings": map[string]any{
				"secretKey": state.WARPPrivateKey,
				"address":   []string{state.WARPAddress},
				"peers":     []map[string]any{peer},
			},
		})
	}

	// Smart Routing Rules
	rules := []map[string]any{
		// Block BT and ads
		{
			"type":        "field",
			"protocol":    []string{"bittorrent"},
			"outboundTag": "block",
		},
	}

	if state.WARPEnabled {
		// Route Google AI / Gemini / OpenAI to WARP Clean IP
		rules = append(rules, map[string]any{
			"type": "field",
			"domain": []string{
				"geosite:google",
				"geosite:openai",
				"domain:googleai.dev",
				"domain:googleusercontent.com",
				"domain:googleapis.com",
				"domain:gstatic.com",
			},
			"outboundTag": "warp_out",
		})
	}

	// China Direct Route
	rules = append(rules, map[string]any{
		"type":        "field",
		"ip":          []string{"geoip:cn", "geoip:private"},
		"outboundTag": "direct",
	})

	configMap := map[string]any{
		"log": map[string]any{
			"loglevel": "warning",
		},
		"inbounds":  inbounds,
		"outbounds": outbounds,
		"routing": map[string]any{
			"domainStrategy": "IPIfNonMatch",
			"rules":          rules,
		},
	}

	return json.MarshalIndent(configMap, "", "  ")
}

// SaveXrayConfig writes standard Xray config.json to disk
func SaveXrayConfig(state *model.GlobalNodeState) error {
	data, err := GenerateStandardXrayConfig(state)
	if err != nil {
		return err
	}
	targetDir := "/etc/v2ray-agent/xray"
	_ = os.MkdirAll(targetDir, 0755)
	return os.WriteFile(filepath.Join(targetDir, "config.json"), data, 0644)
}
