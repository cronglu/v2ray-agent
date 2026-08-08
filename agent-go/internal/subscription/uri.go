package subscription

import (
	"fmt"
	"net/url"
	"v2ray-agent/internal/model"
)

// GenerateUniversalURIs generates universal link strings for all active protocols
func GenerateUniversalURIs(state *model.GlobalNodeState) []string {
	var uris []string

	for _, u := range state.Users {
		// 1. VLESS Reality Vision
		vless := fmt.Sprintf("vless://%s@%s:15393?encryption=none&flow=xtls-rprx-vision&security=reality&sni=%s&fp=chrome&pbk=%s&sid=%s&type=tcp#%s-VLESS_Reality",
			u.UUID,
			state.PublicIP,
			state.RealityServerName,
			state.RealityPublicKey,
			state.RealityShortID,
			url.QueryEscape(u.Email),
		)
		uris = append(uris, vless)

		// 2. Hysteria2
		portField := fmt.Sprintf("%d", state.Hysteria2Port)
		mportParam := ""
		if state.Hysteria2PortHop != "" {
			mportParam = fmt.Sprintf("mport=%s&", state.Hysteria2PortHop)
		}
		hy2 := fmt.Sprintf("hysteria2://%s@%s:%s/?%ssni=%s&alpn=h3&insecure=0#%s-Hysteria2",
			u.Password,
			state.Domain,
			portField,
			mportParam,
			state.Domain,
			url.QueryEscape(u.Email),
		)
		uris = append(uris, hy2)

		// 3. TUIC v5
		tuic := fmt.Sprintf("tuic://%s:%s@%s:%d?congestion_control=bbr&alpn=h3&sni=%s&udp_relay_mode=quic&allow_insecure=0#%s-TUIC_v5",
			u.UUID,
			u.Password,
			state.Domain,
			state.TUICPort,
			state.Domain,
			url.QueryEscape(u.Email),
		)
		uris = append(uris, tuic)

		// 4. Trojan TLS
		trojan := fmt.Sprintf("trojan://%s@%s:443?security=tls&sni=%s&type=tcp#%s-Trojan_TLS",
			u.Password,
			state.Domain,
			state.Domain,
			url.QueryEscape(u.Email),
		)
		uris = append(uris, trojan)
	}

	return uris
}
