# 部署与运维说明

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

## 四、一键部署

```bash
docker compose up -d --build
```

- 前端：浏览器访问 `http://<服务器IP>:8088`
- 后端：`:8080`（可选，仅调试用；生产可在 compose 去掉端口映射）

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
