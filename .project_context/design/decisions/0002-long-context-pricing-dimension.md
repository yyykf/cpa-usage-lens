# 0002 — daily usage 使用 long_context 行维度保留整请求价格档

- 状态：已确认
- 日期：2026-07-12
- 关联任务：trellis:07-12-fix-codex-usage-pricing

## 背景

OpenAI 对 GPT-5.4、GPT-5.5、GPT-5.6 的长上下文不是边际计价：单次请求
`input_tokens > 272000` 后，整个请求的 input/cache/output 都切换到高档价格。
`daily_account_usage` 原本直接汇总一天内的 token，无法再区分一个 300K 请求和
多个小请求，因此查询时无法恢复正确价格档。

## 决策

为 `daily_account_usage` 增加 `long_context boolean`，并把它加入主键：

```text
(usage_date, source, model, key_fingerprint, long_context)
```

模型阈值和两套单价保存在 `model_prices`；rollup 在 hot 明细仍存在时逐请求使用
严格条件 `input_tokens > threshold` 分类。同一日、账号、模型、key 最多形成基础档
和长上下文档两行。查询先按行选择价格，再把两行汇总成既有 API 视图。

阈值来自 LiteLLM 模型元数据，不把 272000 作为全局常量。当前数据模型只支持每个
模型一个阈值；多阈值模型不强行压缩成 boolean。

## 否决的备选

- **在一行中复制一套 long-context token bucket**：主键不变，但几乎所有 token
  字段都要镜像，rollup/query 容易漂移；以后新增价格档会继续扩列。
- **按 daily 总输入判断**：把多个小请求误判为一个长请求，计费语义错误。
- **保存固定 cost**：违背现有 query-time pricing 设计，价格刷新后需要历史回填。
- **全局硬编码 272K**：当前 GPT-5 可用，但无法表达未来模型或其他 provider 的不同阈值。

## 后果

- 迁移把 daily 主键从四列扩为五列，旧 Lens binary 的四列 `ON CONFLICT` 不再可用；
  部署必须停止旧 Lens、迁移、再启动新 Lens，不支持滚动升级。
- 已有 daily 行统一为 `long_context=false`，不追溯历史；hot retention 内的数据可在
  价格元数据刷新后通过原子重建重新分类。
- rollup 改为事务内删除目标 retained window 后重建，避免阈值变化留下陈旧价格桶。
- 未来需要多个阈值时，应以新的 ADR 把 boolean 演进为通用 `pricing_tier`，而不是
  在本结构继续增加布尔列。
