# 仪表盘自动刷新与视觉/视图升级 — 实现与验证

> 日期：2026-06-02 ｜ 任务：06-02-dashboard-refresh-visual-upgrade ｜ 分支：feat/dashboard-refresh-visual-upgrade（基于 main）

## 任务目标

把仪表盘三批升级落地：① 自动刷新选择器 ② KPI 环比角标 ③ 模型总量排行（Token/成本口径）。
设计稿（`.project_context/plan/dashboard/2026-06-02_design-mock-v1.html`）已经 code4j 确认。

## 执行步骤（Trellis 流程）

1. 补齐 `implement.jsonl` / `check.jsonl` 上下文 → `task.py start`（planning→in_progress）→ 基于 main 建 feature 分支。
2. **后端子代理**（trellis-implement）：实现环比「上一等长周期」聚合 + 模型总量排行，产出 `api-contract.md`。
3. **前端第 1 批子代理**：自动刷新选择器 + 修 LiveBadge（与后端并行，文件零交集）。
4. **前端第 2/3 批子代理**：KPI 环比角标 + 模型排行视图（对接已 review 的 contract）。
5. **trellis-check 子代理**：全量审查，0 处需修复。
6. **主线程 playwright 实跑验证**（连生产 Supabase 只读，`COLLECTOR_ENABLED=false`）。

## 关键决策

- **环比百分比后端不算、只下发两段绝对值 + `hasPrevious` 标记**：避免「上期为 0 → ↑∞」「成本未知」算不出，前端按设计稿兜底（「新」/「—」）。
- **模型排行扩展现有 `/api/models`**（不新开 endpoint，KISS）；每项 `tokens`+`cost` 双值都返回，切口径前端就地重排、不二次请求。
- **后端先行定契约**：跨层一致性有据（`api-contract.md`），前端不靠猜。
- **后端 ∥ 前端第 1 批并行**：纯前端的自动刷新与后端 Go 改动零文件交集。
- **本地验证不抢线上队列**：编译临时二进制在仓库根目录运行（读根 `.env`），命令行强制 `COLLECTOR_ENABLED=false`（`godotenv.Load()` 不覆盖已存在 env），日志确认「仅提供查询 API」。

## 发现并修复的 Bug（playwright 实跑专属收获）

**自动刷新默认值未生效**：`useAutoRefresh.ts` 的 `readStored()` 中 `Number(null) === 0`，而 `0` 恰是合法的「关闭」档位，导致 `REFRESH_OPTIONS.includes(0)` 为真 → 首次无 localStorage 时被误判为「用户选了关闭」，永远落不到默认 30s。
**修复**：读取前先排除 `raw === null`（无偏好）再 parse。单测与构建都发现不了（前端无单测、类型合法），只有实跑能逮住——这是坚持 playwright 验证的直接价值。

## 验证结果（全部通过）

| 验收点 | 结果 |
|---|---|
| 自动刷新默认 30s | ✅（修复后 reload 确认「自动刷新 · 30s」） |
| 选档后按间隔拉全部数据 | ✅（5 endpoint 各请求 16 次且数量一致 = 拉全量） |
| 关闭档停止轮询 | ✅（计数 32→32 未增长） |
| LiveBadge 反映自动刷新状态 | ✅（不再写死常亮） |
| KPI 环比兜底 | ✅（7d 无上期 → 「新」，不渲染 ↑∞/NaN） |
| KPI 真实环比% | ✅（today：请求▲18%/Token▲94%/成本▲26%/失败▼79%，颜色语义正确） |
| 模型总量排行 + 默认 Token | ✅（真实数据降序） |
| Token/成本口径切换 | ✅（gpt-5.4 按成本反超 gpt-5.5，重排正确） |
| 排行/每日占比视图切换 | ✅（复用现有 ModelStackChart） |
| 前后端构建 / 单测 / 质检 | ✅（tsc+vite build、go build/vet/test 全绿，trellis-check 0 问题） |

验证截图（本地 e2e 产物，已 gitignore 不入库）：`e2e/screenshots/`（01 总览 / 02 成本排行 / 03 每日占比 / 04 today 环比）。

## 已知小点（非 bug，不阻塞）

- 切周期/刷新瞬间会短暂显示旧数据（loadData 走 silent、不闪骨架的设计权衡）。本地跨境连库放大了该窗口；生产环境后端与库同机房，窗口极小、基本无感。
- 前端无单测设施（vitest 未配）；本次修复的默认值 bug 建议后续补回归测试。

## 安全/卖点保持

- 全程未启动采集器、未消费 CPA 队列（守住「全局单采集器」）。
- 未改动任何 `api_key` 入库行为（明文 api_key 从不入库）。
- 两个 API 均向后兼容（仅新增字段）。
