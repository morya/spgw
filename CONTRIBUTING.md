# 贡献指南

感谢考虑为 spgw 做出贡献！在提交 PR 之前，请阅读以下约定。

## 开发环境

- Go 1.26+
- 本地依赖：MySQL、Redis、NATS、etcd（可用 `compose/` 一键启动）
- 修改 `spgwpb/spgw.proto` 需要 protoc 工具链：`protoc`、`protoc-gen-go`、`protoc-gen-micro`

## 构建与测试

```bash
go build ./...
go vet ./...
go test ./...

# 修改 proto 后重新生成桩代码
./gen.sh

# 代码格式化
go mod tidy && goimports-reviser -excludes spgwpb --format ./...
```

## 提交前检查清单（强制）

每次 commit 前，按顺序执行以下 4 步，任何一步失败都必须修复后继续：

1. `go build ./...`
2. `go mod tidy`
3. `bash gen.sh`（确保 `spgwpb/` 生成代码与 `spgw.proto` 一致）
4. `go build ./...`（验证 tidy + gen.sh 未引入破坏）

## 贡献流程

1. Fork 本仓库并创建特性分支（`git checkout -b feat/your-feature`）
2. 提交变更，commit message 遵循现有风格：`type(scope): 描述`（type 取 feat / fix / refactor / docs / chore / style / test；中英文均可）
3. 确保上述检查清单与测试全部通过；涉及协议报文（CMPP）的改动，建议使用 `stub/spclient` 与 `cmd/iagent` 联调验证
4. Push 到你的 fork 并创建 Pull Request，描述动机与影响的模块

## 代码约定

- 日志统一使用 zap（`internal/glog`，多通道：Run / Bin / Link / Stat），不要引入 logrus
- 错误包装使用 `fmt.Errorf("...: %w", err)`；错误码集中在 `internal/consts/errcode*.go`
- 配置结构体字段必须带 `yaml:` 与 `validate:` tag（启动期强制校验）
- 新增 GORM 模型必须在 `internal/migrate/migrate.go` 登记，否则 `spgw migrate` 不建表
- NATS 消息编解码等热路径使用 `jsoniter`，不要使用标准库 `encoding/json`
- CMPP 接入账号密码明文存储是**协议要求**（鉴权需可逆计算 MD5），不是安全疏漏，详见 [AGENTS.md](AGENTS.md) 的"协议约束与安全例外"

更多仓库内部约定见 [AGENTS.md](AGENTS.md)。

## 安全问题

请勿通过公开 Issue 报告安全漏洞，见 [SECURITY.md](SECURITY.md)。
