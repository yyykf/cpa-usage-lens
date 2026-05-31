# Research: 竞品参考实现（如何消费 usage-queue / 去重 / 断点 / 成本估算）

- **Query**: 重点研究 zhanglunet/cliproxyapi-usage-dashboard（消费方式/去重/中断处理/字段映射/表结构/成本），并快速看 CPA-Manager、Willxup/cpa-usage-keeper 的成本估算与价格同步；提炼对"Supabase 云版"有借鉴价值的设计点和已知坑。
- **Scope**: external（GitHub 源码 + README）
- **Date**: 2026-05-30

---

## 一句话结论

zhanglunet 给出了**最干净的最小可行参考**（纯标准库、RESP + RPOP、`request_id` 去重、UNIQUE 约束幂等插入），其代码可以几乎直译成我们采集器的核心逻辑；但它**不算成本、无断点续采、单机本地**。成本估算与 LiteLLM 价格同步要看 CPA-Manager 和 cpa-usage-keeper（都用 `BerriAI/litellm/.../model_prices_and_context_window.json`，且都只对"用到的模型"持久化价格）。**所有实现共有的两个硬约束必须照搬进我们的云版设计：① 同一队列全局只能一个采集器；② pop 语义不可回放、停机超 retention 即丢数据。**

---

## A. zhanglunet/cliproxyapi-usage-dashboard（主参考，已读全部源码）

源码：单文件 `usage_dashboard.py`（703 行，纯 Python 标准库，无第三方依赖）。是官方 README 推荐工具之一，也是被 issue #3230 用户实际在用的。

### A.1 如何消费 usage-queue —— RESP + RPOP，不是 HTTP

- **走 Redis RESP 裸 TCP**（不是 HTTP 端点），自己手写了 RESP 协议编解码（`RespClient` 类，`resp_command()` 拼 `*N\r\n$len\r\n...`）。连上后先 `AUTH <management_key>`。
- **拉取命令**：`RPOP queue 100` —— 一次最多取 100 条（`rpop(count=100)`）。
- **轮询节奏**：`collect_forever()` 里一个内层 `while True`，不停 `RPOP 100`；**每轮结束 `time.sleep(poll_interval_seconds)`，默认 2 秒**（`DEFAULT_CONFIG["poll_interval_seconds"]=2`）。注意它不是"空了才 sleep"，而是每轮固定 sleep。
- **端口**：`cliproxy_port=8317`（HTTP/RESP 共用端口）。
- 证据（关键代码，`collect_forever` 第 343-366 行）：
  ```python
  client = RespClient(host, port, management_key)   # AUTH
  while True:
      raw_items = client.rpop(100)
      if raw_items:
          inserted = insert_usage(raw_items)
      if now - last_quota >= quota_refresh_seconds:   # 顺带刷配额
          refresh_quota(force=True)
      time.sleep(poll_interval_seconds)               # 默认 2s
  ```

### A.2 去重 —— request_id 优先，否则 sha256(raw)；靠 UNIQUE 约束 + 幂等插入

- `event_key(payload, raw)`：`request_id` 存在就用它；否则 `hashlib.sha256(raw).hexdigest()`（对整条原始 JSON 取哈希）。
- 表上 `event_key TEXT NOT NULL UNIQUE`；插入时 `INSERT ...`，捕获 `sqlite3.IntegrityError: pass` —— **重复就静默跳过**。
- 证据（第 201-205、244-261 行）：
  ```python
  def event_key(payload, raw):
      rid = payload.get("request_id")
      return rid if rid else hashlib.sha256(raw.encode()).hexdigest()
  # ...
  try: conn.execute("INSERT INTO usage_events (...) VALUES (...)", values); inserted += 1
  except sqlite3.IntegrityError: pass
  ```
- **设计含义**：pop 语义本不该重复，但加 UNIQUE 是廉价保险（防止采集器重试/双跑导致重复）。对我们 Supabase 版同样适用——用 `request_id` 作业务主键 + upsert/on-conflict-do-nothing。

### A.3 中断/重启处理 —— 无断点续采，完全依赖 CPA 侧 retention

- **没有任何 checkpoint / offset / 断点续采**。队列是 pop 语义，取走即删，本就没有"位置"可记。
- 重启就是从头跑 `collect_forever`，重连后继续 `RPOP`。
- 异常处理：最外层 `try/except Exception` 捕获后 `time.sleep(5)` 重连（第 364-366 行）。粗暴但够用。
- **明确的已知数据丢失**（README「限制」节原文）：
  - 「只能统计采集器启动之后的请求，历史数据无法补回。」
  - 「CLIProxyAPI 的队列是短期队列，采集器长时间停止会丢失期间事件。」
- **缓解手段**：README 要求用户把 CPA 的 `redis-usage-queue-retention-seconds` 调到 `3600`（最大值），给采集器短暂停机留缓冲：「`redis-usage-queue-retention-seconds` 用于延长队列保留时间，避免采集器短暂停止时丢失事件。」
- 队列堆积应对：靠 `RPOP 100` 批量 + 2s 高频轮询，正常负载下追得上。

### A.4 字段映射 & 表结构（直接可借鉴的 schema）

`usage_events` 表（第 64-95 行），从队列 payload 1:1 映射：

| 队列字段 | 表列 | 备注 |
|---|---|---|
| `request_id` | `request_id` + `event_key`(UNIQUE) | 去重键 |
| `timestamp`(RFC3339) | `timestamp`/`ts_epoch`/`local_date`/`local_hour` | 同时存 UTC、epoch、本地日期、本地小时（预聚合用） |
| `source` | `source` | 账号标识（邮箱等） |
| `auth_index` | `auth_index` | 账号池序号 |
| `provider`/`model`/`endpoint`/`auth_type` | 同名列 | |
| `api_key` | `api_key_hash` | **不存明文**，`sha256(api_key)[:12]` |
| `failed` | `failed`(INTEGER 0/1) | |
| `latency_ms` | `latency_ms` | |
| `tokens.{input,output,reasoning,cached,total}_tokens` | 同名 5 列 | |
| 整条原文 | `raw_json TEXT NOT NULL` | **保留原始 JSON**，schema 演进/补救用 |

索引：`ts_epoch`、`local_date`、`source`、`auth_index`。
时区：硬编码 `Asia/Shanghai`（`LOCAL_TZ`），按本地时区切 today/1h/5h/24h/7d 窗口。
聚合方式：**不预聚合**，查询时现算（`query_summary` 里 `GROUP BY account/model/local_hour` + `SUM`）。

另有 `quota_snapshots` 表：定期查 ChatGPT `backend-api/wham/usage` 拿 Codex 5h/7d 余量（这块对我们价值不大，是 Codex 专属配额，非通用成本统计）。

### A.5 成本 —— **zhanglunet 完全不算成本！**

- 通篇没有任何 price/cost/litellm 逻辑，只统计 tokens 和请求数。
- → 成本估算的参考必须看下面的 CPA-Manager / cpa-usage-keeper。

---

## B. seakee/CPA-Manager 与 CPA-Manager-Plus（成本估算 + 价格同步的最佳参考）

CPA-Manager（Go + 单文件 React 面板 + 可选 Docker "Usage Service"）。CPA-Manager-Plus 是其推荐后继，能力更全。**这是功能最接近我们目标、且最值得抄设计的参考。**

### B.1 消费方式 —— 多模式 auto，HTTP 优先（与 zhanglunet 相反，更适合服务器/反代）

`USAGE_COLLECTOR_MODE`（默认 `auto`）：
- **CPA-Manager**：`auto` = HTTP 队列优先，失败回退 RESP pop。（`http` 强制 HTTP / `resp` 强制 RESP）
- **CPA-Manager-Plus**：`auto` = **RESP Pub/Sub(`subscribe`) → HTTP 队列 → RESP pop** 三级回退。
- 相关参数：`USAGE_BATCH_SIZE=100`（每次 pop 最多条数）、`USAGE_POLL_INTERVAL_MS=500`（空闲轮询间隔，比 zhanglunet 的 2s 更激进）、`USAGE_RESP_POP_SIDE=right`（RPOP）、`USAGE_RESP_QUEUE=usage`（CPA 当前忽略此 arg）。
- 架构（自带一个 `:18317` 服务，前端面板 + 采集器 + SQLite 一体，其余 `/v0/management/*` 反代给 CPA）：
  ```
  Browser -> Usage Service :18317
      -> /v0/management/usage + /model-prices 从 SQLite
      -> 其余 /v0/management/* 反代给 CPA
      -> HTTP/RESP/PubSub consumer -> CPA API port -> SQLite /data/usage.sqlite
  ```

### B.2 成本估算 —— 可编辑模型价格 + 一键 LiteLLM 同步

- 端点：`GET/PUT /v0/management/model-prices`（读/改 SQLite 里的价格表），`POST /v0/management/model-prices/sync`（从 LiteLLM 同步）。
- **LiteLLM 价格源（关键，可直接复用）**：源码常量
  ```go
  const modelPriceSyncSource = "litellm"
  var modelPriceSyncURL = "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"
  ```
- CPA-Manager-Plus 进一步支持**多价格源**（LiteLLM、OpenRouter 等，带 source metadata）。
- 面板"Request Monitoring"展示：persisted usage KPIs、model/channel/account/API-key 维度、requested vs resolved model、**estimated token cost**、失败分析、实时表。
- 价格持久化策略：见 cpa-usage-keeper（只对用到的模型存价格），CPA-Manager 同理（价格存 SQLite，用户可编辑覆盖同步值）。

### B.3 单采集器 / 轮询约束 —— 明确写进了文档（直接抄进我们的约束）

CPA-Manager / Plus README 原文（强约束）：
- 「**Exactly one Usage Service should consume the same CPA usage queue.**」——同一队列只能一个消费者。
- 「CPA keeps queue items in memory for `redis-usage-queue-retention-seconds`, default 60s, max 3600s. **Keep Usage Service running continuously.**」
- CPA-Manager-Plus 额外加了一条工程保护：「**`pollIntervalMs` must be ≤ the CPA queue retention window**. Saves are rejected when the collector would poll too slowly and risk expired queue items.」——**轮询间隔慢于 retention 就直接拒绝保存配置**。这个校验值得我们抄。
- CPA-Manager-Plus 还点出一个微妙行为：停掉采集器只停消费、**不会关 CPA 的 usage publishing**；只要 publishing 还开着，在 retention 窗口内重启采集器还能补回这段时间的事件（相当于 retention 就是唯一的"缓冲区"）。

### B.4 版本/反代经验

- 「CPA `v6.10.8+` 才有 HTTP 队列端点 `/v0/management/usage-queue`，能穿透普通 HTTP 反代；RESP 走 8317 裸 TCP，**过不了 HTTP 反代**。」CPA-Manager-Plus 推荐 CPA `v7.1.0+`。
- 经典报错 `unsupported RESP prefix 'H'`：升级到 v6.10.8+ 并用 `auto`(HTTP 优先) 即可——这是 RESP 客户端连到 HTTP 端点上的症状。

### B.5 历史数据导入（迁移/恢复路径，非续采）

- `GET /v0/management/usage/export`（导出 JSONL/NDJSON）、`POST /v0/management/usage/import`（导入 JSONL 或旧版 CPA `/usage/export` 的 legacy JSON 快照）。
- 重要坑（PR #36 文档原文）：legacy JSON 只有在 `usage.apis.*.models.*.details[]` 有 request 明细时才能转换；只有聚合总量的文件会被拒（无法重建 request 级数据）。且 legacy 文件常缺 `api_key_hash`/channel/request_id/latency/cache tokens/failure reason 等元数据，导入后账号匹配和明细精度会下降。→ **启示：我们若做导入功能，要把它定位成"一次性迁移/恢复"，不是无缝续采。**

---

## C. Willxup/cpa-usage-keeper（Go，523★，维护者亲自背书）

独立 Go 服务（GORM + SQLite + React 前端，Docker/Compose/systemd）。issue #3188 里维护者 @luispater 亲自推荐，称"CPA团队前期和该项目沟通过改进"。

### C.1 消费方式 —— 三模式，默认 auto，**默认走 Redis 队列**

- `USAGE_SYNC_MODE`：`auto`(默认) / `redis` / `legacy_export`。
- `auto` 模式实际以 Redis 队列消费为主，`legacy_export`（轮询 `usage/export`）作为 fallback 节流。
- Redis 队列参数：
  - `REDIS_QUEUE_ADDR`：默认 = `CPA_BASE_URL` 主机名 + `8317`；非默认端口才填 `host:port`。
  - `REDIS_QUEUE_BATCH_SIZE`：**默认 10000**（比 zhanglunet 的 100、CPA-Manager 的 100 大得多——一次尽量抽干）。
  - `REDIS_QUEUE_IDLE_INTERVAL`：默认 `1s`（队列空时的检查间隔）。
  - `REDIS_QUEUE_TLS` / `TLS_SKIP_VERIFY`：CPA 开 HTTPS 时 RESP 也走 TLS。
  - `POLL_INTERVAL`：`30s`（legacy_export 模式 `5m`），auto 模式下也用作 fallback 节流。

### C.2 成本估算 —— "仅对已使用模型"持久化价格

- README Features 原文：「**Configurable pricing persistence for used models only**」/「仅允许对已使用模型进行价格持久化配置」、「Maintain model prices for cost estimation and reporting」。
- Dashboard 展示：request volume、tokens、**cost**、cache hit rate、success rate、latency。
- 代码结构里 `internal/service/` 有专门的 pricing service，`internal/repository/` 负责聚合。
- 价格同步具体源未在 README 明写，但与 CPA-Manager 同生态，大概率同样基于 LiteLLM 表（待确认，见 Caveats）。

### C.3 原始数据备份 —— 落库的同时留原始备份（我们可借鉴的"安全网"）

- `BACKUP_ENABLED=true`（默认）：把从 CPA 拉到的**原始数据**周期性备份到本地磁盘；`BACKUP_INTERVAL`（默认 1h/24h 两个版本不同）、`BACKUP_RETENTION_DAYS`。
- 「Every sync still records a snapshot run and persists usage events」——每次同步都记一次 snapshot run。
- → 对"磁盘紧张"的目标用户这点要权衡：原始备份会占盘。我们云版可考虑只在 Supabase 存结构化数据 + 保留 `raw_json` 列，不再额外落本地备份文件。

### C.4 部署形态

Docker Compose 把 CPA + keeper 一起拉起；keeper 只需 `CPA_BASE_URL` + `CPA_MANAGEMENT_KEY`（+ 公网部署加 `AUTH_ENABLED`/`LOGIN_PASSWORD`）。`/data` 挂 SQLite + 备份 + 日志。

---

## D. ssfun/CLIProxyAPI-Pro（架构参考，非纯采集器）

定制 CPA 后端镜像，**把 SQLite usage service 内嵌进 CPA 自身**（不是外部采集器），暴露 `/v0/management/usage` + `/usage/status` + `/usage/events`(增量轮询) + `/usage/stream`(SSE)。环境变量 `USAGE_BATCH_SIZE`/`USAGE_POLL_INTERVAL_MS`/`USAGE_QUERY_LIMIT`。支持 NDJSON 导入导出、模型价格持久化、WebDAV 备份、SQLite quota cache。
- 与我们路线不同（它改 CPA 本体、强制 `usage-statistics-enabled=true`），但**它的 `/usage/events` 增量轮询 + `/usage/stream` SSE 设计**对我们 Web 展示层的实时刷新有参考价值。

---

## E. 旁证：LiteLLM 价格表的获取方式（成本估算的数据基础）

- 业界标准做法就是拉 **`https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json`**（CPA-Manager 源码常量证实）。这是个 JSON，key 是模型名，value 含 `input_cost_per_token`/`output_cost_per_token` 等（按 per-token 浮点，需自己 ×1M 才是每百万价）。
- 轻量替代：`TechyNilesh/LLMPrice`（Python/TS 库，每日从 LiteLLM 同步，离线 bundled，2500+ 模型）。优点：避免直接依赖 LiteLLM 这个重型包（50+ 依赖）。若我们采集器是 Python，可考虑用它做离线价格查询，或干脆自己定时拉那个 JSON 缓存到 Supabase。
- 价格同步频率：LiteLLM 上游每日更新；LLMPrice 每周一发版，可选 `auto_update` 当日拉取。

---

## 对本项目（Supabase 云版）的设计要点提炼

### 必须照搬的硬约束（来自所有实现的共识）
1. **全局单采集器**：同一个 CPA 队列只能有一个 Lens 采集器消费（pop 语义，多消费者互相抢数据）。云版要在文档/配置层强约束，最好能做"采集器单例锁"（如 Supabase 里一张 lock 表 / advisory lock）。
2. **绝不开 SUBSCRIBE**：用 HTTP `/usage-queue` 或 RESP `RPOP`，永远不要用 `SUBSCRIBE usage`（会让 FIFO 端点取不到数据）。
3. **轮询间隔 << retention**：抄 CPA-Manager-Plus，在保存配置时校验 `poll_interval <= retention`，否则拒绝。引导用户把 retention 设到 3600。
4. **接受"采集器停机 > retention 即丢数据"**，明确写进文档；不要承诺历史回补。

### 强烈建议借鉴的设计
5. **去重用 `request_id` + on-conflict-do-nothing**（zhanglunet 模式）。Supabase/Postgres 用 `INSERT ... ON CONFLICT (request_id) DO NOTHING`；`request_id` 缺失时 fallback `sha256(raw_json)`。
6. **优先 HTTP 端点**（`GET /v0/management/usage-queue?count=N`）而非 RESP——能穿透反代，符合"小服务器 + 反代"的目标场景。批量 `count` 取 100~1000（zhanglunet 100 / keeper 10000，云版考虑网络往返成本取中间值如 200~500）。
7. **schema 保留 `raw_json` 原文列**：schema 演进 + 字段缺失补救的安全网。
8. **预存 `ts_epoch` + 本地日期/小时列**：方便按窗口聚合（虽然 zhanglunet 是查询时现算，云版数据量大可考虑预聚合表）。
9. **成本：可编辑价格 + LiteLLM 一键同步，且只对"出现过的模型"维护价格**（keeper/CPA-Manager 共识）。价格表缓存进 Supabase，定时（每日）从 `BerriAI/litellm` JSON 刷新；允许用户手动覆盖（应对新模型/自定义渠道）。
10. **API key 永远存 hash 不存明文**（zhanglunet `sha256[:12]`；keeper/CPA-Manager 也做脱敏）。

### 云版相比本地版的额外注意
11. **采集器与存储分离**：本地实现都是"采集器 + SQLite + 面板"同进程；我们是"采集器 → Supabase（云）→ Web"。采集器跑在用户服务器（贴着 CPA），网络往 Supabase 写。要考虑：写 Supabase 失败时的本地缓冲/重试（否则一次网络抖动 = 丢掉已 pop 但没写入的数据——**这是 pop 语义下云版独有的新风险点**）。建议：先把 pop 到的批次写本地落盘 WAL/队列，确认 Supabase 写成功再 ack，避免"取出来了但没存上"。
12. **磁盘紧张诉求**：目标用户不想跑本地 DB，所以采集器要尽量无状态/小依赖；本地只留极小的"待上传缓冲"，主数据全在 Supabase。

---

## Caveats / 待确认

1. **cpa-usage-keeper 的价格同步具体数据源**未在 README 明写（只说"maintain model prices for cost estimation"）。是否同样用 `BerriAI/litellm` JSON 需看 `internal/service/` pricing 源码确认（本次未深入读其 Go 源码）。
2. **各实现对 token→成本的具体公式**（是否区分 cached/reasoning token 不同单价、是否处理 cache_read vs cache_creation）未逐一核实。LiteLLM 表里有 `cache_read_input_token_cost` 等字段，精确成本需按这些细分单价算——建议实现时直接读 LiteLLM 表的全部 cost 字段。
3. **CPA-Manager-Plus `pollIntervalMs ≤ retention` 校验的精确实现**（边界、单位换算）凭 README 描述，未读源码确认。
4. zhanglunet 的 `quota_snapshots`（Codex wham/usage 余量）是 **Codex/ChatGPT 专属**，依赖 `chatgpt.com/backend-api/wham/usage` 私有接口，易随上游变动失效——若我们要做"账号余量"功能需单独评估，不属于通用 token 成本统计。

---

## 来源 URL 清单

- zhanglunet/cliproxyapi-usage-dashboard（主参考，已读 usage_dashboard.py 全文）: https://github.com/zhanglunet/cliproxyapi-usage-dashboard
- zhanglunet 源码 raw: https://raw.githubusercontent.com/zhanglunet/cliproxyapi-usage-dashboard/main/usage_dashboard.py
- seakee/CPA-Manager: https://github.com/seakee/CPA-Manager
- seakee/CPA-Manager-Plus（推荐后继，三级回退 + pollInterval 校验）: https://github.com/seakee/CPA-Manager-Plus
- CPA-Manager PR #36（usage import/export + LiteLLM 同步 URL 常量 + legacy 导入坑）: https://github.com/seakee/CPA-Manager/commit/a8e56539d6b3ab6dec42ba5deffa29ace25f2e14
- Willxup/cpa-usage-keeper: https://github.com/Willxup/cpa-usage-keeper
- cpa-usage-keeper README raw（环境变量/REDIS_QUEUE_* 默认值）: https://raw.githubusercontent.com/Willxup/cpa-usage-keeper/main/README.md
- ssfun/CLIProxyAPI-Pro（内嵌 usage service + /usage/events + SSE 参考）: https://github.com/ssfun/CLIProxyAPI-Pro
- LiteLLM 价格表 JSON（成本估算数据源）: https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json
- TechyNilesh/LLMPrice（LiteLLM 价格的轻量离线封装，Python/TS）: https://github.com/TechyNilesh/LLMPrice
- 官方 Redis Usage Queue 文档（RPOP/LPOP/SUBSCRIBE/retention 语义，与采集方式强相关）: https://help.router-for.me/management/redis-usage-queue
