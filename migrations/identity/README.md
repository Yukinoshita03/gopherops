# migrations/identity

身份服务专属迁移；拥有 users、projects、project_members。

迁移顺序：`001_create_users.sql` 创建用户表，`002_create_projects_and_project_members.sql` 创建项目和成员关系表。Compose 会对新建的空数据卷执行这两条迁移；已有本地数据库不会自动执行 002，需要单独应用。MySQL 容器的 initdb 脚本只对空数据卷生效。

成员关系用 `(project_id, user_id)` 保证唯一，`role` 暂为可扩展字符串；当前授权只判断是否存在成员行。项目删除会级联清理成员，用户存在成员关系时不能删除。将来创建项目时，应用层应在同一事务内创建项目和初始 owner 成员；当前 schema 不强制每个项目必须有成员。
