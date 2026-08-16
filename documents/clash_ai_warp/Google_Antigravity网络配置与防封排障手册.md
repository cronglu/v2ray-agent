# Google & Antigravity 网络配置、防封风控与凭证排障全景手册

针对 Google 生态及主流 AI 编程/模型工具（Google Antigravity、Gemini API、Google Workspace、Claude、OpenAI 等），Google 等服务商对**会话 IP 稳定性、地理位置一致性、IPv6 泄漏与长连接保活**有着极其严苛的风控检测机制。

本文档整合了 **Google / AI 代理核心原则、Node.js 代理回退机制分析、Clash/Mihomo 防封调优技巧、两段式/链式进阶代理架构、Antigravity 凭证清理与重置、以及网络诊断自检与故障速查表**，形成一站式运维指南。

---

## 1. 核心原理与 IP 异常风控根因剖析

### 1.1 Google & AI 服务代理的核心四原则
1. **绝对单一出口 IP（No IP Drifting）**：单个鉴权 Session 严禁在多个出口 IP 间频繁跳跃。
2. **零 IPv6 泄漏（No IPv6 Leak）**：必须杜绝「IPv4 走代理、IPv6 走运营商直连」的双栈冲突。
3. **精准分流无遗漏（Complete Rule Coverage）**：Google 认证、模型推理、语言服务器必须走同一组代理规则，避免部分直连、部分代理。
4. **防 DNS 污染与解析泄漏（Fake-IP & Remote DNS）**：域名解析必须交由远端代理节点完成。

### 1.2 故障现象与多 IP 触发机理
- **常见现象**：Antigravity 频繁提示 IP 有问题 / 风险警告；IDE 认证状态偶发失效或频繁要求二次验证。
- **根因分析**：Antigravity 服务端（VS Code Server / Node.js 架构）启动参数默认携带 `--use-host-proxy`，会自动读取环境变量中的 HTTP/HTTPS 代理配置（如 `http://127.0.0.1:7897`）。

当本地代理发生网络抖动或 TLS 握手超时时：
```text
[正常走代理]
Antigravity 发起请求 -> 代理 127.0.0.1:7897 -> 代理出口节点 IP -> Google 认证服务

[代理抖动 / 超时]
Antigravity 发起请求 -> 代理连接超时 -> Node.js 部分网络库 Fallback 直连 -> 服务器真实公网 IP (如 Oracle 出口 IP) -> Google 认证服务
```
**结果**：Google 认证服务检测到同一个鉴权 Session 在短时间内来自多个不同的公网 IP，触发安全风控机制并弹出 IP 异常警告。

### 1.3 Node.js / VS Code 代理 Fallback 机制深度拆解
1. **Chromium / VS Code 网络栈设计**：
   VS Code 底层基于 Electron / Chromium 网络服务，以及 Node.js 的 `ProxyResolver`。其设计哲学通常是**优先保障可用性（Fail-open）**，在代理握手发生 `ECONNREFUSED` 或 `ETIMEDOUT` 时，默认策略往往尝试 `DIRECT`（直连），防止应用彻底断网。
2. **三方 HTTP Client 行为差异**：
   Node.js 社区的各类网络库（如 `https-proxy-agent`、`undici`、`node-fetch`、`axios`）对代理超时的处理各不相同。部分客户端在代理层握手失败后不会直接报错中断，而是尝试直连重试，导致请求在用户无感知的情况下切换了出口 IP。

---

## 2. 多层防御体系与防 Fallback 配置策略

### 2.1 防御方案矩阵对比

| 层次 | 解决手段 | 适用场景 | 效果 |
| :--- | :--- | :--- | :--- |
| **环境变量层（推荐）** | 配置 `no_proxy` 加入 Google 域名 | 服务器在海外（如美国 Oracle / AWS） | **根本消除**：请求从第一步就直接判定直连，永不经过代理，出口 IP 恒定。 |
| **IDE / 配置层** | 设置 `"http.proxySupport": "off"` 或严格代理 | VS Code / Antigravity IDE | 关闭代理介入，彻底杜绝回退切换。 |
| **代码实现层** | 使用 `ProxyAgent` 显式分发并强制报错 | 自研 Node.js 脚本/工具 | 代理失败直接 throw error，杜绝隐式直连重试。 |
| **系统防火墙层** | iptables 拦截非代理用户出站 443 流量 | 服务器在国内、严防真实 IP 泄露 | 物理级硬阻断，一旦 Fallback 直连立即被防火墙掐断。 |

### 2.2 海外服务器场景：环境变量 `no_proxy` 白名单（最推荐）
如果服务器公网 IP 本身在海外（如美国 Oracle / AWS 节点），可直接将 Google API 加入 `no_proxy` 白名单，绕过本地不稳定代理，直接使用优质海外机房原生网络：

编辑 `/root/.bashrc` 或 `/root/.profile`：
```bash
# 添加 Google 域名至不走代理列表
export no_proxy="$no_proxy,googleapis.com,*.googleapis.com,google.com,*.google.com"
export NO_PROXY="$NO_PROXY,googleapis.com,*.googleapis.com,google.com,*.google.com"
```
使环境变量生效：
```bash
source /root/.profile
# 或
source /root/.bashrc
```

### 2.3 代码层实现示例（Strict Proxy 模式）
对于自定义脚本或服务，防止代理挂掉后隐式直连泄露本地 IP：
```javascript
import { ProxyAgent, fetch } from 'undici';

// 显式绑定代理 Dispatcher，代理异常时直接报错，绝不隐式直连
const proxyAgent = new ProxyAgent('http://127.0.0.1:7897');

try {
  const res = await fetch("https://generativelanguage.googleapis.com/v1beta/models", {
    dispatcher: proxyAgent
  });
} catch (err) {
  // 代理不可用时抛出异常，阻止使用本地真实 IP 发起请求
  console.error("代理连接失败，已安全终止请求:", err);
}
```

### 2.4 系统防火墙硬拦截（iptables 示例）
```bash
# 仅允许代理进程专用 UID 发起外部 443 连接，其余进程直连全部拦截
iptables -A OUTPUT -p tcp --dport 443 -m owner ! --uid-owner proxy_user -j REJECT
```

---

## 3. Clash / Mihomo 核心配置与避坑五大实战

### 3.1 技巧一：创建专用的 Google / AI 独立策略组（严禁使用 AUTO / 负载均衡）

#### ❌ 常见误区：
将 Google 或全局规则指向 `AUTO`（自动测速 / url-test）或 `Load-Balance`（负载均衡）。
- **后果**：Clash 后台每隔几分钟测速一次，一旦新节点延迟稍低就会自动切换，导致长达数小时的编程/对话 Session 遭遇中途 IP 漂移，触发 Google 账号风控或强制掉线。

#### ✅ 正确做法：
为 Google 建立专属策略组，**固定选择单一优质静态节点**（手动选择 `select` 类型）：
```yaml
proxy-groups:
  # 专为 Google / AI 分配的策略组
  - name: "Google-AI"
    type: select
    proxies:
      - "🇺🇸 美国 01 [专线]"
      - "🇺🇸 美国 02 [静态]"
      - "🇸🇬 新加坡 01"
      - "DIRECT"
```

---

### 3.2 技巧二：TUN 模式核心参数调优

使用 TUN 虚拟网卡模式接管系统全局流量时，推荐以下参数组合（以 Clash Meta / Mihomo 为例）：

```yaml
tun:
  enable: true
  stack: mixed                  # mixed 兼顾性能与兼容性，若有异常可切换为 gvisor
  auto-route: true              # 自动设置系统默认路由
  auto-detect-interface: true   # 自动检测主物理网卡，防止路由环路
  dns-hijack:
    - "tcp://any:53"
    - "udp://any:53"
  strict-route: true            # 严格路由模式，阻止直连旁路流量漏出
```

---

### 3.3 技巧三：DNS 防污染与全局禁用 IPv6（终极防泄漏）

很多国内宽带（电信/联通/移动）已默认普及公网 IPv6。如果代理未配置拦截，客户端会优先发起 IPv6 直连，造成重大风控事故。

#### 关键配置（Clash 配置文件的 `dns` 字段）：
```yaml
dns:
  enable: true
  listen: :1053
  ipv6: false                   # 核心：彻底禁用 AAAA (IPv6) 解析，强制纯 IPv4 链路
  enhanced-mode: fake-ip        # 核心：使用 Fake-IP，避免本地 DNS 污染与解析延迟
  fake-ip-range: 198.18.0.1/16
  nameserver:
    - https://dns.google/dns-query
    - https://1.1.1.1/dns-query
  # 针对 Google 域名指定由海外加密 DNS 解析
  nameserver-policy:
    "geosite:google": "https://dns.google/dns-query"
    "geosite:openai": "https://1.1.1.1/dns-query"
```

---

### 3.4 技巧四：Linux / WSL2 双网卡宿主环境内核级禁用 IPv6

在 Windows + WSL2（或带有两张物理/虚拟网卡的 Linux 主机）环境下，物理网卡的 IPv6 默认路由优先级往往高于 TUN 网卡。

**路由冲突机理**：
```text
IPv4 路由：default via 198.18.0.2 dev eth0 (TUN 网卡) metric 256  -> 走代理（海外 IP）
IPv6 路由：default via fe80::... dev eth2 (物理网卡) metric 281    -> 直连物理出口（中国运营商 IPv6）
```

#### 必须在 Linux/WSL 内执行系统级关闭：
```bash
# 1. 立即关闭系统 IPv6 栈
sysctl -w net.ipv6.conf.all.disable_ipv6=1
sysctl -w net.ipv6.conf.default.disable_ipv6=1
sysctl -w net.ipv6.conf.lo.disable_ipv6=1
sysctl -w net.ipv6.conf.eth2.disable_ipv6=1

# 2. 持久化至系统配置
cat << 'EOF' >> /etc/sysctl.conf
net.ipv6.conf.all.disable_ipv6 = 1
net.ipv6.conf.default.disable_ipv6 = 1
net.ipv6.conf.lo.disable_ipv6 = 1
net.ipv6.conf.eth2.disable_ipv6 = 1
EOF
sysctl -p
```

---

### 3.5 技巧五：Google & Antigravity 专属分流规则补全

Antigravity 底层依赖多个 Google 内部子系统（账号登录、核心模型推理、Language Server 插件通信）。确保分流规则完整覆盖所有端点，避免部分请求落入直连：

```yaml
rules:
  # Antigravity 核心与 Gemini API
  - DOMAIN-SUFFIX,generativelanguage.googleapis.com,Google-AI
  - DOMAIN-SUFFIX,cloudcode-pa.googleapis.com,Google-AI
  - DOMAIN-SUFFIX,alkalimakersuite-pa.googleapis.com,Google-AI
  
  # Google 账号体系与认证
  - DOMAIN-SUFFIX,accounts.google.com,Google-AI
  - DOMAIN-SUFFIX,oauth2.googleapis.com,Google-AI
  - DOMAIN-SUFFIX,googleusercontent.com,Google-AI
  - DOMAIN-SUFFIX,gstatic.com,Google-AI
  
  # Google 全生态规则集
  - GEOSITE,google,Google-AI
  - GEOIP,google,Google-AI
  
  # 兜底规则
  - MATCH,Google-AI   # 或 PROXY
```

---

## 4. 进阶网络架构：抗 GFW 封锁与 AI/流媒体解锁终极方案

要同时实现 **「抗 GFW 封锁（保证连接不断）」** 与 **「完美解锁 Google（不弹验证码/不报 IP 异常）、ChatGPT、Netflix（看非自制剧）、Claude」**，工业级方案推荐 **「前置抗封协议 + 服务端智能分流 WARP / 解锁落地」的两段式架构**。

```text
  [你的手机 / 电脑 (Clash)]
         │
         ▼  【第一段：抗封入口】 VLESS-Reality 或 VLESS-XHTTP (+Cloudflare CDN)
  [你的海外 VPS 主机]
         │
         ├── [普通流量：YouTube / GitHub / 网页] ──> 【直连 VPS 原生 IP】
         │
         ├── [AI 流量：ChatGPT / Claude / Google] ──> 【分流到本地 Cloudflare WARP SOCKS5】
         │
         └── [流媒体：Netflix / Disney+] ──────────> 【分流到 WARP IPv6 / 原生解锁落地】
```

### 4.1 方案 A：VPS 被墙时的救活方案（VLESS-XHTTP + Cloudflare CDN 小黄云）
如果你的 VPS 真实 IP 已经被 GFW 彻底阻断：
1. 在 Cloudflare 上将二级域名（如 `proxy.yourdomain.com`）解析到 VPS IP，并**开启 Proxy（点亮橙色小黄云）**。
2. 服务端开启 **`VLESS + XHTTP`**（或 WebSocket）。
3. 客户端连接 `proxy.yourdomain.com:443`：
   - 流量先到 Cloudflare 任意 Anycast 节点（在国内永远能连上），再由 Cloudflare 内网隧道转发至你的 VPS。
   - 实现 **零成本复活被封 IP**。

---

### 4.2 方案 B：前置中继 VPS + 廉价住宅/原生 IP 链式代理 (Chain Proxy)
如果部分极端严格的流媒体或银行风控识别了数据中心 IP：
```text
[国内客户端] ──(Hy2 / Reality 高速)──> [Oracle/优质 VPS (中继入口)] ──(SOCKS5 转发)──> [廉价原生住宅/家宽小鸡 (落地出口)] ──> [Google / Netflix / GPT]
```
- **入口 VPS**：只负责抗丢包、高吞吐（运行 Hysteria2 或 Reality）；
- **落地 VPS**：使用几块钱一个月的非知名原生 IP / 住宅代理，专门承载解封流量；
- **效果**：兼顾「国内连接极速」与「出口 100% 纯正海外家宽 IP」。

---

## 5. Antigravity 认证凭证清理与重置操作指南

Antigravity 的登录与配置信息保存在 `/root/.gemini/` 目录下。

### 5.1 备份并清理登录凭证
```bash
# 备份旧凭证（带时间戳）
if [ -f /root/.gemini/gemini-credentials.json ]; then
    mv /root/.gemini/gemini-credentials.json /root/.gemini/gemini-credentials.json.bak.$(date +%Y%m%d)
    echo "旧凭证已备份"
fi

# 如果需要彻底删除凭证
rm -f /root/.gemini/gemini-credentials.json
```

### 5.2 检查关联账户状态
```bash
# 查看账户状态
cat /root/.gemini/google_accounts.json

# 清理后账户文件应呈现未登录状态：
# {
#   "active": null,
#   "old": []
# }
```

### 5.3 （可选）清理历史会话与上下文缓存
若需要进一步释放磁盘空间或彻底清理历史对话记录：
```bash
# 查看各目录占用体积
du -sh /root/.gemini/antigravity/brain \
       /root/.gemini/antigravity/conversations \
       /root/.gemini/antigravity-ide/brain \
       /root/.gemini/antigravity-ide/conversations 2>/dev/null

# 清空历史对话会话（如需彻底清理）
# rm -rf /root/.gemini/antigravity/conversations/*
# rm -rf /root/.gemini/antigravity-ide/conversations/*
```

### 5.4 重新授权绑定新环境
在完成网络优化（锁定单一节点 / 配置 no_proxy / 禁用 IPv6）并清理旧凭证后：
1. 重启 Antigravity IDE 或重启 Language Server 进程。
2. 触发需要认证的功能，界面会引导重新发起登录/授权。
3. 新生成的凭证将与当前锁定的稳定单一出口 IP 强绑定，不再触发风控。

---

## 6. 网络连通性健康度自检与故障排查速查表

### 6.1 一键健康度与连通性自检命令集
每次配置或切换网络后，可在终端运行以下命令进行全链路诊断：

```bash
# 1. 验证直连/当前公网 IPv4 出口 IP 与归属地
curl -s --max-time 5 https://ipinfo.io/json | python3 -m json.tool | grep -E '"ip"|"country"|"region"|"org"'

# 2. 检查环境变量中的代理设置
env | grep -iE "proxy|http_proxy|https_proxy|all_proxy|no_proxy"

# 3. 验证指定代理端口（如 7897）的出口 IP 与地理位置
curl -s --max-time 5 --proxy http://127.0.0.1:7897 https://ipinfo.io/json | python3 -m json.tool | grep -E '"ip"|"country"|"region"|"org"'

# 4. 验证 IPv6 是否已安全阻断（应返回无法连接或超时，绝不能返回国内公网 IPv6）
curl -6 --max-time 3 https://ifconfig.co 2>&1 || echo "IPv6 阻断正常，无泄漏"

# 5. 验证 Google API 端点握手状态（返回 403 PERMISSION_DENIED 即表示链路 100% 连通）
curl -s -o /dev/null -w "Google API HTTP Code: %{http_code}\n" --max-time 5 https://generativelanguage.googleapis.com/v1beta/models

# 6. 验证 ChatGPT / OpenAI 连通性状态
curl -s -o /dev/null -w "OpenAI HTTP Code: %{http_code}\n" --max-time 5 https://api.openai.com/v1/models
```

---

### 6.2 常见问题排查速查表

| 故障现象 | 可能原因 | 解决办法 |
| :--- | :--- | :--- |
| **提示「IP 异常 / 频繁要求二次验证」** | 策略组选了 `AUTO` 节点或 IPv6 泄漏 | ① 策略组改为手动单选静态节点；② 禁用 Linux IPv6；③ 服务端开启 WARP 出口分流；④ 清理 `/root/.gemini/` 凭证并重新授权。 |
| **Antigravity 报 Connection Timeout (403/5001ms)** | 本地 7897 端口握手阻塞或节点质量差 | 更换为低延迟专线节点；海外服务器直接配 `no_proxy` 直连；检查 Clash 是否开启 TUN。 |
| **Node.js 请求偶发泄露真实 IP** | 网络抖动触发 Node.js 代理 Fail-open 直连 | 代码中使用 `ProxyAgent` 显式抛错，或通过 iptables 阻断非代理出站。 |
| **ChatGPT 报 Access Denied (403) / 拦截** | 机房 IP (如 Oracle/AWS) 被 OpenAI 风控 | 在 VPS 上配置 WARP SOCKS5 分流，OpenAI 流量走 WARP。 |
| **Netflix 只能看自制剧（无法看非自制剧）** | VPS IP 被 Netflix 识别为数据中心代理 | 开启 WARP IPv6 出口分流，解锁流媒体非自制剧版权。 |
| **部分网页能开，但 API 请求走直连** | 规则库遗漏了子域名或 Fake-IP 范围不生效 | 添加 `DOMAIN-SUFFIX,googleapis.com,Google-AI` 并开启 `enhanced-mode: fake-ip`。 |
| **海外 VPS/服务器访问速度慢** | 服务器本身在海外却套了本地代理 | 在海外机器上对 Google 配置 `no_proxy`，直接使用机房原生出口。 |
