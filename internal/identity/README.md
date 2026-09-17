# identity

职责：身份与项目权限。

identity 负责注册、登录、用户和项目成员权限。当前已有 MySQL 用户存取、注册/登录 HTTP 流程；登录成功后使用 RS256 签发短期访问令牌。`GET /v1/me` 只接受 `Authorization: Bearer <token>`，验证签名和标准声明后，从 `sub` 取得用户 ID 并返回。Bearer 格式错误、验签失败、过期令牌和无效用户 ID 都返回通用 401。阶段证据见 [总体计划](../../docs/project-plan.md)。
