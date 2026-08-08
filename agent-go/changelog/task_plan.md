# xraycli (v2ray-agent Go 重构版) 任务拆解与开发进度日志

---

## 1. 任务拆分总览 (Milestones & Granular Tasks)

- [x] **Phase 1: 基础设施与系统底座 (System & Core Ops)**
  - [x] 1.1 初始化 Go 模块与目录工程骨架 (`go.mod`, `build.sh`)
  - [x] 1.2 系统环境探测与发行版识别 (`pkg/system/distro.go`)
  - [x] 1.3 Linux 内核 16MB UDP Buffer 与 BBR 拥塞控制调优 (`pkg/system/sysctl.go`)
  - [x] 1.4 防火墙与 NAT 端口跳跃管理 (`pkg/system/firewall.go`)
  - [x] 1.5 Xray / Sing-box 下载器、多源代理池与 Systemd 守护 (`pkg/core/`)
  - [x] 1.6 基础辅助工具：IP 探测、Reality x25519 密钥生成、ANSI 彩色日志 (`pkg/util/`, `pkg/tls/`)
  - [x] 1.7 引入 `spf13/viper` 实现多路径自动寻址与配置文件持久化 (`pkg/config/loader.go`)

- [x] **Phase 2: 官方标准配置编排与修复 (Standard Protocol Engine)**
  - [x] 2.1 强类型领域数据模型与 `mapstructure` 标签映射 (`internal/model/node.go`)
  - [x] 2.2 Xray 官方标准 JSON 构建器（修复 443 Fallback ALPN 闭环） (`internal/config/xray_config.go`)
  - [x] 2.3 Sing-box 官方标准 JSON 构建器（Hysteria2 端口跳跃、TUIC v5 SNI 规范） (`internal/config/singbox_config.go`)
  - [x] 2.4 Cloudflare WARP WireGuard 出站与 Google AI (Gemini) / OpenAI 智能分流规则 (`internal/config/xray_config.go`, `internal/config/singbox_config.go`)

- [x] **Phase 3: Go 原生内嵌 Web 引擎与订阅中心 (Zero-Nginx Web & Sub)**
  - [x] 3.1 `embed.FS` 静态内嵌 3D 极客展示伪装网站 (`internal/web/static/index.html`)
  - [x] 3.2 Proxy Protocol v1 解析与原生 HTTP/HTTPS 伪装服务器 (`internal/web/server.go`)
  - [x] 3.3 智能订阅分发引擎：自适应 UA 导出 Clash.Meta YAML、Sing-box JSON、通用 URI (`internal/subscription/`)

- [x] **Phase 4: CLI 命令行工具与交互式 TUI (Console & Main Entry)**
  - [x] 4.1 CLI 命令行标志解析与子命令路由 (`internal/app/cli.go`)
  - [x] 4.2 交互式控制台菜单与服务状态监控仪表盘 (`internal/app/menu.go`)
  - [x] 4.3 主程序入口装配 (`cmd/xraycli/main.go`) 与全功能二进制构建

- [x] **Phase 5: 验证、构建与全流程测试 (Verification & Deliverable)**
  - [x] 5.1 本地编译通过（`agent-go/bin/xraycli`，零外部依赖静态二进制）
  - [x] 5.2 核心配置语法与 JSON/YAML 校验测试（`xraycli sub`、`xraycli help` 输出验证）
  - [x] 5.3 编写 `build.sh` 一键构建脚本替代 Makefile
  - [x] 5.4 输出技术档案与进度记录 (`agent-go/changelog/`)

---

## 2. 详细执行日志 (Step-by-step Progress Log)

### 2026-08-09: 全量重构落地、Viper 集成与发布测试
- **任务目标**：完成 `xraycli` 从架构设计、Viper 配置引擎集成、Build.sh 脚本到完整 Go 源码的工程化实现与验证。
- **实施进展**：
  1. **架构与别名确立**：确立以 `xraycli` 为核心主快捷命令，软链接 `xcli` / `v2cli`。
  2. **配置管理器升级 (Viper)**：
     - 引入 `github.com/spf13/viper`，实现 `/etc/v2ray-agent/config.yaml` 磁盘配置的安全加载、默认值兜底与回写持久化。
     - 为 `model.GlobalNodeState` 与 `model.UserCredential` 补充 `mapstructure` 标签映射。
  3. **系统底层实现**：
     - `pkg/system/distro.go`：多发行版（Debian, Ubuntu, CentOS, Alpine, Arch）与 CPU 架构自适应探测。
     - `pkg/system/sysctl.go`：原生持久化 Linux 内核 16MB UDP 缓冲区（`net.core.rmem_max=16777216`）与 `bbr + fq` 拥塞控制。
     - `pkg/system/firewall.go`：自动安装并管理 `iptables-persistent`，基于 comment 标签精准维护 Hysteria2 端口跳跃 NAT 规则。
  4. **官方标准配置编排**：
     - `internal/config/xray_config.go`：构建 100% 官方标准 Xray JSON，修复 443 Fallback 冲突（仅保留 `dest: 31296`，移除 `alpn: h2`）。
     - `internal/config/singbox_config.go`：构建标准 Sing-box JSON，规范 TUIC v5 的 `server_name` 与 Hysteria2 `up_mbps/down_mbps`。
  5. **Zero-Nginx Web 引擎**：
     - `internal/web/server.go`：基于 Go `embed.FS` 内嵌极客 HTML5 伪装网站，原生监听 31300 端口并处理 `/s/` 动态 UA 订阅下发。
  6. **跨客户端订阅导出**：
     - `internal/subscription/clash_meta.go`：导出规范的 Clash.Meta / Mihomo YAML 订阅。
     - `internal/subscription/singbox_client.go`：导出标准的 Sing-box Client JSON。
     - `internal/subscription/uri.go`：导出 `vless://`, `hysteria2://`, `tuic://`, `trojan://` 通用链接。
  7. **编译脚本与运行测试**：
     - 编写 `build.sh` 取代 Makefile，支持 `build`、`install`、`release`（跨平台编译）与 `clean`。
     - 执行 `./build.sh install` 安装至 `/usr/local/bin/xraycli`。
     - 测试 `xraycli help`、`xraycli sub`，验证节点导出、TUIC SNI、Hysteria2 端口跳跃与 Clash.Meta 语法完全正确。
