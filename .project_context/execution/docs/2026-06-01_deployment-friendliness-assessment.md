# Deployment Friendliness Assessment

- 日期：2026-06-01
- 范围：README、部署文档、Compose、环境变量模板、Dockerfile、Release workflow、GHCR 镜像 manifest

## 任务目标

评估当前项目的部署方式对普通使用者是否友好，重点关注首次部署路径、免构建镜像、配置门槛、风险提示和运维可恢复性。

## 执行步骤

- 阅读 `README.md`、`README.zh-CN.md`、`docs/deployment.md` 和 `.env.example`。
- 阅读 `docker-compose.yml`、`docker-compose.prod.yml`、`backend/Dockerfile`、`frontend/Dockerfile`、`frontend/nginx.conf`。
- 检查 `.github/workflows/release.yml` 与 GitHub Release 状态。
- 使用 `docker manifest inspect` 验证 `v0.1.1` 的 backend/frontend GHCR 镜像均包含 `linux/amd64` 与 `linux/arm64`。

## 关键决策

- 当前对外推荐路径应优先使用预构建镜像，而不是要求用户本地构建 Go/Node 前端。
- 版本示例应跟随当前最新正式发布，避免 README 中的旧 tag 让用户部署到较旧镜像。
- 应为“不 clone 源码部署”的用户提供可直接下载的 `docker-compose.prod.yml` 与 `.env.example` 获取方式。

## 结果总结

当前部署方式总体对技术用户友好：两容器 Compose、Supabase 云数据库、预构建 GHCR 镜像、多架构镜像和双语部署文档都已经具备。主要不友好点集中在首次上手的细节：文档示例仍写 `v0.1.0`，但当前最新发布为 `v0.1.1`；Release 没有附带 compose/env 示例资产；文档缺少部署后 smoke test、日志排障和反向代理/HTTPS 示例。
