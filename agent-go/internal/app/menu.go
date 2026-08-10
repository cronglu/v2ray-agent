package app

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"v2ray-agent/internal/model"
	"v2ray-agent/internal/monitor"
	"v2ray-agent/internal/subscription"
	"v2ray-agent/internal/web"
	"v2ray-agent/pkg/core"
	"v2ray-agent/pkg/system"
	"v2ray-agent/pkg/util"
)

// ShowDashboard prints the interactive status dashboard and menu
func ShowDashboard() {
	state := LoadState()
	sysInfo := system.DetectSystem()

	xrayStatus := "🔴 未运行"
	if core.IsServiceRunning("xray") {
		xrayStatus = "🟢 运行中"
	}
	singboxStatus := "🔴 未运行"
	if core.IsServiceRunning("sing-box") {
		singboxStatus = "🟢 运行中"
	}

	for {
		fmt.Println()
		util.PrintDivider()
		fmt.Println(util.ColorCyan + util.ColorBold + "      xraycli (v2ray-agent Go 重构版) 管理控制台" + util.ColorReset)
		fmt.Printf(util.ColorWhite+"  系统发行版: %-10s | 架构: %-8s | 公网 IP: %s\n"+util.ColorReset, sysInfo.Distro, sysInfo.Arch, state.PublicIP)
		fmt.Printf(util.ColorWhite+"  Xray 状态: %s   | Sing-box 状态: %s\n"+util.ColorReset, xrayStatus, singboxStatus)
		util.PrintDivider()

		fmt.Println(util.ColorYellow + " 1. 一键全协议安装 (Xray + Sing-box + 16MB UDP + BBR)" + util.ColorReset)
		fmt.Println(util.ColorYellow + " 2. 查看客户端订阅 (Clash.Meta / Sing-box / 通用链接)" + util.ColorReset)
		fmt.Println(util.ColorYellow + " 3. 启动/管理 Go 内置 Web 伪装站与订阅分发服务" + util.ColorReset)
		fmt.Println(util.ColorYellow + " 4. 一键执行 Linux 内核 16MB UDP 缓冲区与 BBR 优化" + util.ColorReset)
		fmt.Println(util.ColorYellow + " 5. 重新载入并校验官方标准 JSON 配置 (Hot Reload)" + util.ColorReset)
		fmt.Println(util.ColorYellow + " 6. 重启核心服务 (Restart Xray & Sing-box)" + util.ColorReset)
		fmt.Println(util.ColorYellow + " 7. 各协议健康监控 (TCP/TLS/QUIC/证书/配置/日志)" + util.ColorReset)
		fmt.Println(util.ColorYellow + " 0. 退出管理控制台" + util.ColorReset)
		util.PrintDivider()
		fmt.Print("请选择操作 [0-7]: ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			_ = RunFullInstallation(state, sysInfo)
		case "2":
			ShowSubscriptions(state)
		case "3":
			go func() {
				_ = web.StartCamouflageServer(state, 31300)
			}()
			util.PrintSuccess("Go 内置 Web 伪装服务器已在后台启动 (127.0.0.1:31300)")
		case "4":
			_ = system.OptimizeKernel()
		case "5":
			_ = RunFullInstallation(state, sysInfo)
			util.PrintSuccess("所有官方标准配置重新编译完成并已平滑生效")
		case "6":
			_ = core.RestartService("xray")
			_ = core.RestartService("sing-box")
		case "7":
			mon := monitor.New(state)
			rep := mon.Run(context.Background())
			fmt.Print(rep.Render())
		case "0":
			fmt.Println("感谢使用 xraycli，再见！")
			return
		default:
			util.PrintError("无效的选择，请重新输入")
		}
	}
}

// ShowSubscriptions prints clean, formatted subscriptions for all clients
func ShowSubscriptions(state *model.GlobalNodeState) {
	util.PrintDivider()
	util.PrintCyan("📦 客户端订阅与节点导出信息:")
	util.PrintDivider()

	// 1. Universal URIs
	uris := subscription.GenerateUniversalURIs(state)
	fmt.Println(util.ColorGreen + "【通用节点链接 (Universal URIs)】:" + util.ColorReset)
	for _, u := range uris {
		fmt.Println("  " + u)
	}

	fmt.Println()
	fmt.Println(util.ColorCyan + "【Clash.Meta (Mihomo) YAML 订阅预览】:" + util.ColorReset)
	fmt.Println(subscription.GenerateClashMetaConfig(state))

	util.PrintDivider()
}
