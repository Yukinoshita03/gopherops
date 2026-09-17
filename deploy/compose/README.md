# 本地开发环境

Compose 在本机启动 MySQL 和 identity 两个独立容器。MySQL 使用持久化数据卷；新建空数据卷时，会自动执行 `migrations/identity/001_create_users.sql`。

## 启动

需要 Docker Engine 和 Docker Compose。首次配置时创建本地环境文件：

```sh
cd deploy/compose
if [ ! -f .env ]; then cp .env.example .env; fi
```

首次配置时分别运行两次 `openssl rand -hex 24`，把两个不同的输出填入 `.env` 的 `MYSQL_ROOT_PASSWORD` 和 `IDENTITY_DB_PASSWORD`。十六进制字符可安全放进当前 MySQL DSN 配置。已有 `.env` 时保留原密码；改写密码变量不会自动轮换已初始化数据库中的账号密码。

JWT 私钥保存在本机未跟踪的 PEM 文件中，不要提交该文件。首次配置时生成私钥：

```sh
if [ ! -f identity-jwt-private.pem ]; then
  openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out identity-jwt-private.pem
  chmod 600 identity-jwt-private.pem
fi
```

把 `id -u` 和 `id -g` 显示的数字填入 `.env` 的 `IDENTITY_RUNTIME_UID` 和 `IDENTITY_RUNTIME_GID`，让容器以本机文件所有者身份读取私钥。`.env.example` 已提供私钥路径、签发方、受众和 15 分钟有效期；如果 `.env` 是之前创建的，需要补上这些新项。

然后构建并启动服务：

如果 Docker Desktop 报 `desktop-linux` context 不存在，或 CLI 连接 `/var/run/docker.sock` 失败，在 macOS 先指定 Docker Desktop 的本机 socket：

```sh
export DOCKER_HOST="unix://$HOME/.docker/run/docker.sock"
```

然后运行：

```sh
docker compose up --build -d
docker compose ps
curl -i http://127.0.0.1:8081/healthz
```

identity 通过 Compose 网络中的 `mysql:3306` 连接数据库。MySQL 只绑定到本机 `127.0.0.1:3307`，identity API 绑定到本机 `127.0.0.1:8081`。修改 `.env` 中 `MYSQL_HOST_PORT` 可以更改本机 MySQL 端口。

注册接口可这样试用：

```sh
curl -i http://127.0.0.1:8081/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"demo","password":"change-this-demo-password"}'
```

注册后可用同一凭据验证登录：

```sh
curl -i http://127.0.0.1:8081/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"demo","password":"change-this-demo-password"}'
```

登录成功响应包含 `user_id`、`access_token` 和 RFC3339 格式的 `expires_at`。用返回的令牌请求当前身份：

```sh
curl -i http://127.0.0.1:8081/v1/me \
  -H 'Authorization: Bearer <access_token>'
```

`/v1/me` 验证 RS256 签名、issuer、audience 和时间声明，通过后返回令牌主体对应的 `user_id`；缺少或无效令牌会得到 HTTP 401。

查看日志并停止容器：

```sh
docker compose logs -f mysql identity
docker compose down
```

`docker compose down` 会保留数据库数据卷。MySQL 官方镜像只在数据目录为空时执行初始化环境变量和 `/docker-entrypoint-initdb.d` 中的脚本；已有数据卷不会自动重跑迁移，修改 `.env` 中的密码也不会更新已有数据库账号。后续新增迁移时，应按迁移顺序对本地数据库执行，不要靠重建容器期待已有卷重新初始化。

`.env` 和 `*.pem` 已被 Git 忽略。不要把密码或私钥复制进 `.env.example`、Compose 文件或提交记录。
