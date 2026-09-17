# 集成测试

服务与数据库、队列的集成验证。

## Identity MySQL Repository and Registration

测试位于 `internal/identity/repository/mysql_user_repository_integration_test.go`，使用 `IDENTITY_TEST_MYSQL_DSN` 连接真实 MySQL。它只接受名为 `identity_test_db` 的数据库；没有设置 DSN 时会明确 skip，不会回退到服务正在使用的 `identity_db`。

准备独立测试库时，为 `identity_test_db` 创建一个仅有 `SELECT`、`INSERT`、`UPDATE`、`DELETE` 权限的测试用户，并执行 `migrations/identity/001_create_users.sql`。例如 DSN 格式为：

```text
identity_test:<local-password>@tcp(127.0.0.1:3307)/identity_test_db?charset=utf8mb4&parseTime=True&loc=UTC
```

密码可用 `openssl rand -hex 24` 生成。将完整 DSN 放入本地忽略文件 `deploy/compose/.env.test` 后，从仓库根目录运行：

```sh
IDENTITY_TEST_MYSQL_DSN="$(sed -n 's/^IDENTITY_TEST_MYSQL_DSN=//p' deploy/compose/.env.test)" \
  go test -v -count=1 ./internal/identity/repository
```

测试使用独立用户名，并只删除自己插入的记录；不会清空或重建数据库。并发注册用例通过 HTTP Router 和真实 MySQL 验证同名请求得到一个 201、一个 409，且只保存一条记录。
