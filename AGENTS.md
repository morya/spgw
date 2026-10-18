# AGENTS.md

spgw（SP 短信网关）。Go 1.25 + go-micro v5 + CMPP 2.0/3.0 + GORM + Gin + NATS + etcd + Redis + MySQL。

## 功能简述

这是一个支持SP短信转发的proxy系统，命名spgw。

- 上游协议接入：
  - 支持 cmpp2.0
  - 支持 cmpp3.0
  - 支持 http api（`cmd/api` 提供 `/api/sms/send`、`/api/sms/batchSend`）
- 下游协议支持：
  - ~~支持 cmpp2.0~~ （待实现）
  - ~~支持 cmpp3.0~~ （待实现）
  - 支持 http api（`internal/httpsend` + `adapters/`，附带通用 HTTP API 适配示例 `adapters/example_api.go`）

## 沟通约定

- 与用户交互使用中文。
- 代码、注释、commit 沿用现有风格（中英文混用，注释多为中文）。

## 仓库布局

入口在根目录 `main.go`，通过 `urfave/cli/v2` 注册多个子命令；每个子命令对应 `cmd/<name>/` 下的独立进程。

```
main.go            cli 入口，注册所有子命令
before.go          App.Before：日志（glog）、pprof、加载 config.yaml、连接 redis
flags.go           全局 flag：--loglevel/--logformat/--logdir/--config/--pprof/--report-func
config/            配置样例（spgw-sample.yaml / portal.yaml）；本地 config.yaml 不入库
compose/           本地 docker compose（mariadb + nats + spgw）
Procfile           foreman 启动模板
gen.sh             生成 spgwpb 的 protobuf + go-micro 桩
scripts/           建库 SQL、protofmt 等
stub/              辅助测试桩（http-server/ 模拟下游、spclient/ 模拟 SP）

cmd/
  cmcenter/    控制中心：gin HTTP :8080 + go-micro 服务 spgw.cmcenter（BindAccount/AllocSendBudget）
  api/         SP HTTP API 接入（Gin），订阅 NATS spgw.api
  server/      总线消息中枢：订阅 NATS spgw.server，按 CmdID 分发，含路由/HTTP下发/waitAck
  iagent/      CMPP 接入端（北向，netpoll TCP server，双监听 cmpp20+cmpp30）
  oagent/      CMPP 出局端（南向，netpoll TCP client）
  migrate/     GORM AutoMigrate
  billdumper/  账单导出（空实现桩）
  demo/spclient/  SP 模拟器（空实现桩）
  demo/ss/     go-micro demo server（未在 main.go 注册）

internal/
  gconfig/   全局配置 + yaml 加载 + validator
  glog/      zap Logger 初始化 + GORM logger 适配（多通道：Run/Bin/Link/Stat）
  db/        GORM MySQL 连接（表前缀 gw_，单数表名）
  rconn/     Redis 客户端 + 序列号断点续号 + 分布式锁（含 keys/ 子包）
  link/      CMPP Link（client/server 双模、心跳、绑定、收发、etcd 心跳注册）
             含 codec.go / codec_cmpp20.go / codec_cmpp30.go（编解码已并入）
  linkid/    全局 link id 分配
  seq/       CMPP SequenceID / MsgID 生成器
  encoding/  GB18030/UCS2 编码转换
  router/    Gin 路由（middle/ + response/ + router_helper/ + routes/{auth,user,menu,role,api}_route）
  model/     GORM 数据模型（含 user_model/）
  service/   业务服务层（user_service/ + my_jwt/）
  daox/      数据访问层（含 user_dao/）
  consts/    状态常量 + 错误码（errcode/errcode_cmpp/errcode_cmpp_status/errcode_http）
  httpsend/  下游 HTTP 发送框架（Adapter 接口 + Manager + Sign + adapters/）
  counter/   提交/报告成功失败计数器（cache2go，I/O 双向 8 个计数函数）
  jsonx/     Gin JSON helper（jsoniter 解码 + jettison 编码）
  waitack/   等待确认（空桩，实际逻辑在 cmd/server/waitack.go）
  etcdsrc/   go-micro etcd config source
  util/{ipx,k8s,microx,ptr,timex}
  validatex/ validator/v10 单例 + universal-translator 中文翻译 + eachIn/eachIPv4 自定义验证器

spgwpb/   spgw.proto + 生成的 .pb.go / .pb.micro.go / newmsg.go
```

## 子命令与进程模型

各进程通过 NATS 传业务消息、etcd 注册链路存活、MySQL 共享元数据。所有进程共享 `before.go` 的初始化（日志 / pprof / config / redis）。

| 子命令 | 别名 | 职责 |
| --- | --- | --- |
| `spgw cmcenter` | `c` | 控制中心：gin HTTP（默认 8080）+ go-micro 服务 `spgw.cmcenter`，提供 BindAccount/AllocSendBudget RPC（雏形） |
| `spgw api` | - | SP HTTP API 接入（Gin，默认端口 80），订阅 NATS `spgw.api`，路由 `/api/sms/*` + `/api/v1/callback/:vendor` |
| `spgw server` | `s` | 总线中枢：订阅 NATS `spgw.server`，处理 Submit/Deliver/CallbackReport，含路由分发与 HTTP 下发 |
| `spgw iagent` | `i`,`ia` | 北向 CMPP 服务端，双监听 `--cmpp30port`（默认 7891）+ `--cmpp20port`（默认 7890），SP 接入并投递 NATS |
| `spgw oagent` | `o`,`oa` | 南向 CMPP 客户端，连接上游通道并下发 |
| `spgw migrate` | `m` | GORM AutoMigrate（表前缀 `gw_`） |
| `spgw billdumper` | - | 账单导出（当前空实现桩） |
| `spgw demo-spclient` | - | SP 模拟器（当前空实现桩） |

环境变量：所有连接 etcd/nats 的子命令需要 `MICRO_REGISTRY=etcd`（见 `Procfile`、`compose.yaml`）。

NodeID：iagent / server / oagent 的 `before.go` 通过 `internal/util/k8s` 判断是否在 K8s。K8s 中以 `HOSTNAME` 末段（StatefulSet 序号）+1 作为 `NodeID`，本地默认 `1`。

## 关键运行依赖

- MySQL（默认 `localhost:3606`，库 `spgw`，时区 `Asia/Shanghai`）
- Redis（默认 `localhost:6379`）
- NATS（默认 `localhost:4222,4223`）
- etcd（默认 `localhost:2379`，go-micro 服务发现 + 链路心跳 `/spgw/ilink-*` `/spgw/olink-*`）

修改任何依赖前先看 `internal/gconfig/gconfig.go` 的 `validate` tag —— `gconfig.Load` 会强制校验。

## 公共契约：spgwpb

跨进程消息统一为 `spgwpb.Msg{Head, Raw, ...}`。`Head.CommandId` 使用 `CmdID` 枚举（CMPP 风格，请求位 0、响应位 0x8000）。NATS payload 是 `Msg` 的 **JSON 编码**（`jsoniter` / `gostring.JsonEncodeString`），不是 protobuf 二进制。

`spgwpb/spgw.proto` 定义了两个 go-micro service：
- `Spgw` -- Greeting 示例（demo 用）
- `Cmcenter` -- BindAccount / AllocSendBudget（控制中心 RPC）

修改 `spgwpb/spgw.proto` 后必须运行 `./gen.sh`（需 `protoc`、`protoc-gen-go` 与 `protoc-gen-micro`）。

## 常用命令

```bash
go build -o spgw .

# 本地运行（任选其一，需要 MICRO_REGISTRY=etcd）
MICRO_REGISTRY=etcd ./spgw migrate
MICRO_REGISTRY=etcd ./spgw cmcenter --port 8080
MICRO_REGISTRY=etcd ./spgw api    --port 80 --pprof 6060
MICRO_REGISTRY=etcd ./spgw server
MICRO_REGISTRY=etcd ./spgw iagent --cmpp30port 7891 --cmpp20port 7890
MICRO_REGISTRY=etcd ./spgw oagent

# Air 热重载
air -c air-api.toml
air -c air-iagent.toml
air -c air-oagent.toml
air -c air-server.toml

# docker（构建 + compose up）
./run-in-docker.sh

go test ./...
go vet ./...
./gen.sh
```

## Go 编码约定

> 遵循通用 Go 规范（go mod tidy、goimports-reviser、现代 Go 语法）。下列为本仓库特有补充。
>
> Go 代码格式化必须使用：
>
> ```bash
> go mod tidy && goimports-reviser -excludes spgwpb  --format ./...
> ```
>
> **格式化后必须重新生成 proto 并验证编译**：每次执行完上述代码格式化后，必须再执行一次 `bash gen.sh` 重新生成 `spgwpb/` 下的 proto 对应文件，然后运行 `go build ./...` 验证；若报错则修复后重复此步骤，直到 `go build ./...` 无任何报错为止。

- `go 1.25.0`，可使用 `any`、`min/max`、`for range N` 等新语法。
- **日志统一使用 zap**：`zap.S().With(zap.Any("k", v)).Errorf(...)`。日志包在 `internal/glog`（非 `internal/log`），提供多通道 logger：`glog.RunS()` / `glog.BinS()` / `glog.LinkS()` / `glog.StatS()`，分别对应运行日志、二进制报文日志、链路日志、统计日志，底层用 lumberjack 做文件切割。仓库已**全面禁用 logrus**（无任何 `.go` 文件 import，`go.mod` 也不再 require），新代码不要重新引入。
- **错误包装**：标准库 `fmt.Errorf("...: %w", err)` 为主。错误码集中在 `internal/consts/errcode*.go`（errcode / errcode_cmpp / errcode_cmpp_status / errcode_http），不存在 `internal/errors` 或 `internal/errs` 目录。
- **配置**：所有配置走 `internal/gconfig`，结构体字段必须带 `yaml:` 与 `validate:` tag。`GConfig` 包含 `SPGwId`、`RouteMode`（1:按比例 2:按归属地）、`WaitAckTimeout`、`WaitRptTimeout`、`MaxServerConcurrency`、`MaxAgentConcurrency`、`HeartbeatInterval` 等字段。
- **校验**：统一走 `internal/validatex`（`go-playground/validator/v10` + `universal-translator` 中文翻译）。`validate:` tag 使用 **逗号分隔规则、等号传参** 语法（如 `validate:"required,min=1,max=100"`），不是管道 `|` 语法。枚举用 `oneof=a b c`（空格分隔值）。自定义验证器 `eachIn` / `eachIPv4` 已在 validatex 注册。可选字段须加 `omitempty`。校验入口：`validatex.Validate(s)`，返回第一个错误的中文字符串。
- **数据库**：通过 `internal/db.MainDB`（GORM）访问；模型放 `internal/model`，表前缀固定 `gw_`、单数表名（`MyNaming`）。**新增模型必须在 `internal/migrate/migrate.go` 的 `arr` 中登记**，否则 `spgw migrate` 不建表。当前已登记：IAccount、OAccount、LinkInfo、Route、user_model 下全部表。
- **HTTP**：Gin（`github.com/gin-gonic/gin` + `gin-contrib/cors`）。SP 接入 API 路由在 `cmd/api/app.go` 的 `Run()` 方法中通过 `initApi(g *gin.RouterGroup)` 注册；管理后台路由集中在 `internal/router/routes/<resource>_route/`。handler 使用泛型封装 `routerhelper.HandleWithModel[T]`（自动绑定+校验请求体）；请求体经 `Validator`（go-playground/validator/v10）校验。中间件 `middle.BodyDump()` + `gin.Recovery()` + `cors.Default()`。SP HTTP API 鉴权走 `App.signMiddle`（IAccount/OAccount 双向 MD5 验签）。
- **CMPP 协议**：复用 `github.com/hujm2023/go-sms-protocol`（cmpp30 子包，`go.mod` 中 replace 指向 `github.com/morya/go-sms-protocol` fork）。编解码在 `internal/link/` 内（`codec.go` / `codec_cmpp20.go` / `codec_cmpp30.go`，入口 `NewMyCodec(proto)`）--原 `internal/codec/` 已删除并并入 link。
- **公共工具**：`github.com/morya/gommon`（gostring / strmd5 / wait_group_wrapper / config / mytrace 等泛型工具包）。
- **链路抽象**：TCP 连接统一走 `cloudwego/netpoll` + `internal/link.Link`（`ModeClient` / `ModeServer`）。新增协议需扩展 `link.newLink` 的 `info.Proto` 分支并补 codec。
- **序列号**：`internal/seq` 提供 SeqID/MsgID 生成器，`rconn.GetLastSeqID` 从 Redis 取断点续号。新增节点类型必须沿用同样的持久化方式，避免重启冲突。
- **并发**：iagent/oagent 内部状态用 `sync.Map` 或 `sync.RWMutex + map`（见 `cmd/oagent/app.go`），保持现状即可。
- **关闭流程**：`closeOnce sync.Once` + `cancel()` + 关闭 channel/连接，参考 `cmd/oagent/app.go` 的 `Stop()`。
- **下游 HTTP 发送**：`internal/httpsend` 提供 `Adapter` 接口（`Vendor()` / `BuildSubmitRequest` / `ParseSubmitResponse` / `ParseReports`），`Manager` 负责注册与查找，`adapters/` 下放具体渠道实现（内置示例 `ExampleAdapter`，vendor 标识为 `example`）。新增下游渠道需实现 `Adapter` 接口并在 `cmd/server` 的 `Init()` 中注册。
- **计数器**：`internal/counter` 基于 cache2go 提供 I/O 双向 submit/report 成功失败计数，通过 `counter.IncrISubmitSuccCnt(id)` 等函数调用。
- **JSON**：`internal/jsonx` 提供 `DecodeRequest`（jsoniter + fuzzy decoders 兼容 PHP 风格）与 `WriteJSON`（jettison 编码，nil slice -> []、nil map -> {}）。进程间 NATS 消息用 `jsoniter`（`cmd/oagent/run.go`、`cmd/server/app.go`）或 `gostring.JsonEncodeString`，热路径不要切到标准库 `encoding/json`。

## 提交前检查清单（强制）

每次 `git commit` 前，**必须按顺序**执行以下 4 步，任一步骤失败都必须修复后继续，不得跳过：

1. **确保编译通过**：`go build ./...`
2. **执行 tidy**：`go mod tidy`
3. **重新生成 proto**：`bash gen.sh`（确保 `spgwpb/` 下的 `.pb.go` / `.pb.micro.go` 与 `spgw.proto` 一致）
4. **再次编译**：`go build ./...`（验证 tidy + gen.sh 未引入破坏）

> 若第 4 步报错，修复后回到第 1 步重新走一遍，直到全流程通过再 commit。

## 测试与验证

- 单测分布在 `internal/seq/msgid_test.go`、`internal/linkid/lkid_test.go`、`internal/util/ptr/ptr_test.go`、`unittest/bind_test.go`。提交前至少跑 `go test ./...` 与 `go vet ./...`。
- 改 `internal/gconfig` 后用本仓库 `config.yaml` 跑一次 `MICRO_REGISTRY=etcd ./spgw migrate`，确认 `validate` tag 不会拒绝现有配置。
- 涉及 CMPP 报文流转的改动，建议 `stub/spclient` + `cmd/iagent` 联调。

## 构建 / 容器化

- 多架构构建：`Dockerfile-multi-arch` 产出 `linux/amd64,linux/arm64` 镜像；本地一键构建运行见 `./run-in-docker.sh` 与 `compose/`。
- `version.GitRev` 通过 `-ldflags "-X github.com/morya/spgw/version.GitRev=<sha>"` 注入。

## 协议约束与安全例外

以下做法看起来违反通用安全最佳实践，但因协议要求**必须保留**。任何代码评审或安全审计在指出"问题"前请先看这里。

### CMPP 接入账号密码以明文形式存储

- **现象**：`internal/model` 中 SP 接入账号、上游通道账号的 `Password` 字段为明文（不做 bcrypt/argon2 hash）。
- **为什么必须如此**：CMPP 协议鉴权依赖 `AuthenticatorSource = MD5(SourceAddr + 9 字节 0 + Password + Timestamp)`（见 `go-sms-protocol/cmpp/cmpp30/pdu_connect.go`）。ISMG 侧（也就是 spgw）必须能在收到 connect_req 时用对端预先约定的明文 `Password` 重新计算 MD5 并比对。一旦库里只剩单向 hash，重算无法进行，链路鉴权直接失败。CMPP 2.0 / SMGP / SGIP 几乎同构，约束相同。
- **缓解措施（已经在做或应当做的）**：
  - 库表权限收紧：只有 spgw 进程账号可读 `gw_iaccount` / `gw_oaccount` 等账号表；
  - 配置文件中的同类口令（`config.yaml` 的 DB / Redis / JWT key）使用 K8s Secret 或环境变量注入，不进 git；
  - 备份加密、审计日志脱敏；
  - 管理后台的**管理员账号**（`gw_user` 表）走的是 JWT 登录路径，**不受**此约束，应当用 bcrypt/argon2 单向 hash 存储——这两类账号性质不同，请勿混淆。

### 其他类似的"协议高于通用安全建议"场景

- **CMPP `SequenceID` 允许 uint32 回绕**：高并发下序号循环是协议允许的，不要套用"序号永不重复"的直觉。
- **CMPP 心跳 / 长连接保持**：违反通用 HTTP 短连接思路，但属于协议要求。

- 不要在 spgw 自己的 `.go` 文件中 import `github.com/go-redis/redis/v8`。它是 `go-micro.dev/v5/cmd` → `cache/redis` 的传递依赖，编译进二进制但运行时不会被实例化；业务侧 redis 客户端**统一用** `github.com/redis/go-redis/v9`（见 `internal/rconn/`）。同理，`amqp091-go`、`lib/pq`、`hashicorp/consul/api` 等也属于 go-micro 框架副作用，不要在 spgw 代码中直接 import 它们。
- 不要把热路径上的 `encoding/json` 扩散到 NATS 消息编解码等性能敏感路径；改用 `jsoniter`（`cmd/oagent/run.go`）或 `gostring.JsonEncodeString`。logrus 已彻底移除，不要再引入。
- 不要在 `internal/link` 的 `link.go`/`conn.go` 等通用层硬编码具体 CMPP 报文逻辑--协议编解码属于 `internal/link/codec*.go` 与 `go-sms-protocol`。
- 不要绕开 `internal/db.MainDB` 直接 `database/sql.Open`；表名/前缀策略由 `MyNaming` 统一管理。
- 不要在 `gconfig` 字段上省略 `validate` tag —— 启动期会强制校验，缺失会导致进程启动直接失败。
- 不要 `commit` 自己产生的 `spgwpb/.pb.go` 修改而不重新跑 `./gen.sh`，proto 与生成代码必须一致。
