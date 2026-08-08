package core

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"v2ray-agent/pkg/util"
)

const xrayService = `[Unit]
Description=Xray Service
After=network.target nss-lookup.target

[Service]
User=root
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_BIND_SERVICE
NoNewPrivileges=true
ExecStart=/etc/v2ray-agent/xray/xray run -c /etc/v2ray-agent/xray/config.json
Restart=on-failure
RestartPreventExitStatus=23
LimitNPROC=10000
LimitNOFILE=1000000

[Install]
WantedBy=multi-user.target
`

const singboxService = `[Unit]
Description=sing-box service
Documentation=https://sing-box.sagernet.org
After=network.target nss-lookup.target

[Service]
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_SYS_PTRACE CAP_DAC_READ_SEARCH
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_SYS_PTRACE CAP_DAC_READ_SEARCH
ExecStart=/etc/v2ray-agent/sing-box/sing-box run -c /etc/v2ray-agent/sing-box/config.json
Restart=on-failure
RestartSec=10s
LimitNOFILE=infinity

[Install]
WantedBy=multi-user.target
`

// SetupServices registers systemd service units
func SetupServices() error {
	_ = os.WriteFile("/etc/systemd/system/xray.service", []byte(xrayService), 0644)
	_ = os.WriteFile("/etc/systemd/system/sing-box.service", []byte(singboxService), 0644)
	_ = exec.Command("systemctl", "daemon-reload").Run()
	return nil
}

// RestartService restarts and enables systemd service
func RestartService(serviceName string) error {
	_ = exec.Command("systemctl", "enable", serviceName).Run()
	cmd := exec.Command("systemctl", "restart", serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restart %s: %w", serviceName, err)
	}
	util.PrintSuccess(fmt.Sprintf("服务 %s 运行正常", serviceName))
	return nil
}

// IsServiceRunning checks if systemd service is active
func IsServiceRunning(serviceName string) bool {
	out, err := exec.Command("systemctl", "is-active", serviceName).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "active"
}
