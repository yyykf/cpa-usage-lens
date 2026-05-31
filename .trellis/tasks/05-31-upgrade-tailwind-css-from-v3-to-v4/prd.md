# Upgrade Tailwind CSS from v3 to v4

## Goal

把 `frontend/` 的 Tailwind CSS 从 v3.4 升级到 v4，吃到 Rust（Oxide）引擎的构建 / HMR 性能红利并简化配置依赖，**同时保证碳黑暗色仪表盘视觉零回归**。走 shadcn 官方 V4 标准的 **CSS-first 路线**：配置统一进 `src/index.css`，删除 `tailwind.config.js`。

## What I already know

- 当前 Tailwind **3.4.19**（package.json 声明 `^3.4.15`）。
- 构建链：Vite 5.4 + PostCSS（`tailwindcss` + `autoprefixer`）。
- UI 体系：shadcn/ui（`radix-nova`），全套 `hsl(var(--xxx))` 通道色；`tailwindcss-animate` 插件。
- 单入口 `src/index.css`：`@tailwind` 三件套 + `@layer base` + `@apply`；`tailwind.config.js` 含语义色 / keyframes / radius 派生 / fontFamily / `darkMode: ['class']`。
- 规模小：1 个 CSS 文件 + ~23 个 tsx，迁移可控。
- Node **v20.19.3** ✓ 满足官方 codemod（`npx @tailwindcss/upgrade`）要求。
- **服务形态**：`docker-compose.yml` 中 frontend 为 nginx **生产容器**（`npm run build` → 静态托管），端口 **8088:80**；backend 8080，Supabase 云。验证 = `docker compose up -d --build frontend` 后访问 `http://localhost:8088`。
- 已有 `dashboard-e2e.png` 基线 + `.playwright-cli`，可做视觉比对。

## Assumptions

- 仅内部 / 现代浏览器访问，满足 V4 门槛（Safari 16.4+ / Chrome 111+ / Firefox 128+）。**[已确认]**
- 暗色为唯一主题（MVP），`:root` 直接铺暗色，页面无 `.dark` 祖先类。
- 视觉零回归是头号验收目标。

## Requirements

- Tailwind 升级到 v4 最新稳定版，`npm run build`（`tsc -b && vite build`）通过。
- 配置从 `tailwind.config.js` 迁进 `src/index.css` 的 `@theme`，删除 JS 配置文件。
- 构建集成换 `@tailwindcss/vite` 插件，删除 `postcss.config.js` + `autoprefixer`。
- shadcn 色彩 token：`:root` 保留现有 hsl 通道值不动，`@theme inline` 桥接（色值零变化、保 `/alpha`）。
- **darkMode 行为保持**：用 `@custom-variant dark (&:is(.dark *))` 锁定 class 策略，确保组件里的 `dark:` 变体在无 `.dark` 祖先时**不生效**（与 V3 现状一致），杜绝"跟随系统暗色"导致的回归。
- 动画：保留 `tailwindcss-animate`，在 CSS 中用 `@plugin "tailwindcss-animate"` 加载；自定义 keyframes（accordion / caret-blink / pulse）迁进 `@theme` + `@keyframes`。
- 类名 breaking change 全部修正（codemod + 人工核对）。
- 视觉零回归（与升级前 8088 截图比对一致）。

## Acceptance Criteria

- [x] `npm run build` 通过，无 Tailwind / PostCSS 报错。（tsc + vite，1.31s）
- [x] `docker compose up -d --build frontend` 成功，8088 页面正常渲染（CSS 哈希与本地 V4 构建一致）。
- [~] 视觉零回归：依据为 dist CSS 颜色值与 V3 逐字节一致 + 类名均 V4 等价替换（强推断）；playwright 像素级截图本次未跑成（环境缺 `timeout`、浏览器未启动），像素实测待补或人工在 8088 核对。
- [x] `dark:` 变体行为不变（`@custom-variant dark` 锁定，`.dark` 已注册，无祖先即不生效）。
- [x] shadcn 组件动画正常（`@plugin tailwindcss-animate`，CSS 含 `animate-in` 等）。
- [x] `tailwind.config.js`、`postcss.config.js`、`autoprefixer`、`@tailwindcss/postcss` 已移除，改用 `@tailwindcss/vite`；依赖 / 锁文件一致。
- [x] 颜色桥接正确：非 alpha 走 `var(--color-*)` 解析回原 hsl；alpha（如 `/10`）内联为 `hsl(var(--*)/.1)`。
- [x] `vite.config.js` / `.d.ts`（`tsc -b` 增量产物）已被根 `.gitignore` 忽略，不入库。（未改 `tsconfig.node.json`：`noEmit` 与 `composite` 冲突，且产物已忽略，故按 YAGNI 不动）

## Definition of Done

- 构建 / 类型检查通过。
- 视觉比对无回归。
- 依赖清理干净，`package.json` / `package-lock.json` 一致。
- 执行摘要更新到 `.project_context/execution/`。

## Out of Scope (explicit)

- 亮色主题。
- shadcn 组件库整体升级到最新 registry。
- 业务功能 / 布局改动。
- `vite.config.js` / `vite.config.d.ts` 编译产物入库问题（仅记录，单独处理）。

## Decision (ADR-lite)

**Context**: V3.4 → V4 升级；用户非前端背景，诉求为「升级 + 继续用 shadcn + 视觉别坏」，技术取舍授权由我决定。

**Decision**:
- 走 **CSS-first（路径 B）**：codemod 打底，`tailwind.config.js` 的 theme 迁进 `src/index.css` 的 `@theme`，删除 JS 配置——对齐 shadcn 官方 V4 标准。
- 浏览器 **仅现代浏览器**，接受 V4 基线。
- 构建集成换 **`@tailwindcss/vite`** 插件，删除 `postcss.config.js` + `autoprefixer`。
- 色彩 token **保留 hsl 数值 + `@theme inline` 桥接**：色值零变化，碳黑 / 哑光观感零回归。
- 动画 **保留 `tailwindcss-animate`（`@plugin`）**：动画类名 / 行为零变化。
- darkMode 用 **`@custom-variant dark`** 锁定 class 策略，杜绝 `dark:` 变体被系统暗色误触发。

**Consequences**:
- 收益：配置统一进 CSS、依赖更少、构建更快、面向未来、对齐 shadcn 官方。
- 成本 / 风险：改动面较大但体量小可控；shadcn `hsl(var())` 用 `@theme inline` 重接；视觉需逐项截图比对。

## Technical Notes

- codemod：`npx @tailwindcss/upgrade`（Node ≥20 ✓），自动处理依赖 / 配置迁移 / 类名重命名 / 指令替换的大部分；shadcn token、darkMode 变体、视觉细节人工核对。
- 关键变化：`@tailwind` → `@import "tailwindcss"`；构建集成 → `@tailwindcss/vite`；`autoprefixer` / `postcss-import` 内置可删。
- 类名 breaking：`shadow-sm→shadow-xs`、`rounded-sm→rounded-xs`（sm 档整体下移一级）、`outline-none→outline-hidden`、`ring` 默认 3px→1px、默认 border 色 `gray-200→currentColor`。
  - 注：组件多处显式写 `ring-[3px]` / `focus-visible:ring-[3px]`，为任意值，**不受** `ring` 默认宽度变化影响。
- shadcn token 桥接：`:root` 保留 hsl 通道值不动，`@theme inline` 内写 `--color-*: hsl(var(--*))`。
- darkMode：V3 `darkMode: ['class']` → V4 `@custom-variant dark (&:is(.dark *))`；项目无 `.dark` 祖先，故 `dark:` 变体保持不生效。
- 验证：`docker compose up -d --build frontend` → playwright 访问 `http://localhost:8088` 截 Dashboard + Login，与升级前基线比对。
- 附带（out of scope）：`tsconfig.node.json` 把 `vite.config.ts` 编译出 `vite.config.js` / `.d.ts`，疑似产物入库，单独评估。
