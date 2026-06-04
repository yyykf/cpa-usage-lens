# API 契约：仪表盘刷新与视图升级（后端第 2、3 批）

> 供前端子代理对接。仅列出**本 task 新增/变更**的字段；未提及字段保持原样。
> 所有 endpoint 均需 `Authorization: Bearer <token>`；周期参数沿用 `period=today|7d|30d|custom`
> （custom 另带 `from`/`to`，格式 `YYYY-MM-DD`，含端点）。未传 `period` 默认 `7d`。

---

## 1. `GET /api/overview` —— KPI 环比（第 2 批）

在原有绝对值字段之外，**新增** `hasPrevious` 与 `previous` 两个字段，用于 4 个 KPI 的环比角标。

### 请求参数

| 参数 | 说明 |
|------|------|
| `period` | `today` / `7d` / `30d` / `custom`（默认 `7d`） |
| `from` / `to` | 仅 `period=custom` 时必填，`YYYY-MM-DD` 含端点 |

无新增请求参数。环比对比的「上一周期」由后端自动推算：与当前周期**紧邻且等长**的前一段
（如 7d 的 `[05-25, 06-01)` 对应 `[05-18, 05-25)`；custom 任意天数同样等长平移）。

### 响应 JSON（新增字段标注）

```jsonc
{
  // —— 本周期绝对值（原有，未变）——
  "requests": 1234,
  "tokens": 5678901,
  "cost": 12.34,           // number | null（存在缺价模型时为 null = 成本未知）
  "failed": 12,
  "inputTokens": 3000000,
  "outputTokens": 2000000,
  "reasoningTokens": 500000,
  "cachedTokens": 100000,
  "cacheReadTokens": 78000,
  "cacheCreationTokens": 22000,

  // —— 新增：环比 ——
  "hasPrevious": true,     // [新增] bool；false=上一周期完全无数据（无可比基准，见兜底）
  "previous": {            // [新增] object | null；上一等长周期的可比指标（仅 4 个 KPI 维度）
    "requests": 1000,
    "tokens": 5000000,
    "cost": 10.00,         // number | null（上一周期缺价模型时为 null）
    "failed": 20
  }
}
```

### 字段类型与语义

| 字段 | 类型 | 说明 |
|------|------|------|
| `hasPrevious` | `bool` | 上一周期是否有任何数据。`false` → 无可比基准 |
| `previous` | `object \| null` | `hasPrevious=false` 时为 `null`；否则为上一周期汇总 |
| `previous.requests` | `int` | 上一周期总请求数 |
| `previous.tokens` | `int` | 上一周期总 token |
| `previous.cost` | `number \| null` | 上一周期总成本；缺价时 `null`（成本未知） |
| `previous.failed` | `int` | 上一周期失败请求数 |

### 环比兜底语义（后端只给原料，不算百分比 —— 前端按设计稿决定呈现）

> 设计动机：后端若直接算百分比，遇「上一周期为 0」会得到 `↑∞`，遇「成本未知」无从下手。
> 故后端只下发两段绝对值 + 标记位，把兜底**呈现**交给前端设计稿。

1. **无上一周期数据**：`hasPrevious=false` 且 `previous=null`
   → 前端**不应**显示百分比角标（无基准）。建议显示「—」或「新」之类占位（设计稿定）。
2. **上一周期有数据、某指标为 0**（如上一周期 `failed=0`，本周期 `failed>0`）：
   `hasPrevious=true`，`previous.failed=0`。这是**真实的 0**，不是缺数据。
   分母为 0 时百分比无意义 → 前端兜底（如显示「新增」「—」），**不要**渲染 `↑∞` / `↑NaN`。
3. **成本未知**：`cost`（本周期）或 `previous.cost`（上一周期）任一为 `null`
   → 成本 KPI 的环比走「未知」兜底（如「—」），不参与百分比计算。
4. 环比百分比建议算法（前端）：`(本 − 上) / 上 × 100`，仅当 `hasPrevious && 上一周期该指标 > 0 && 两侧成本均非 null（成本卡）` 时才渲染；否则走兜底占位。

---

## 2. `GET /api/models` —— 模型总量排行（第 3 批）

在原有「每日堆叠柱」数据（`models` / `daily`）之外，**新增** `ranking` 与 `metric` 两个字段，
用于「模型总量排行」（水平条形）。**优先扩展现有 endpoint**（KISS），未新增独立 endpoint。

### 请求参数（新增 `metric`）

| 参数 | 取值 | 默认 | 说明 |
|------|------|------|------|
| `period` / `from` / `to` | 同上 | `7d` | 周期 |
| `metric` | `token` / `cost` | `token` | **[新增]** 决定 `ranking` 的**排序口径**。非法值一律按 `token` |

> `metric` 只影响 `ranking` 的**排序依据**；每项始终同时返回 `tokens` 与 `cost` 两个值，
> 前端切换口径只需改排序/展示，**无需二次请求**。
> `cost` 口径用 `model_prices` 价格表 **query-time 实时计算**（复用后端 `internal/pricing`，不在库里存死成本）。

### 响应 JSON（新增字段标注）

```jsonc
{
  // —— 每日 100% 堆叠柱（原有，未变；恒按 token，与 metric 无关）——
  "models": ["gpt-5.4", "claude-...", "..."],   // 周期内模型，按总 token 降序（图例/配色索引）
  "daily": [
    { "date": "2026-05-25", "tokens": { "gpt-5.4": 12000, "claude-...": 3400 } }
    // 每天一项，日期升序；tokens map 仅含当天有数据的模型
  ],

  // —— 新增：模型总量排行 ——
  "metric": "token",        // [新增] string；本次实际生效口径 "token" | "cost"（已归一化）
  "ranking": [              // [新增] array；按 metric 口径降序
    { "model": "gpt-5.4", "tokens": 3000000, "cost": 12.34 },
    { "model": "claude-...", "tokens": 500000, "cost": 8.90 },
    { "model": "legacy-x", "tokens": 9999,  "cost": null }   // cost=null：该模型缺价
  ]
}
```

### 字段类型与语义

| 字段 | 类型 | 说明 |
|------|------|------|
| `metric` | `string` | 实际生效口径，`"token"` 或 `"cost"`（已把非法/空值归一化为 `token`） |
| `ranking` | `array` | 模型总量排行；按 `metric` 口径**降序** |
| `ranking[].model` | `string` | 模型名 |
| `ranking[].tokens` | `int` | 周期内该模型总 token（**始终返回**） |
| `ranking[].cost` | `number \| null` | 周期内该模型总成本；该模型有任一行缺价 → `null`（成本未知） |

### 排序细节（确定性，前端可直接渲染）

- `metric=token`（默认）：按 `tokens` 降序；相同则按 `model` 名字典序。
- `metric=cost`：按 `cost` 降序；**缺价（`cost=null`）的模型一律排到末尾**；
  成本相同则按 `tokens` 降序、再按名字典序。
- 前端**无需**自行重排，直接按 `ranking` 数组顺序绘制水平条形即可；
  若用户切口径，可前端就地按另一字段重排（两值都在），或带 `metric` 重新请求，二者结果一致。

---

## 变更影响汇总（前端需关注）

| Endpoint | 变更 | 前端动作 |
|----------|------|----------|
| `GET /api/overview` | 响应新增 `hasPrevious` / `previous` | 渲染 4 个 KPI 环比角标 + 兜底 |
| `GET /api/models` | 请求新增 `metric`；响应新增 `ranking` / `metric` | 新增「模型总量排行」视图 + token/cost 口径切换（默认 token） |

- 两个 endpoint 均**向后兼容**：仅新增字段，未删除/改名旧字段。
- 未改动 `/api/accounts`、`/api/trend`、`/api/collector`、`/api/prices/refresh`。
- 第 1 批（自动刷新档位）为**纯前端**，后端无对应改动。
- 未改动任何 `api_key` 入库行为（明文 api_key 仍从不入库）。
