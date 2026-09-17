# Docker 镜像

`identity.Dockerfile` 使用多阶段构建编译 `cmd/identity`，最终镜像只包含运行时证书和 identity 可执行文件。构建入口由 `deploy/compose/compose.yaml` 调用。
