# xraycli (v2ray-agent Go 重构版) 技术与架构档案 (Technical & Feature Archive)

本文档记录 `xraycli` 的核心技术选型、架构决策、协议拓扑模型以及高阶特性设计。

---

## 1. 技术栈与架构选型 (Technical Stack)

| 层次 | 技术选型 | 选型理由 |
| :--- | :--- | :--- |
| **开发语言** | Go 1.24+ | 静态编译单一二进制、跨平台交叉编译、原生高并发协程支持、零外部脚本依赖 |
| **标准库** | `net/http`, `embed.FS`, `os/exec`, `crypto/tls` | 零第三方重型依赖，直接内嵌静态资源，原生支持 HTTP/1.1 与 HTTP/2 Web 服务 |
| **配置标准** | 100% 官方标准 Xray / Sing-box JSON | 完全与官方标准生态对齐，零私有中间语法学习成本 |
| **终端交互** | 原生 ANSI 终端控制序列与交互式 CLI | 极速响应、跨所有 Linux 控制台（包括轻量终端）无乱码 |
| **订阅生成** | 强类型数据模型逆向导出 | 实时自适应客户端 UA，精准输出 Clash.Meta YAML、Sing-box JSON 与 URI |

---

## 2. 核心架构决策记录 (Architectural Decision Records - ADR)

### ADR-001: 彻底放弃 Shell 字符串拼接，拥抱强类型 Go 引擎
- **背景**：原 `install.sh` 超过 10,000 行，大量使用 `cat <<EOF` 拼接 JSON/YAML 字符串，极易因为转义、变量未初始化、逗号缺失引发服务崩溃。
- **决策**：使用 Go 结构体与标准序列化器组装官方配置，从编译期与运行时双重保证语法 100% 正确。

### ADR-002: 剔除 Xray 443 Fallback 中的 `alpn: h2` 规则
- **背景**：现代 Clash.Meta / Mihomo 模拟 Chrome 指纹时携带 `ALPN: ["h2", "http/1.1"]`，导致 Xray 优先将 Trojan 加密流分发至 Nginx 31302 端口，触发 Nginx RST 重置与客户端超时。
- **决策**：443 Inbound 仅保留无条件回落 `{"dest": 31296, "xver": 1}`，所有非 VLESS 流量闭环交由 Trojan 处理，Trojan 自身再回落至伪装网站。

### ADR-003: 使用 Go 原生内嵌 Web 引擎替代 Nginx (Zero-Nginx Architecture)
- **背景**：Nginx 在 VPS 上占用 30MB+ 内存且需 50MB 软件包安装，CentOS 上极易触发 SELinux 端口权限错误。
- **决策**：使用 Go `embed.FS` 内嵌静态 3D 游戏/博客网站，结合 `net/http` 原生监听 31300 端口，并支持 Proxy Protocol v1 与动态 UA 识别智能订阅分发。

### ADR-004: 内置 Linux 内核 16MB UDP 缓冲区与 BBR 原生调优
- **背景**：Linux 默认 UDP 缓冲区仅 212KB，高并发 QUIC (Hysteria2 / TUIC) 产生大量内核层丢包。
- **决策**：Go 程序直接接管 `/etc/sysctl.d/99-vps-optimization.conf` 的生成与持久化激活，固化 16MB 缓冲区与 BBR + FQ。

### ADR-005: 命令行快捷方式命名为 `xraycli`
- **背景**：原 `vasma` 缩写生僻晦涩。
- **决策**：统一使用直观的 `xraycli`（软链接别名 `xcli` / `v2cli`），终端直接运行。

---

## 3. 服务端回落与网络拓扑流向

```text
客户端连接 443 端口 (TLS 解密)
   │
   ├─ VLESS-Vision 流量 ──────────────> [Xray VLESS 核心转发]
   │
   └─ 非 VLESS 流量 (无条件回落 31296)
         │
         ▼
     [Xray Trojan 31296 端口]
         │
         ├─ 密码校验正确 ────────────> [Trojan 核心正常代理]
         │
         └─ 密码错误 / 探测流量 ─────> (回落 31300 Proxy Protocol)
                                              │
                                              ▼
                                 [Go 原生内嵌 Web 伪装站]
                                (展示 3D 极客网页 / 博客)
```
