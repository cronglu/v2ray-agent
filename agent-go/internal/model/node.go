package model

// ProtocolType defines proxy protocol kind
type ProtocolType string

const (
	ProtoVLESSReality ProtocolType = "vless_reality"
	ProtoVLESSTLS     ProtocolType = "vless_tls"
	ProtoTrojan       ProtocolType = "trojan"
	ProtoHysteria2    ProtocolType = "hysteria2"
	ProtoTUIC         ProtocolType = "tuic"
	ProtoVMess        ProtocolType = "vmess"
)

// UserCredential holds user UUID and credentials
type UserCredential struct {
	UUID     string `json:"uuid" mapstructure:"uuid"`
	Password string `json:"password" mapstructure:"password"`
	Email    string `json:"email" mapstructure:"email"`
}

// GlobalNodeState stores the current server node information
type GlobalNodeState struct {
	Domain             string           `json:"domain" mapstructure:"domain"`
	PublicIP           string           `json:"public_ip" mapstructure:"public_ip"`
	Users              []UserCredential `json:"users" mapstructure:"users"`
	RealityPublicKey   string           `json:"reality_public_key" mapstructure:"reality_public_key"`
	RealityPrivateKey  string           `json:"reality_private_key" mapstructure:"reality_private_key"`
	RealityShortID     string           `json:"reality_short_id" mapstructure:"reality_short_id"`
	RealityServerName  string           `json:"reality_server_name" mapstructure:"reality_server_name"`
	Hysteria2Port      int              `json:"hysteria2_port" mapstructure:"hysteria2_port"`
	Hysteria2PortHop   string           `json:"hysteria2_port_hop" mapstructure:"hysteria2_port_hop"`
	Hysteria2UpMbps    int              `json:"hysteria2_up_mbps" mapstructure:"hysteria2_up_mbps"`
	Hysteria2DownMbps  int              `json:"hysteria2_down_mbps" mapstructure:"hysteria2_down_mbps"`
	TUICPort           int              `json:"tuic_port" mapstructure:"tuic_port"`
	TrojanPort         int              `json:"trojan_port" mapstructure:"trojan_port"`
	VLESSPort          int              `json:"vless_port" mapstructure:"vless_port"`
	WARPEnabled        bool             `json:"warp_enabled" mapstructure:"warp_enabled"`
	WARPPrivateKey     string           `json:"warp_private_key" mapstructure:"warp_private_key"`
	WARPReserved       string           `json:"warp_reserved" mapstructure:"warp_reserved"`
	WARPAddress        string           `json:"warp_address" mapstructure:"warp_address"`
}
