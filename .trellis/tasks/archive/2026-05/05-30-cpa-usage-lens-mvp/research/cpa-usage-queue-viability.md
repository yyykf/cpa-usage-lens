# Research: CLIProxyAPI usage-queue 数据源可靠性（issue #3048 "移除统计功能"的确切范围）

- **Query**: CLIProxyAPI 维护者在 issue #3048 说 "We will completely remove the statistics feature in mid-to-early June 2026"，要查清"移除"的确切范围——是只删内存聚合统计，还是连 usage-queue 遥测队列 + Redis RESP 队列 + usage publishing 一起砍。
- **Scope**: external（GitHub releases/commits/源码/issue + 官方文档）
- **Date**: 2026-05-30

---

## 一句话结论（最重要）

**usage-queue 数据源是安全的，不会随 issue #3048 的"移除"而消失。** issue #3048 说的"completely remove the statistics feature"指的是**已删除的内存聚合统计**（`internal/usage/` 里的 `RequestStatistics`/Snapshot + 旧的 `/usage` `/usage/export` `/usage/import` 聚合端点）。而我们要依赖的**单条请求遥测队列（`GET /v0/management/usage-queue` + Redis RESP 队列 + usage publishing）是一套完全不同、且仍在被积极开发增强的基础设施**，截至今天（2026-05-30，最新版本 v7.1.32）仍然存在、仍是官方文档推荐的对接方式、仍是所有第三方统计工具（含官方 README 列出的 3 个）的唯一数据源。

> 重要的反直觉点：很多 issue 标题（如 #3188「求还原使用统计」、#3230「Request to add back usage statistics」）让人误以为"统计功能被整体砍掉了"。**被砍的是 CPA 自带的 Web 面板内置统计 UI + 内存聚合**；底层的 per-request 遥测队列不但没删，反而是官方明确指引大家改用的"新统计数据输出方式"。两者是相反方向的事，不要混淆。

---

## Findings

### 1. 时间线已经走完——"移除"早已发生（提前于"6 月中上旬"），但删的不是队列

issue #3048 维护者 @luispater 说的是 "mid-to-early June 2026"。但实际上**移除动作在 v6.10.0 就已经完成了，提前于宣称的时间**。今天（2026-05-30）仓库已经迭代到 **v7.1.32**（2026-05-30 当天发布），早就越过了 v6.10.0 这个分水岭。

证据 1：CLIProxyAPI README 当前正文（`https://raw.githubusercontent.com/router-for-me/CLIProxyAPI/refs/heads/main/README.md`，第 75-89 行）：

```
## Usage Statistics
Since v6.10.0, CLIProxyAPI and CPAMC no longer ship built-in usage statistics. If you need usage statistics, use:
### CPA Usage Keeper ...
### CLIProxyAPI Usage Dashboard ... collects per-request token usage from the Redis-compatible usage queue into SQLite ...
### CPA-Manager ...
```

→ 关键：官方在宣布"不再内置统计"的**同一段话里**，直接把"从 Redis-compatible usage queue 采集 per-request token usage"的第三方工具列为推荐方案。这本身就证明队列是被有意保留、并指定为对接入口的。

证据 2（最有力）：issue #3188（2026-05-02）维护者 @luispater 亲自回复，明确区分了"被删的"和"保留的"：
> 如果确定需要统计功能，可以自行部署这套持久化统计方案：`https://github.com/Willxup/cpa-usage-keeper`。方案完整，支持持久化，CPA团队前期和该项目进行过沟通改进，**支持全新的统计数据输出方式，不再依赖过去基于长期内存存储统计信息的接口，而且采用队列输出的方式获取单条统计明细。**

→ 维护者原话：删的是"长期内存存储统计信息的接口"，新方向是"队列输出 + 单条统计明细"——正是 usage-queue。

### 2. 源码级铁证：删了什么 / 留了什么（基于 main 分支当前源码树，2026-05-30）

通过 GitHub Contents API 逐目录核对当前 `main`：

| 路径 | 状态 | 含义 |
|---|---|---|
| `internal/usage/` | **404 Not Found（已删除）** | 这是内存聚合统计（`RequestStatistics`/Snapshot/persistence）所在目录，正是 #3048 抱怨的"无界内存增长"根因。**被删的就是它。** |
| `internal/api/handlers/management/usage.go` | **存在（1274 字节）** | 队列拉取 handler `GetUsageQueue`，仍在。 |
| `internal/redisqueue/` | **存在**（`queue.go` / `plugin.go` / `usage_toggle.go` / 测试） | per-request 遥测队列的全部基础设施，完好。 |
| `sdk/cliproxy/usage/manager.go` | **存在（7268 字节）** | usage publishing 的发布管线（Manager/Plugin/Record），完好。 |

证据 3：路由注册（`internal/api/server.go`，main 分支）：
- 第 604 行：`mgmt.GET("/usage-queue", s.mgmt.GetUsageQueue)` → `/v0/management/usage-queue` **仍然注册**。
- 第 580-582 行：`/usage-statistics-enabled` GET/PUT/PATCH 仍在（控制队列发布开关）。
- 第 1380-1385 行：热重载时 `redisqueue.SetUsageStatisticsEnabled` 和 `redisqueue.SetRetentionSeconds` 仍被调用。
- 第 36 行 import `internal/redisqueue`，第 320/1444 行 `redisqueue.SetEnabled(...)` 仍在 server 装配里。

> 注意：旧 commit `61b39d4`（2026-05-04）里这个端点叫 `/usage` / `GetUsage`，**之后被改名为 `/usage-queue` / `GetUsageQueue`**。所以网上能搜到的 mintlify 旧文档里写的 `GET /v0/management/usage` / `/usage/export` / `/usage/import` 是**过期信息**，以 help.router-for.me 和源码为准。

### 3. usage publishing 不仅没删，还在被持续增强（决定性反证）

如果队列要被砍，不会有人继续给它加功能。但最近两周（v7.1.13 ~ v7.1.32，2026-05-18 ~ 05-30）的 release changelog 里全是 `feat(usage)`：

| 版本 | 日期 | usage 相关 commit |
|---|---|---|
| v7.1.13 | 05-18 | `feat(runtime): track upstream response headers in logging and usage reporting` |
| v7.1.18 | 05-20 | `Add reasoning effort to usage events` |
| v7.1.24 | 05-27 | `feat(executor): add TTFT tracking and reporting`；`SetTranslatedReasoningEffort to track reasoning levels in usage reporting` |
| v7.1.26 | 05-28 | `feat(usage): include cache tokens in total token calculation and add tests` |
| v7.1.27 | 05-28 | `feat(usage): add service tier tracking and defaults in usage reporting` |

证据 4：对应到 `sdk/cliproxy/usage/manager.go` 的 `Record` 结构体——`TTFT`、`ServiceTier`、`ReasoningEffort`、`ResponseHeaders`、`CacheReadTokens`/`CacheCreationTokens` 都是最近新增字段，与上面 changelog 一一对应。**这是一套活跃维护的产线代码，不是待删的遗留物。**

### 4. 官方文档当前仍把 usage-queue 作为推荐采集方式

证据 5：`https://help.router-for.me/management/api`（Usage Telemetry Queue 一节，当前在线）：
> Legacy aggregated usage endpoints (`/usage`, `/usage/export`, `/usage/import`) are no longer available. **Use `GET /usage-queue` for per-request queue records.**
> `GET /usage-queue?count=10` — Pop up to `count` usage records from the queue.

证据 6：`https://help.router-for.me/management/redis-usage-queue`（Redis Usage Queue RESP 页，当前在线）：
> CLIProxyAPI exposes a minimal Redis RESP interface on the same TCP port as the HTTP API (default 8317). It is designed for pulling recent per-request usage records as JSON so external collectors can ingest telemetry without scraping logs.

→ 两个文档页都把队列定位为"给外部采集器用的"，且把旧聚合端点明确标注为 "no longer available"。文档方向与代码、与维护者表态完全一致。

### 5. usage-queue 的工作机制（决定我们采集器设计的硬约束，来自源码 `internal/redisqueue/queue.go`）

- **pop 语义确认**：`PopOldest(count)` 从队头取走 `count` 条并 `head += count`——**取一条少一条，无法回放**。HTTP 端点 `/usage-queue?count=N` 和 RESP 的 `LPOP`/`RPOP` 都走这条路径。
- **保留窗口**：`redis-usage-queue-retention-seconds`，**默认 60 秒，最大 3600 秒**（`defaultRetentionSeconds=60`，`maxRetentionSeconds=3600`）。每次 enqueue/pop 都会 `pruneLocked` 清掉超过窗口的旧条目——**采集器停机超过这个窗口，期间的数据就永久丢失**。
- **纯内存、不落盘**：队列就是进程内一个切片，**CPA 进程重启 → 队列清空**（`SetEnabled(false)` 会 `clear()`）。
- **发布开关**：`usage-statistics-enabled`（`usage_toggle.go` 的 `usageStatisticsEnabled`，默认 true）。为 false 时 `Enqueue` 直接 return，队列收不到任何记录。
- **队列启用条件**：`redisqueue.SetEnabled` = `hasManagementSecret || Home.Enabled`（server.go:320）。即**必须配置了 `remote-management.secret-key`（Management 启用）队列才工作**；Management 关掉时 RESP 连接直接断开。
- **SUBSCRIBE 与 FIFO 互斥（大坑）**：`Enqueue` 里 `if global.publishToSubscribers(payload) { return }`——**只要有任何一个 `SUBSCRIBE usage` 订阅者在线，新记录就只走 pub/sub 广播、不进 FIFO 队列**，事后无法再用 `LPOP`/`RPOP`/HTTP 端点补取。多个采集方式不能混用。

### 6. 队列 payload 字段（来自官方文档 + 源码 Record 结构）

`/usage-queue` 返回 JSON 数组（`count=1` 也是数组，空队列返回 `[]`）。单条字段：
```json
{
  "timestamp": "2026-05-05T12:00:00Z", "latency_ms": 1234,
  "source": "user@example.com", "auth_index": "0",
  "tokens": {"input_tokens":10,"output_tokens":20,"reasoning_tokens":0,"cached_tokens":0,"total_tokens":30},
  "failed": false, "provider": "openai", "model": "gpt-5.4", "alias": "gpt-5.4",
  "endpoint": "POST /v1/chat/completions", "auth_type": "api_key",
  "api_key": "sk-...", "request_id": "req_..."
}
```
近期新增字段（来自 SDK Record）：`reasoning_effort`、`service_tier`、`cache_read_tokens`/`cache_creation_tokens`、`ttft`、`response_headers` —— 实际 JSON 里是否全部输出需在目标 CPA 版本上抓一次真实样本确认（见下方待确认项）。

---

## 对本项目（CPA Usage Lens / Supabase 云版）的影响与建议

1. **可以放心开工。** 地基风险解除：usage-queue 不会因 #3048 消失。我们的"外部采集器 + Supabase 存储 + Web 展示"架构方向与官方指引一致（官方主动把大家往 per-request 队列 + 外部持久化方案上引）。

2. **优先用 HTTP `GET /v0/management/usage-queue?count=N`，而不是 Redis RESP。** 理由：
   - HTTP 端点能穿透普通 HTTP 反代（小服务器常见部署），RESP 走的是 8317 裸 TCP，过不了 HTTP 反代（CPA-Manager 文档反复强调这点）。
   - HTTP 端点是 v6.10.8+ 才有的（早期只有 RESP）。我们的目标用户大概率已在 v7.x，没问题。
   - 二者数据等价（同一个队列同一套 pop 语义）。

3. **必须把"单采集器 + pop 语义不可回放"当成一等约束来设计**（详见 reference-implementations.md）。Supabase 云版尤其要注意：
   - 全局只能有**一个**采集器消费同一个 CPA 队列（多实例会互相抢走对方的数据）。
   - 绝不能开 `SUBSCRIBE usage`（否则 FIFO 端点取不到数据）。
   - 采集器轮询间隔必须 **远小于** `redis-usage-queue-retention-seconds`；建议引导用户把 retention 调到 3600，采集间隔几秒级。

4. **建议在文档里要求用户 CPA 配置：** `usage-statistics-enabled: true` + `redis-usage-queue-retention-seconds: 3600` + 已设置 `remote-management.secret-key`。三者缺一队列就拿不到数据。

5. **版本下限建议 CPA v6.10.8+（实测目标 v7.1.x）。** 低于 v6.10.8 没有 HTTP 队列端点，只能 RESP。

---

## Caveats / 待人工确认（不确定项）

1. **config.example.yaml 注释措辞有歧义/疑似过期**：第 66 行写 `# When false, disable in-memory usage statistics aggregation`，第 70 行写 `# The local Redis RESP usage output is disabled.`。但源码 `usage_toggle.go` 的实际语义是：`usage-statistics-enabled` 现在控制的是**队列发布（enqueue）**，不是"内存聚合"（聚合代码 `internal/usage/` 已删、不存在了）。**结论不变**（这个开关就是我们队列数据的总闸，必须为 true），但官方注释文字本身已落后于代码，不要被它误导。

2. **"是否会在更晚的版本里连队列也删"无法 100% 排除**，但当前所有信号都指向相反方向（官方主动推、持续加功能、文档推荐、第三方生态依赖）。维护者从未表达过要删队列。风险等级：低。缓解：采集器对 payload 字段做容错解析，保留 `raw_json` 原文，未来 schema 变了也能补救。

3. **队列 payload 的完整真实字段**需要在**实际目标 CPA 版本**上 `curl /v0/management/usage-queue?count=1` 抓一次真实样本核对（尤其 `service_tier`/`ttft`/`cache_read_tokens` 等新字段在不同 provider 下是否都有、命名是否与 SDK 字段一致）。文档示例可能滞后于代码。

4. **CPA 生态本身的稳定性**有杂音：issue #3188 评论提到 CPA Plus 已删库、CPA Business 疑似停更。但这是**周边分支**的事，主仓 `router-for-me/CLIProxyAPI` 本身更新极其活跃（5 月几乎每天发版），无需担心主仓与队列功能。

---

## 来源 URL 清单

- issue #3048（被引用的"移除统计"原话）: https://github.com/router-for-me/CLIProxyAPI/issues/3048
- issue #3188（维护者明确指向队列方案/cpa-usage-keeper）: https://github.com/router-for-me/CLIProxyAPI/issues/3188
- issue #3230（用户求恢复面板统计，侧证面板统计被删）: https://github.com/router-for-me/CLIProxyAPI/issues/3230
- README（Since v6.10.0 不再内置统计 + 推荐队列工具）: https://github.com/router-for-me/CLIProxyAPI/blob/main/README.md
- Management API 文档（usage-queue 推荐 + 旧端点 no longer available）: https://help.router-for.me/management/api
- Redis Usage Queue (RESP) 文档（retention/LPOP/SUBSCRIBE 语义）: https://help.router-for.me/management/redis-usage-queue
- 源码 `internal/redisqueue/queue.go`（pop/retention/subscribe 实现）: https://github.com/router-for-me/CLIProxyAPI/blob/main/internal/redisqueue/queue.go
- 源码 `internal/redisqueue/usage_toggle.go`（发布开关语义）: https://github.com/router-for-me/CLIProxyAPI/blob/main/internal/redisqueue/usage_toggle.go
- 源码 `internal/api/handlers/management/usage.go`（GetUsageQueue handler）: https://github.com/router-for-me/CLIProxyAPI/blob/main/internal/api/handlers/management/usage.go
- 源码 `sdk/cliproxy/usage/manager.go`（usage publishing 管线 + Record 字段）: https://github.com/router-for-me/CLIProxyAPI/blob/main/sdk/cliproxy/usage/manager.go
- 路由注册 `internal/api/server.go`（确认 /usage-queue 仍注册）: https://github.com/router-for-me/CLIProxyAPI/blob/main/internal/api/server.go
- Releases（v7.1.13~v7.1.32 的 feat(usage) 证据）: https://github.com/router-for-me/CLIProxyAPI/releases
- 改名前的旧 commit 61b39d4（/usage → /usage-queue 演变佐证）: https://github.com/router-for-me/CLIProxyAPI/commit/61b39d49bd8cad26c8d74eb0bd0f6b8fda16ab2c
