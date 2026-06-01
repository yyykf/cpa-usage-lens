# Backend Security and Reliability Review

## 结论

当前后端没有发现明显 SQL 注入或数据 API 鉴权遗漏；`/api/overview`、`/api/accounts`、`/api/trend`、`/api/collector`、`/api/prices/refresh` 均经 `requireAuth` 包裹。

需要优先修复 2 个高严重度数据丢失风险：采集缓冲失败后的 pop 数据丢失，以及 rollup 清理并未严格保证被删明细已聚合。

## 核心发现

1. `backend/internal/collector/collector.go:103` - 高 - 缓冲落盘失败后仍继续走写库路径；如果随后 `InsertEvents` 失败，已从 CPA pop 的事件既不在队列里，也没有缓冲文件，直接丢失。修复建议：落盘失败时不要进入普通轮询路径；对内存批次做阻塞重试直到 DB 写入成功，或将采集器置为 fatal/degraded 并停止继续 pop，同时在启动前健康检查缓冲目录可写。

2. `backend/internal/rollup/rollup.go:55` - 高 - 调度只 rollup `[cutoff, today]`，但第 60 行删除 `< cutoff` 的明细；首次接入已有历史热明细、或某天 rollup 连续失败后，下一次成功 tick 会删除窗口左侧未聚合数据。修复建议：删除前先聚合实际待删除日期范围，或引入 rollup watermark/按日确认表，只删除已确认聚合的日期。

3. `backend/internal/rollup/rollup.go:55` - 高 - `RollupRange` 与 `DeleteHotBefore` 是两个独立 SQL 调用，没有共享事务快照；并发 collector 可能在 rollup 查询完成后、delete 执行前插入 `event_ts < cutoff` 的迟到事件，随后被 delete 删除但未进入 daily 聚合。修复建议：把 rollup+delete 合并到 DB 层单事务，至少使用 REPEATABLE READ 快照；或在 rollup 开始时记录 `ingested_at` 上界，delete 只删该上界之前已被 rollup 覆盖的行。

4. `backend/internal/pricing/cost.go:19` - 中 - 成本计算忽略 `Tokens.Cached`。CPA 的 OpenAI-style usage 会把 `cached_tokens` 填到 `cached_tokens` 字段，而不是 `cache_read_tokens`；当前公式按全量 `Input` 收 input 单价，又没有把 `Cached` 作为 cache-read 折扣，缓存命中成本会被高估。修复建议：当 `Cached > 0` 且 `CacheRead == 0` 时，将 `Cached` 归一化为 cache-read，并用 `(Input - Cached)` 计算普通 input 成本；Claude-style `CacheRead/CacheCreation` 已显式存在时不要重复扣减。

5. `backend/internal/pricing/cost.go:32` - 中 - `cacheCost` 在 cache token > 0 且专价与 input fallback 都缺失时返回 0，但 `Cost` 仍返回 `ok=true`；这会把缺价场景显示成 `$0` 而不是“未知”。修复建议：cache token > 0 时如果既无专价也无 input 价，应返回 `ok=false`，由 report 层继续显示未知成本。

6. `backend/internal/api/handlers.go:12` - 中 - `/api/login` 没有速率限制、失败计数或请求体大小限制；前端端口公开后，单用户密码可以被持续暴力尝试，bcrypt 只能提高单次成本，不能限制尝试次数。修复建议：加 IP 级 token bucket/失败退避，或在 nginx/backend 都限制登录频率；同时用 `http.MaxBytesReader` 限制 JSON body。

7. `backend/internal/collector/buffer.go:40` - 中 - 缓冲文件直接 `os.WriteFile` 到最终 `.json` 路径，没有临时文件、rename、fsync；进程或宿主机崩溃时可能留下半写文件，`recoverPending` 只能反复记录加载失败，无法恢复这批已 pop 数据。修复建议：写 `*.tmp`、fsync 文件、atomic rename 到 `.json`、fsync 目录；加载失败的文件移动到 quarantine，并把 collector 状态置错以便人工处理。

## 详细内容

- SQL 注入：`QueryDailyUsage` 使用 `$1/$2` 参数，`RollupRange`/`DeleteHotBefore` 对日期和时区也使用参数绑定，本次未发现字符串拼 SQL 的用户输入路径。
- 鉴权遗漏：除公开的 `/healthz` 与 `/api/login` 外，数据与刷新接口均经 `requireAuth`，本次未发现未鉴权数据接口。
- 验证：已执行 `go test ./...`，全部通过。当前测试没有覆盖 rollup/cleanup 的故障恢复与并发迟到事件场景，也没有覆盖 cached token 成本语义。
