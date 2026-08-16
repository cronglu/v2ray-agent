# USA 服务器配置与 3X-UI 运维管理档案

本文档记录了 `usa` 服务器（RackNerd VPS）的 3X-UI 历史配置、面板访问凭证、内核优化参数、服务清单、WARP 状态、以及 **3X-UI 下的全套节点参数、Subconverter 订阅转换与客户端导入指南**。

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

## 2. 3X-UI 可视化管理面板配置参考

3X-UI（v3.6.0）基于 **Xray-core** 驱动。

| 配置项 | 详细信息 |
| :--- | :--- |
| **面板访问 URL** | 👉 `http://107.172.87.203:6313/vSoayE7luL0REsERN7/` |
| **默认登录账号** | `s9yug5Exm2` |
| **默认登录密码** | `hA78j96wir` |
| **面板监听端口** | `6313` |
| **URL 基础路径** | `/vSoayE7luL0REsERN7/` |
| **聚合订阅服务端口** | `2096` |
| **数据库路径** | `/etc/x-ui/x-ui.db`（SQLite 存储） |
| **API Token** | `C2YxjXTgTWu96viuUGctK9oX3ug28rvZJrLKeBREreaya7Ol` |

---

## 3. 3X-UI 节点参数与客户端导入配置

### 3.1 核心连接参数表 (VLESS + Reality + Vision)

| 参数名 | 填入值 / 配置 |
| :--- | :--- |
| **节点名称** | `🇺🇸 USA-VLESS-Reality-Vision` |
| **服务器地址 (Server)** | `107.172.87.203` |
| **端口 (Port)** | `40338` |
| **协议 (Protocol)** | `vless` |
| **用户 ID (UUID)** | `9fd85f55-a29a-4dce-8956-3c3bee91fd72` |
| **流控 (Flow)** | `xtls-rprx-vision` |
| **传输协议 (Network)** | `tcp` |
| **TLS / 安全** | `reality` |
| **伪装域名 (SNI)** | `www.apple.com` |
| **Public Key (公钥)** | `wEgL9gxdk0K_HmcSBmWCpKi_8CCVrgSEi4BZ-Apf7V4` |
| **Short ID** | `781ea893` |
| **客户端指纹 (Fingerprint)** | `chrome` |

---

### 3.2 客户端标准导入格式

#### ① VLESS 快捷导入链接：
```text
vless://9fd85f55-a29a-4dce-8956-3c3bee91fd72@107.172.87.203:40338?flow=xtls-rprx-vision&fp=chrome&pbk=wEgL9gxdk0K_HmcSBmWCpKi_8CCVrgSEi4BZ-Apf7V4&security=reality&sid=781ea893&sni=www.apple.com&spx=%2Fc88b1a2067702b7&type=tcp#%F0%9F%87%BA%F0%9F%87%B8%20USA-VLESS-Reality-Vision
```

#### ② Clash Meta / Mihomo 本地配置文件片段：
```yaml
proxies:
  - name: "🇺🇸 USA-VLESS-Reality-Vision"
    type: vless
    server: 107.172.87.203
    port: 40338
    uuid: 9fd85f55-a29a-4dce-8956-3c3bee91fd72
    network: tcp
    tls: true
    udp: true
    flow: xtls-rprx-vision
    servername: www.apple.com
    client-fingerprint: chrome
    reality-opts:
      public-key: wEgL9gxdk0K_HmcSBmWCpKi_8CCVrgSEi4BZ-Apf7V4
      short-id: 781ea893
```

---

### 3.3 聚合订阅与 Clash 订阅转换详解

#### ① 3X-UI 原生订阅源（Base64 纯文本格式）
- **原生订阅 URL**：
  `http://107.172.87.203:2096/sub/e4f38f4db1d4c4e7`

#### ② Clash 专属自动转换订阅 URL（直接导入 Clash Verge 成功拉取 ⭐⭐⭐⭐⭐）
由于 Clash 客户端只认 YAML 配置文件格式，直接填入原生 Base64 订阅会提示 `invalid yaml` 错误。因此需要通过 Subconverter 转换格式：

- **Clash 专属订阅链接**：
  ```text
  https://api.v1.mk/sub?target=clash&url=http%3A%2F%2F107.172.87.203%3A2096%2Fsub%2Fe4f38f4db1d4c4e7&insert=false
  ```

#### ③ 订阅转换链接的参数结构拆解：
```text
https://api.v1.mk/sub ? target=clash & url=http%3A%2F%2F107.172.87.203... & insert=false
       │                      │                                 │                    │
  【转换后端 API】      【目标转为 Clash 格式】       【你的 3X-UI 原始订阅地址】    【不插入额外多余规则】
```

1. **`https://api.v1.mk/sub`**：开源 Subconverter 订阅转换服务的后端 API 接口；
2. **`target=clash`**：声明目标输出为标准 Clash YAML 格式（自动生成 `proxies:`、`proxy-groups:` 与基础规则）；
3. **`url=http://107.172.87.203:2096/sub/...`**：指向你 VPS 上的 3X-UI 原始订阅地址（已做标准 URL 编码）；
4. **`insert=false`**：指示转换器不要强行插入第三方的拦截规则，保持纯净直出。

---

## 4. 3X-UI 套 WARP 智能分流配置（两段式出口）

在 3X-UI 的「面板设置」➔「Xray 配置」中，添加以下出站与路由规则：

```json
// 1. 在 outbounds 数组中添加 WARP 出口
{
  "tag": "warp-out",
  "protocol": "socks",
  "settings": {
    "servers": [
      {
        "address": "127.0.0.1",
        "port": 31303
      }
    ]
  }
}
```

```json
// 2. 在 routing.rules 规则数组头部添加分流策略
[
  {
    "type": "field",
    "outboundTag": "warp-out",
    "domain": [
      "geosite:openai",
      "geosite:netflix",
      "geosite:disney",
      "domain:anthropic.com",
      "domain:claude.ai"
    ]
  },
  {
    "type": "field",
    "outboundTag": "direct",
    "network": "tcp,udp"
  }
]
```
