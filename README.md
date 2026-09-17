# GopherOps

基于 Go 的集群诊断 Agent 平台，参考 [GopherAI-v2](https://github.com/youngyangyang04/GopherAI/tree/main/GopherAI-v2) 的分层组织与功能，独立重写。

## 当前状态

identity 已接入注册、登录密码校验、RS256 访问令牌签发与验证，以及受保护的 `GET /v1/me`；`deploy/compose` 提供本地 MySQL 与 identity 容器编排和初始化迁移。认证测试覆盖 Bearer 头解析、无效/过期令牌拒绝和登录到 `/v1/me` 的完整请求链。最小项目授权、Agent 和 Kubernetes 部署尚未实现。

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

运行 `make check` 可执行当前单元测试和静态检查。Compose 启动步骤见 [`deploy/compose/README.md`](deploy/compose/README.md)。

## 开发顺序

1. 身份与会话：接口契约、数据库迁移、登录与消息历史。
2. 单次 Agent：模型适配、一个工具、步骤记录。
3. 异步任务：RabbitMQ、Worker、状态查询。
4. 可靠性：Outbox、幂等、租约、取消与崩溃恢复。
5. 集群诊断：只读工具、证据报告、SSE 与固定故障评测。
6. Kubernetes：镜像、权限、探针与多副本验证。

开发阶段、验收标准和当前下一步以 [总体设计与开发计划](docs/project-plan.md) 为准；另见 [架构概要](docs/architecture.md) 和 [开发与教学交接](AGENTS.md)。
