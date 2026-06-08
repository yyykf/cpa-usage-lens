# 按 API key 看用量（脱敏存储）

## Goal（目标）

把用量统计维度从「账号(source) × 模型 × 天」**下钻到 API key 这一层**，让同一账号下分发的多把 key 各自的用量（请求 / token / 成本 / 失败）可见。
核心约束：**坚持「明文 api_key 绝不入库」的隐私卖点**——只存不可逆的脱敏形态（指纹 / 掩码），不存明文。

## What I already know（已掌握的事实）

### 功能定位与历史
- 这是 `06-02-dashboard-refresh-visual-upgrade` 明确拆出的 Out of Scope 项（见其 `prd.md:76-80`），不是遗漏，是当时主动单独排期。
- 现有统计维度：账号(source) × 模型 × 天。看不到「每把客户端 API key 各用了多少」。
- **关键认知（code4j 指正）：account 与 api_key 是正交的两个视角，不是层级关系**：
  - `source`(账号) = CPA 内部配置的上游账号 → 运营者视角「我哪个上游在扛量」；
  - `api_key` = 客户端调 CPA 用的 key → 对外视角「谁在用我的服务」；客户端不知道 CPA 内部有哪些账号。
  - 故 key 维度做**独立并列的榜单**（与账号榜平级），**不是**账号下钻出 key（下钻假设错误）。

### 「api_key 从不入库」卖点的现状（很硬，是结构性保证）
- README.zh-CN.md:37 白纸黑字卖点：「写库前剥离敏感字段：api_key / response_headers / fail.body 绝不入库」。
- CPA 队列原始 payload **确实含明文 api_key**（`collector/payload.go:22`，认证后的调用方 key，形如 `sk-...`）。
- 采集器 `collector/sanitize.go:toEvent()` 转换时，目标结构 `model.UsageEvent`（`model/types.go:20-37`）**从字段层就不含 api_key**——想存都没地方存。
- DB 两张表 `request_events_hot` / `daily_account_usage`（`supabase/migrations/20260530185206_init_schema.sql`）**无任何 key 相关列**。
- 有单测 `collector/sanitize_test.go` 守门。

### 已踩过的坑（不要重犯）
- `auth_index` 字段曾被误判可当 key 维度，**已纠正**：它是**账号(source)的哈希**，与 api_key 无关，顶不上（`06-02 prd.md:78`）。

### 存量数据无法还原
- 库里从没存过 key；CPA 队列默认 **60 秒即删**（采过即焚，见 `research/cpa-usage-queue-viability.md:87`）→ 历史数据补不回真实 key，只能回填。

### code4j 已拍板的决策
- **隐私底线**：明文绝不入库；**不可逆的脱敏形态可以接受**（前缀 + 后几位）。「不入库」指的是不接受明文，只要不泄露明文即可。
- **存量回填**：统一刷成 code4j 当前在用的这把 key；若有他人在用，回填脚本默认刷成 `default`。

## Assumptions（待验证假设）
- 这个功能要在前端有可视化呈现（不只是后端能查）。
- 生产采集器在生产采集器主机上运行，改采集逻辑 + 改 schema 后需要重新部署 + 跑生产 Supabase migration。

## Open Questions（仅保留阻塞/偏好类）
- ~~Q1 脱敏形态~~ → **已定：指纹(哈希) + 掩码 两列都存**（指纹精确区分/做聚合主键，掩码 `sk-…后4位` 界面展示；均不可逆、不含明文）。
- ~~Q2 前端展示形态~~ → **已定：独立「API key 用量榜」**（与账号榜并列，单页，不下钻、不引路由）。
- ~~Q3 聚合粒度~~ → **已定：key 进 `daily_account_usage` 主键**，长期可见（主键变「天×账号×模型×key」）。
- ~~Q4 回填策略~~ → **已定：存量统一刷当前 key**（无法分辨归属，只能整体刷；明文执行时本地喂入算指纹，不落库）。
- ~~Q5 MVP 边界~~ → **已定：6 件事**（脱敏存储 / key 进主键 / 后端聚合+API / 前端 key 榜 / 回填脚本 / README 更新）；别名不做、不下钻、不引路由。

## Requirements（evolving）
- 采集时对明文 api_key 做不可逆脱敏，**存两列：指纹(哈希) + 掩码(`sk-…后4位`)**，明文绝不落库（卖点措辞更新为「明文绝不入库，仅留不可逆指纹」）。
- key 进 `daily_account_usage` 主键（天×账号×模型×key），后端支持按 key 维度**长期**聚合查询。
- 前端新增独立「API key 用量榜」（与账号榜并列、单页），按 key 看请求/token/成本/失败。
- key 榜指标口径（请求/token/成本/失败 + token/成本切换）**对齐现有账号榜/模型榜**，不另立一套（DRY）。
- 非 api_key 认证（oauth 等无 key）的请求归一个「其他/非 key 认证」桶，不丢数据、不报错。
- 提供存量回填迁移脚本：存量无法分辨 key 归属 → **统一回填成 code4j 当前在用的 key**（`default` 作通用兜底）；明文 key 在执行迁移时本地喂入、当场算指纹+掩码，明文不落库。

## Acceptance Criteria（evolving）
- [ ] 采集器对明文 key 脱敏后入库，明文不出现在任何表/日志（单测守门）。
- [ ] 后端可按 key 维度返回用量聚合。
- [ ] 前端展示按 key 的用量。
- [ ] 回填脚本可把存量数据按约定策略补上 key 维度。
- [ ] 卖点文案（README）同步更新为脱敏口径。
- [ ] 前后端质量检查（lint / type-check / 构建 / 相关测试）通过。

## Definition of Done（完成定义）
- 代码在基于 `main` 的 feature 分支提交（不动 `main`）。
- schema 变更走 `supabase/migrations` 新增迁移文件。
- 「明文 api_key 绝不入库」底线不被破坏（单测 + 代码审查双重保证）。
- **回填明文 key 安全**：仅执行时内存使用，绝不写入代码/commit/PRD/任何文件，绝不明文落库（只写指纹+掩码）。
- **双重验收（code4j 强制要求）**：① 自测——本地起前后端（`COLLECTOR_ENABLED=false` 只读生产库）打开浏览器核实 key 榜正确；② 自测通过后交 **Codex review**（走插件、xhigh effort）审查代码 + 打开浏览器判断是否正确。
- 执行摘要写入 `.project_context/execution/changes/06-04-api-key-usage-breakdown/`。

## Technical Approach（技术方案概要）

- **脱敏**：采集 `sanitize.go:toEvent()` 对明文 api_key 算 `sha256`→指纹(hex)，按「前缀+后4位」取掩码；`model.UsageEvent` + `request_events_hot` + `daily_account_usage` 各加 `key_fingerprint` / `key_mask` 两列。
- **聚合**：`rollup.go:RollupRange` 的 GROUP BY 加 `key_fingerprint`（掩码随指纹带出）；`daily_account_usage` 主键扩为 `(usage_date, source, model, key_fingerprint)`。
- **查询/API**：`queries.go` 加按 key 聚合；`handlers.go` 加 `/api/keys`（对齐 `/api/accounts`）。
- **前端**：仿 `AccountTable` 加「API key 用量榜」组件，挂进单页 `Dashboard.tsx`。
- **回填**：新增 `supabase/migrations` 迁移 + 一次性回填；明文 key 执行时本地喂入算指纹，不落库。
- **非 key 认证**：`key_fingerprint` 为空时落到固定哨兵值（如 `none`）归「其他」桶（主键非空约束）。

## Decision（ADR-lite）

**Context**：要按 key 看用量，但「明文 api_key 绝不入库」是硬卖点；且 account 与 key 是正交视角、存量从未存过 key。

**Decision**：采集时存**不可逆的指纹(哈希)+掩码**两列（明文仍不入库）；key 作为**独立维度**进日聚合主键，前端做**独立 key 榜**（不下钻）；存量**统一回填当前 key**。卖点措辞从「绝不入库」改为「明文绝不入库，仅留不可逆指纹」（`response_headers`/`fail.body` 仍完全不入库，不变）。

**Consequences**：功能落地且隐私底线（明文不落库）不破；聚合主键多一维、行数可控；存量 key 归属为近似（统一刷），新数据起真实记录；README 卖点需同步改文案。

## Out of Scope（本 task 不做）

- **key 别名/备注**：界面只显示掩码 `sk-…789`；不做「key→人话名字」映射（YAGNI，未来多人/记不住再加）。
- **账号→key 下钻**：account 与 key 正交，做独立并列榜单，不做层级下钻。
- **前端路由/子页面/侧边栏**：保持单页（延续 06-02 约束）。
- **后端已存未展示的其他维度**（latency/ttft 性能、fail_status_code、provider/endpoint 等）：本次不做。

## Technical Notes（约束 / 文件 / 引用）
- 采集脱敏：`backend/internal/collector/{sanitize.go,payload.go}`、`backend/internal/model/types.go`、单测 `sanitize_test.go`。
- DB：`supabase/migrations/20260530185206_init_schema.sql`、`backend/internal/db/queries.go`（聚合查询）。
- API：`backend/internal/api/handlers.go`（现有 `/api/{overview,accounts,trend,models,collector}`）。
- 成本：`backend/internal/pricing`（缺价标未知）。
- 前端：`frontend/src/pages/Dashboard.tsx` 及 `components/`。
- 本地验证规约：生产采集器在生产采集器主机，本地 `COLLECTOR_ENABLED=false` 只读；写入类验证用独立 Supabase / 本地 PG / 单测喂假 payload，**绝不连生产 CPA 队列**。
- **指纹算法（采集与回填必须一致）**：指纹 = `sha256(明文 api_key)` 小写 hex 全长；掩码 = `sk-…后4位`（实现时可含少量前缀，如 `sk-前缀…后4位`）。两端同算法，否则同把 key 指纹对不上、被当两把。
- **回填目标 key**：code4j 已在对话提供明文（掩码 `sk-…2216`，目前在用的那把），并授权 AI 经 Supabase 执行回填；回填时用上述算法本地算指纹，**明文绝不写入任何文件/commit/PRD/库**。已建议 code4j 回填后轮换该 key。

## Research References
- [`../archive/2026-05/05-30-cpa-usage-lens-mvp/research/cpa-usage-queue-viability.md`] — CPA 队列 payload 字段、60s 保留、pop 即删等硬约束。
