# 仪表盘自动刷新与视觉/视图升级

> 方向参考 README 概念图 `docs/assets/product-intro.png`（仅视觉方向，非实现截图）。
> 本 task 仅含「前三批」；按 API key 看用量、子页面/侧边栏、项目介绍落地页均**不在本 task**。

## Goal（目标）

把仪表盘从「静态单屏、卡片朴素」升级为「会自动刷新、信息更丰富、卡片更精致」的版本：
沿用现有配色（碳黑底 + 青绿渐变 + 琥珀/红点缀），**收敛辉光、避免 AI 味**。
视觉类改动**先出设计稿、经 code4j 确认后再实现**。

## Requirements（需求）

### 1. 自动刷新选择器（纯前端，可直接做）
- 顶栏提供**可切换**的自动刷新档位：`关闭 / 5s / 10s / 30s / 60s`，**默认 30s**。
- 选中某档后，按该间隔自动重新拉取全部数据；选「关闭」则停止轮询。
- 修正现有「采集中」徽标（`LivePulse.tsx` 的 `LiveBadge`）的**误导**：它当前写死常亮绿，
  既不反映真实采集状态、也不触发刷新。需改为反映「自动刷新状态」或「真实采集健康」。

### 2. KPI 环比角标
- 顶部 4 个 KPI（请求 / Token / 成本 / 失败）在绝对值之外，显示与**上一个等长周期**对比的增减百分比（如 `↑12%` / `↓8%`）。
- 形式：一个总数 + 一个增减角标（参考概念图）。
- 需后端新增「上一周期」对比数据（当前 `queries.go` 只取单区间）。
- 无上一周期数据 / 成本未知 时的兜底显示，在设计稿中定。

### 3. 模型用量增强
- 新增**「模型总量排行」**视图（水平条形，按周期内总量降序），与现有「每日 100% 堆叠柱」互补
  （前者看「整段时间谁用得最多」，后者看「每天占比变化」）。
- 支持**按 Token / 按成本**两种口径切换（成本口径用 `model_prices` 实时算），**默认按 Token**。

### 4. 采集器健康卡（不做圆环）
- code4j 决定**不做圆环**：圆环只能做成象征性健康环，语义偏装饰、价值不高。
- 保持现有纯文字健康卡（`CollectorHealth.tsx`），仅随整体做轻度视觉对齐。

### 5. 设计稿先行（流程约束）
- 第 2/3 项及卡片排版属于视觉改动，**实现前必须先出设计稿**，经 code4j 确认。
- 第 1 项（自动刷新）为标准交互，可不出独立设计稿直接实现（或在整体稿中带一笔）。

## Acceptance Criteria（验收标准）

- [ ] 顶栏有可切换自动刷新选择器（含「关闭」档），**默认 30s**，选中后按间隔自动刷新数据，关闭后停止轮询。
- [ ] 「采集中」徽标不再写死常亮，能反映真实采集状态或自动刷新状态。
- [ ] 4 个 KPI 显示对比上一等长周期的增减百分比，含无数据兜底。
- [ ] 新增「模型总量排行」视图，支持 Token / 成本 两种口径切换，**默认 Token**。
- [ ] 第 2/3 项视觉改动在实现前已有设计稿且经 code4j 确认。
- [ ] 前后端质量检查（lint / type-check / 构建 / 相关测试）通过。

## Definition of Done（完成定义）

- 代码在基于 `main` 的分支提交（feature 分支，不动 `main`）。
- 设计稿留档于 `.project_context/`（如 `plan/dashboard/` 或 `explore/dashboard/`）。
- 未引入子页面 / 侧边栏 / 前端路由。
- 未改动 API key 入库行为（保持「明文 api_key 从不入库」卖点）。
- 执行摘要写入 `.project_context/execution/changes/06-02-dashboard-refresh-visual-upgrade/`。

## Technical Approach（技术方案）

- **前端**：自动刷新用 `setInterval` + 卸载清理；档位状态可存 `localStorage` 记住偏好；复用现有 `loadData`。
- **后端**：环比需新增「上一等长周期」聚合查询（`queries.go` 现仅 `QueryDailyUsage` 单区间）；
  模型成本口径复用 `internal/pricing`。
- **图表**：圆环与水平条形复用现有 `recharts`，或轻量 SVG（避免引入新依赖，遵循 KISS/YAGNI）。
- **配色**：不改色板（`accent` 色相 ~186 青、`data-success` 绿、成本琥珀、失败红），仅收敛辉光质感。

## Decision（ADR-lite）

**Context**：README 概念图视觉/信息更丰富，但产品当前为静态单屏、卡片朴素，且「采集中」徽标存在「看似实时、实则不刷新」的体验矛盾。

**Decision**：先做「自动刷新 + 视觉/视图增强」三批，视觉部分先出设计稿确认；
按 API key 看用量、子页面/侧边栏单独排期，保持当前单页架构。

**Consequences**：体验明显提升且不引入路由复杂度；API key 安全卖点不受影响；
环比需后端多一次「上一周期」查询（成本可控）。

## Out of Scope（不在本 task）

- **按 API key 看用量**：单独 task（基于 CPA 源码调研）。结论：
  - CPA 队列事件**含调用方明文 api_key**（`api_key` 字段 = CPA 内部 `userApiKey`，认证后写入），但本项目第一版**已剥离不存**；
  - `auth_index` 是**账号(source)哈希**、与 api_key 无关，不能当 key 维度（之前误判已纠正）；
  - 故只能走「采集时脱敏存 key」，**会松动『api_key 从不入库』卖点，待 code4j 拍板**；
  - 存量无法还原（库未存过 + CPA 队列默认 60s 即删），按 code4j 方案**统一回填最常用的那把 key**（迁移脚本）。
- **子页面 / 侧边栏 / 前端路由**：等将来做 API key 详情等独立页面时再一起引入。
- **项目介绍落地页**：概念图左栏文案，确认丢弃。
- **后端已存但未展示的其他维度**（`latency_ms`/`ttft_ms` 性能、`fail_status_code` 失败分析、`provider`/`endpoint` 等）：后续可选，本次不做。

## Technical Notes（技术备注）

- 概念图：`docs/assets/product-intro.png`；真实截图：`docs/assets/dashboard-screenshot.jpg`。
- 相关前端：`frontend/src/pages/Dashboard.tsx`、`components/{StatRail,TrendChart,ModelStackChart,CollectorHealth,PeriodSwitcher}`、`components/dashboard/LivePulse.tsx`。
- 相关后端：`internal/db/queries.go`（需加上一周期查询）、`internal/api/handlers.go`、`internal/pricing`。
- 现有数据 API：`/api/overview`、`/api/accounts`、`/api/trend`、`/api/models`、`/api/collector`。
- 周期参数：`today / 7d / 30d / custom`（`periodQuery` 已支持）。

## 本地验证规约（重要：勿抢生产队列）

- 生产采集器跑在 **vmrack**（`COLLECTOR_ENABLED=true`），CPA 队列 **pop 即删**，全局仅允许一个采集器——本地**绝不可**再起采集器。
- **本 task 本地验证**：`.env` 置 **`COLLECTOR_ENABLED=false`**（见 `backend/cmd/server/main.go:49-64`——此时仅跑查询 API，**不采集 / 不 rollup / 不清理**），`DATABASE_URL` 连**同一生产 Supabase**，读 vmrack 写入的真实数据即可验证（前三批均为**读路径**，不需要消费队列）。
- **自动刷新验证**：靠 vmrack 持续写入观察数字变化；或打开浏览器 Network 面板确认每 30s 发出一次 API 请求。
- **勿点前端「刷新价格表」**（会写 `model_prices`）；`pricing` 的 `RunDaily` 在 `COLLECTOR_ENABLED` 开关之外、每日自动刷一次，开发期一般不触发。
- **彻底隔离 / 写入类验证**（如将来 API key 脱敏采集）：用独立 Supabase 或本地 Postgres + 跑 `supabase/migrations` 造测试数据，或写**单元测试喂假 payload**（参考 `collector/sanitize_test.go`），**绝不连生产 CPA 队列**。
