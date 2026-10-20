<meta name="keywords" content="SMS gateway,SP SMS gateway,SMS forwarding,SMS proxy,CMPP,CMPP 2.0,CMPP 3.0,China Mobile SMS protocol,ISMG,SMSC,delivery report,status report,MO,MT,verification code SMS,notification SMS,marketing SMS,SMS channel,channel routing,HTTP SMS API,Go,Golang,high concurrency,netpoll,NATS,etcd,MySQL,Redis,GORM,Gin,go-micro">

# spgw — SP SMS Gateway

[简体中文](README.md) | English

spgw is a high-performance **SP SMS gateway (SMS forwarding proxy)** written in Go. It provides CMPP 2.0 / CMPP 3.0 protocol and HTTP API access for upstream SPs (Service Providers), receives their submitted messages (submit, including mobile-terminating SMS such as verification codes, notifications and marketing), routes each message to the best next-hop (southbound) SMS channel / carrier gateway (ISMG) according to routing and weight configuration, and delivers SMS status reports (Delivery Reports) and mobile-originated messages (MO) back to the SP.

## Features

- CMPP 2.0 / 3.0 inbound
- HTTP API inbound
- HTTP API outbound
- CMPP 2.0 / 3.0 outbound (planned)

## Screenshots

The web admin console covers northbound (SP access) and southbound (downstream channels) management:

**Northbound · SP Management** — manage SP accounts: login account, IP whitelist, CMPP interface settings (Spid/Pswd, ServiceCode, max connections, submit rate limit, protocol version)

![SP Management](screenshot/sc001.png)

**Northbound · Message Management** — query submitted messages, delivery receipts and unknown records

![Message Management](screenshot/sc002.png)

**Southbound · Channel Management** — configure downstream channels: protocol type, status and sending weight

![Channel Management](screenshot/sc003.png)

**Southbound · Report Management** — submission statistics per channel, per day

![Report Management](screenshot/sc004.png)

## Why spgw

- Automatic best-channel selection based on pricing (in development for v0.2)
- Automatic best-channel selection based on routing rules (in development for v0.2)

## Protocol Support

- Northbound (upstream)
  - CMPP 2.0 inbound
  - CMPP 3.0 inbound
  - HTTP API inbound (`/api/sms/send`, `/api/sms/batchSend`)
- Southbound (downstream)
  - HTTP API outbound (`internal/httpsend` adapter framework + per-vendor adapters)
  - CMPP 2.0 / 3.0 outbound (planned)

## Architecture

Multi-process design: business messages flow between processes over NATS, etcd provides service discovery and link heartbeats, MySQL stores shared metadata, and Redis provides resumable sequence numbers and distributed locks.

| Subcommand | Responsibility |
| --- | --- |
| `spgw server` | Message bus core: dispatch by CmdID, routing / HTTP delivery / waitAck |
| `spgw iagent` | Northbound CMPP server (dual listeners: CMPP 2.0 + CMPP 3.0) |
| `spgw oagent` | Southbound channel client |
| `spgw api` | SP HTTP API access (Gin) |
| `spgw cmcenter` | Control center (account binding / send-budget RPC) |
| `spgw migrate` | GORM AutoMigrate (table prefix `gw_`) |
| `spgw billdumper` | Billing export |

## Quick Start

Requirements: Go 1.25+, MySQL, Redis, NATS, etcd (for local development, `compose/` starts everything in one click).

```bash
# 1. Prepare config: copy config/spgw-sample.yaml to ./config.yaml and adjust connection parameters
cp config/spgw-sample.yaml config.yaml

# 2. Initialize tables
MICRO_REGISTRY=etcd go run . migrate

# 3. Start the processes (MICRO_REGISTRY=etcd is required)
MICRO_REGISTRY=etcd go run . server
MICRO_REGISTRY=etcd go run . iagent --cmpp30port 7891 --cmpp20port 7890
MICRO_REGISTRY=etcd go run . api --port 8080
```

## Development & Build

```bash
go build ./...
go test ./...
go vet ./...

# Regenerate after editing spgwpb/spgw.proto
./gen.sh

# Hot reload (air)
air -c air-api.toml

# Local docker build
./run-in-docker.sh
```

## Directory Structure

- `cmd/`: entry points of the subcommand processes (server / iagent / oagent / api / cmcenter / migrate / billdumper)
- `internal/`: config, logging, link layer (CMPP Link + codec), routing, models, services, downstream HTTP sending framework, etc.
- `spgwpb/`: cross-process message contract (protobuf definitions and generated code)
- `stub/`: test stubs (simulated SP / simulated downstream HTTP service)
- `scripts/`: database init SQL, protofmt utilities
- `AGENTS.md`: repository development conventions (code style, pre-commit checklist, protocol constraints)

## Contributing

Issues and pull requests are welcome. Before committing, please read the pre-commit checklist in [AGENTS.md](AGENTS.md) (build → tidy → gen.sh → build) and make sure `go test ./...` and `go vet ./...` pass.

## Keywords

> Listed for search engines and AI retrieval.

- **Domain**: SMS gateway / SP SMS gateway / SMS forwarding gateway / SMS proxy / SMSC / ISMG / SMS channel / channel routing / SMS load balancing
- **Protocols**: CMPP / CMPP 2.0 / CMPP 3.0 / China Mobile SMS protocol / delivery report (status report) / submit / response / mobile-originated (MO) / mobile-terminating (MT) / HTTP SMS API
- **Use cases**: verification code SMS / notification SMS / marketing SMS / SP access / channel management / submission log query / billing reports
- **Tech stack**: Go / Golang / high concurrency / netpoll / NATS / etcd / MySQL / Redis / GORM / Gin / go-micro / protobuf

## License

This project is licensed under the [Mulan PSL v2](http://license.coscl.org.cn/MulanPSL2). See [LICENSE](LICENSE) for details.

Copyright (c) 2026 Nanjing Guboyun Network Technology Co., Ltd.

contact: wechat: moryaden
