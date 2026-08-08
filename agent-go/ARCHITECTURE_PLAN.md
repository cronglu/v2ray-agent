# Go 重构 v2ray-agent (xraycli) 架构方案与技术规范 (V3.2 纯 Go 极简版)

> 本次更新：
> 1. **协议按需启闭**：明确 Trojan、Hysteria2、TUIC、VLESS 各协议完全解耦，用户未安装 Trojan 没有任何影响。
> 2. **Go 内置轻量 Web 伪装站与订阅分发引擎**：彻底告别庞大冗余的 Nginx 依赖，由 Go 原生 `net/http` + `embed.FS` 统一接管伪装站与订阅分发。

---

## 1. 快捷方式与命令命名 (Shortcut: xraycli)

- **主命令快捷方式**：**`xraycli`**。
- **辅助别名**：`xcli` / `v2cli`。
- **命令体系**：
  ```bash
  xraycli            # 呼出交互式 TUI 状态大盘与管理控制台
  xraycli status     # 检查 Xray / Sing-box 核心状态与端口占用
  xraycli sub        # 输出/管理客户端订阅与二维码
  xraycli reload     # 重新校验并加载官方标准 JSON 配置
  xraycli log        # 实时查看核心服务日志
  ```

---

## 2. 协议解耦：没有安装 Trojan 协议会有影响吗？

**结论：没有任何问题，系统完全支持协议按需自由组合。**

- **协议完全解耦**：
  - 用户可以只开启 **Hysteria2**、只开启 **TUIC v5**、或只开启 **VLESS-Reality**。
  - **当未启用 Trojan 时**：443 端口的 VLESS-TCP-TLS 流量在非代理认证时，直接无条件回落到本地 Go 内置的伪装网站（`127.0.0.1:31300`），链路甚至比开启 Trojan 更加极简与纯粹。
  - **当主打 Reality + Hysteria2 时**：Reality 本身偷取第三方大厂域名（如 `itunes.apple.com`），甚至无需本地开启 443 端口和申请证书。

---

## 3. Go 替代 Nginx 的核心实现原理 (Zero-Nginx Architecture)

原 Shell 脚本中安装 Nginx 主要是为了做两件事：
1. 提供一个 **HTTP 伪装网页**（防止 GFW 或主动探针探测到裸代理）。
2. 提供一个 **订阅服务分发地址**（提供 HTTPS 订阅链接下载）。

在 Go 重构版本中，我们使用 **Go 原生标准库 + 内置 Web 引擎** 彻底替代 Nginx，带来巨大的架构优势：

```
┌─────────────────────────────────────────────────────────────┐
│                 Go 原生内嵌 Web 引擎 (xraycli)              │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ 1. 内嵌静态伪装站 (embed.FS)                         │  │
│  │    - 3D WebGL 游戏 / HTML5 极客博客 / 现代作品集      │  │
│  │    - 监听 127.0.0.1:31300 (支持 ProxyProtocol v1)     │  │
│  └───────────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ 2. 智能订阅分发服务 (Smart Subscription Server)       │  │
│  │    - 自动识别 UA (Clash -> YAML, Singbox -> JSON)     │  │
│  │    - Token 鉴权与安全防爬                             │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 3.1 Go 替代 Nginx 的四大核心优势

| 维度 | 传统 Nginx 方案 | Go 原生内置 Web 方案 |
| :--- | :--- | :--- |
| **系统体积与开销** | 需安装 50MB+ Nginx 软件包，常驻 30MB+ 内存 | **0 外部依赖**，打包进单个二进制，额外内存占用 < 5MB |
| **伪装站资源管理** | 需下载静态网页包，路径分散，常发生 404/权限错误 | **`embed.FS` 静态内嵌**，网页资源随二进制打包，永不丢失 |
| **SELinux 与系统冲突** | CentOS 下经常因 `http_port_t` 导致 31302 权限拒绝 | **原生 Socket 绑定**，彻底告别 SELinux Nginx 冲突 |
| **智能订阅分发** | 只能返回静态文件，不同客户端需生成多个独立文件 | **动态识别客户端 UA**：Clash 请求返 YAML，Singbox 返 JSON |
| **Proxy Protocol** | 需要在 `nginx.conf` 中配置 `proxy_protocol` | Go 原生支持 Proxy Protocol，精确还原 Xray 回落的真实 IP |

---

## 4. 标准服务端配置与回落拓扑

```
客户端连接 (443 端口)
   │
   ├─ VLESS 认证成功 ───────────────> [Xray 代理转发]
   │
   └─ 非 VLESS 流量 (回落 dest: 31300)
         │
         ▼
   [Go 原生内嵌伪装站点] ─────────────> 返回逼真的 3D 游戏 / 博客网页 (HTTP/1.1 & HTTP/2)
```

---

## 5. 目录结构规划 (`agent-go/`)

```text
agent-go/
├── ARCHITECTURE_PLAN.md     # 本技术规范与对比分析
├── build.sh                 # 跨平台一键编译并安装至 /usr/local/bin/xraycli
├── go.mod
├── go.sum
├── cmd/
│   └── xraycli/             # 主程序入口
│       └── main.go
├── internal/
│   ├── app/                 # CLI 标志解析与 TUI 控制台
│   │   ├── cli.go
│   │   ├── menu.go
│   │   └── installer.go
│   ├── web/                 # Go 原生替代 Nginx 的 Web 引擎
│   │   ├── server.go        # 伪装站与订阅分发 HTTP/HTTPS 服务
│   │   ├── proxy_proto.go   # Proxy Protocol v1/v2 协议解析
│   │   ├── static/          # embed.FS 内嵌精美伪装网页资源
│   │   │   └── index.html
│   │   └── sub_handler.go   # 智能 UA 订阅分发处理器
│   ├── config/              # 官方标准 JSON 结构体与装配
│   │   ├── xray_config.go   # Xray 标准 JSON 构建
│   │   ├── singbox_config.go# Sing-box 标准 JSON 构建
│   │   └── warp_config.go   # WARP WireGuard 出站与分流
│   └── subscription/        # 客户端订阅生成器
│       ├── clash_meta.go
│       ├── singbox_client.go
│       └── uri.go
└── pkg/
    ├── core/                # Xray / Sing-box 二进制拉取与 Systemd 服务管理
    ├── system/              # Sysctl 16MB UDP Buffer、BBR、Iptables 端口跳跃
    ├── tls/                 # ACME 证书与 Reality 算法
    └── util/                # IP 检测、彩色日志、二维码渲染
```
