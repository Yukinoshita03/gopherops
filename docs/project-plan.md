# GopherOps 项目设计与开发计划

更新：2026-09-17。面向开发者本人及后续接手指导的 Luna。

## 1. 项目定位

做一个 Go 编写的 Kubernetes 只读诊断 Agent 平台：用户选择有权限的集群与 namespace，提出诊断问题，平台创建任务，Agent 调用有限的诊断工具，持续返回执行进度，最后给出带证据的报告。

这是当前仓库的主线，不是知识库问答或工单平台。参考 GopherAI 的组织思路，不以复制功能数量为目标。此前 go-llm-gateway 是独立项目，可作为模型访问入口，但不得把其已实现功能算作本项目进度。

目标是能演示、能解释、能测试的工程项目，不预先宣称独家创新、高性能或生产可用。简历中的效果、吞吐量、恢复能力必须来自实际验证。

### 核心演示

注册/登录 → 创建诊断任务 → 查看持续更新的步骤 → 读取带证据的报告 → 查询历史任务。进阶演示包括取消执行、重复消息、Worker 崩溃与恢复。

### 首版范围

- 单集群、限定 namespace；本地或隔离测试集群，不直接操作学校生产集群。
- 只读工具，不开放任意 shell、kubectl 命令或自动修复。
- 文本模型加工具调用循环；限制调用次数、单次耗时、总耗时和输出大小。
- HTTP API 优先，用命令行验证；前端不是后端主链路的前置条件。
- RAG、长期记忆、自动上下文压缩、多 Agent 协作暂缓。首版先用有限历史和工具输出上限控制上下文。

## 2. 架构与职责

保留仓库的四个独立入口，分阶段接通，不要求第一阶段把所有服务和中间件同时启动。

| 服务 | 模块 | 负责 | 不负责 |
|---|---|---|---|
| identity | identity | 注册、登录、用户、项目成员与权限 | Agent 执行、任务表 |
| platform | conversation、run、report | 对外任务 API、会话、任务状态、事件、报告、Outbox | 直接持有集群高权限凭据 |
| agent-worker | agent | 模型请求、工具选择与调用循环、执行预算、提交步骤 | 直接修改 platform 的业务表 |
| cluster-tools | clustertool | 校验工具参数和授权范围，读取 Kubernetes API | 任意脚本、修改集群资源 |

最终目标路径：

```text
客户端 → identity：获取身份凭证
客户端 → platform：授权检查，创建 run + outbox（同一事务）
投递器 → RabbitMQ → agent-worker
agent-worker → platform 内部 API：领取、续租、记录步骤、提交结果
agent-worker → 模型适配器 → DeepSeek 或独立模型网关
agent-worker → cluster-tools → 测试集群 Kubernetes API
客户端 ← platform：查询 / SSE 读取持久化事件和报告
```

MySQL 是任务状态的权威来源；Redis 只存可重建缓存和限流数据；RabbitMQ 传递任务通知，不替代数据库。
identity_db 和 platform_db 可以共用 MySQL 实例，但分库、分账号。其他服务不得跨库 JOIN 或共享业务 Repository。
Worker 通过内部 API 写入 platform 的数据；MVP 可以先用进程内接口验证逻辑，但必须明确只是临时测试装配，不算四服务联调完成。

### 分层约定

- domain：业务对象、状态、领域错误，不依赖 Gin/GORM。
- application：业务用例以及所需接口，组织校验、调用和状态变化。
- repository：实现持久化接口，数据库模型与 domain 对象互转，转换数据库错误。
- transport：HTTP/MCP 等协议输入输出、身份提取、状态码转换，不把 SQL 和 Agent 循环写进 handler。
- cmd：配置、依赖组装、服务启动和退出。
- platformkit：确实被复用的日志、配置等基础设施；不要提前建立万能公共层。

## 3. 关键数据和接口草案

以下为设计草案，只有 users 迁移已经存在。其他字段和路由在对应阶段实现前定稿。

| 数据 | 所属 | 核心内容 |
|---|---|---|
| users | identity | id、username 唯一索引、password_hash、created_at |
| projects / project_members | identity | 用户所属项目和角色 |
| cluster_bindings | platform | 项目可使用的集群引用和 namespace 白名单；不对外暴露凭据 |
| conversations / messages | platform | 用户输入和模型回答，必须带资源所属项目 |
| runs | platform | project_id、请求者、目标范围、状态、deadline、取消标记、执行版本、租约 |
| run_steps | platform | 模型/工具步骤、耗时、结构化状态、受限大小的结果或证据引用 |
| run_events | platform | run_id、seq、event_type、payload、created_at；(run_id, seq) 唯一 |
| reports | platform | 结论、证据引用、建议、不确定性；关联 run |
| outbox | platform | event_id 唯一、事件版本、载荷、投递状态、重试时间 |

建议公开 API：

- POST /v1/auth/register、POST /v1/auth/login：注册与登录。
- GET /v1/me：验证当前身份。
- POST /v1/runs：创建任务；后续支持按项目和调用方限定作用域的幂等键。
- GET /v1/runs/:id：任务状态，强制检查项目权限。
- POST /v1/runs/:id/cancel：持久化取消请求；重复调用应安全。
- GET /v1/runs/:id/events：SSE，支持按事件序号补读。
- GET /v1/runs/:id/report：报告及证据。

统一错误使用稳定业务 code 和可读 message，不直接把 SQL、凭据或上游原始错误返回客户端。日志与客户端错误分离。
注册冲突由数据库唯一索引兜底，不能只靠“先查询、再插入”。实现阶段补充用户已存在领域错误及 MySQL 错误映射。
登录令牌建议先做短期有效的签名访问令牌；明确算法、签发方、受众、有效期与密钥管理，刷新/撤销策略单独设计，不默认已支持。
在开放任务接口前必须接上项目授权；早期单用户演示只能绑定固定开发身份，不能伪装为完整权限系统。

## 4. Agent、Provider 与 SSE 的边界

- Provider/模型适配器负责认证、请求格式、HTTP 调用、模型响应及工具调用格式转换。
- Agent 负责决定下一步：调用模型 → 判断工具请求 → 执行允许的工具 → 带结果继续请求 → 输出结论。
- 模型的 SSE 是模型传输协议；平台的 SSE 是任务事件协议，两者不等同。
- 不能原样把模型 token 流当作全部任务进度；平台还需要工具开始/结束、失败、取消和最终状态事件。
- 首版可先用非流式模型结果完成工具循环，再增加平台进度 SSE；token 流是后续体验优化。
- 不能只依赖旧网关的 choices[0].message.content 文本接口完成工具调用：先核对 tool_calls、tool_call_id、工具定义等是否被完整保留。
- 请求超时不是 Agent 总预算。分别限制模型调用、工具调用、步骤数、上下文大小和整次任务时间。
- 日志、资源注释、工具返回文本都是不可信数据，不能成为扩大工具权限的指令。

首批工具建议只做三个：查询 Pod 状态、查询 namespace 事件、读取限定长度的 Pod 日志。工具 schema 必须校验 namespace、资源名和大小限制，服务端根据可信授权信息确定可访问范围，不能相信模型提供的权限声明。
Service selector 诊断需要额外的 Service/EndpointSlice 查询工具，在扩展阶段加入。

## 5. 任务状态与故障语义

建议状态：

```text
queued → running → succeeded / failed / cancelled
queued → cancelled
running --租约过期且可重试--> queued（撤销旧执行版本）
```

- cancel_requested 是取消意图，不等于已经停止。running 任务由执行者观察取消或由恢复器完成收敛。
- terminal 状态不允许被迟到的结果覆盖。取消与成功并发时，以数据库条件更新的获胜结果为准。
- 每次领取生成新的执行版本；步骤写入、续租、完成提交都必须校验版本和当前状态。
- 超时先以 failed + timeout 原因表达，不同时维护多个含义重复的状态。
- 至少一次投递意味着同一任务可能重复到达。用 event_id 去重、条件领取和终态判断控制重复执行。
- Outbox 发布确认后标记已发送；发布成功但未标记时可能重发，消费者仍必须幂等。
- 初版消费者确认可在结果持久化后执行；崩溃重投结合租约处理，不得形成无延迟 requeue 热循环。长任务需要核对队列确认超时配置。
- 租约恢复器必须持续扫描过期执行，原子撤销旧版本并重新投递，设置最大尝试次数。仅有消息 ACK 不代表恢复闭环完成。
- 无法保证模型调用和工具调用 exactly-once；可能产生重复成本，须记录且限制重试预算。

### 取消与断线

异步任务提交成功后，任务生命周期不绑定创建它的 HTTP 请求。HTTP handler 返回不能把任务一起取消。
SSE 客户端断线只停止订阅，不默认取消任务。显式取消写入数据库，由 Worker 的取消监测触发执行 context.cancel，再传递给模型和工具请求。
服务进程退出、任务截止时间、显式取消是不同原因，要可区分记录。

### SSE 重连

事件先持久化，再对订阅者可见。每个 run 的 seq 单调递增，终态事件与终态提交保持一致性。
客户端使用 Last-Event-ID 补读；用同一持久化事件序列避免“历史读取到实时订阅之间”漏事件。首版允许小批轮询数据库，暂不依赖 Redis Pub/Sub。
设置订阅权限、慢客户端和写超时策略，重连不重新执行任务；过期事件应明确返回不可补读提示。

## 6. 开发阶段与验收门槛

按完成证据推进，不承诺固定日期。三个月安排中应留出 Go/网络/OS、MySQL/Redis 和算法时间，不能把所有时间投入功能数量。

| 阶段 | 交付 | 必须验证 | 本阶段暂不做 |
|---|---|---|---|
| P0 用户存取 | Repository、迁移、测试库装配 | 创建后回填、两种查询、未找到、重复用户名、DB 错误、取消 | Redis、MQ、前端 |
| P1 身份闭环 | 注册/登录/me、密码哈希、认证、最小项目授权 | 错误凭证、过期令牌、重名并发、跨项目拒绝 | 完整 RBAC 控制台 |
| P2 单次诊断 | 模型适配器、一个只读工具、有限 Agent 循环、结构化报告 | 假模型测试、坏参数、模型/工具失败、超时、步数上限 | MQ、恢复、RAG |
| P3 持久化与订阅 | run/step/event/report、任务查询、进度 SSE、显式取消 | 事件顺序、重连补读、断线不取消、取消传播 | 多副本高可用 |
| P4 异步可靠性 | RabbitMQ、Outbox、领取/租约/版本、恢复器 | 重复消息、崩溃、租约接管、旧结果拒绝、发布故障、取消竞争 | exactly-once 宣称 |
| P5 诊断与工程验证 | 扩充工具、固定故障集、指标、必要的 Redis 限流 | 证据质量、无权限访问、成本/延迟、过载行为 | 无依据的性能优化 |
| P6 部署与展示 | Compose、镜像、测试 K8s 部署、演示脚本、项目总结 | 最小权限、探针、退出、故障演示、可复现启动 | 直接部署生产集群 |

P2 的执行器可以同步验证；P3 的进程内后台执行是过渡方案，进程崩溃不保证恢复，直到 P4 通过故障测试才可宣称支持恢复。
Redis 只在明确需要限流/缓存时引入；RabbitMQ 只在执行逻辑已可测试后接入；MCP 在只读工具本身能独立测试后接入。

### 验证方法

- 单元测试：假 Repository、假模型、假工具，默认不花真实模型费用。
- Repository 集成测试：独立测试库，真实 MySQL；只清理明确归属测试的数据，不连接生产库。
- 系统测试：固定测试环境注入断网、重复消息、Worker 中断等故障。
- 模型评测：保存输入、环境、模型配置和证据，区分单次示例与重复测量。
- 固定故障集：镜像拉取失败、CrashLoopBackOff、OOM、探针错误、Service selector 不匹配；工具不够时不能标记全部支持。
- 压测区分假模型下的平台吞吐量和真实模型端到端延迟，记录并发、机器配置、p50/p95、错误率，不把供应商延迟称为网关性能。

## 7. 当前实现证据

截至 2026-09-17，以本地工作区为准，不代表已经提交或推送。

- 已有四个服务入口；identity 已装配 MySQL Repository、注册用例和 HTTP 路由。
- 已写 User、UserRepository、MySQLUserRepository、users SQL 迁移；GORM/MySQL 依赖已引入。
- 已检查：两种查询会映射 ErrUserNotFound 并透传其他错误。
- 最新 Create 已改成检查 db.Error，成功后回填 ID/CreatedAt；此前提前返回问题已在代码中修正。
- Create 仍接收传入对象的 ID/CreatedAt。普通注册应由服务端生成，后续明确创建契约，不能直接把客户端 JSON 绑定成可任意指定字段的 domain.User。
- 注册用例已有 fake Repository 单元测试。MySQL Repository 集成测试已增加，要求 `IDENTITY_TEST_MYSQL_DSN` 指向独立的 `identity_test_db`，不会回退到服务数据库。
- 集成测试已在本地 MySQL 8.4.11 测试库运行通过，覆盖创建回填、按名/ID 查询、未找到、重复用户名领域错误映射、取消 context、数据库关闭错误；并发注册实测恰好一个 HTTP 201、一个 HTTP 409，数据库保留一条用户记录。
- 注册 Handler 将 `domain.ErrUserAlreadyExists` 映射为稳定的 `username_already_exists` 响应，不暴露数据库错误；其他内部错误仍返回通用 500。
- 本轮验证：设置独立测试库 DSN 后 `go test -count=1 ./...` 通过；真实 MySQL Repository 集成套件 `go test -race -count=1 ./internal/identity/repository` 通过；`go vet ./...` 和 `git diff --check` 通过。
- `deploy/compose` 包含 MySQL 与 identity 的本地 Compose 编排、identity Dockerfile、空数据库初始化迁移和健康检查。本机实际启动后，MySQL 和 identity 均通过健康检查；`GET /healthz` 返回 204，注册接口返回 201 并写入 MySQL，测试用户已清理。
- P1 登录基础闭环已实现：`POST /v1/auth/login` 按用户名读取用户并用 bcrypt 校验；未知用户与错误密码映射到统一凭证错误，HTTP 响应均为通用 401。Fake Repository 单元测试和 HTTP handler 测试覆盖成功、错误凭证、无效输入、坏 JSON、Repository 错误及内部错误隐藏。
- 访问令牌、认证中间件、Agent、队列、SSE 与 Kubernetes 部署尚未实现。

### 当前阶段：P0 用户存取验收完成

重复用户名的 MySQL 错误已映射到 `domain.ErrUserAlreadyExists`，Handler 返回 HTTP 409；真实 MySQL 并发注册测试验证数据库唯一索引兜底。

### 下一步：进入 P1 身份闭环

登录密码校验接口已完成。接下来确定访问令牌的算法、有效期和密钥配置，接入令牌签发、认证与 `/v1/me`，再补齐过期令牌和最小项目授权验证。项目成员授权继续放在开放诊断任务 API 之前完成。

## 8. 变更管理与待决项

已有边界：四服务、MySQL 权威状态、只读集群工具、至少一次投递、不承诺端到端 exactly-once。
待对应阶段定稿：认证与内部服务凭证、项目角色矩阵、模型直连还是网关、MCP 传输、MQ 确认/重试参数、事件保留时间、执行预算默认值。
设计取舍需要说明原因与影响；大幅改变项目定位、拆服务、增加基础设施先向用户确认。
每完成一个可验证阶段更新本节进度和验证证据；重要决策写入 docs/decisions，不按每轮问答重复记录。
