# CPA Usage Lens MVP

## Goal

为运行 CLIProxyAPI(CPA) 的小服务器用户，提供一个**不占用本地资源**的账号级用量分析工具：用外部采集器消费 CPA 的用量队列，把精简数据写入 Supabase(云数据库)，并提供一个**美观的 Web 页面**，随时查看每个账号在一段周期内的请求数、token 用量与估算成本。核心动机：用户服务器性能紧张、不愿再本地跑数据库。

## What I already know（已确认）

### 来自用户的决策
- 一定要做。差异化 = 数据上 Supabase 云、本地近乎零负担（现有同类项目几乎都用本地 SQLite）。
- 必须有 Web 页面，且要有**一定美观度**（不是简陋 HTML）。
- 核心诉求：看每个账号在一段周期内**花多少钱、用多少 token**。
- 热明细保留天数**不写死**，默认 7 天、**可配置**（额度大的人想看更久）。

### 来自调研（已查实）
- **数据源**：`GET /v0/management/usage-queue`（需 management key），返回每条请求的用量 JSON；也可用同端口的 Redis RESP 队列消费。
- **pop 语义**：取出即删除 → 采集器必须常驻且勤快消费；同一队列**不能多采集器并行**（会互吃数据）。
- **队列真实字段（v7.1.31 实测）**：`timestamp(带本地时区偏移), latency_ms, ttft_ms, source(账号邮箱), auth_index, tokens{input/output/reasoning/cached/cache_read/cache_creation/total}, failed, fail{status_code, body}, response_headers(⚠️大+含Set-Cookie等敏感,须剥离), provider, model, alias, endpoint, auth_type, api_key(⚠️敏感,须剥离), request_id, reasoning_effort, service_tier`。
- **成本不由 CPA 给**：需自己用价格表（LiteLLM `model_prices_and_context_window.json`）按 token 估算。ccusage / CPA-Manager 均如此。
- **已有同类项目**（多为本地 SQLite）：zhanglunet/cliproxyapi-usage-dashboard、Willxup/cpa-usage-keeper、CPA-Manager 等。

## Assumptions（部分已验证）
- ✅ **usage-queue 在 CPA「2026-06 移除内置统计」后仍保留**（已查实：删的是内存聚合统计 `internal/usage/`，队列完好且持续加功能至 v7.1.32；官方反而主推"队列+外部持久化"路线）。
- 单 CPA 实例、单采集器（第一版）。
- 目标量级约 1000 请求/天，7 天明细 + 紧凑聚合可舒适待在 Supabase Free 500MB 内。
- ✅ 已在 v7.1.31 实例抓真实样本核对字段：发现缓存 token 实际拆分（cache_read/cache_creation）、`fail.status_code` 真实存在、并多出 `response_headers`（大+敏感，须剥离）等，详见 Technical Notes。

## Open Questions（已全部收敛 ✅）
- ✅ 整体架构形态 → 见 Decision D1
- ✅ 技术栈 → 见 Decision D2
- ✅ MVP 页面范围与周期 → 模块 1-5 + 今天/近7天/近30天/自定义
- ✅ 鉴权 → 单用户密码登录 + bcrypt + token，明文不落库
- ✅ 成本 query-time + 价格表每日/手动刷新 + 时区默认 Asia/Shanghai 可配
- ✅ 容量可见性 → 健康模块显示真实占用大小（不用百分比）

## Requirements（evolving）
- 通过 HTTP `GET /v0/management/usage-queue?count=N` 轮询消费单个 CPA 实例的用量事件（穿透反代、适合小服务器），写精简明细到 Supabase
- **防丢数据**：采集器对"已 pop 但尚未确认写入 Supabase"的数据先本地落盘缓冲，确认写入成功后再丢弃（云版独有风险：pop 不可回放）
- 写库前剥离敏感/大字段：`api_key`、`response_headers`（含 Set-Cookie 等）、`fail.body`（spec 非目标 + 隐私 + 容量）
- 用 `request_id` 去重（幂等插入，参考 zhanglunet 的 UNIQUE 约束做法）
- 明细保留可配置短窗口（默认 7 天）；按账号+模型聚合每日用量并长期保留
- 删明细不丢聚合（幂等 rollup，可重算最近几天处理延迟事件）
- 容量有界：backend 每日定时清理过期明细；**先确保该日已 rollup 进聚合表、再删该日明细**（删除窗口 > 聚合重算窗口，避免误删未聚合数据）。明细体积 ≈ 保留天数 × 日请求量（有上限、不随时间增长）；聚合行极小且增长缓，长期稳在 Supabase Free 500MB 内
- 健康模块显示数据库占用的**真实大小**（Postgres 可查，明细/聚合分别显示），**不显示百分比**（用户容量套餐不一，分母无法写死）
- 用 LiteLLM 价格表估算成本：**查询时实时计算（token × 当前单价），不在明细/聚合里存死 cost**；价格表启动拉取 + 每日自动刷新 + 手动刷新；只存用过的模型；缺价模型先标"未知"，补价后自动生效（无需重启/回填）
- 暴露采集器健康状态（延迟/游标/最后错误）
- 提供美观 Web 页面（dashboard），MVP 含 5 个模块：① 顶部总览(周期内 总请求/token/成本/失败) ② 周期切换 ③ 各账号用量榜(请求/token/成本/失败，**核心**) ④ 每日趋势图 ⑤ 采集器健康
- 总览/账号榜/趋势均**跟随选定周期**；周期 = 今天 / 近7天 / 近30天 / 自定义日期范围（滚动窗口，不做"本周/本月"）；"额度"= 看用量统计，不做预算管理
- 单用户密码登录保护：部署时经环境变量注入登录密码，backend 用 bcrypt 校验（明文不落库/不落文件），登录后签发有时效 token；所有数据 API 需带 token

## Acceptance Criteria（evolving）
- [ ] 约 1000 请求/天下，存储舒适低于 Supabase Free 上限
- [ ] 页面可看「某账号 今天/近 N 天 请求数 / token / 成本」
- [ ] 可查看热窗口内近期失败 / 高成本请求
- [ ] 删旧明细后长期聚合指标不丢
- [ ] 采集器异常/重启后不丢"已 pop"的数据

## Definition of Done
- 测试覆盖关键路径（去重、聚合幂等、保留删除、落盘缓冲恢复）
- Lint / typecheck / CI 通过
- 文档：部署说明、Supabase 容量与保留假设、配置项说明、停机丢数据风险告知
- 风险回滚：采集器中断重启、队列堆积的应对

## Technical Approach

- **架构**：单 backend(Go) = 后台采集循环 + rollup/清理定时任务 + 价格表刷新 + 对外 HTTP API + 鉴权；frontend(React/shadcn) 静态站；数据在 Supabase 云。Docker Compose 编排 backend+frontend 两容器（详见 D1/D2）。
- **数据流**：CPA `/usage-queue` --轮询 pop--> 采集器(剥 api_key / 按 request_id 去重 / 落盘缓冲确认写入) --> `request_events_hot`(留 7 天,可配) --每日 rollup--> `daily_account_usage`(按 账号+模型+天 存 token,长期) --超期--> 清理(先确认已聚合再删)。
- **成本**：query-time = token × `model_prices`(LiteLLM,每日+手动刷新) 当前单价；缺价标"未知"。
- **容量**：明细有界(保留天数×日请求量) + 聚合极小 → 稳在 500MB；健康模块显示 Postgres 真实表大小。
- **鉴权**：单用户密码(env 注入) → bcrypt 校验 → 签发 token → API 校验。
- **时区**：可配(默认 Asia/Shanghai)，所有"天"按它界定。

## Decision (ADR-lite)

### D1: 架构形态 = 前后端分离 + Docker Compose + Supabase 云
- **Context**: 用户服务器性能紧张、不愿本地跑数据库；习惯前后端分离，不接受前端直连数据库。
- **Decision**: 采用「优化版 C」——前端项目 + backend 项目代码分离，用 Docker Compose 编排为两个轻量容器（`backend` + `frontend`）一键部署在用户服务器；数据库用 Supabase 云，**不进 Compose**。采集器与后端 API **合并为一个 `backend` 服务**（同进程：后台采集循环 + 对外 HTTP API），因采集器受"全局单实例"硬约束、无独立扩展需求。前端只调 backend API，**不直连数据库**。
- **Consequences**: 比"前端直连 DB"多常驻 backend+frontend 两个轻量容器（量级几十 MB），换来前后端分离、不暴露 DB、一键部署；Compose 内无数据库容器，本地负担仍很轻。竞品多为单体（前端由后端 serve 或单脚本），本方案更规范。

### D2: 技术栈 = Go(backend) + React/shadcn(frontend)
- **Context**: 用户对 Node/Python/Go/前端均不熟、全程靠 AI 辅助；核心诉求是占用小、页面要现代好看。
- **Decision**: backend = **Go**（单二进制，最省资源；同进程跑后台采集循环 + 对外 HTTP API）。frontend = **React + Vite + TypeScript + Tailwind CSS + shadcn/ui**（现代审美），图表用 **Recharts**（shadcn chart 即基于它）。部署 = Docker Compose 编排 backend + frontend 两容器。
- **Consequences**: Go 内存占用最低、契合小服务器；前后端语言不同无法共享类型（但用户两边都不熟，此优势本就不适用）。shadcn 现代感强但需少量 Tailwind 配置，AI 辅助可消化；React 生态 AI 辅助质量最高。

## Implementation Plan（小步拆分）

- **PR1 脚手架 + Supabase 表**：仓库结构(backend/ frontend/)、Compose 骨架、表(`request_events_hot` / `daily_account_usage` / `model_prices` / `collector_state`)、配置(env)、backend 连通 Supabase。
- **PR2 采集器**：轮询 `/usage-queue` → 剥敏感字段 → request_id 去重 → 落盘缓冲+确认写入 → 写明细；更新 `collector_state`；测试(去重/缓冲恢复)。
- **PR3 rollup + 保留清理**：每日聚合(幂等/重算近几日)；先确认已聚合再删过期明细；测试。
- **PR4 价格表 + 成本**：拉 LiteLLM → `model_prices`(只存用过)；每日+手动刷新；query-time 成本；缺价标未知。
- **PR5 查询 API + 鉴权**：登录(bcrypt+token)+中间件；数据 API(总览/账号榜/趋势/健康+容量,带周期参数)。
- **PR6 前端 dashboard**：React+Vite+shadcn+Tailwind；登录页；5 模块；调 API。**视觉按 `.project_context/explore/frontend/design-system.md`（暗色 Bento 控制台）实现，用 ui-ux-pro-max + frontend-design skill 辅助。**
- **PR7 部署 + 文档**：Compose 完善；部署/配置/容量假设/停机丢数据风险文档。

## Out of Scope
- 不存原始 prompt/响应、完整请求响应体、完整失败 body
- 不存密钥 / 原始 token
- 不替代 CPA 路由 / 认证
- 不支持多采集器消费同一队列
- 第一版不做企业级高规模可观测平台
- （待定）多 CPA 实例、告警、配额预测
- （第二批，非 MVP）页面⑥按模型分布、⑦近期请求明细（排障）

## Technical Notes
- 字段已按 v7.1.31 真实样本核对（修正早期假设）：`fail.status_code` **真实存在**（HTTP 状态码，可存）；`cache_read_tokens`/`cache_creation_tokens` **真实拆分**（成本可按读/写不同单价精确算，无需近似）；`cost_usd` **不入库，查询时按最新价格表实时计算**（聚合表保留 model 维度按模型单价算）。
- ⚠️ 队列含 `response_headers`（完整响应头，体积大且含 `Set-Cookie` 等敏感信息）与 `api_key`、`fail.body`：采集器**必须剥离，绝不入库**。
- `timestamp` 带本地时区偏移（如 `+08:00`），非 UTC；时区换算据此（聚合"天"按可配时区）。
- v7.1.31 额外字段：`ttft_ms`（首字节延迟，可选存为性能指标）、`reasoning_effort`、`service_tier`。
- 项目编码规范：`.trellis/spec/{frontend,backend}/`。
- **采集硬约束（必须遵守）**：① 同一 CPA 队列全局只能跑一个采集器；② 绝不开 `SUBSCRIBE usage`（会让 FIFO 端点取不到数据）；③ 采集器停机超过 `redis-usage-queue-retention-seconds`（默认 60s，最大 3600s）期间的数据**永久丢失**（pop 不可回放）——文档需明确告知，并尽量保证采集器高可用/自动重启。
- 采集方式优先 HTTP `GET /usage-queue?count=N`（穿透反代），而非 RESP 裸 TCP。CPA 版本下限 v6.10.8+，目标 v7.1.x。
- 坑：`config.example.yaml` 注释把 `usage-statistics-enabled` 写成"内存聚合开关"，实际是"队列发布总闸"，必须为 `true`，否则队列无数据。

## Research References
- `research/cpa-usage-queue-viability.md` — ✅ 数据源安全可开工；删的是内存聚合，队列保留并持续迭代（v7.1.32）
- `research/reference-implementations.md` — zhanglunet 最干净(RESP+RPOP / request_id 去重 / UNIQUE 幂等)但不算成本；成本估算参考 CPA-Manager / cpa-usage-keeper（LiteLLM 价格表，只存用过的模型）
- `.project_context/explore/frontend/design-system.md` — 前端视觉风格规范（暗色 Bento 控制台：配色变量/字体/Bento 布局/各组件视觉/图表/动效/a11y 清单）
