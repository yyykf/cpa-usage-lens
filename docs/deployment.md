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

Pull the published images from GHCR and run them directly. Use the latest release tag from [GitHub Releases](https://github.com/yyykf/cpa-usage-lens/releases); pinning a tag makes rollback predictable.

#### No source checkout

The normal production run only needs `.env` and `docker-compose.prod.yml`; the debug override is downloaded as a convenience for later troubleshooting. The Compose templates are fetched from `main`, while the images are pinned by `CUL_VERSION` to the latest release tag.

```bash
mkdir -p cpa-usage-lens
cd cpa-usage-lens

# Replace this with the latest release tag, for example v0.1.1 or newer.
export CUL_VERSION=<latest-release-tag>

curl -fsSLO "https://raw.githubusercontent.com/yyykf/cpa-usage-lens/main/docker-compose.prod.yml"
curl -fsSLO "https://raw.githubusercontent.com/yyykf/cpa-usage-lens/main/docker-compose.debug.yml"
curl -fsSLO "https://raw.githubusercontent.com/yyykf/cpa-usage-lens/main/.env.example"
cp .env.example .env

# Fill in CPA_BASE_URL, CPA_MANAGEMENT_KEY, DATABASE_URL, DASHBOARD_PASSWORD, AUTH_TOKEN_SECRET.
nano .env

docker compose -f docker-compose.prod.yml up -d
```

#### Existing source checkout

```bash
export CUL_VERSION=<latest-release-tag>
docker compose -f docker-compose.prod.yml up -d
```

Images are published to GHCR automatically on every `v*` release tag (see [the Release workflow](../.github/workflows/release.yml)).

### Option B — Build from source

```bash
docker compose up -d --build
```

Either way:

- Frontend: open `http://<server-ip>:8088`
- Backend: not exposed by default. This is intentional: the frontend nginx reaches `backend:8080` over the internal Compose network, so normal users only need to expose the dashboard port.

If you intentionally need direct backend access for debugging:

```bash
docker compose -f docker-compose.prod.yml -f docker-compose.debug.yml up -d
curl http://<server-ip>:8080/healthz
```

---

## 5. Verify deployment

Run these checks after starting the stack:

```bash
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs --tail=100 backend
docker compose -f docker-compose.prod.yml exec -T backend wget -qO- http://127.0.0.1:8080/healthz
```

Expected results:

- `backend` and `frontend` are both `running` or `up`.
- `/healthz` prints `ok`.
- `http://<server-ip>:8088` shows the login page.
- After login, the collector card should show recent polling. If CPA has live traffic, `events_ingested` should increase.

---

## 6. Troubleshooting

| Symptom | What to check | Why |
|---------|---------------|-----|
| Login page does not open | `docker compose ps`, server firewall, cloud security group, and whether port `8088` is published | Only the frontend port is exposed in production |
| Backend health fails | `docker compose logs backend`, `DATABASE_URL`, and whether the Supabase migration was run | The backend exits if required config or DB access is invalid |
| Login password does not work | Update `DASHBOARD_PASSWORD` in `.env`, then restart with `docker compose -f docker-compose.prod.yml up -d` | The password is read from environment at backend startup |
| Collector shows no new events | CPA `usage-statistics-enabled`, `remote-management.secret-key`, `redis-usage-queue-retention-seconds`, and backend logs | CPA only publishes queue events when the management queue is enabled |
| Data is split or missing | Make sure only one instance has `COLLECTOR_ENABLED=true` for this CPA queue | The CPA usage queue is destructive pop-on-read |
| Cost shows unknown | Click refresh prices and check outbound access to GitHub raw content | Prices are fetched from the LiteLLM price table |

---

## 7. Critical constraints & data-loss risks (must read)

1. **Globally single collector** — only **one** instance of this tool may run against a given CPA queue. The queue has pop (take-and-delete) semantics; multiple instances steal each other's data.
2. **Pop is not replayable** — requests produced while the collector is down for **longer than `redis-usage-queue-retention-seconds`** (default 60s, recommend 3600s) are **lost permanently**: CPA's queue is purely in-memory, never persisted, and cleared on expiry.
3. **Disk buffer (cloud-edition protection)** — batches already popped from the queue but not yet confirmed written to Supabase are first buffered to the `backend-buffer` volume; they're deleted only after a successful write and auto-recovered on collector restart — avoiding "popped but never stored".
4. **Never enable `SUBSCRIBE usage`** — as long as a subscriber is online, new records go only through pub/sub and never enter the FIFO queue, so the HTTP endpoint can't fetch them. This tool uses only `GET /usage-queue`; don't run another subscription-style consumer alongside it.
5. **Keep the collector highly available** — compose sets `restart: unless-stopped`, so it comes back up automatically after a host reboot.

---

## 8. Read-only instance / iterative validation (`COLLECTOR_ENABLED`)

The backend is a single process that by default runs everything at once: the background collector loop + rollup/cleanup scheduler + price refresh + query API. But the CPA queue is **pop-to-delete** and **only one collector may run globally** (constraint 1 above) — so you **cannot** simply spin up a second instance for validation (it would steal the queue and also trigger rollup/cleanup writes).

For this, the `COLLECTOR_ENABLED` toggle (in `.env`, default `true`) exists:

| Value | Behavior |
|-------|----------|
| `true` (default) | Normal instance: collect + rollup/cleanup + price refresh + query API (existing behavior, no change) |
| `false` | **Read-only instance**: price refresh + query API only. **Does not consume the CPA queue, write to the DB, or rollup/cleanup** — so it won't steal from the running collector |

Use case: during iteration/debugging (e.g. validating the frontend or a new query API), start a `COLLECTOR_ENABLED=false` instance on another machine or locally, pointed at the same Supabase, to query data read-only **without contending for the same CPA queue** as the production collector. Keep the production collector unique and `COLLECTOR_ENABLED=true`.

> Note: `COLLECTOR_ENABLED=false` merely stops queue consumption — it does **not** relax the hard "one collector per CPA queue" constraint; it exists precisely to let you validate without violating it. Price-table upserts are idempotent, so a read-only instance can still compute cost.

---

## 9. Capacity assumptions

- **Bounded detail** — size ≈ retention days × daily request volume (default 7 days); it has a ceiling and does **not** grow unbounded over time.
- **Tiny aggregates** — each `daily_account_usage` row is small and grows by account × model × day, slowly.
- At ~**1,000 requests/day**, 7 days of detail + long-term aggregates **fit comfortably within Supabase Free's 500 MB**.
- The dashboard's collector-health card shows **real table sizes** (detail / aggregate separately, as absolute values — no percentages, since plan quotas differ).

---

## 10. Cost estimation

- Uses the **LiteLLM price table** (`model_prices_and_context_window.json` from `BerriAI/litellm`).
- **Query-time calculation**: cost = tokens × current unit price; cost is never stored in the DB. Change a price and historical data automatically reflects it — no backfill.
- **Stores only used models**: fetched on startup + auto-refreshed daily + manually refreshable from the page.
- Models without a price show **"unknown"** and take effect automatically once a price is added.

---

## 11. Shutdown & rollback

- Collector interruption/restart: the disk buffer auto-recovers; but data lost during downtime exceeding retention can't be recovered (see risk 2).
- Queue backlog: under normal load, `count=200` + 3s polling is enough to keep up; if the backlog is severe, temporarily raise `COLLECTOR_BATCH_SIZE`.
- Deleting old detail never loses aggregates: cleanup always first confirms the day has been rolled up (the deletion window > the aggregate-recompute window).

---

## 12. Breaking migration: `daily_account_usage` primary key 3 → 4 columns (no zero-downtime)

> Applies when upgrading an existing deployment across the migration `20260605002633_add_api_key_dimension.sql` (the "API key dimension"). **A fresh install is unaffected** — just run all migrations before the first start.

**Why it is breaking.** This migration widens the `daily_account_usage` primary key from `(usage_date, source, model)` to `(usage_date, source, model, key_fingerprint)`. The old collector binary's rollup uses `INSERT ... ON CONFLICT (usage_date, source, model) DO UPDATE`. Postgres requires the `ON CONFLICT` inference columns to exactly match an existing unique constraint; once the primary key becomes 4 columns, the old 3-column `ON CONFLICT` no longer matches any constraint and **every rollup fails** (`there is no unique or exclusion constraint matching the ON CONFLICT specification`). So a running old binary against the migrated schema errors continuously — **rolling/zero-downtime upgrade is not supported here**.

**Required upgrade order (do not reorder).**

1. **Stop the collector** — set `COLLECTOR_ENABLED=false` on the production instance (or stop it). This halts queue consumption and rollup so nothing writes with the stale 3-column `ON CONFLICT`.
   ```bash
   # in the production .env
   COLLECTOR_ENABLED=false
   docker compose -f docker-compose.prod.yml up -d
   ```
   > Brief data-loss caveat: while the collector is down longer than CPA's `redis-usage-queue-retention-seconds`, queued requests expire and are lost (the usual pop-on-read risk, see risk 2). Keep this window short.
2. **Run the migration** on the production database (CLI `supabase db push`, or paste `20260605002633_add_api_key_dimension.sql` into the SQL Editor — it is fully idempotent and safe to re-run).
3. **Deploy the new code** — bump `CUL_VERSION` to the release that contains the 4-column rollup and `up -d`. The new binary's `ON CONFLICT (usage_date, source, model, key_fingerprint)` matches the new primary key.
4. **(Optional) Backfill** historical rows from the sentinel `key_fingerprint='none'` bucket into the designated current key, per the backfill plan (plaintext is fed in only at run time; only the fingerprint + mask are written — never the plaintext).
5. **Re-enable the collector** — set `COLLECTOR_ENABLED=true` and `up -d` to resume collection on the new schema + code.

**Do not** migrate the database while the old binary is still collecting: the moment the primary key flips to 4 columns, the still-running old rollup starts failing on the 3-column `ON CONFLICT`.

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

从 GHCR 拉取已发布的镜像直接运行。请使用 [GitHub Releases](https://github.com/yyykf/cpa-usage-lens/releases) 里的最新发布 tag；固定 tag 部署比直接用浮动 `latest` 更便于回滚。

#### 不 clone 源码部署

生产正常运行只需要 `.env` 和 `docker-compose.prod.yml`；这里顺手下载调试 override，方便后续排障时使用。Compose 模板从 `main` 获取，实际运行的镜像版本仍由 `CUL_VERSION` 固定到最新发布 tag。

```bash
mkdir -p cpa-usage-lens
cd cpa-usage-lens

# 换成当前最新发布 tag，例如 v0.1.1 或更新版本。
export CUL_VERSION=<latest-release-tag>

curl -fsSLO "https://raw.githubusercontent.com/yyykf/cpa-usage-lens/main/docker-compose.prod.yml"
curl -fsSLO "https://raw.githubusercontent.com/yyykf/cpa-usage-lens/main/docker-compose.debug.yml"
curl -fsSLO "https://raw.githubusercontent.com/yyykf/cpa-usage-lens/main/.env.example"
cp .env.example .env

# 填写 CPA_BASE_URL、CPA_MANAGEMENT_KEY、DATABASE_URL、DASHBOARD_PASSWORD、AUTH_TOKEN_SECRET。
nano .env

docker compose -f docker-compose.prod.yml up -d
```

#### 已经 clone 源码

```bash
export CUL_VERSION=<latest-release-tag>
docker compose -f docker-compose.prod.yml up -d
```

镜像在每次 `v*` 发布 tag 时自动推送到 GHCR（见 [Release workflow](../.github/workflows/release.yml)）。

### 方式 B —— 从源码构建

```bash
docker compose up -d --build
```

两种方式部署后：

- 前端：浏览器访问 `http://<服务器IP>:8088`
- 后端：默认不暴露到宿主机。这是刻意的生产默认值：frontend nginx 会在 Compose 内网访问 `backend:8080`，普通使用者只需要暴露 dashboard 端口。

如确实需要直连 backend 做调试：

```bash
docker compose -f docker-compose.prod.yml -f docker-compose.debug.yml up -d
curl http://<服务器IP>:8080/healthz
```

---

## 五、部署后检查

启动后先跑这几条：

```bash
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs --tail=100 backend
docker compose -f docker-compose.prod.yml exec -T backend wget -qO- http://127.0.0.1:8080/healthz
```

正常结果：

- `backend` 和 `frontend` 都是 `running` / `up`。
- `/healthz` 输出 `ok`。
- 浏览器打开 `http://<服务器IP>:8088` 能看到登录页。
- 登录后，采集器卡片应显示最近轮询；如果 CPA 有真实流量，`events_ingested` 应该增长。

---

## 六、常见问题排查

| 现象 | 检查项 | 原因 |
|------|--------|------|
| 登录页打不开 | `docker compose ps`、服务器防火墙、云厂商安全组、`8088` 是否暴露 | 生产默认只暴露 frontend 端口 |
| backend health 失败 | `docker compose logs backend`、`DATABASE_URL`、Supabase migration 是否执行 | 必填配置或数据库连接错误会导致 backend 退出 |
| 登录密码不生效 | 修改 `.env` 的 `DASHBOARD_PASSWORD` 后执行 `docker compose -f docker-compose.prod.yml up -d` 重启 | 登录密码在 backend 启动时读取 |
| 采集器没有新数据 | CPA 的 `usage-statistics-enabled`、`remote-management.secret-key`、`redis-usage-queue-retention-seconds`、backend 日志 | CPA 只有启用 management queue 后才会发布用量事件 |
| 数据缺失或被拆散 | 确保同一个 CPA 队列只有一个实例 `COLLECTOR_ENABLED=true` | CPA usage queue 是 pop 即删，多采集器会互相抢数据 |
| 成本显示未知 | 点击刷新价格表，并确认容器能访问 GitHub raw 内容 | 价格来自 LiteLLM price table |

---

## 七、⚠️ 关键约束与丢数据风险（务必阅读）

1. **全局单采集器**：同一个 CPA 队列**只能跑一个**本工具实例。队列是 pop（取走即删）语义，多实例会互相抢走对方的数据。
2. **pop 不可回放**：采集器停机**超过 `redis-usage-queue-retention-seconds`**（默认 60s，建议 3600s）期间产生的请求，会**永久丢失**——CPA 队列纯内存、不落盘，过期即清。
3. **落盘缓冲（云版独有保护）**：已从队列 pop、但还没确认写入 Supabase 的批次，会先落盘到 `backend-buffer` volume；写库成功才删除，采集器重启时自动恢复——避免"取出来了但没存上"。
4. **绝不开 `SUBSCRIBE usage`**：只要有订阅者在线，新记录就只走 pub/sub、不进 FIFO 队列，HTTP 端点取不到。本工具只用 `GET /usage-queue`，不要同时跑别的订阅式消费者。
5. **保证采集器高可用**：compose 配了 `restart: unless-stopped`，宿主重启后自动拉起。

---

## 八、只读实例 / 迭代验证（`COLLECTOR_ENABLED`）

backend 是单进程，默认同时跑：后台采集循环 + rollup/清理调度 + 价格刷新 + 查询 API。但 CPA 队列是 **pop 即删** 且**全局只能跑一个采集器**（见上节约束 1）——所以**不能**简单地再起第二个实例来做验证（会抢队列、还会触发 rollup/清理写库）。

为此提供开关 `COLLECTOR_ENABLED`（`.env`，默认 `true`）：

| 值 | 行为 |
|------|------|
| `true`（默认） | 正常实例：采集 + rollup/清理 + 价格刷新 + 查询 API（现有行为，零变化） |
| `false` | **只读实例**：仅价格刷新 + 查询 API。**不消费 CPA 队列、不写库、不 rollup/清理**——因此不抢正在运行的采集器的数据 |

用途：迭代/调试时（如验证前端或新查询 API），在另一台机器或本地起一个 `COLLECTOR_ENABLED=false` 的实例，连同一个 Supabase 只读地查数据，**不会与生产采集器争抢同一个 CPA 队列**。生产采集实例始终保持唯一且 `COLLECTOR_ENABLED=true`。

> 注意：`COLLECTOR_ENABLED=false` 只是不再消费队列，并**不改变**「同一 CPA 队列全局单采集器」这条硬约束——它正是为了让你在不违反该约束的前提下做验证。价格表 upsert 是幂等的，只读实例照样能算成本。

---

## 九、容量假设

- **明细有界**：体积 ≈ 保留天数 × 日请求量（默认 7 天），**有上限、不随时间无限增长**。
- **聚合极小**：`daily_account_usage` 每行很小，按 账号×模型×天 增长，缓慢。
- 约 **1000 请求/天**下，7 天明细 + 长期聚合可**舒适待在 Supabase Free 500MB 内**。
- Dashboard 采集器健康卡显示**真实表大小**（明细/聚合分别，绝对值，不显示百分比——因为容量套餐分母不一）。

---

## 十、成本估算

- 用 **LiteLLM 价格表**（`BerriAI/litellm` 的 `model_prices_and_context_window.json`）。
- **query-time 计算**：成本 = token × 当前单价，不在库里存死 cost；改了价格历史数据自动按新价显示，无需回填。
- **只存用过的模型**：启动拉取 + 每日自动刷新 + 页面可手动刷新。
- 缺价模型显示**"未知"**，补价后自动生效。

---

## 十一、停机与回滚

- 采集器中断重启：自动恢复落盘缓冲；但停机超 retention 的数据无法补回（见风险 2）。
- 队列堆积：正常负载下 `count=200` + 3s 轮询足以追上；积压严重时可临时调大 `COLLECTOR_BATCH_SIZE`。
- 删旧明细不丢聚合：清理前必先确认该日已 rollup（删除窗口 > 聚合重算窗口）。

---

## 十二、破坏性迁移：`daily_account_usage` 主键 3 → 4 列（不支持零停机）

> 仅在「升级既有部署、跨过迁移 `20260605002633_add_api_key_dimension.sql`（API key 维度）」时适用。**全新部署不受影响**——首次启动前把所有迁移跑完即可。

**为什么是破坏性的。** 这条迁移把 `daily_account_usage` 主键从 `(usage_date, source, model)` 扩为 `(usage_date, source, model, key_fingerprint)`。旧采集器 binary 的 rollup 用的是 `INSERT ... ON CONFLICT (usage_date, source, model) DO UPDATE`。Postgres 要求 `ON CONFLICT` 的推断列必须与某个现有唯一约束**完全匹配**；主键一旦变成 4 列，旧的 3 列 `ON CONFLICT` 就匹配不到任何约束，**每次 rollup 都会失败**（报 `there is no unique or exclusion constraint matching the ON CONFLICT specification`）。所以「旧 binary 跑在已迁移的 schema 上」会持续报错——**这里不支持滚动/零停机升级**。

**必须按序执行（不要调换顺序）。**

1. **停采集器**——把生产实例的 `COLLECTOR_ENABLED` 置为 `false`（或直接停掉）。这会停掉队列消费与 rollup，确保没有任何写入再走旧的 3 列 `ON CONFLICT`。
   ```bash
   # 生产 .env 里
   COLLECTOR_ENABLED=false
   docker compose -f docker-compose.prod.yml up -d
   ```
   > 短暂丢数据提醒：采集器停机超过 CPA 的 `redis-usage-queue-retention-seconds` 期间，队列里的请求会过期丢失（即 pop 即删的固有风险，见风险 2）。务必把这个窗口压短。
2. **跑迁移**——在生产库执行（CLI `supabase db push`，或把 `20260605002633_add_api_key_dimension.sql` 贴进 SQL Editor——它全幂等，可安全重跑）。
3. **部署新代码**——把 `CUL_VERSION` 升到包含 4 列 rollup 的发布版并 `up -d`。新 binary 的 `ON CONFLICT (usage_date, source, model, key_fingerprint)` 与新主键匹配。
4. **（可选）回填**——按回填方案把存量哨兵桶 `key_fingerprint='none'` 的行归并到指定的当前在用 key（明文仅执行时本地喂入，只写指纹+掩码，**绝不写明文**）。
5. **重启采集器**——把 `COLLECTOR_ENABLED` 置回 `true` 并 `up -d`，在新 schema + 新代码上恢复采集。

**切勿**在旧 binary 还在采集时就迁移数据库：主键一旦切到 4 列，仍在运行的旧 rollup 立刻会因 3 列 `ON CONFLICT` 失配而报错。
