package util

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

var ipEndpointsV4 = []string{
	"https://api.ipify.org",
	"https://ipinfo.io/ip",
	"https://ifconfig.me/ip",
	"https://icanhazip.com",
	"https://checkip.amazonaws.com",
}

// GetPublicIPv4 retrieves public IPv4 address with fallback across multiple providers
func GetPublicIPv4() string {
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, "tcp4", addr)
			},
		},
	}

	for _, endpoint := range ipEndpointsV4 {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "curl/7.88.1")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err == nil {
				ip := strings.TrimSpace(string(body))
				if parsed := net.ParseIP(ip); parsed != nil && parsed.To4() != nil {
					return ip
				}
			}
		}
	}
	return "127.0.0.1"
}
