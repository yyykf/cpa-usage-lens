# README 双语化 + 部署文档 + 徽章 + CI 初始化

- 日期：2026-05-31
- 分支：feature/tailwind-v4-upgrade（注：本批改动与 Tailwind 升级无关，建议单独成一个原子 commit）

## 任务目标

为仓库提前初始化对外门面：README 双语（英文为主）、双语部署文档、README 徽章、GitHub Actions CI、开源协议。当前处于开发阶段，尚未发布到 main。

## 关键决策（与 code4j 确认）

| 决策点 | 选择 |
|--------|------|
| 文档双语组织 | README 拆分（`README.md` 英文 + `README.zh-CN.md` 中文）+ 部署文档单文件双语 |
| 开源协议 | MIT |
| CI 范围 | 构建 + 测试（backend `go build/vet/test`；frontend `tsc` + `vite build`） |
| 徽章风格 | flat-square；状态行仅 WIP 用暖色，License 与技术栈统一碳黑底（`#0d1117`）+ 品牌色 logo |

## 执行步骤 / 产出

- 新增 `LICENSE`（MIT，© 2026 KaiFan Yu）
- 重写 `README.md` 为英文门面：徽章 + WIP 提示 + 语言切换 + 特性/架构/快速开始/技术栈/结构/约束/许可证
- 新增 `README.zh-CN.md`：中文版，结构与英文对齐
- 重写 `docs/deployment.md` 为英中双语单文件（顶部锚点导航，英文 9 节 + 中文 9 节）
- 新增 `.github/workflows/ci.yml`：backend / frontend 两个独立 job，`go-version-file` 跟随 go.mod，`concurrency` 取消同 ref 堆积的旧 run

## 验证

- `ci.yml` YAML 语法有效
- 本机 go 1.26.2 与 go.mod 一致；本地原样复现 CI backend job：build ✓ / vet ✓ / test -race 全过
- frontend CI job 待当前 Tailwind 升级 task 完成、build 恢复后自然转绿（非 CI 配置问题）

## 待办 / 注意

- CI 徽章在首次合并 main 并跑过 CI 前会显示 "no status"，属正常
- `go.mod` module path 用 `github.com/code4j/...`，但 git remote owner 是 `yyykf` —— 徽章/链接已统一用 `yyykf`；module path 是否改名另议
