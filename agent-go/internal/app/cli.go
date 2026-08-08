package app

import (
	"fmt"
	"v2ray-agent/internal/web"
	"v2ray-agent/pkg/core"
	"v2ray-agent/pkg/system"
	"v2ray-agent/pkg/util"
)

// HandleCLI processes command-line flags and subcommands
func HandleCLI(args []string) bool {
	if len(args) < 2 {
		return false // No subcommands passed, fallback to TUI menu
	}

	state := LoadState()
	sysInfo := system.DetectSystem()

	switch args[1] {
	case "status":
		util.PrintInfo(fmt.Sprintf("Xray 运行状态: %v", core.IsServiceRunning("xray")))
		util.PrintInfo(fmt.Sprintf("Sing-box 运行状态: %v", core.IsServiceRunning("sing-box")))
		return true

	case "sub", "subscription":
		ShowSubscriptions(state)
		return true

	case "optimize", "bbr":
		_ = system.OptimizeKernel()
		return true

	case "reload":
		_ = RunFullInstallation(state, sysInfo)
		util.PrintSuccess("配置重新编译与平滑加载完成")
		return true

	case "web":
		_ = web.StartCamouflageServer(state, 31300)
		return true

	case "install":
		_ = RunFullInstallation(state, sysInfo)
		return true

	case "help", "-h", "--help":
		fmt.Println("用法: xraycli [子命令]")
		fmt.Println("  xraycli          - 启动交互式 TUI 管理大盘")
		fmt.Println("  xraycli status   - 检查服务运行状态")
		fmt.Println("  xraycli sub      - 输出全部客户端订阅")
		fmt.Println("  xraycli optimize - 调优 Linux 内核 16MB UDP 缓冲区与 BBR")
		fmt.Println("  xraycli reload   - 重新生成并加载配置")
		fmt.Println("  xraycli web      - 启动内置 Web 伪装服务器")
		return true

	default:
		return false
	}
}
