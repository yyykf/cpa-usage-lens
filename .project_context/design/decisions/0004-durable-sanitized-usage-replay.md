# 0004 — 在强类型解析前持久化脱敏 usage replay envelope

- 状态：已确认
- 日期：2026-08-09
- 关联任务：trellis:08-09-durable-cpa-queue-buffer

## 背景

CPA 的 management usage queue 是 destructive pop：响应中的记录在 Lens
确认、校验或写库前已从上游删除。旧 collector 先把响应强类型解码为
`rawQueueItem`、执行 `toEvent`，然后才把 `UsageEvent` 写入 disk buffer。

这个顺序只保护“规范化成功后发生 DB 故障”的场景。若字段类型漂移、
`toEvent` 拒绝记录或进程在 buffer 前退出，已经 pop 的协议证据无法重放。
它也与项目已有的 `pop -> persist -> DB` 规范不一致。

原始 CPA JSON 不能直接落盘，因为它可能包含明文 `api_key`、响应头和失败正文。

## 决策

CPA HTTP client 只把成功响应切成逐项 `json.RawMessage`。collector 随即：

1. 从每项提取 key 指纹与安全掩码；
2. 删除 `api_key`、`response_headers` 和 `fail.body`；
3. 清空内存中的原始 secret-bearing message；
4. 将脱敏 payload 写入带 schema version 的 replay envelope；
5. 完成 file fsync、同目录 atomic rename 和 directory fsync；
6. 再进行 `rawQueueItem` 强类型解析、accounting 校验和 DB insert。

交付语义是 at-least-once replay，加数据库复合键
`(request_id, event_ts, total_tokens)` 幂等，不宣称 exactly-once。

无法规范化的项不静默丢弃：同批有效项确认入库后，脱敏批次改名为
`.rejected` 供排查；损坏或不支持的 envelope 改名为 `.corrupt`。

读取路径兼容旧版 `[]UsageEvent` buffer；新版本只写 replay envelope。

## 否决的备选

### 继续保存规范化后的 `[]UsageEvent`

改动最小，但字段漂移、解析 bug 和 buffer 前崩溃仍会永久丢失数据，不能满足
destructive source 的核心约束。

### 原样保存 CPA JSON

重放信息最完整，但会把明文凭据和其他敏感/大字段带到磁盘，安全风险不可接受。

### 引入通用本地消息队列或 exactly-once 协议

CPA 没有 ack/cursor，Lens 单方面无法实现 exactly-once。引入通用队列会增加部署、
迁移和运维复杂度，不能关闭 source 在响应后、Lens 落盘前的固有窗口，违反 KISS/YAGNI。

## 后果

- CPA 新增未知字段或单字段类型漂移时，脱敏协议证据可先持久化，再决定是否接受。
- DB 成功、buffer 删除前崩溃会重放；现有复合主键负责去重。
- buffer schema 成为需要版本兼容的持久化契约。
- `.rejected` / `.corrupt` 文件需要运维观察和人工清理，不能作为自动重试输入。
- 仍存在三个不可消除窗口：CPA 已 pop 但 envelope rename 尚未完成、整个 HTTP
  响应无法切分为合法 JSON array、Lens 停机超过 CPA retention。
- 此决策不改变数据库、API、前端、环境变量或部署拓扑；单 collector 约束继续有效。
