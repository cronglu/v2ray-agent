# USA 服务器配置与 S-UI (Sing-box UI) 全协议运维档案

本文档记录了 `usa` 服务器（RackNerd VPS）由 **S-UI（Sing-box UI）** 原生内核驱动的 **5 大主流协议（VLESS-Reality-TCP / VLESS-Reality-XHTTP / Hysteria 2 / Tuic v5 / AnyTLS）** 的实时运行端口、连接凭证、WARP 智能分流规则、Subconverter 订阅转换机制与自建全流程。

---

## 1. 服务器基础信息

| 属性 | 配置值 / 说明 |
| :--- | :--- |
| **主机别名** | `usa`（配置在 `~/.ssh/config`） |
| **公网 IP** | `107.172.87.203` |
| **SSH 登录** | `ssh usa` 或 `ssh root@107.172.87.203 -p 28222` |
| **SSH 端口** | `28222` |
| **硬件与系统** | 1 核 CPU / 1GB 内存 / 16GB SSD / Ubuntu 20.04.6 LTS (x86_64) |
| **服务商** | RackNerd (美西机房) |

---

## 2. S-UI (Sing-box UI) 可视化管理面板

S-UI 基于 **Sing-box 官方原生内核** 驱动，已作为 systemd 守护进程常驻运行（开机自启）。

| 配置项 | 详细信息 |
| :--- | :--- |
| **面板访问 URL** | 👉 `http://107.172.87.203:6313/admin/` |
| **登录账号** | `admin` |
| **登录密码** | `sui_pass_202688` |
| **面板监听端口** | `6313` |
| **URL 基础路径** | `/admin/` |
| **订阅服务端口** | `2096`（路径 `/sub_eggbr/`） |
| **数据库路径** | `/usr/local/s-ui/db/s-ui.db` |
| **核心程序路径** | `/usr/local/s-ui/sui`（内置 Sing-box 1.5.5 内核） |

---

## 3. 当前已部署生效的 5 大协议配置总览（当前用户：`eggbr`）

| 协议名称 | 监听端口 / 传输 | 安全/TLS 类型 | 伪装 / 证书策略 | 核心适用场景 | 运行状态 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **1. VLESS-Reality-Vision** | `19348` (TCP) | Reality | 借用 `www.apple.com` | **日常主力推荐**，防封能力最强 | 🟢 正常运行 |
| **2. VLESS-Reality-XHTTP** | `41304` (HTTP/2) | Reality | 借用 `www.apple.com` (`/kxgjxhttp`) | **极端严苛内网/校园网/套CDN穿透** | 🟢 正常运行 |
| **3. Hysteria 2** | `48618` (UDP) | 标准 TLS | 自签名证书（跳过证书校验） | **晚高峰/弱网抗丢包极速神器**，看 4K/8K 视频 | 🟢 正常运行 |
| **4. Tuic v5** | `17285` (UDP) | QUIC TLS | 自签名证书 + Cubic 拥塞控制 | **QUIC 多路复用**，零 RTT 极速握手 | 🟢 正常运行 |
| **5. AnyTLS** | `31335` (TCP) | AnyTLS | Sing-box 原生 TLS 封装 | 适用于 **Sing-box 官方客户端 / Karing** | 🟢 正常运行 |

---

## 4. 客户端完整导入配置

### 4.1 单节点标准链接（复制进客户端直接导入）：

```text
vless://cce0c2f5-c9cf-4d71-8754-b98b45f6dd36@107.172.87.203:19348?type=tcp&security=reality&pbk=xfj0LI42PfKawX5RDl6wZJTrdE5XNPqkZ7ZLLGL9gQE&sid=8cabcc6a&sni=www.apple.com&flow=xtls-rprx-vision#USA-VLESS-Reality
vless://cce0c2f5-c9cf-4d71-8754-b98b45f6dd36@107.172.87.203:41304?type=http&host=www.apple.com&path=%2Fkxgjxhttp&security=reality&pbk=xfj0LI42PfKawX5RDl6wZJTrdE5XNPqkZ7ZLLGL9gQE&sid=8cabcc6a&sni=www.apple.com#USA-VLESS-XHTTP
hysteria2://uM2TEVV2AH@107.172.87.203:48618?downmbps=100&upmbps=100&security=tls&insecure=1&sni=usa.proxy.local#USA-Hysteria2
tuic://cce0c2f5-c9cf-4d71-8754-b98b45f6dd36:uM2TEVV2AH@107.172.87.203:17285?security=tls&insecure=1&sni=usa.proxy.local&congestion_control=cubic&version=5&alpn=h3#USA-Tuic-v5
anytls://uM2TEVV2AH@107.172.87.203:31335?security=tls&insecure=1&sni=usa.proxy.local#USA-AnyTLS
```

---

### 4.2 Clash Meta / Mihomo 配置文件（可直接保存为 Local YAML）：

```yaml
port: 7890
socks-port: 7891
allow-lan: false
mode: rule
log-level: info

proxies:
  # 节点 1：VLESS + Reality + Vision（免证书，日常稳定首选）
  - name: "🇺🇸 01-USA-VLESS-Reality"
    type: vless
    server: 107.172.87.203
    port: 19348
    uuid: cce0c2f5-c9cf-4d71-8754-b98b45f6dd36
    network: tcp
    tls: true
    udp: true
    flow: xtls-rprx-vision
    servername: www.apple.com
    client-fingerprint: chrome
    reality-opts:
      public-key: xfj0LI42PfKawX5RDl6wZJTrdE5XNPqkZ7ZLLGL9gQE
      short-id: 8cabcc6a

  # 节点 2：VLESS + Reality + XHTTP（强力内网穿透 / 流式 HTTP）
  - name: "🇺🇸 02-USA-VLESS-XHTTP"
    type: vless
    server: 107.172.87.203
    port: 41304
    uuid: cce0c2f5-c9cf-4d71-8754-b98b45f6dd36
    network: http
    tls: true
    udp: true
    servername: www.apple.com
    client-fingerprint: chrome
    http-opts:
      path: "/kxgjxhttp"
      headers:
        Host: ["www.apple.com"]
    reality-opts:
      public-key: xfj0LI42PfKawX5RDl6wZJTrdE5XNPqkZ7ZLLGL9gQE
      short-id: 8cabcc6a

  # 节点 3：Hysteria 2（UDP 自研拥塞算法，晚高峰极速抗丢包）
  - name: "🇺🇸 03-USA-Hysteria2"
    type: hysteria2
    server: 107.172.87.203
    port: 48618
    password: uM2TEVV2AH
    sni: usa.proxy.local
    skip-cert-verify: true
    up: "100 Mbps"
    down: "100 Mbps"

  # 节点 4：Tuic v5（QUIC 多路复用，超低连接时延）
  - name: "🇺🇸 04-USA-Tuic-v5"
    type: tuic
    server: 107.172.87.203
    port: 17285
    uuid: cce0c2f5-c9cf-4d71-8754-b98b45f6dd36
    password: uM2TEVV2AH
    congestion-controller: cubic
    sni: usa.proxy.local
    skip-cert-verify: true
    version: 5
    alpn:
      - h3


proxy-groups:
  - name: 🚀 节点选择
    type: select
    proxies:
      - ⚡ 自动选择
      - 🇺🇸 01-USA-VLESS-Reality
      - 🇺🇸 02-USA-VLESS-XHTTP
      - 🇺🇸 03-USA-Hysteria2
      - 🇺🇸 04-USA-Tuic-v5

  - name: ⚡ 自动选择
    type: url-test
    url: http://www.gstatic.com/generate_204
    interval: 300
    proxies:
      - 🇺🇸 01-USA-VLESS-Reality
      - 🇺🇸 02-USA-VLESS-XHTTP
      - 🇺🇸 03-USA-Hysteria2
      - 🇺🇸 04-USA-Tuic-v5

rules:
  - MATCH,🚀 节点选择
```

---

### 4.3 Sing-box 官方客户端 JSON 格式（Outbounds 节点定义）：

```json
[
  {
    "type": "vless",
    "tag": "01-USA-VLESS-Reality",
    "server": "107.172.87.203",
    "server_port": 19348,
    "uuid": "cce0c2f5-c9cf-4d71-8754-b98b45f6dd36",
    "flow": "xtls-rprx-vision",
    "tls": {
      "enabled": true,
      "server_name": "www.apple.com",
      "reality": {
        "enabled": true,
        "public_key": "xfj0LI42PfKawX5RDl6wZJTrdE5XNPqkZ7ZLLGL9gQE",
        "short_id": "8cabcc6a"
      },
      "utls": { "enabled": true, "fingerprint": "chrome" }
    }
  },
  {
    "type": "vless",
    "tag": "02-USA-VLESS-XHTTP",
    "server": "107.172.87.203",
    "server_port": 41304,
    "uuid": "cce0c2f5-c9cf-4d71-8754-b98b45f6dd36",
    "transport": {
      "type": "http",
      "host": ["www.apple.com"],
      "path": "/kxgjxhttp"
    },
    "tls": {
      "enabled": true,
      "server_name": "www.apple.com",
      "reality": {
        "enabled": true,
        "public_key": "xfj0LI42PfKawX5RDl6wZJTrdE5XNPqkZ7ZLLGL9gQE",
        "short_id": "8cabcc6a"
      },
      "utls": { "enabled": true, "fingerprint": "chrome" }
    }
  },
  {
    "type": "hysteria2",
    "tag": "03-USA-Hysteria2",
    "server": "107.172.87.203",
    "server_port": 48618,
    "password": "uM2TEVV2AH",
    "tls": {
      "enabled": true,
      "server_name": "usa.proxy.local",
      "insecure": true
    }
  },
  {
    "type": "tuic",
    "tag": "04-USA-Tuic-v5",
    "server": "107.172.87.203",
    "server_port": 17285,
    "uuid": "cce0c2f5-c9cf-4d71-8754-b98b45f6dd36",
    "password": "uM2TEVV2AH",
    "congestion_control": "cubic",
    "tls": {
      "enabled": true,
      "server_name": "usa.proxy.local",
      "insecure": true
    }
  },
  {
    "type": "anytls",
    "tag": "05-USA-AnyTLS",
    "server": "107.172.87.203",
    "server_port": 31335,
    "password": "uM2TEVV2AH",
    "tls": {
      "enabled": true,
      "server_name": "usa.proxy.local",
      "insecure": true
    }
  }
]
```

---

## 5. 当前已生效的 WARP 智能分流规则（两段式出口）

在 Sing-box 底层已配置好对 **OpenAI、Claude、Netflix、Disney+、Google AI** 的自动分流：

```json
{
  "outbounds": [
    { "type": "direct", "tag": "direct" },
    { "type": "socks", "tag": "warp-out", "server": "127.0.0.1", "server_port": 31303 }
  ],
  "route": {
    "rules": [
      { "action": "sniff" },
      { "protocol": ["dns"], "action": "hijack-dns" },
      {
        "domain_suffix": [
          "openai.com",
          "chatgpt.com",
          "oaistatic.com",
          "oaiusercontent.com",
          "anthropic.com",
          "claude.ai",
          "gemini.google.com",
          "aistudio.google.com",
          "notebooklm.google",
          "ai.google.dev",
          "generativelanguage.googleapis.com",
          "netflix.com",
          "netflix.net",
          "nflximg.net",
          "nflxvideo.net",
          "nflxext.com",
          "nflxso.net",
          "disneyplus.com"
        ],
        "outbound": "warp-out"
      }
    ]
  }
}
```

- **访问日常网站（YouTube/Twitter/GitHub）**：走 VPS 默认 IP 直连，享受最低延迟；
- **访问 OpenAI / Claude / Gemini / Netflix**：自动切换到本地 WARP SOCKS5（`127.0.0.1:31303`）发出，纯净解封。

---

## 6. Subconverter 订阅转换与自建实战

由于 S-UI 原生内置的 `format=clash` 转换器对现代协议（Reality / Hysteria 2 / Tuic）的生成支持有限，业界最通用的做法是借助 **Subconverter** 将通用订阅转换为标准 Clash Meta / Mihomo 配置文件。

---

### 6.1 现成可用订阅链接（直接导入客户端）

#### 👉 A. 自建专属全能分流版（强烈推荐 ⭐⭐⭐⭐⭐，完全复刻 `v2ray-agent` 工业级分组标准）

基于你在 VPS（`107.172.87.203:25500`）自建的专属 Subconverter 服务，**默认已加载 `v2ray-agent` 规则模板**，无需在 URL 后面附带冗长的 `&config=`：

```text
http://107.172.87.203:25500/sub?target=clash&url=http%3A%2F%2F107.172.87.203%3A2096%2Fsub_eggbr%2Feggbr
```

- **自动生成的纯净策略组清单（100% 对齐 `v2ray-agent` 规范）**：
  - `手动切换`（手动点选指定节点）
  - `自动选择`（自动测速优选低延迟节点）
  - `全球代理`（所有未识别的海外流量）
  - `Google`（Google 搜索 / Play / Gemini 专属分流）
  - `YouTube`（YouTube 视频流）
  - `OpenAI`（ChatGPT / OpenAI API）
  - `ClaudeAI`（Anthropic Claude）
  - `Telegram`（电报通信）
  - `Netflix` / `Disney` / `Spotify` / `HBO`（流媒体专属）
  - `GitHub` / `Bing`（开发者服务）
  - `国内媒体` / `本地直连`（国内百度/淘宝等直连不耗流量）
  - `漏网之鱼`（终极兜底策略）

---


---

### 6.2 Subconverter 订阅链接通用拼接公式与参数字典

你可以根据需要自由组装任何订阅链接，通用拼接公式如下：

#### 🔗 通用拼接公式：
```text
{转换后端API}/sub?target={目标客户端}&url={URL编码后的订阅源}&insert={false}&emoji={true}&config={URL编码后的规则模板}
```

#### 📋 核心传参字典速查表：

| 参数项 (Key) | 推荐取值 / 示例 | 作用与技术含义 |
| :--- | :--- | :--- |
| **转换后端 API** | `https://api.v1.mk/sub` | 运行 Subconverter 核心引擎的服务器接口 |
| **`target`** *(必填)* | `clash`、`singbox`、`surge`、`v2ray` | **目标客户端格式**。例如 `clash` 生成标准 Clash YAML，`singbox` 生成 JSON。 |
| **`url`** *(必填)* | `http%3A%2F%2F...` | **原始订阅源地址**（URL 编码后）。支持用 `%7C`（即 `|`）拼接多个订阅或单节点链接。 |
| **`config`** | `https%3A%2F%2Fraw.github...` | **远程分流规则模板**（URL 编码后）。决定生成哪些策略组（如谷歌、AI、油管等）。 |
| **`insert`** | `false` | **是否插入服务商推广广告节点**。设为 `false` 保证节点纯净。 |
| **`emoji`** | `true` | **是否自动匹配国旗 Emoji**。设为 `true` 自动在节点前添加 🇺🇸、🇭🇰、🇯🇵 等图标。 |
| **`list`** | `false` | **输出模式**。`false` 生成带策略组与规则的完整配置；`true` 仅输出节点列表。 |
| **`udp`** | `true` | **强制开启 UDP 转发**。为所有节点开启 UDP 支持。 |
| **`scv`** | `true` | **跳过证书校验**（Skip Cert Verify）。用于自签名证书节点。 |
| **`include`** | `美国\|USA` | **节点筛选（仅保留）**。支持正则，只保留匹配名称的节点。 |
| **`exclude`** | `过期\|剩余` | **节点排除**。排除名称中包含特定关键字的节点。 |
| **`filename`** | `USA-Proxy.yaml` | **下载文件名**。指定客户端下载该订阅时的默认保存文件名。 |

---

### 6.3 常用权威分流规则模板库（ACL4SSR / Blackmatrix7）

分流模板存放在各大权威开源仓库中，可直接将其 URL 编码后填入 `config=` 参数：

| 规则模板名称 | 远程模板 URL 地址 | 包含特色策略组 |
| :--- | :--- | :--- |
| **ACL4SSR Google 版 (最推荐 ⭐⭐⭐⭐⭐)** | `https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/config/ACL4SSR_Online_Full_Google.ini` | 独立 **Google、Ai平台、YouTube、Telegram、Netflix、广告拦截** |
| **ACL4SSR 全分组多模式版** | `https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/config/ACL4SSR_Online_Full_MultiMode.ini` | 增加 **负载均衡、故障转移、多地区测速** |
| **ACL4SSR 极简白名单版** | `https://raw.githubusercontent.com/ACL4SSR/ACL4SSR/master/Clash/config/ACL4SSR_Online_Mini.ini` | 极简，仅国外媒体与国内直连 |

- **GitHub 官方规则仓库**：
  - 👉 [ACL4SSR/ACL4SSR](https://github.com/ACL4SSR/ACL4SSR)（Subconverter 规则模板）
  - 👉 [blackmatrix7/ios_rule_script](https://github.com/blackmatrix7/ios_rule_script)（全球最全模块化原子规则）
  - 👉 [Loyalsoldier/clash-rules](https://github.com/Loyalsoldier/clash-rules)（权威 GFW 与国内外直连基准）

---

### 6.4 如何自建专属的 Subconverter 订阅转换服务（防泄露最安全）

使用公共转换后端（如 `api.v1.mk`）存在将 VPS 节点信息暴露给第三方的风险。在自己的 VPS 上自建 Subconverter 服务是最佳方案：

#### 🚀 方法一：Docker 一键部署（最推荐 ⭐⭐⭐⭐⭐）

```bash
# 1. 运行 Subconverter 官方转换后端容器 (监听 25500 端口)
docker run -d \
  --name subconverter \
  --restart=always \
  -p 25500:25500 \
  tindy2013/subconverter:latest

# 2. (可选) 运行配套的前端可视化配置网页 sub-web (监听 25580 端口)
docker run -d \
  --name sub-web \
  --restart=always \
  -p 25580:80 \
  -e SUB_BACKEND="http://107.172.87.203:25500" \
  careywong/sub-web:latest
```

- **自建后端使用**：将 `https://api.v1.mk/sub?...` 替换为 `http://107.172.87.203:25500/sub?...` 即可。
- **自建前端网页**：浏览器访问 `http://107.172.87.203:25580`，可视化勾选生成！

---

### 6.5 如何自行在线转换并获取全新的 Clash YAML 源码（实操教程）

1. **准备节点链接**：复制你的节点标准链接（以 `vless://`、`hysteria2://`、`tuic://` 开头）；
2. **打开在线转换平台**：浏览器访问 👉 `https://acl4ssr-sub.github.io/`（或 `https://sub.xeton.dev/`）；
3. **填入节点信息**：
   - 粘贴进 **「订阅链接」** 输入框中；
   - **「客户端」** 选择 **`Clash`**；
   - **「远程配置」** 建议选择 **`ACL4SSR_Online_Full_Google`**；
4. **生成并复制链接**：点击 **「生成订阅链接」** 并复制该长链接；
5. **提取完整 YAML 配置文件**：
   - **直接将该链接粘贴到浏览器地址栏中按回车打开**；
   - 浏览器页面显示的整篇纯文本就是完整的 Clash YAML 配置；
   - 按 `Ctrl + A`（全选）➔ `Ctrl + C`（复制）保存为本地文件即可！

---

### 6.6 TUIC v5 协议版本与 Clash 握手兼容性要点（避坑指南 ⚠️）


#### 1. 核心问题根源：
- **Sing-box 原生使用的是最新的【TUIC v5】标准**；
- 在线转换器（Subconverter）根据 `tuic://` 链接生成 Clash Meta (Mihomo) 配置时，如果原始链接没有声明版本，转换器默认会漏掉 `version: 5` 和 `alpn: [h3]`；
- **Clash Meta 缺少 `version: 5` 时会默认按旧版【TUIC v4】去握手**，导致两端协议版本不兼容被直接拒绝连接。

#### 2. 规范配置解决方案：
- **在单节点 URI 中携带参数（从源头解决 ⭐⭐⭐⭐⭐）**：
  必须在 `tuic://` 链接末尾明确附加 `&version=5&alpn=h3`：
  ```text
  tuic://UUID:PASSWORD@107.172.87.203:17285?security=tls&insecure=1&sni=usa.proxy.local&congestion_control=cubic&version=5&alpn=h3#USA-Tuic-v5
  ```
  *(注：当前 S-UI 数据库中客户端 `eggbr` 的链接已在服务端配置此参数，通过 Subconverter 转换时已全自动注入 `alpn: ['h3']`)*。

- **在 Clash 本地 YAML 中显式声明**：
  ```yaml
  - name: "🇺🇸 04-USA-Tuic-v5"
    type: tuic
    server: 107.172.87.203
    port: 17285
    uuid: cce0c2f5-c9cf-4d71-8754-b98b45f6dd36
    password: uM2TEVV2AH
    congestion-controller: cubic
    sni: usa.proxy.local
    skip-cert-verify: true
    version: 5            # 👈 核心：必须显式声明为 v5
    alpn:
      - h3                # 👈 核心：必须声明 QUIC h3 协商
  ```

---



## 7. 常用运维管理命令

```bash
# 1. 【一键导出全部节点链接】(无需打开网页，终端直接打印所有 vless/hy2/tuic/anytls 链接)
sui-export

# 2. 查看 S-UI 运行状态
s-ui status
# 或
systemctl status s-ui

# 3. 重启 S-UI 服务
s-ui restart

# 4. 查看 Sing-box 实时运行日志
s-ui log

# 5. 查看/修改 S-UI 管理员密码
/usr/local/s-ui/sui admin -show
/usr/local/s-ui/sui admin -username admin -password <新密码>
```