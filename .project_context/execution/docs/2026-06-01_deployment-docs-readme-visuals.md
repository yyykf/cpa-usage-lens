# Deployment Docs and README Visuals

- 日期：2026-06-01
- 分支：codex/docs-deployment-friendly-assets

## 任务目标

提升 CPA Usage Lens 的部署友好度和 README 产品展示效果：部署文档改为推荐使用最新 release tag，补充免 clone 源码部署路径、部署后自检、常见排障，并让生产 Compose 默认不暴露 backend 调试端口；同时加入产品介绍图和真实打码截图。

## 执行步骤

- 从 `main` 创建 `codex/docs-deployment-friendly-assets` 分支。
- 创建 Trellis 任务并记录 PRD：`.trellis/tasks/06-01-docs-deployment-friendly-assets/prd.md`。
- 检查项目仪表盘截图 `docs/assets/dashboard-screenshot.jpg`，确认适合作为 README 真实产品截图。
- 使用 imagegen 生成无真实账号数据的产品介绍图；最终采用视觉效果更好的第一版，并在 README 明确标注为概念预览，避免被理解为当前 UI 的精确截图。
- 将图片资产保存到 `docs/assets/`。
- 更新 `README.md`、`README.zh-CN.md`、`docs/deployment.md`、`docker-compose.prod.yml`。
- 新增 `docker-compose.debug.yml` 作为显式调试 override。
- 新增 `.trellis/spec/backend/deployment-guidelines.md`，沉淀生产 Compose 与 debug override 的部署合同。

## 关键决策

- README 使用两类图：生成的产品介绍图负责第一印象，真实打码截图负责可信度。
- README 明确标注生成图是概念预览 / 后续视觉方向，不代表当前界面已完全实现；真实打码截图才是当前产品实际效果。
- 部署命令不再硬编码旧版本，改为提示用户从 GitHub Releases 复制最新 tag。
- 免 clone 源码路径从 `main` 下载 Compose/env 模板，镜像仍通过 `CUL_VERSION=<latest-release-tag>` 固定到最新发布版本，避免文档合并后但新 tag 尚未发布时 raw tag 文件缺失。
- 生产默认只暴露 frontend `8088`，backend `8080` 通过 `docker-compose.debug.yml` 按需开启。

## 结果总结

- README 已加入 `docs/assets/product-intro.png` 和 `docs/assets/dashboard-screenshot.jpg`。
- 部署文档已补充免 clone 部署、部署后检查和常见问题排查。
- `docker-compose.prod.yml` 已移除 backend host port，`docker-compose.debug.yml` 提供调试入口。
- 已通过 `docker compose -f docker-compose.prod.yml config --quiet` 与 debug override 组合校验。
