package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"v2ray-agent/internal/config"
	"v2ray-agent/internal/model"
	pkgConfig "v2ray-agent/pkg/config"
	"v2ray-agent/pkg/core"
	"v2ray-agent/pkg/system"
	"v2ray-agent/pkg/tls"
	"v2ray-agent/pkg/util"
)

// LoadState reads global node state from disk via Viper or returns default
func LoadState() *model.GlobalNodeState {
	viperManager := pkgConfig.NewViperConfigManager()
	state, err := viperManager.LoadConfig()
	if err == nil && len(state.Users) > 0 {
		return state
	}

	// Generate default Reality keys & UUID if missing
	realityKeys, _ := tls.GenerateRealityKeyPair()
	pubIP := util.GetPublicIPv4()

	uuidBytes := make([]byte, 16)
	_, _ = rand.Read(uuidBytes)
	defaultUUID := fmt.Sprintf("%x-%x-%x-%x-%x", uuidBytes[0:4], uuidBytes[4:6], uuidBytes[6:8], uuidBytes[8:10], uuidBytes[10:])

	defaultState := &model.GlobalNodeState{
		Domain:            "www.eggbr.top",
		PublicIP:          pubIP,
		RealityPublicKey:  realityKeys.PublicKey,
		RealityPrivateKey: realityKeys.PrivateKey,
		RealityShortID:    realityKeys.ShortID,
		RealityServerName: "itunes.apple.com",
		Hysteria2Port:     20505,
		Hysteria2PortHop:  "30000-40000",
		Hysteria2UpMbps:   50,
		Hysteria2DownMbps: 100,
		TUICPort:          20185,
		TrojanPort:        443,
		VLESSPort:         443,
		WARPEnabled:       true,
		WARPAddress:       "172.16.0.2/32",
		Users: []model.UserCredential{
			{
				UUID:     defaultUUID,
				Password: hex.EncodeToString(uuidBytes[:8]),
				Email:    "eggbr",
			},
		},
	}
	_ = viperManager.SaveConfig(defaultState)
	return defaultState
}

// SaveState persists node state via Viper to disk
func SaveState(state *model.GlobalNodeState) error {
	viperManager := pkgConfig.NewViperConfigManager()
	return viperManager.SaveConfig(state)
}

// RunFullInstallation deploys all components and optimizes the system
func RunFullInstallation(state *model.GlobalNodeState, sysInfo *system.SystemInfo) error {
	util.PrintDivider()
	util.PrintCyan("🚀 开始部署 xraycli 高性能双内核与协议矩阵...")
	util.PrintDivider()

	// 1. Sysctl 16MB UDP & BBR
	util.PrintStep(1, 6, "应用 Linux 内核 16MB UDP 缓冲区与 BBR 拥塞控制")
	_ = system.OptimizeKernel()

	// 2. Download Xray & Sing-box
	util.PrintStep(2, 6, "拉取 Xray-core 与 Sing-box 官方核心二进制")
	if err := core.InstallXray(sysInfo); err != nil {
		util.PrintError(fmt.Sprintf("Xray 安装失败: %v", err))
	}
	if err := core.InstallSingBox(sysInfo); err != nil {
		util.PrintError(fmt.Sprintf("Sing-box 安装失败: %v", err))
	}

	// 3. Write Standard Configs
	util.PrintStep(3, 6, "构建官方标准 JSON 配置 (修复 443 Fallback & TUIC SNI)")
	if err := config.SaveXrayConfig(state); err != nil {
		return fmt.Errorf("failed to save Xray config: %w", err)
	}
	if err := config.SaveSingBoxConfig(state); err != nil {
		return fmt.Errorf("failed to save Sing-box config: %w", err)
	}

	// 4. Firewall & Port Hopping
	util.PrintStep(4, 6, "配置防火墙与 UDP 端口跳跃 NAT 转发")
	_ = system.AllowPort(443, "tcp")
	_ = system.AllowPort(15393, "tcp")
	_ = system.AllowPort(state.Hysteria2Port, "udp")
	_ = system.AllowPort(state.TUICPort, "udp")
	_ = system.AddPortHopping("hysteria2", 30000, 40000, state.Hysteria2Port, sysInfo.Distro)

	// 5. Systemd Setup
	util.PrintStep(5, 6, "注册并启动 Systemd 守护进程")
	_ = core.SetupServices()
	_ = core.RestartService("xray")
	_ = core.RestartService("sing-box")

	// 6. Save State
	util.PrintStep(6, 6, "持久化节点状态与凭据")
	_ = SaveState(state)

	util.PrintDivider()
	util.PrintGreen("🎉 全协议与双内核部署完成！终端输入 [xraycli] 随时管理。")
	util.PrintDivider()
	return nil
}
