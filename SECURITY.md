# 安全政策

## 报告漏洞

请勿通过公开 Issue、Pull Request 或讨论区报告任何安全漏洞。

如发现 spgw 中的安全问题，请通过以下任一渠道私密报告：

1. **GitHub 私密漏洞报告**：[Report a vulnerability](https://github.com/morya/spgw/security/advisories/new)（推荐）
2. **邮件**：[morya](mailto:moryaden@qq.com)

请在报告中包含：问题描述、受影响版本、复现步骤、可能的影响面。我们会尽快响应（目标 72 小时内确认受理）。

## 特别提醒：短信网关协议面

spgw 处理 CMPP 2.0/3.0 与 HTTP 接入，评估问题时请注意本项目的一些**协议固有设计**（并非漏洞）：

- **CMPP 接入账号密码在库中明文存储**：CMPP 鉴权要求 ISMG 侧能用明文密码重算 `MD5(SourceAddr + 9字节0 + Password + Timestamp)`，单向 hash 会使链路鉴权无法工作。因此账号表的可访问面（库表权限、备份、审计）就是安全边界本身，部署时务必收紧。
- **CMPP `SequenceID` 允许 uint32 回绕**、长连接与心跳保活：均为协议要求。
- 管理后台（`cmd/portal` 路由、`gw_user`）的登录凭据应使用 bcrypt/argon2 单向散列存储，与上述 CMPP 约束无关。

如果你发现的问题与以上说明一致，且仅在企业内网可信环境暴露，请仍先行报告，由维护者评估实际风险。

## 支持版本

当前处于 1.x 之前的孵化阶段，仅最新 `main` 分支接受安全修复。
