package config

import (
	"fmt"
	"os"
	"path/filepath"
	"v2ray-agent/internal/model"
	"v2ray-agent/pkg/util"

	"github.com/spf13/viper"
)

const (
	DefaultConfigDir  = "/etc/v2ray-agent"
	DefaultConfigName = "config" // matches config.yaml / config.json / config.toml
	DefaultConfigType = "yaml"
)

// ViperConfigManager handles configuration loading and persistence via spf13/viper
type ViperConfigManager struct {
	v *viper.Viper
}

// NewViperConfigManager initializes Viper with standard search paths and defaults
func NewViperConfigManager() *ViperConfigManager {
	v := viper.New()

	// 1. Set configuration file names & search paths
	v.SetConfigName(DefaultConfigName)
	v.SetConfigType(DefaultConfigType)
	v.AddConfigPath(DefaultConfigDir)
	v.AddConfigPath(".")
	v.AddConfigPath("$HOME/.v2ray-agent")

	// 2. Set default values
	v.SetDefault("domain", "www.eggbr.top")
	v.SetDefault("public_ip", "127.0.0.1")
	v.SetDefault("reality_server_name", "itunes.apple.com")
	v.SetDefault("hysteria2_port", 20505)
	v.SetDefault("hysteria2_port_hop", "30000-40000")
	v.SetDefault("hysteria2_up_mbps", 50)
	v.SetDefault("hysteria2_down_mbps", 100)
	v.SetDefault("tuic_port", 20185)
	v.SetDefault("trojan_port", 443)
	v.SetDefault("vless_port", 443)
	v.SetDefault("warp_enabled", true)
	v.SetDefault("warp_address", "172.16.0.2/32")

	// Support environment variables with prefix XRAYCLI_
	v.SetEnvPrefix("XRAYCLI")
	v.AutomaticEnv()

	return &ViperConfigManager{v: v}
}

// LoadConfig reads configuration file from disk into model.GlobalNodeState
func (m *ViperConfigManager) LoadConfig() (*model.GlobalNodeState, error) {
	var state model.GlobalNodeState

	if err := m.v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, use defaults
			util.PrintWarning("未检测到已有配置文件，将使用默认参数初始化...")
		} else {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	} else {
		util.PrintInfo(fmt.Sprintf("成功通过 Viper 加载配置文件: %s", m.v.ConfigFileUsed()))
	}

	if err := m.v.Unmarshal(&state); err != nil {
		return nil, fmt.Errorf("反序列化配置失败: %w", err)
	}

	return &state, nil
}

// SaveConfig persists the current GlobalNodeState to disk using Viper
func (m *ViperConfigManager) SaveConfig(state *model.GlobalNodeState) error {
	_ = os.MkdirAll(DefaultConfigDir, 0755)
	targetFile := filepath.Join(DefaultConfigDir, fmt.Sprintf("%s.%s", DefaultConfigName, DefaultConfigType))

	// Map struct to viper keys
	m.v.Set("domain", state.Domain)
	m.v.Set("public_ip", state.PublicIP)
	m.v.Set("users", state.Users)
	m.v.Set("reality_public_key", state.RealityPublicKey)
	m.v.Set("reality_private_key", state.RealityPrivateKey)
	m.v.Set("reality_short_id", state.RealityShortID)
	m.v.Set("reality_server_name", state.RealityServerName)
	m.v.Set("hysteria2_port", state.Hysteria2Port)
	m.v.Set("hysteria2_port_hop", state.Hysteria2PortHop)
	m.v.Set("hysteria2_up_mbps", state.Hysteria2UpMbps)
	m.v.Set("hysteria2_down_mbps", state.Hysteria2DownMbps)
	m.v.Set("tuic_port", state.TUICPort)
	m.v.Set("trojan_port", state.TrojanPort)
	m.v.Set("vless_port", state.VLESSPort)
	m.v.Set("warp_enabled", state.WARPEnabled)
	m.v.Set("warp_private_key", state.WARPPrivateKey)
	m.v.Set("warp_reserved", state.WARPReserved)
	m.v.Set("warp_address", state.WARPAddress)

	if err := m.v.WriteConfigAs(targetFile); err != nil {
		// If file already exists, overwrite it
		return m.v.WriteConfig()
	}

	util.PrintSuccess(fmt.Sprintf("配置文件已成功写入: %s", targetFile))
	return nil
}
