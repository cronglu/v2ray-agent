# xraycli (v2ray-agent Go 重构版) 用户使用手册

`xraycli` 是使用 Go 语言全面重构的高性能、零外部依赖、双内核（Xray-core / Sing-box）一键式代理管理套件。

---

## 1. 核心特性

- **零外部依赖**：单静态编译二进制，无需安装 `jq`、`sed`、`awk` 或 `nginx` 软件包。
- **100% 官方标准配置**：完全对齐 Xray 与 Sing-box 官方标准 JSON 结构，无任何私有中间语法阻碍。
- **内置 Web 伪装站**：使用 Go `embed.FS` 内嵌静态极客伪装网页，开箱即用，零文件丢失。
- **协议级修复**：
  - 彻底修复 443 端口 Xray Fallbacks 与 ALPN 冲突引起的 Trojan 断流超时。
  - 彻底修复 TUIC v5 订阅中 SNI 错误填入用户标识导致的证书验证失败。
  - 内置原生 Linux 16MB UDP 缓冲区与 BBR 拥塞控制调优。
  - 原生管理 `iptables-persistent`，解决 Debian/Ubuntu 重启丢失端口跳跃规则。
  - 内置 Google AI (Gemini) / OpenAI 智能 WARP WireGuard 分流出站。

---

## 2. 快速安装与编译

### 2.1 从源码一键编译
```bash
cd agent-go
chmod +x build.sh
./build.sh install
```
安装后，可直接在终端使用 `xraycli` 或快捷别名 `xcli`、`v2cli`。

---

## 3. 常用命令操作

```bash
# 启动交互式管理控制面板
xraycli

# 常用快捷子命令
xraycli status      # 查看 Xray / Sing-box / 伪装 Web 运行状态与端口
xraycli reload      # 校验并平滑重载所有官方标准 JSON 配置
xraycli sub         # 输出 Clash.Meta YAML、Sing-box JSON 及通用 URI 订阅
xraycli log         # 实时跟踪滚动查看核心服务日志
xraycli optimize    # 一键执行 Linux 内核 16MB UDP 缓冲区与 BBR 调优
```

---

## 4. 默认目录与标准配置文件

- **Xray 官方配置**：`/etc/v2ray-agent/xray/config.json`
- **Sing-box 官方配置**：`/etc/v2ray-agent/sing-box/config.json`
- **TLS 证书目录**：`/etc/v2ray-agent/tls/`
- **全局状态记录**：`/etc/v2ray-agent/agent_state.json`
