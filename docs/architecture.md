# 架构设计

## 参考与实现边界

参考项目：youngyangyang04/GopherAI，检查时提交 `1d892e4afffca02e3232f8928c2cdafafee5dac8`。
借鉴 controller/service/dao 的分层思路，将代码职责明确为 domain/application/repository/transport。
本项目正在实现 identity 用户存取层，其余主流程仍为设计目标。阶段验收和实际进度详见 [总体设计与开发计划](project-plan.md)。

## 主流程

用户 → platform → MySQL（任务与 Outbox 同事务）→ 投递器 → RabbitMQ → agent-worker → cluster-tools → Kubernetes API。
Worker 通过 platform 的内部接口领取执行权、续租、保存步骤与报告。前端通过 platform 查询或订阅进度。
identity 负责身份与项目成员信息；当前 identity HTTP 服务使用 RS256 令牌验证 `/v1/me`。platform 的项目成员授权和跨服务令牌验证仍待实现。

## 数据所有权

identity 拥有 identity_db；platform 拥有 platform_db。可共用一个 MySQL 实例，但使用独立库和账号。
Worker 与工具服务不直接访问其他服务的表。不使用跨服务 JOIN 或共享业务 DAO。
MySQL 是任务状态的权威存储；Redis 用于限流及可重建缓存。

## 任务可靠性

- 以至少一次投递为前提，event_id 去重，领取任务与结果提交幂等。
- Outbox 发送端使用发布确认；消费者确认策略需与任务持久化及恢复逻辑一起设计。
- Worker 租约包含过期时间和执行版本；旧执行者不能覆盖新执行者的结果。
- 外部模型与工具调用可能重复，不宣称端到端 exactly-once。
- 取消请求、超时与 context 传播需覆盖模型和工具调用。
- run_events 使用稳定序号，为客户端断线补读保留依据。

## 服务边界

API 请求与耗时 Agent 执行分开，集群访问权限隔离在工具服务。
初期单集群、指定 namespace、只读工具，不提供任意 shell 执行。
所有服务可独立构建与部署；共享仓库和 Go module 不代表共享运行进程。

## 验证目标

固定故障集：镜像拉取失败、CrashLoopBackOff、OOM、探针错误、Service selector 不匹配。
比较固定巡检与 Agent 诊断，记录正确率、证据完整性、耗时、工具调用次数与模型成本。
