# Deployment & Operations · 部署与运维

[English](#english) · [简体中文](#chinese)

<a id="english"></a>

> **English** — 简体中文 version is [below](#chinese).

## Prerequisites

- A **Supabase project** (the free tier is enough — 500 MB comfortably holds ~1,000 requests/day)
- A server that can run **Docker + Docker Compose**
- A **CLIProxyAPI (CPA)** instance, version **v6.10.8+** (v7.1.x recommended), with management enabled

---

## 1. Create tables in Supabase

Using the Supabase CLI (recommended — reproducible):

```bash
supabase link --project-ref <your-project-ref> -p '<db-password>'
supabase db push --db-url "<Session pooler connection string>"
```

Or run `supabase/migrations/20260530185206_init_schema.sql` directly in the Supabase Dashboard's SQL Editor.

> This creates 4 tables: `request_events_hot` (hot detail) / `daily_account_usage` (daily rollup) / `model_prices` (prices) / `collector_state` (collector state). All have RLS enabled with **no** policies — the backend connects directly via the connection string, the frontend never connects directly, and the Data API denies access by default.

---

## 2. Required CPA configuration (⚠️ miss any and the queue stays empty)

In CPA's `config.yaml`, confirm:

| Setting | Value | Notes |
|---------|-------|-------|
| `usage-statistics-enabled` | `true` | **Master switch for queue publishing** (the official comment calling this an "in-memory aggregation switch" is outdated — it actually gates the queue and must be `true`) |
| `redis-usage-queue-retention-seconds` | `3600` | Queue retention window — **set it to the max of 3600** to give the collector a buffer for brief downtime |
| `remote-management.secret-key` | set | Management key — a prerequisite for enabling the queue |

---

## 3. Configure `.env`

Copy `.env.example` to `.env` and fill in:

| Variable | Description |
|----------|-------------|
| `CPA_BASE_URL` | CPA address, e.g. `https://your-cpa-host.com` |
| `CPA_MANAGEMENT_KEY` | CPA management secret key |
| `DATABASE_URL` | Supabase **Session pooler** connection string (include `?sslmode=require`) |
| `DASHBOARD_PASSWORD` | Login password (injected in plaintext; the backend verifies with bcrypt and never stores it) |
| `AUTH_TOKEN_SECRET` | Token-signing secret — generate with `openssl rand -hex 32` |
| `COLLECTOR_POLL_INTERVAL_SECONDS` | Collector poll interval (default 3, **must be far smaller than retention**) |
| `COLLECTOR_BATCH_SIZE` | Records popped per poll (default 200) |
| `HOT_RETENTION_DAYS` | Hot-detail retention in days (default 7, can be increased) |
| `ROLLUP_INTERVAL_SECONDS` | Rollup + cleanup interval (default 60) |
| `TIMEZONE` | Timezone — defines the boundaries of a "day" (default `Asia/Shanghai`) |

---

## 4. Deploy

Two ways — pick one.

### Option A — Pre-built images (recommended, no build)

Pull the published images from GHCR and run them directly. The server only needs `.env` and `docker-compose.prod.yml` (no source checkout):

```bash
# CUL_VERSION selects the release tag; omit it to use :latest
CUL_VERSION=v0.1.0 docker compose -f docker-compose.prod.yml up -d
```

Images are published to GHCR automatically on every `v*` release tag (see [the Release workflow](../.github/workflows/release.yml)).

### Option B — Build from source

```bash
docker compose up -d --build
```

Either way:

- Frontend: open `http://<server-ip>:8088`
- Backend: `:8080` (optional, debug only; you can drop the port mapping in production and let the frontend nginx reach it over the internal network)

---

## 5. Critical constraints & data-loss risks (must read)

1. **Globally single collector** — only **one** instance of this tool may run against a given CPA queue. The queue has pop (take-and-delete) semantics; multiple instances steal each other's data.
2. **Pop is not replayable** — requests produced while the collector is down for **longer than `redis-usage-queue-retention-seconds`** (default 60s, recommend 3600s) are **lost permanently**: CPA's queue is purely in-memory, never persisted, and cleared on expiry.
3. **Disk buffer (cloud-edition protection)** — batches already popped from the queue but not yet confirmed written to Supabase are first buffered to the `backend-buffer` volume; they're deleted only after a successful write and auto-recovered on collector restart — avoiding "popped but never stored".
4. **Never enable `SUBSCRIBE usage`** — as long as a subscriber is online, new records go only through pub/sub and never enter the FIFO queue, so the HTTP endpoint can't fetch them. This tool uses only `GET /usage-queue`; don't run another subscription-style consumer alongside it.
5. **Keep the collector highly available** — compose sets `restart: unless-stopped`, so it comes back up automatically after a host reboot.

---

## 6. Read-only instance / iterative validation (`COLLECTOR_ENABLED`)

The backend is a single process that by default runs everything at once: the background collector loop + rollup/cleanup scheduler + price refresh + query API. But the CPA queue is **pop-to-delete** and **only one collector may run globally** (constraint 1 above) — so you **cannot** simply spin up a second instance for validation (it would steal the queue and also trigger rollup/cleanup writes).

For this, the `COLLECTOR_ENABLED` toggle (in `.env`, default `true`) exists:

| Value | Behavior |
|-------|----------|
| `true` (default) | Normal instance: collect + rollup/cleanup + price refresh + query API (existing behavior, no change) |
| `false` | **Read-only instance**: price refresh + query API only. **Does not consume the CPA queue, write to the DB, or rollup/cleanup** — so it won't steal from the running collector |

Use case: during iteration/debugging (e.g. validating the frontend or a new query API), start a `COLLECTOR_ENABLED=false` instance on another machine or locally, pointed at the same Supabase, to query data read-only **without contending for the same CPA queue** as the production collector. Keep the production collector unique and `COLLECTOR_ENABLED=true`.

> Note: `COLLECTOR_ENABLED=false` merely stops queue consumption — it does **not** relax the hard "one collector per CPA queue" constraint; it exists precisely to let you validate without violating it. Price-table upserts are idempotent, so a read-only instance can still compute cost.

---

## 7. Capacity assumptions

- **Bounded detail** — size ≈ retention days × daily request volume (default 7 days); it has a ceiling and does **not** grow unbounded over time.
- **Tiny aggregates** — each `daily_account_usage` row is small and grows by account × model × day, slowly.
- At ~**1,000 requests/day**, 7 days of detail + long-term aggregates **fit comfortably within Supabase Free's 500 MB**.
- The dashboard's collector-health card shows **real table sizes** (detail / aggregate separately, as absolute values — no percentages, since plan quotas differ).

---

## 8. Cost estimation

- Uses the **LiteLLM price table** (`model_prices_and_context_window.json` from `BerriAI/litellm`).
- **Query-time calculation**: cost = tokens × current unit price; cost is never stored in the DB. Change a price and historical data automatically reflects it — no backfill.
- **Stores only used models**: fetched on startup + auto-refreshed daily + manually refreshable from the page.
- Models without a price show **"unknown"** and take effect automatically once a price is added.

---

## 9. Shutdown & rollback

- Collector interruption/restart: the disk buffer auto-recovers; but data lost during downtime exceeding retention can't be recovered (see risk 2).
- Queue backlog: under normal load, `count=200` + 3s polling is enough to keep up; if the backlog is severe, temporarily raise `COLLECTOR_BATCH_SIZE`.
- Deleting old detail never loses aggregates: cleanup always first confirms the day has been rolled up (the deletion window > the aggregate-recompute window).

---

<a id="chinese"></a>

> **简体中文** —— English version is [above](#english).

## 前置要求

- 一个 **Supabase 项目**（免费版即可，500MB 足够约 1000 请求/天）
- 一台能跑 **Docker + Docker Compose** 的服务器
- 一个 **CLIProxyAPI (CPA)** 实例，版本 **v6.10.8+**（推荐 v7.1.x），已启用 management

---

## 一、Supabase 建表

用 Supabase CLI（推荐，可追溯）：

```bash
supabase link --project-ref <你的项目ref> -p '<数据库密码>'
supabase db push --db-url "<Session pooler 连接串>"
```

或直接在 Supabase Dashboard 的 SQL Editor 里执行 `supabase/migrations/20260530185206_init_schema.sql`。

> 建好后有 4 张表：`request_events_hot`（热明细）/ `daily_account_usage`（日聚合）/ `model_prices`（价格）/ `collector_state`（采集器状态）。全部启用 RLS 且无策略——backend 用连接串直连，前端永不直连，Data API 默认拒绝。

---

## 二、CPA 必须配置（⚠️ 缺一则队列无数据）

在 CPA 的 `config.yaml` 中确认：

| 配置 | 值 | 说明 |
|------|-----|------|
| `usage-statistics-enabled` | `true` | **队列发布总闸**（注意：官方注释把它写成"内存聚合开关"是过期的，实际是队列开关，必须 true） |
| `redis-usage-queue-retention-seconds` | `3600` | 队列保留窗口，**建议设到最大值 3600**，给采集器短暂停机留缓冲 |
| `remote-management.secret-key` | 已设置 | Management key，队列启用的前提 |

---

## 三、配置 `.env`

复制 `.env.example` 为 `.env` 并填写：

| 变量 | 说明 |
|------|------|
| `CPA_BASE_URL` | CPA 地址，如 `https://your-cpa-host.com` |
| `CPA_MANAGEMENT_KEY` | CPA 的 management secret key |
| `DATABASE_URL` | Supabase **Session pooler** 连接串（含 `?sslmode=require`） |
| `DASHBOARD_PASSWORD` | 登录密码（明文注入，backend 用 bcrypt 校验，不落库） |
| `AUTH_TOKEN_SECRET` | token 签名密钥，用 `openssl rand -hex 32` 生成 |
| `COLLECTOR_POLL_INTERVAL_SECONDS` | 采集轮询间隔（默认 3，**必须远小于 retention**） |
| `COLLECTOR_BATCH_SIZE` | 每次 pop 条数（默认 200） |
| `HOT_RETENTION_DAYS` | 热明细保留天数（默认 7，可调大） |
| `ROLLUP_INTERVAL_SECONDS` | 聚合+清理间隔（默认 60） |
| `TIMEZONE` | 时区，"天"按它界定（默认 `Asia/Shanghai`） |

---

## 四、部署（两种方式，二选一）

### 方式 A —— 用预构建镜像（推荐，免构建）

从 GHCR 拉取已发布的镜像直接运行。服务器上只需要 `.env` 和 `docker-compose.prod.yml`（无需 clone 源码）：

```bash
# CUL_VERSION 指定发布版本号；省略则用 :latest
CUL_VERSION=v0.1.0 docker compose -f docker-compose.prod.yml up -d
```

镜像在每次 `v*` 发布 tag 时自动推送到 GHCR（见 [Release workflow](../.github/workflows/release.yml)）。

### 方式 B —— 从源码构建

```bash
docker compose up -d --build
```

两种方式部署后：

- 前端：浏览器访问 `http://<服务器IP>:8088`
- 后端：`:8080`（可选，仅调试用；生产可在 compose 去掉端口映射，仅由 frontend nginx 内网访问）

---

## 五、⚠️ 关键约束与丢数据风险（务必阅读）

1. **全局单采集器**：同一个 CPA 队列**只能跑一个**本工具实例。队列是 pop（取走即删）语义，多实例会互相抢走对方的数据。
2. **pop 不可回放**：采集器停机**超过 `redis-usage-queue-retention-seconds`**（默认 60s，建议 3600s）期间产生的请求，会**永久丢失**——CPA 队列纯内存、不落盘，过期即清。
3. **落盘缓冲（云版独有保护）**：已从队列 pop、但还没确认写入 Supabase 的批次，会先落盘到 `backend-buffer` volume；写库成功才删除，采集器重启时自动恢复——避免"取出来了但没存上"。
4. **绝不开 `SUBSCRIBE usage`**：只要有订阅者在线，新记录就只走 pub/sub、不进 FIFO 队列，HTTP 端点取不到。本工具只用 `GET /usage-queue`，不要同时跑别的订阅式消费者。
5. **保证采集器高可用**：compose 配了 `restart: unless-stopped`，宿主重启后自动拉起。

---

## 六、只读实例 / 迭代验证（`COLLECTOR_ENABLED`）

backend 是单进程，默认同时跑：后台采集循环 + rollup/清理调度 + 价格刷新 + 查询 API。但 CPA 队列是 **pop 即删** 且**全局只能跑一个采集器**（见上节约束 1）——所以**不能**简单地再起第二个实例来做验证（会抢队列、还会触发 rollup/清理写库）。

为此提供开关 `COLLECTOR_ENABLED`（`.env`，默认 `true`）：

| 值 | 行为 |
|------|------|
| `true`（默认） | 正常实例：采集 + rollup/清理 + 价格刷新 + 查询 API（现有行为，零变化） |
| `false` | **只读实例**：仅价格刷新 + 查询 API。**不消费 CPA 队列、不写库、不 rollup/清理**——因此不抢正在运行的采集器的数据 |

用途：迭代/调试时（如验证前端或新查询 API），在另一台机器或本地起一个 `COLLECTOR_ENABLED=false` 的实例，连同一个 Supabase 只读地查数据，**不会与生产采集器争抢同一个 CPA 队列**。生产采集实例始终保持唯一且 `COLLECTOR_ENABLED=true`。

> 注意：`COLLECTOR_ENABLED=false` 只是不再消费队列，并**不改变**「同一 CPA 队列全局单采集器」这条硬约束——它正是为了让你在不违反该约束的前提下做验证。价格表 upsert 是幂等的，只读实例照样能算成本。

---

## 七、容量假设

- **明细有界**：体积 ≈ 保留天数 × 日请求量（默认 7 天），**有上限、不随时间无限增长**。
- **聚合极小**：`daily_account_usage` 每行很小，按 账号×模型×天 增长，缓慢。
- 约 **1000 请求/天**下，7 天明细 + 长期聚合可**舒适待在 Supabase Free 500MB 内**。
- Dashboard 采集器健康卡显示**真实表大小**（明细/聚合分别，绝对值，不显示百分比——因为容量套餐分母不一）。

---

## 八、成本估算

- 用 **LiteLLM 价格表**（`BerriAI/litellm` 的 `model_prices_and_context_window.json`）。
- **query-time 计算**：成本 = token × 当前单价，不在库里存死 cost；改了价格历史数据自动按新价显示，无需回填。
- **只存用过的模型**：启动拉取 + 每日自动刷新 + 页面可手动刷新。
- 缺价模型显示**"未知"**，补价后自动生效。

---

## 九、停机与回滚

- 采集器中断重启：自动恢复落盘缓冲；但停机超 retention 的数据无法补回（见风险 2）。
- 队列堆积：正常负载下 `count=200` + 3s 轮询足以追上；积压严重时可临时调大 `COLLECTOR_BATCH_SIZE`。
- 删旧明细不丢聚合：清理前必先确认该日已 rollup（删除窗口 > 聚合重算窗口）。
