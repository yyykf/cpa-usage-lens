# 06-04 按 API key 看用量（脱敏存储）— 实现与验收摘要

> 日期：2026-06-05　分支：`feat/api-key-usage-breakdown`（基于 main）
> 关联：`.trellis/tasks/06-04-api-key-usage-breakdown/prd.md`

## 概述

给 CPA Usage Lens 新增「按 API key 看用量」维度，坚守隐私底线 **「明文 api_key 绝不入库」**——只存不可逆的 `sha256` 指纹 + 掩码（`sk-…后4位`）。account 与 api_key 是正交的两个视角，key 做**独立榜单**（不下钻）。

## 完成范围（对应 PRD 6 件事）

| # | 范围 | 状态 |
|---|---|---|
| 1 | 采集脱敏存 key（指纹+掩码） | ✅ |
| 2 | key 进日聚合主键（长期可见） | ✅ |
| 3 | 后端 key 聚合 + `/api/keys` | ✅ |
| 4 | 前端「API key 用量榜」（独立、与账号榜并列） | ✅ |
| 5 | 存量回填 | ⏳ 逻辑就绪，**实际回填待上线 rollout**（需 code4j + 生产库） |
| 6 | README 卖点文案更新 | ✅ |

## 关键决策（详见 prd.md Decision）

- **指纹**（sha256 全长 hex，做精确区分/聚合主键）+ **掩码**（`sk-…后4位`，界面展示）两列，明文绝不入库。
- account 与 key 正交 → **独立 key 榜**，不下钻、不引路由（延续单页架构）。
- 存量**无原始 key 可还原**（库从未存过 + CPA 队列 60s 即删）→ 按 code4j 决策统一回填其当前 key；未回填前归 `none`（「非 key 认证」）桶。
- 卖点文案：「绝不入库」→「**明文绝不入库，仅留不可逆指纹**」（`response_headers`/`fail.body` 仍完全不入库）。

## 改动文件

- **迁移**：`supabase/migrations/20260605002633_add_api_key_dimension.sql`（hot/daily 加 `key_fingerprint`/`key_mask`、daily 主键扩为 4 列、索引、幂等 DO 块按列名判定）
- **后端**：`collector/{sanitize,sanitize_test,collector}.go`、`model/types.go`、`db/{events,rollup,queries}.go`、`report/{report,report_test}.go`（`BuildKeys`）、`api/{handlers,server}.go`（`/api/keys`）
- **前端**：`types.ts`、`lib/api.ts`、`components/KeyTable.tsx`（新）、`pages/Dashboard.tsx`
- **文档**：`README.md`、`README.zh-CN.md`（卖点文案）、`docs/deployment.md`（破坏性上线顺序）

## 质量验收（双关，code4j 强制要求）

### 关 1：AI 自测（本地隔离 + 浏览器核实）
- docker 隔离 Postgres + 4 把 key（含 `none` 桶）× 3 天测试数据（**未连生产**，用完即删）。
- 浏览器登录核实：key 榜数据全对、按成本降序、「非 key 认证」桶正确显示、视觉两榜对称、配色（碳黑+青绿+琥珀）协调。
- **正交性印证**：账号榜 `alice@example.com`=1,560 请求 = key 榜 `f001`(1,200)+`f003`(360)——同一账号的量分散在两把 key，两个视角各自独立、对得上。
- `go build/vet/test` + 前端 `npm run build` 全绿。

### 关 2：Codex review（插件，xhigh effort）
- **首轮 Block**（2 必修）：① 短 key 明文会进 `key_mask`（安全一票否决）；② 主键 3→4 列扩维对旧采集器破坏性、不可滚动。
- 修复 7 项后**复审 RESOLVED**：两 Block 解除，确认无新明文泄露路径、无新问题。

## Codex 发现 → 已修（7 项）

1. **[CRITICAL]** 短 key → 定长占位 `"****"`（不回显任何原文片段）+ 守门测试。
2. **[安全/贴 PRD]** 掩码前缀收紧为 `sk-…后4位`（原来多暴露 4 位）。
3. **[健壮]** 掩码用 `[]rune`（防非 ASCII key 切坏 UTF-8）；指纹仍用原始 bytes。
4. **[内存]** 明文提取指纹+掩码后立即置空，缩短内存生命周期。
5. **[迁移]** 幂等块改为按主键实际列名集合判定（不只列数）。
6. **[文档]** 迁移注释澄清回填语义（无原始 key 可还原）。
7. **[部署]** `docs/deployment.md` 新增破坏性上线顺序章节（中英双语）。

## 观察项 / 技术债（留 code4j 定）

- **DRY**：`BuildKeys`/`BuildAccounts`、`KeyTable`/`AccountTable` 结构性重复，Codex 建议抽公共组件。**本轮按 YAGNI 未抽**——account 榜与 key 榜未来可能分化（如 key 加别名）。若确定不分化，可抽 `UsageTable` 基础组件 + 通用聚合 helper（account/key 各传维度 key + 标签 getter）。

## ⚠️ 待办：生产上线 rollout（需 code4j + vmrack，**不支持零停机**）

daily 主键 3→4 列对生产正在跑的旧采集器（旧 rollup 用 3 列 `ON CONFLICT`）是破坏性变更。**必须按序**（详见 `docs/deployment.md`）：

1. **code4j** 停 vmrack 采集器（`COLLECTOR_ENABLED=false`）
2. 生产库跑迁移 `20260605002633_add_api_key_dimension.sql`
3. **code4j** 部署新代码到 vmrack
4. **（AI 可执行）回填存量**：用 code4j 当前 key 算指纹，经 Supabase 把存量 daily 的 `none` 桶刷成当前 key；明文本地算、绝不落库
5. **code4j** 重启采集器（`COLLECTOR_ENABLED=true`）

## 安全提醒

- code4j 的明文 key 在 brainstorm 对话中出现过（已视为暴露），**建议上线后在 CPA 轮换该 key**；轮换不影响回填（历史归旧 key、新数据走新 key）。

## 本地验证规约遵守

- 全程**未连生产 CPA 队列 / 生产 Supabase**；自测用 docker 隔离 PG，用完即删。迁移复验、rollup 幂等、列名守护均在本地隔离库完成。
