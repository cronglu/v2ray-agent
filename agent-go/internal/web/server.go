package web

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"v2ray-agent/internal/model"
	"v2ray-agent/internal/subscription"
	"v2ray-agent/pkg/util"
)

//go:embed static/*
var staticFS embed.FS

// WebServer handles camouflage site and dynamic smart subscriptions
type WebServer struct {
	State *model.GlobalNodeState
}

// StartCamouflageServer starts the local web server on port 31300
func StartCamouflageServer(state *model.GlobalNodeState, port int) error {
	subFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		return err
	}

	mux := http.NewServeMux()

	// 1. Camouflage website
	fileServer := http.FileServer(http.FS(subFS))
	mux.Handle("/", fileServer)

	// 2. Smart Subscription Handler (/s/)
	mux.HandleFunc("/s/", func(w http.ResponseWriter, r *http.Request) {
		ua := strings.ToLower(r.UserAgent())
		path := r.URL.Path

		// Return Clash.Meta YAML if requested by Clash/Mihomo or path contains clash
		if strings.Contains(ua, "clash") || strings.Contains(ua, "mihomo") || strings.Contains(path, "clash") {
			w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
			w.Header().Set("Subscription-Userinfo", "upload=0; download=0; total=107374182400; expire=0")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(subscription.GenerateClashMetaConfig(state)))
			return
		}

		// Return Sing-box JSON if requested by Sing-box or path contains singbox
		if strings.Contains(ua, "sing-box") || strings.Contains(path, "singbox") {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			data, _ := subscription.GenerateSingBoxClientConfig(state)
			_, _ = w.Write(data)
			return
		}

		// Universal Plain URIs for Shadowrocket / v2rayN / browser
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		uris := subscription.GenerateUniversalURIs(state)
		_, _ = w.Write([]byte(strings.Join(uris, "\n")))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", port),
		Handler: mux,
	}

	util.PrintInfo(fmt.Sprintf("Go 内置 Web 伪装站与智能订阅引擎启动 [127.0.0.1:%d]", port))
	return server.ListenAndServe()
}
