# VPS (107.172.87.203) 节点排查与网络优化记录

## 1. 基础信息与免密配置
- **服务器地址**：`root@107.172.87.203`
- **SSH 免密状态**：已配置公钥 `~/.ssh/id_rsa_wsl`，本地已配置 `~/.ssh/config`，可通过 `ssh 107.172.87.203` 直接免密连接。
- **自有域名证书**：`www.eggbr.top`（Let's Encrypt ECC 256 位有效证书）。
- **Clash.Meta 订阅地址**：
  `https://www.eggbr.top:35270/s/clashMeta/37b99eef2aefae20a6aa1db4bb40ce00`

---

## 2. 节点超时排查与核心修复

### 2.1 Trojan+TCP[TLS] (端口 443)：Xray 回落规则与 ALPN 的逻辑冲突详解

#### ① vasma 脚本的原始回落设计意图
在 `vasma` 脚本的架构中，443 端口由 Xray 统一接管（VLESS-TCP-Vision 协议），其回落链条原本设计如下：
```text
客户端连接 443 端口 (TLS 握手解密)
  │
  ├─ 识别为 VLESS 协议流量 ──> [Xray VLESS 核心处理转发]
  │
  └─ 非 VLESS 流量 ──> 进入 fallbacks 回落规则判定：
        ├─ 规则 A：{"alpn": "h2", "dest": 31302} ──> [Nginx HTTP/2 网页]
        └─ 规则 B (无条件兜底)：{"dest": 31296} ──> [Xray Trojan 核心处理]
                                                        └─ 若密码错误 ──> [Nginx 31300 网页]
```
脚本作者原本假设：正常浏览器访问网站会走 HTTP/2（ALPN 为 `h2`），因此设置了规则 A 转发给 Nginx 伪装站；而 Trojan 客户端会走 HTTP/1.1 或无 ALPN，从而落入规则 B 的 Trojan。

#### ② 为什么会导致 Clash 连接 Trojan 100% 超时/断开？
1. **现代 Clash.Meta / Mihomo 的指纹模拟机制**：
   Clash 配置中默认启用了客户端指纹模拟（如 `client-fingerprint: chrome`），Go 语言的 TLS 栈在握手 ClientHello 时会自动携带 `ALPN: ["h2", "http/1.1"]`。
2. **Xray 的 Fallbacks 匹配优先级**：
   TLS 握手协商成功后，协商出的应用层协议为 `h2`。Xray 在回落列表中优先匹配具有特定条件（`alpn: "h2"`）的规则 A，**导致 Xray 将 Trojan 的加密数据流直接扔给了 31302 端口上的 Nginx HTTP/2 引擎**。
3. **协议解析崩溃**：
   Nginx 31302 端口期待的是标准 HTTP/2 连接序言（Connection Preface `PRI * HTTP/2.0...`），而客户端发送的是 Trojan 协议头（`56 字节 Hex 密码 Hash + \r\n + 目标地址端口`）。Nginx 判定为非法 HTTP/2 帧，直接发送 `GOAWAY / PROTOCOL_ERROR` 并重置（RST）连接。
4. **客户端现象**：
   Clash 握手刚完成就被服务端 Nginx 强行掐断，在 Clash 控制台报错表现为 `connection reset by peer` 或持续处于 `dial tcp timeout`（超时）。

#### ③ 正确的修复方案
- 修改 `/etc/v2ray-agent/xray/conf/02_VLESS_TCP_inbounds.json`：
  **彻底移除 `{"alpn": "h2", "dest": 31302}` 规则**，只保留无条件回落 `{"dest": 31296, "xver": 1}`。
- **运行逻辑恢复闭环**：
  所有非 VLESS 流量无条件直接给 31296 端口的 Trojan。Trojan 会对密码做校验：
  - 密码正确：作为 Trojan 正常代理转发（无论客户端 ALPN 是 `h2` 还是 `http/1.1` 均完美支持）。
  - 密码错误（如外部探针或普通浏览器访问）：由 Trojan 自身的 fallback 规则转发给 31300 端口的 Nginx 展示正常伪装网页。

### 2.2 TUIC v5 (UDP 20185)
- **超时根因**：脚本在生成 ClashMeta 订阅时存在 Bug，写入了 `disable-sni: true` 并将 SNI 错填为节点 tag 名 `0c3df94e-singbox_tuic`，导致客户端在建立 QUIC TLS 握手时无法完成对服务端域名 `www.eggbr.top` 的证书验证而超时。
- **修复方案**：已修正订阅文件及脚本模板 `/etc/v2ray-agent/install.sh`，将 SNI 纠正为 `www.eggbr.top` 并开启有效 SNI。

### 2.3 Hysteria2 (UDP 20505 / 端口跳跃 30000-40000)
- **超时/卡顿根因**：
  1. Linux 系统默认内核 UDP 缓冲区（`rmem_max`/`wmem_max` 仅 212 KB）过小，QUIC 高速爆发流量导致内核层严重丢包。
  2. 国内部分运营商对单一固定 UDP 端口（如 20505）进行 QoS 限速或断流。
- **修复方案**：
  1. 内核 UDP 缓冲区调优：`net.core.rmem_max` 和 `wmem_max` 调整为 **16 MB**，并保持开启 **BBR + fq** 拥塞控制。
  2. 启用端口跳跃 (Port Hopping)：配置 `iptables` 将 UDP 端口段 `30000:40000` 映射转发到 `20505` 并持久化保存。订阅中已同步新增支持端口跳跃的 Hysteria2 节点。

### 2.4 VLESS+Reality+XHTTP (端口 22261) 停用说明
- **为什么容易超时**：
  XHTTP（基于 splitHTTP / 分块流传输）采用双向 HTTP POST/GET 分块数据流机制。在国内多数运营商网络及中间审查节点（Middlebox）环境下，HTTP/2 分块流长连接极易被代理缓存拦截、重组延迟或直接截断缓冲；同时 Clash.Meta / Mihomo 对 XHTTP 的流式重连兼容性不如纯 TCP/UDP 传输层协议成熟，常出现上传流卡死和握手超时。
- **操作结果**：
  已彻底**关闭并下线**该协议配置（端口 22261），并从所有订阅中剔除，减少服务端开销与端口暴露。

---

## 3. WARP 分流策略 (Google AI / CDN 解锁)

- **分流路由规则**：
  - **WARP IPv4 (Cloudflare Clean IP) 出口**：
    - `geosite:google`（Google 全站搜索与服务）
    - `geosite:openai`（ChatGPT / OpenAI 接口）
    - `googleapis.com` / `gstatic.com`（Google API / CDN 资源）
    - `google.com` / `googleai.dev` / `googleusercontent.com` / `googlevideo.com`（Gemini、AI Studio、Google AI 搜索）
  - **VPS 原生直连出口**：
    - 国内直连流量及其他常规国外网站，保障最低延迟与最大带宽。

---

## 4. 证书与 SNI 区别说明

| 协议类型 | 节点名称 | 状态 | 伪装/证书 SNI | 原理说明 |
| :--- | :--- | :--- | :--- | :--- |
| **VLESS Reality** | `vless_reality_vision` (15393) | 正常开启 | `itunes.apple.com` | Reality 协议无需本地私钥，借用第三方苹果服务器 TLS 握手特征做混淆。 |
| **Trojan+TLS** | `trojan_tcp` (443) | 正常开启 | `www.eggbr.top` | 标准 TLS 协议，服务端必须持有域名的私钥完成签名，必须使用自有域名。 |
| **Hysteria2** | `singbox_hysteria2` (20505/30000-40000) | 正常开启 | `www.eggbr.top` | 基于 QUIC / TLS 1.3，支持端口跳跃，必须使用拥有私钥的自有域名证书。 |
| **TUIC v5** | `singbox_tuic` (20185) | 正常开启 | `www.eggbr.top` | 基于 QUIC / TLS 1.3，同样必须使用拥有私钥的自有域名证书。 |
| **VLESS XHTTP** | `VLESS_Reality_XHTTP` (22261) | **已关闭停用** | - | 分块流在部分运营商网络及客户端下易丢包超时，已彻底下线。 |

---

## 5. 待办事项 (Todo Checklist)

- [x] 配置 SSH 免密登录 (`root@107.172.87.203`)
- [x] 修复 Xray 443 端口 VLESS Fallback 冲突导致的 Trojan 超时
- [x] 修复 TUIC 订阅模板 SNI 错误
- [x] 优化 Linux 内核 UDP 参数 (16MB buffer) 与 BBR 算法
- [x] 配置 Hysteria2 iptables UDP 端口段映射 (30000-40000)
- [x] 将 Google API / CDN / Google AI 完整加入 WARP WireGuard 出口分流
- [x] 关闭并下线不稳定的 VLESS+Reality+XHTTP 协议 (端口 22261)
- [ ] **客户端操作**：在 Clash / Clash.Meta / Mihomo 中点击 **「更新订阅」**，刷新节点配置并测试连通性。
