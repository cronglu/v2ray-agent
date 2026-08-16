# Cloudflare WARP 安装配置与运维实战指南

本文档全面记录了 **Cloudflare WARP** 在 Linux VPS 上的安装部署、工作原理、当前服务器配置项深度解析、日常运维管理指令，以及与 Sing-box / S-UI / Xray 代理内核的链式智能分流实操。

---

## 1. 为什么在 VPS 上使用 Cloudflare WARP？（核心价值与原理）

### 1.1 核心痛点与解决思路
- **机房 IP 被大厂风控**：绝大多数高性价比 VPS（如 RackNerd、Oracle、DigitalOcean、Linode、Hetzner）属于数据中心 IP（Hosting / Datacenter IP）。在访问 **OpenAI (ChatGPT)、Claude、Netflix、Disney+** 或 Google 时，极易遇到 **`403 Access Denied`** 报错、强制人机验证码或限制仅看自制剧。
- **WARP 的洗白机制**：Cloudflare WARP 是 Cloudflare 官方提供的安全隧道服务。在 VPS 上启用 WARP 后，发往特定大厂服务的流量会通过 Cloudflare 的 Anycast 全球边缘网络发出，**出口 IP 变成 Cloudflare 原生纯净双栈 IP**，从而 100% 解决 403 拦截与风控。

```text
                                              【智能分流机制】
                                              ┌── [访问普通网站 (YouTube/GitHub)] ──> VPS 原生 IP 直连 (最快速度/最低时延)
[国内客户端] ──(VLESS/Hysteria2)──> [VPS 节点] ┤
                                              └── [访问 AI/流媒体 (OpenAI/Claude)] ──> 本地 WARP SOCKS5 (127.0.0.1:31303)
                                                                                            │
                                                                                            ▼
                                                                                   Cloudflare 纯净 Anycast IP
                                                                                   (100% 解封 403 / 无感解锁)
```

---

## 2. Cloudflare WARP 官方安装与部署流程

### 2.1 Ubuntu / Debian 系统安装（官方源方式）

```bash
# 1. 安装基础依赖与 gpg 工具
apt-get update && apt-get install -y curl gnupg lsb-release

# 2. 导入 Cloudflare 官方 GPG 密钥
curl -fsSL https://pkg.cloudflareclient.com/pubkey.gpg | gpg --yes --dearmor --output /usr/share/keyrings/cloudflare-warp-archive-keyring.gpg

# 3. 添加 Cloudflare 官方 apt 软件源
echo "deb [signed-by=/usr/share/keyrings/cloudflare-warp-archive-keyring.gpg] https://pkg.cloudflareclient.com/ $(lsb_release -cs) main" | tee /etc/apt/sources.list.d/cloudflare-client.list

# 4. 安装 cloudflare-warp 客户端
apt-get update && apt-get install -y cloudflare-warp

# 5. 验证安装版本
warp-cli --version
```

### 2.2 CentOS / RHEL 系统安装

```bash
# 1. 添加 yum 软件源
curl -fsSL https://pkg.cloudflareclient.com/cloudflare-warp-ascii.repo | tee /etc/yum.repos.d/cloudflare-warp.repo

# 2. 安装 cloudflare-warp
yum install -y cloudflare-warp
```

---

## 3. 当前 USA 服务器上的 WARP 实时配置与参数深度解析

当前在 `usa` 服务器（`107.172.87.203`）上运行的 WARP 配置参数如下（通过 `warp-cli settings` 查看）：

```text
Merged configuration:
(not set)       Compliance Environment: Normal
(derived)       Always On: true
(override)      Switch Locked: false
(user set)      Mode: WarpProxy on port 31303
(network policy) WARP tunnel protocol: MASQUE
(not set)       MASQUE Protocol Settings: 
  HTTP Version: MASQUE (HTTP/3 with HTTP/2 fallback)
(network policy) Post-quantum support for MASQUE: Enabled (downgrades allowed)
(not set)       Resolve via: cloudflare-dns.com @ [162.159.36.1, 162.159.46.1, 2606:4700:4700::1111, 2606:4700:4700::1001]
(api defaults)  Exclude mode, with hosts/ips: [10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, ...]
```

### 核心参数逐项拆解：

| 配置参数项 | 当前设置值 | 深度技术解释与为什么这样配？ |
| :--- | :--- | :--- |
| **`Mode` (运行模式)** | **`WarpProxy on port 31303`** | **最核心的安全策略**！WARP 运行为 **本地 SOCKS5 代理模式**，仅在 `127.0.0.1:31303` 监听。<br>❌ **绝对不要在 VPS 上开全局网卡模式 (WARP Interface)**，否则默认网关被改会导致 SSH 瞬间失联断开；<br>🟢 **Proxy 模式** 配合 Sing-box/Xray 可实现按需智能分流，安全可控。 |
| **`WARP tunnel protocol`** | **`MASQUE`** | Cloudflare 最新的下一代隧道协议（基于 **HTTP/3 QUIC** 与 HTTP/2 回落），取代了传统的 WireGuard 协议，抗弱网丢包和重连能力更强。 |
| **`Post-quantum support`** | **`Enabled`** | 开启后量子密码学加密支持，防止量子计算抓包破解。 |
| **`Resolve via` (DNS)** | **`cloudflare-dns.com`** | 使用 Cloudflare 专属安全 DoH DNS（1.1.1.1 / 1.0.0.1）解析域名，防止 DNS 污染。 |
| **`Exclude mode` (排除路由)** | **私有内网网段** | 自动排除 `10.0.0.0/8`、`172.16.0.0/12`、`192.168.0.0/16`、`127.0.0.1` 等私有网段，确保 VPS 本地进程间通信不受阻碍。 |
| **`Always On`** | **`true`** | 守护进程开机自动重连，无需人工干预。 |

---

## 4. WARP 日常运维管理与控制命令

### 4.1 查看状态与账户信息

```bash
# 1. 查看当前连接状态 (Connected / Disconnected)
warp-cli status

# 2. 查看完整配置参数 (模式、SOCKS5 端口、隧道协议等)
warp-cli settings

# 3. 查看当前注册的客户端账户信息与密钥公钥
warp-cli account

# 4. 查看后台 systemd 服务运行状态
systemctl status warp-svc
```

### 4.2 常用配置与控制指令

```bash
# 1. 注册新账户 / 同意服务条款 (首次安装必须执行)
warp-cli registration new
# 或
warp-cli terms set accept

# 2. 连接 WARP
warp-cli connect

# 3. 断开 WARP 连接
warp-cli disconnect

# 4. 设置为本地 SOCKS5 代理模式 (防失联核心命令)
warp-cli mode proxy

# 5. 设置本地 SOCKS5 代理端口 (例如设置为 31303)
warp-cli proxy port 31303

# 6. 切换隧道协议为 MASQUE (QUIC 高性能模式)
warp-cli tunnel protocol set masque

# 7. 重启 WARP 系统服务
systemctl restart warp-svc
```

---

## 5. 出口 IP 查看、连通性测试与 IP 伪装效果验证

在 VPS 终端执行以下命令，查看 WARP 出口 IP 或对比直连与走 WARP 出口的效果：

### 5.1 官方对照测试：原生机房 IP (直连) vs WARP 出口 IP (走 SOCKS5)

```bash
# [直连] 显示 VPS 原生机房 IP (warp=off，当前为 107.172.87.203)
curl -s https://www.cloudflare.com/cdn-cgi/trace

# [WARP 出口] 显示 Cloudflare 纯净 IP (warp=on，出口为 104.28.201.80)
curl -sx socks5h://127.0.0.1:31303 https://www.cloudflare.com/cdn-cgi/trace
```

### 5.2 纯 IP 与中文地理位置速查 (日常快速确认)

```bash
# 1. 快速输出纯 IP 地址 (104.28.201.80)
curl -sx socks5h://127.0.0.1:31303 https://api.ipify.org

# 2. 查看详细地理位置与所属运营商 (中文展示)
curl -sx socks5h://127.0.0.1:31303 https://myip.ipip.net
# 输出示例：当前 IP：104.28.201.80  来自于：美国 加利福尼亚州 圣何塞  cloudflare.com
```

### 5.3 原生机房 IP vs WARP 出口 IP 效果对比表

| 查看对象 | 执行命令 | 输出 IP 与属性 |
| :--- | :--- | :--- |
| **原生机房 IP (直连)** | `curl -s https://api.ipify.org` | `107.172.87.203` (RackNerd 机房 IP，易被风控) |
| **WARP 出口 IP (洗白)** | `curl -sx socks5h://127.0.0.1:31303 https://api.ipify.org` | `104.28.201.80` (Cloudflare 纯净 IP，解锁 AI / 奈飞) |

### 5.4 验证 OpenAI / ChatGPT 解锁效果（解决 403 报错）
```bash
# 1. 用 VPS 原生机房 IP 探测 OpenAI API (部分机房可能直接返回 403 Access Denied)
curl -s -o /dev/null -w "Direct OpenAI Status Code: %{http_code}\n" https://api.openai.com/v1/models

# 2. 通过 WARP SOCKS5 出口探测 OpenAI (返回 401 说明鉴权正常通过，403 表示被拦截)
curl -sx socks5h://127.0.0.1:31303 -o /dev/null -w "WARP OpenAI Status Code: %{http_code}\n" https://api.openai.com/v1/models
```



---

## 6. 与代理服务端（S-UI / Sing-box）的智能分流集成实战

在当前 `usa` 服务器上，已将 WARP 与 **S-UI (Sing-box 内核)** 做了无感智能联动：

### 6.1 Sing-box 配置结构（两段式出口）：

```json
{
  "outbounds": [
    // 默认主力出站：VPS 原生网络直连 (速度最快、延迟最低)
    {
      "type": "direct",
      "tag": "direct"
    },
    // WARP 洗白出站：转发给本地 31303 SOCKS5
    {
      "type": "socks",
      "tag": "warp-out",
      "server": "127.0.0.1",
      "server_port": 31303
    }
  ],
  "route": {
    "rules": [
      {
        "action": "sniff"
      },
      {
        "protocol": ["dns"],
        "action": "hijack-dns"
      },
      // 匹配到 AI 平台与流媒体服务，无感走 WARP 落地
      {
        "domain_suffix": [
          "openai.com",
          "chatgpt.com",
          "oaistatic.com",
          "oaiusercontent.com",
          "anthropic.com",
          "claude.ai",
          "netflix.com",
          "netflix.net",
          "nflximg.net",
          "nflxvideo.net",
          "nflxext.com",
          "nflxso.net",
          "disneyplus.com"
        ],
        "outbound": "warp-out"
      },
      // 其余所有网站 (YouTube、GitHub、Twitter 等) 走默认直连
      {
        "outbound": "direct"
      }
    ]
  }
}
```

---

## 7. 常见问题排查 (FAQ)

### Q1：为什么执行 `curl -sx socks5h://127.0.0.1:31303 ...` 提示连接被拒绝 (Connection Refused)？
- **排查步骤**：
  1. 执行 `warp-cli status` 检查是否显示 `Connected`；
  2. 若显示 `Disconnected`，执行 `warp-cli connect`；
  3. 若依然连接不上，执行 `systemctl restart warp-svc` 重启服务。

### Q2：WARP 开启后会不会导致服务器 SSH 端口断开？
- **不会**！因为我们在配置中明确使用了 **`warp-cli mode proxy`（本地 SOCKS5 代理模式）**，它只在 `127.0.0.1:31303` 监听，**绝不修改系统默认路由网卡（default gateway）**，SSH 永远安全稳定。
