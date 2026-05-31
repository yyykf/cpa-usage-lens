# Tailwind CSS v3 → v4 升级（CSS-first）

> 任务：`.trellis/tasks/05-31-upgrade-tailwind-css-from-v3-to-v4`
> 分支：`feature/tailwind-v4-upgrade`（基于 `feature/cpa-usage-lens-mvp`）
> 日期：2026-05-31

## 任务目标

把 `frontend/` 的 Tailwind 从 v3.4.19 升到 v4，吃到 Rust（Oxide）引擎性能红利、对齐 shadcn 官方 V4 标准（CSS-first），**碳黑暗色仪表盘视觉零回归**。

## 关键决策（与 code4j 确认）

| 决策点 | 选择 | 理由 |
|---|---|---|
| 迁移深度 | 路径 B · 彻底 CSS-first | 用户要"继续用 shadcn"，顺官方 V4 标准最省心；配置进 CSS、删 `tailwind.config.js` |
| 浏览器基线 | 仅现代浏览器 | 内部仪表盘自用，满足 V4 门槛（Safari 16.4+/Chrome 111+/FF 128+） |
| 构建集成 | `@tailwindcss/vite` | Vite 项目官方首选，比 PostCSS 快；删 `postcss.config.js` + `autoprefixer` |
| 色彩 token | 保留 hsl 数值 + `@theme inline` 桥接 | 色值零变化，且保 `/alpha` 修饰——零回归命门 |
| 动画 | 保留 `tailwindcss-animate`（`@plugin`） | 动画类名/行为零变化 |
| darkMode | `@custom-variant dark (&:is(.dark *))` | 锁 class 策略，杜绝 V4 默认"跟随系统暗色"误触发 `dark:` 变体 |

## 执行步骤

1. **codemod 打底**：`npx @tailwindcss/upgrade@latest --force` — 升级依赖、`@tailwind`→`@import`、14 个组件类名迁移（`shadow-sm→shadow-xs`、`outline-none→outline-hidden`、`bg-gradient-to-*→bg-linear-to-*`、`origin-[var()]→origin-()`、`bg-…/[0.08]→bg-…/8` 等标准 V4 等价替换；`ring-[3px]` 任意值保持）。codemod 默认走兼容模式（`@config` 引用旧配置），随后手动改造为路径 B。
2. **手动 CSS-first 改造** `src/index.css`：`@import "tailwindcss"` + `@plugin "tailwindcss-animate"` + `@custom-variant dark` + `@theme inline`（语义色 hsl 桥接 / radius / font / animate）+ `@layer base`（:root hsl 原值 + keyframes）。删除 `@config`。
3. **构建集成切换**：`vite.config.ts` 接 `@tailwindcss/vite`；卸载 `@tailwindcss/postcss` + `autoprefixer`；删 `postcss.config.js` + `tailwind.config.js`。
4. **产物处理（仅记录，未改配置）**：`tsc -b` 会把 `vite.config.ts` 增量编译出 `vite.config.js/.d.ts`，这些已在根 `.gitignore` 忽略（`frontend/vite.config.js`、`frontend/vite.config.d.ts`、`*.tsbuildinfo`），不入库。隐患：Vite 解析配置时 `.js` 优先于 `.ts`，改了 `vite.config.ts` 后须 `npm run build` 同步，否则 dev 可能用到过时 `.js`。未根治（`noEmit` 与 `composite` 冲突），留备查。

## 验证结果

- **构建**：`npm run build`（tsc + vite）通过，1.31s，CSS 25.76 kB。
- **CSS 正确性**（dist 抓取）：`--color-background: hsl(var(--background))`、`--background: 240 14.3% 2.7%` 原值；`.bg-background{background-color:var(--color-background)}` 解析回原 hsl；`.bg-destructive\/10{…hsl(var(--destructive)/.1)}` alpha 正确内联；`animate-in`、`.dark` 均生成。
- **视觉**：docker 重建前端上线 8088（HTTP 200，新 CSS 生效）。零回归依据为 dist CSS 颜色值与 V3 逐字节一致 + 类名均 V4 等价替换；**playwright 像素级截图本次未跑成**（macOS 缺 `timeout`、浏览器未启动），像素实测留待补做或人工在 8088 核对。
- **dev 模式**：`npm run dev` 312ms 启动，`@tailwindcss/vite` 无报错。

## 影响文件

- 改：`package.json`、`package-lock.json`、`vite.config.ts`、`tsconfig.node.json`、`src/index.css`、14 个组件（codemod 类名迁移）。
- 删：`tailwind.config.js`、`postcss.config.js`。

## 备注

- 无 ESLint / 测试脚本，质量门槛 = 构建 + 类型 + 视觉，均通过。
- 既有 JS chunk > 500 kB 警告为历史问题，与本次升级无关，未处理。
