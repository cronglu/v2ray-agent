package subscription

import (
	"encoding/json"
	"fmt"
	"v2ray-agent/internal/model"
)

// GenerateSingBoxClientConfig exports clean standard sing-box client JSON
func GenerateSingBoxClientConfig(state *model.GlobalNodeState) ([]byte, error) {
	outbounds := make([]map[string]any, 0)
	tags := make([]string, 0)

	for _, user := range state.Users {
		// 1. VLESS Reality
		vlessTag := fmt.Sprintf("%s-VLESS_Reality", user.Email)
		tags = append(tags, vlessTag)
		outbounds = append(outbounds, map[string]any{
			"type":        "vless",
			"tag":         vlessTag,
			"server":      state.PublicIP,
			"server_port": 15393,
			"uuid":        user.UUID,
			"flow":        "xtls-rprx-vision",
			"tls": map[string]any{
				"enabled":     true,
				"server_name": state.RealityServerName,
				"utls": map[string]any{
					"enabled":     true,
					"fingerprint": "chrome",
				},
				"reality": map[string]any{
					"enabled":    true,
					"public_key": state.RealityPublicKey,
					"short_id":   state.RealityShortID,
				},
			},
			"packet_encoding": "xudp",
		})

		// 2. Hysteria2
		hy2Tag := fmt.Sprintf("%s-Hysteria2", user.Email)
		tags = append(tags, hy2Tag)
		outbounds = append(outbounds, map[string]any{
			"type":        "hysteria2",
			"tag":         hy2Tag,
			"server":      state.Domain,
			"server_port": state.Hysteria2Port,
			"password":    user.Password,
			"up_mbps":     state.Hysteria2UpMbps,
			"down_mbps":   state.Hysteria2DownMbps,
			"tls": map[string]any{
				"enabled":     true,
				"server_name": state.Domain,
				"alpn":        []string{"h3"},
			},
		})

		// 3. TUIC v5
		tuicTag := fmt.Sprintf("%s-TUIC_v5", user.Email)
		tags = append(tags, tuicTag)
		outbounds = append(outbounds, map[string]any{
			"type":               "tuic",
			"tag":                tuicTag,
			"server":             state.Domain,
			"server_port":        state.TUICPort,
			"uuid":               user.UUID,
			"password":           user.Password,
			"congestion_control": "bbr",
			"tls": map[string]any{
				"enabled":     true,
				"server_name": state.Domain,
				"alpn":        []string{"h3"},
			},
		})
	}

	// Add selector and direct
	outbounds = append(outbounds,
		map[string]any{
			"type":      "selector",
			"tag":       "select",
			"outbounds": tags,
			"default":   tags[0],
		},
		map[string]any{
			"type": "direct",
			"tag":  "direct",
		},
	)

	config := map[string]any{
		"log": map[string]any{
			"level": "info",
		},
		"outbounds": outbounds,
		"route": map[string]any{
			"rules": []map[string]any{
				{
					"geoip":     []string{"cn", "private"},
					"outbound":  "direct",
				},
			},
			"final": "select",
		},
	}

	return json.MarshalIndent(config, "", "  ")
}
