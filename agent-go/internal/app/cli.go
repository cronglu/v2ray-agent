package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"v2ray-agent/internal/model"
	"v2ray-agent/internal/monitor"
	"v2ray-agent/internal/web"
	"v2ray-agent/pkg/system"
	"v2ray-agent/pkg/util"
)

// HandleCLI processes command-line flags and subcommands.
// Returns true when a subcommand was handled. For the monitor subcommand the
// process exits with a health-derived code so it can be used by cron/alerting.
func HandleCLI(args []string) bool {
	if len(args) < 2 {
		return false // No subcommands passed, fallback to TUI menu
	}

	state := LoadState()
	sysInfo := system.DetectSystem()

	switch args[1] {
	case "status", "monitor":
		runMonitor(state, args[2:])
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

	case "log":
		util.PrintInfo("提示: 使用 journalctl -u xray -f 或 journalctl -u sing-box -f 跟踪日志")
		return true

	case "help", "-h", "--help":
		printHelp()
		return true

	default:
		return false
	}
}

// runMonitor builds the per-protocol health report and prints it. Supports a
// -j/--json flag for machine-readable output, and exits with a status code
// (0 healthy / 1 degraded / 2 down) so it can drive cron alerts.
func runMonitor(state *model.GlobalNodeState, args []string) {
	asJSON := false
	for _, a := range args {
		switch a {
		case "-j", "--json", "json":
			asJSON = true
		}
	}

	mon := monitor.New(state)
	rep := mon.Run(context.Background())

	if asJSON {
		out, err := rep.JSON()
		if err != nil {
			util.PrintError(fmt.Sprintf("生成 JSON 报告失败: %v", err))
			os.Exit(3)
		}
		fmt.Println(out)
	} else {
		fmt.Print(rep.Render())
	}

	// Keep a friendly summary line for the plain-text output.
	if !asJSON {
		util.PrintInfo(fmt.Sprintf("Xray: %v  Sing-box: %v  总体: %s",
			rep.XrayActive, rep.SingBoxActive, strings.ToLower(string(rep.Overall))))
	}
	os.Exit(rep.ExitCode())
}

func printHelp() {
	fmt.Println("用法: xraycli [子命令]")
	fmt.Println("  xraycli            - 启动交互式 TUI 管理大盘")
	fmt.Println("  xraycli status     - 各协议健康监控 (TCP/TLS/QUIC/证书/配置/日志)")
	fmt.Println("  xraycli monitor    - 同 status，别名")
	fmt.Println("  xraycli monitor -j - JSON 机器可读输出，退出码反映健康度")
	fmt.Println("  xraycli sub        - 输出全部客户端订阅")
	fmt.Println("  xraycli log        - 查看核心服务日志")
	fmt.Println("  xraycli optimize   - 调优 Linux 内核 16MB UDP 缓冲区与 BBR")
	fmt.Println("  xraycli reload     - 重新生成并加载配置")
	fmt.Println("  xraycli web        - 启动内置 Web 伪装服务器")
	fmt.Println("  xraycli install    - 一键全协议部署")
}
