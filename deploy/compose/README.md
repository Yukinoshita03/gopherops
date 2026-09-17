# 本地开发环境

Compose 在本机启动 MySQL 和 identity 两个独立容器。MySQL 使用持久化数据卷；新建空数据卷时，会自动执行 `migrations/identity/001_create_users.sql`。

## 启动

需要 Docker Engine 和 Docker Compose。首次配置时创建本地环境文件：

```sh
cd deploy/compose
if [ ! -f .env ]; then cp .env.example .env; fi
```

首次配置时分别运行两次 `openssl rand -hex 24`，把两个不同的输出填入 `.env` 的 `MYSQL_ROOT_PASSWORD` 和 `IDENTITY_DB_PASSWORD`。十六进制字符可安全放进当前 MySQL DSN 配置。已有 `.env` 时保留原密码；改写密码变量不会自动轮换已初始化数据库中的账号密码。

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

当前登录接口返回 `user_id`；访问令牌签发和 `/v1/me` 认证将在 P1 后续接入。

查看日志并停止容器：

```sh
docker compose logs -f mysql identity
docker compose down
```

`docker compose down` 会保留数据库数据卷。MySQL 官方镜像只在数据目录为空时执行初始化环境变量和 `/docker-entrypoint-initdb.d` 中的脚本；已有数据卷不会自动重跑迁移，修改 `.env` 中的密码也不会更新已有数据库账号。后续新增迁移时，应按迁移顺序对本地数据库执行，不要靠重建容器期待已有卷重新初始化。

`.env` 已被 Git 忽略。不要把真实密码复制进 `.env.example`、Compose 文件或提交记录。
