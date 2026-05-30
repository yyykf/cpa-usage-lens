# CPA Usage Lens — 前端视觉风格规范（Design System）

> **方向**：暗色 Bento 控制台（Dark Bento Console）
> **技术栈**：React + Vite + TypeScript + Tailwind CSS + shadcn/ui + Recharts + lucide-react
> **用途**：前端实现的视觉蓝图。新会话实现前端时，**照本文件落地即可**，无需重新决策视觉。
> **依据**：ui-ux-pro-max 搜索（Data-Dense Dashboard + Dark Mode OLED + Bento Box Grid 三者融合）
> **类比产品**：Vercel / Linear / Railway 的暗色后台观感。

---

## 1. 整体基调

- 暗色基底 + Bento 卡片网格：模块化卡片拼成不对称网格，深色背景上卡片轻微"上浮"。
- 气质：专业、技术、克制、现代。是"开发者 API 用量台"，不是营销页——**信息密度优先，但留呼吸感**。
- 默认**暗色单主题**（MVP 不做亮/暗切换；后续可加）。
- 避免：华丽装饰、渐变滥用、emoji 当图标、霓虹/赛博朋克花哨效果。

---

## 2. 配色（shadcn CSS 变量，暗色为默认）

直接写入 `globals.css` 的 `:root`（本项目暗色即默认）。值给的是 hex；若用旧版 shadcn（HSL 模式）按需转换，新版 shadcn CLI 可直接用 hex/oklch。

| 变量 | 值 | 用途 |
|------|-----|------|
| `--background` | `#0A0E1A` | 页面深底（午夜蓝黑，比纯黑柔和） |
| `--foreground` | `#E6E9F0` | 主文字 |
| `--card` | `#111627` | 卡片背景（比底略亮，形成上浮感） |
| `--card-foreground` | `#E6E9F0` | 卡片内文字 |
| `--popover` | `#111627` | 弹层/下拉背景 |
| `--muted` | `#1A2036` | 次级背景 / 行 hover 底 |
| `--muted-foreground` | `#8B93A7` | 次要文字、标签、轴标签（对比≈6:1，达标） |
| `--border` | `#232A40` | 卡片/分隔线边框（微妙亮边） |
| `--input` | `#232A40` | 输入框边框 |
| `--primary` | `#3B82F6` | 主强调（电光蓝）：主按钮、链接、选中态 |
| `--primary-foreground` | `#FFFFFF` | 主强调上的文字 |
| `--accent` | `#1E2540` | hover 高亮背景 |
| `--accent-foreground` | `#E6E9F0` | accent 上文字 |
| `--ring` | `#3B82F6` | focus 焦点环 |
| `--destructive` | `#EF4444` | 失败/错误（红） |
| `--radius` | `0.875rem`（14px） | 全局圆角基准（偏大，Bento 感） |

### 数据语义色（图表 / 指标，固定语义，全站一致）

| 语义 | 色值 | 用在 |
|------|------|------|
| 请求数 | `#3B82F6`（蓝） | 趋势线、请求相关指标 |
| token | `#22D3EE`（青） | 趋势线、token 相关指标 |
| 成本 | `#F97316`（橙） | 趋势线、成本相关指标 |
| 失败 | `#EF4444`（红） | 失败数、错误态、告警 |
| 成功/健康 | `#10B981`（绿） | 采集器"正常"状态点 |

> 颜色不作为唯一指示：失败/异常除红色外，必须同时配图标或文字（色盲友好）。

---

## 3. 字体

辨识度强、技术感足，刻意避开满大街的 Inter。

```css
@import url('https://fonts.googleapis.com/css2?family=Fira+Code:wght@400;500;600;700&family=Fira+Sans:wght@300;400;500;600;700&display=swap');
```

Tailwind 配置：
```js
fontFamily: {
  sans: ['"Fira Sans"', 'system-ui', 'sans-serif'],
  mono: ['"Fira Code"', 'ui-monospace', 'monospace'],
}
```

用法约定：
- **Fira Code（mono）**：所有**数字**（KPI 大数、表格数值、成本、token、图表轴）、以及标题——等宽让数字纵向对齐、技术感强。
- **Fira Sans（sans）**：正文、标签、说明文字、按钮文字。
- 正文 line-height 1.5–1.6；正文最小 14px（移动端 16px）。

---

## 4. 布局与栅格（Bento）

- 容器：`max-w-7xl mx-auto px-4 md:px-6`，垂直 `py-6`。
- 顶部 Header：左=项目名/Logo，右=周期切换器（+ 可选账号筛选）。
- **KPI 行**：`grid grid-cols-2 lg:grid-cols-4 gap-4`（4 个总览卡）。
- **Bento 主区**：`grid grid-cols-1 lg:grid-cols-3 gap-4`
  - 趋势图卡 → `lg:col-span-2`（宽）
  - 采集器健康卡 → `lg:col-span-1`（窄）
  - 账号用量榜卡 → `lg:col-span-3`（全宽）
- 卡片统一：`rounded-2xl border bg-card p-5 md:p-6`；卡片间 `gap-4`。
- 卡片 hover：`hover:border-primary/40 transition-colors duration-200`（暗色下用边框提亮区分，不靠重阴影）。

---

## 5. 组件视觉规范

### 5.1 KPI 总览卡（×4：总请求 / 总 token / 总成本 / 失败数）
- 结构：顶部小标签（muted，Fira Sans）+ 中部大数字（Fira Code，`text-3xl md:text-4xl font-semibold`）+ 底部可选环比/迷你说明。
- 失败数卡：数字 > 0 时用 `--destructive` 红色 + 小三角/警告图标。
- 可选：右上角放一个对应语义色的小图标（lucide：Activity / Coins / DollarSign / AlertTriangle）。

### 5.2 周期切换器（今天 / 近7天 / 近30天 / 自定义）
- 用 shadcn `Tabs` 或 `ToggleGroup` 做快捷项（今天/近7/近30）。
- "自定义"→ `Popover` + `Calendar`（范围选择）。
- 选中态：`bg-primary text-primary-foreground`；未选：`text-muted-foreground hover:text-foreground`。

### 5.3 账号用量榜（核心模块，shadcn Table）
- 列：账号（邮箱/标签，左对齐，Fira Sans）｜ 请求数 ｜ token ｜ 成本 ｜ 失败数（数值列**右对齐**，Fira Code）。
- 成本列：`$` 前缀；缺价时显示灰色"未知"而非 0。
- 失败数 > 0：红色 + 细角标。
- 行 hover：`hover:bg-muted/50`，`cursor-default`（整行不可点则不加 pointer；列头可点排序加 pointer）。
- 列头支持点击排序（默认按成本或请求降序），带升降箭头图标。
- 表头：`text-muted-foreground text-xs uppercase tracking-wide`。

### 5.4 趋势图（Recharts，详见第 7 节）
- 卡片标题 + 右侧图例（请求/token/成本，三个语义色小圆点 + 文字）。
- 支持切换显示哪条线（点图例 toggle）。

### 5.5 采集器健康卡
- 顶部状态行：圆形状态点（绿`#10B981`=采集中 / 红`#EF4444`=异常）+ 文字（"采集中" / "异常"）。
- 指标行（Fira Code 数值）：采集延迟（如"3s 前"）、最后错误（有则红字截断显示，hover 看全文）。
- **数据库占用**：分别显示明细表、聚合表的真实大小（如 `明细 2.3 MB / 聚合 11.8 MB`），**绝对值、不显示百分比**。

### 5.6 登录页（单用户密码）
- 全屏居中，暗背景可叠加极低调的径向渐变或点阵网格（增加现代质感，克制）。
- 居中 `Card`（约 `max-w-sm`）：项目名/Logo + 密码 `Input`（带 label）+ 主色登录 `Button`（loading 时禁用并显示 spinner）。
- 错误提示在输入框下方，红字 + 图标，靠近问题处。

---

## 6. 交互与动效

- 过渡统一 `transition-colors`/`transition-all duration-200`（150–300ms 区间）。
- hover 反馈：卡片边框提亮 / 可选 `hover:-translate-y-0.5`（轻微）；**禁止**会引起布局位移的 scale。
- 加载态：用 shadcn `Skeleton` 占位（KPI、表格、图表各自骨架），**预留空间避免内容跳动**。
- 图表：hover tooltip；可选数字 count-up（必须尊重 `prefers-reduced-motion`）。
- 所有可点元素加 `cursor-pointer`；async 按钮点击后禁用 + spinner。
- 尊重 `prefers-reduced-motion: reduce`：关闭非必要动画。

---

## 7. 图表规范（Recharts）

- 主图：`LineChart` 或 `AreaChart`（趋势）；多账号对比可用 Grouped Bar（第二批）。
- `CartesianGrid`：`stroke="#232A40"`，低对比、虚线可选。
- 坐标轴：tick 用 `--muted-foreground`，字体 Fira Code 小号；Y 轴数值简化（如 1.2k / $3.40）。
- 线：`strokeWidth={2}`，颜色用第 2 节语义色；`dot={false}`，hover 显示 active dot。
- 面积填充：对应语义色 **20% opacity**（用 linearGradient 从 20%→0%）。
- Tooltip：背景 `#111627`、边框 `#232A40`、`rounded-lg`，数值 Fira Code；标题为日期。
- 图例：Fira Sans 小字 + 语义色圆点；可点击 toggle 系列。
- 优先用 shadcn 官方 `Chart` 组件（对 Recharts 的封装，自带主题变量对接）。
- 无障碍：提供数据的表格替代（账号榜本身即承担部分此职责）。

---

## 8. 图标

- 统一用 **lucide-react**（shadcn 生态默认），**禁止 emoji 当图标**。
- 尺寸统一 `w-4 h-4`（16）或 `w-5 h-5`（20），同一区域不混用尺寸。
- 常用：Activity（请求）、Coins（token）、DollarSign（成本）、AlertTriangle（失败）、Database（容量）、RefreshCw（刷新价格表）、Clock（延迟）、CheckCircle/XCircle（健康状态）。

---

## 9. Accessibility & 交付质量检查清单

- [ ] 文字对比 ≥ 4.5:1（暗色下逐项验证，尤其 muted 文字）
- [ ] focus 焦点环可见（`--ring` 蓝），Tab 顺序合理
- [ ] 图标为 SVG（lucide），非 emoji；图标按钮加 `aria-label`
- [ ] 触摸目标 ≥ 44×44px
- [ ] 失败/异常不只靠红色，配图标或文字
- [ ] 图表配色对色盲友好 + 有表格替代
- [ ] 表单输入有 `label`（登录密码）
- [ ] hover 不造成布局位移；过渡 150–300ms
- [ ] `prefers-reduced-motion` 已处理
- [ ] 响应式自测：375 / 768 / 1024 / 1440px，无横向滚动

---

## 10. shadcn 组件映射（实现时按需 add）

| 模块 | shadcn 组件 |
|------|-------------|
| 卡片（KPI / Bento） | `card` |
| 账号用量榜 | `table` |
| 周期切换 | `tabs` 或 `toggle-group` + `popover` + `calendar` |
| 登录 | `input` `label` `button` |
| 状态/失败标记 | `badge` |
| 加载占位 | `skeleton` |
| 图表 tooltip / 说明 | `tooltip` |
| 趋势图 | `chart`（shadcn 对 Recharts 的封装） |
| 错误提示（可选） | `sonner` |

---

## 实现提示

- 实现前端时可用 `ui-ux-pro-max`（按本规范取风格/组件细节）+ `frontend-design`（提升代码设计质量、避免通用感）两个 skill 辅助。
- 主题色先落进 `globals.css` 与 `tailwind.config`，再搭组件，保证全站一致。
