# GopherOps

基于 Go 的集群诊断 Agent 平台，参考 [GopherAI-v2](https://github.com/youngyangyang04/GopherAI/tree/main/GopherAI-v2) 的分层组织与功能，独立重写。

## 当前状态

已创建项目结构与四个可编译的入口，正在实现 identity 的 MySQL 用户存取层：已有领域对象、Repository 接口与实现、users 迁移。尚无业务测试或数据库联调证据；没有实现登录、HTTP 服务、Agent、数据库启动装配或 Kubernetes 部署。入口运行后输出占位信息并退出。

本次骨架未复制 GopherAI 源码；后续引用上游代码时需保留来源并核对适用许可证。

## 技术目标

Go + Gin + MySQL + Redis + RabbitMQ + Agent/MCP + Kubernetes，前端预留 Vue。当前已引入 GORM/MySQL 依赖，其余按阶段接入，不代表已实现。

## 服务与模块

| 服务 | 模块 | 职责 |
|---|---|---|
| identity | identity | 用户、项目与权限 |
| platform | conversation、run、report | 会话、任务、SSE 与报告 |
| agent-worker | agent | 消费任务并驱动 Agent |
| cluster-tools | clustertool | 只读集群诊断工具 |

## 目录

```text
cmd/                 四个服务入口
internal/            六个业务模块与少量公共基础设施
api/                 HTTP、RPC、事件、MCP 契约
migrations/          按服务归属划分的 SQL 迁移
web/                 前端
configs/             配置示例
build/docker/        镜像构建
deploy/              Compose 与 Kubernetes
scripts/             开发脚本
tests/               集成与故障场景
docs/                架构与决策
```

## 本地验证

需要 Go 1.26 或更高版本：

```sh
make check
make build
go run ./cmd/platform
```

目前 `go test` 仅验证包可编译，没有业务测试用例。

## 开发顺序

1. 身份与会话：接口契约、数据库迁移、登录与消息历史。
2. 单次 Agent：模型适配、一个工具、步骤记录。
3. 异步任务：RabbitMQ、Worker、状态查询。
4. 可靠性：Outbox、幂等、租约、取消与崩溃恢复。
5. 集群诊断：只读工具、证据报告、SSE 与固定故障评测。
6. Kubernetes：镜像、权限、探针与多副本验证。

开发阶段、验收标准和当前下一步以 [总体设计与开发计划](docs/project-plan.md) 为准；另见 [架构概要](docs/architecture.md) 和 [开发与教学交接](AGENTS.md)。
