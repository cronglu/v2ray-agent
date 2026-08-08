package system

import (
	"fmt"
	"os/exec"
	"strings"
	"v2ray-agent/pkg/util"
)

// EnsureFirewallPersistence installs iptables-persistent on Debian/Ubuntu if missing
func EnsureFirewallPersistence(distro Distro) {
	if distro == DistroDebian || distro == DistroUbuntu {
		// Check if netfilter-persistent is available
		if _, err := exec.LookPath("netfilter-persistent"); err != nil {
			util.PrintInfo("正在安装 iptables-persistent 防火墙持久化组件...")
			// Non-interactive install
			cmd := exec.Command("bash", "-c", "DEBIAN_FRONTEND=noninteractive apt-get -y install iptables-persistent")
			_ = cmd.Run()
			_ = exec.Command("systemctl", "enable", "netfilter-persistent").Run()
		}
	}
}

// AllowPort opens a single TCP or UDP port
func AllowPort(port int, protocol string) error {
	// 1. Try ufw if active
	if _, err := exec.LookPath("ufw"); err == nil {
		_ = exec.Command("ufw", "allow", fmt.Sprintf("%d/%s", port, protocol)).Run()
	}

	// 2. Try firewall-cmd (CentOS/RHEL)
	if _, err := exec.LookPath("firewall-cmd"); err == nil {
		_ = exec.Command("firewall-cmd", "--permanent", fmt.Sprintf("--add-port=%d/%s", port, protocol)).Run()
		_ = exec.Command("firewall-cmd", "--reload").Run()
	}

	// 3. Fallback to standard iptables
	if _, err := exec.LookPath("iptables"); err == nil {
		comment := fmt.Sprintf("xraycli_allow_%d_%s", port, protocol)
		// Check if rule already exists
		out, _ := exec.Command("iptables", "-L", "INPUT", "-n").Output()
		if !strings.Contains(string(out), fmt.Sprintf("dpt:%d", port)) {
			_ = exec.Command("iptables", "-I", "INPUT", "-p", protocol, "--dport", fmt.Sprintf("%d", port), "-m", "comment", "--comment", comment, "-j", "ACCEPT").Run()
			SaveIptablesRules()
		}
	}
	return nil
}

// AddPortHopping adds UDP port hopping forwarding (e.g. 30000:40000 -> targetPort)
func AddPortHopping(protocol string, portStart, portEnd, targetPort int, distro Distro) error {
	EnsureFirewallPersistence(distro)

	comment := fmt.Sprintf("xraycli_%s_portHopping", protocol)

	// Clean up old rules first
	DeletePortHopping(protocol)

	if distro == DistroCentOS {
		_ = exec.Command("firewall-cmd", "--permanent", "--add-masquerade").Run()
		_ = exec.Command("firewall-cmd", "--permanent", fmt.Sprintf("--add-forward-port=port=%d-%d:proto=udp:toport=%d", portStart, portEnd, targetPort)).Run()
		_ = exec.Command("firewall-cmd", "--reload").Run()
	} else {
		// iptables NAT PREROUTING
		cmd := exec.Command("iptables", "-t", "nat", "-A", "PREROUTING", "-p", "udp",
			"--dport", fmt.Sprintf("%d:%d", portStart, portEnd),
			"-m", "comment", "--comment", comment,
			"-j", "DNAT", "--to-destination", fmt.Sprintf(":%d", targetPort))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("iptables port hopping failed: %w", err)
		}
		SaveIptablesRules()
	}

	util.PrintSuccess(fmt.Sprintf("UDP 端口跳跃添加成功: [%d - %d] ──> %d", portStart, portEnd, targetPort))
	return nil
}

// DeletePortHopping removes port hopping rules by comment tag
func DeletePortHopping(protocol string) {
	comment := fmt.Sprintf("xraycli_%s_portHopping", protocol)
	for i := 0; i < 10; i++ {
		out, err := exec.Command("iptables", "-t", "nat", "-L", "PREROUTING", "--line-numbers", "-n").Output()
		if err != nil {
			break
		}
		lines := strings.Split(string(out), "\n")
		found := false
		for _, line := range lines {
			if strings.Contains(line, comment) {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					lineNum := fields[0]
					_ = exec.Command("iptables", "-t", "nat", "-D", "PREROUTING", lineNum).Run()
					found = true
					break
				}
			}
		}
		if !found {
			break
		}
	}
	SaveIptablesRules()
}

// SaveIptablesRules persists iptables rules across system reboots
func SaveIptablesRules() {
	if _, err := exec.LookPath("netfilter-persistent"); err == nil {
		_ = exec.Command("netfilter-persistent", "save").Run()
	}
}
