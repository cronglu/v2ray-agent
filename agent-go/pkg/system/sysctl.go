package system

import (
	"fmt"
	"os"
	"os/exec"
	"v2ray-agent/pkg/util"
)

const sysctlConfPath = "/etc/sysctl.d/99-vps-optimization.conf"

const sysctlConfigContent = `# Managed by xraycli - High Performance UDP & TCP Optimization
# 16MB UDP Buffer for Hysteria2 / TUIC / WireGuard high throughput
net.core.rmem_max = 16777216
net.core.wmem_max = 16777216
net.core.rmem_default = 262144
net.core.wmem_default = 262144
net.ipv4.udp_rmem_min = 16384
net.ipv4.udp_wmem_min = 16384

# TCP BBR Congestion Control & FQ
net.core.default_qdisc = fq
net.ipv4.tcp_congestion_control = bbr
net.ipv4.tcp_fastopen = 3

# Connection Queue Optimization
net.core.somaxconn = 32768
net.core.netdev_max_backlog = 16384
net.ipv4.tcp_max_syn_backlog = 8192
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_fin_timeout = 15
`

// OptimizeKernel applies sysctl parameters for 16MB UDP buffers and BBR
func OptimizeKernel() error {
	util.PrintInfo("正在应用 Linux 内核 16MB UDP 缓冲区与 BBR 拥塞控制调优...")

	// Write persistent sysctl file
	if err := os.WriteFile(sysctlConfPath, []byte(sysctlConfigContent), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", sysctlConfPath, err)
	}

	// Apply immediately via sysctl --system or sysctl -p
	cmd := exec.Command("sysctl", "--system")
	_ = cmd.Run()

	util.PrintSuccess("内核网络优化已生效 (net.core.rmem_max=16MB, BBR+FQ 拥塞控制)")
	return nil
}
