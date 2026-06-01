# Styling Guidelines

> CSS, theming, and Tailwind conventions for this project.

This project uses **Tailwind CSS v4 (CSS-first)** + **shadcn/ui**. All theme
configuration lives in `src/index.css` — there is **no `tailwind.config.js`**.
Design tokens are shadcn HSL-channel CSS variables declared in `:root`
(`--background`, `--primary`, … ; dark-only MVP, light theme reserved).

---

## Convention: Theme tokens MUST use `@theme inline`

**What**: When bridging shadcn token variables into Tailwind's theme in
`src/index.css`, always use `@theme inline { … }` — never bare `@theme { … }`.

**Why**: shadcn color bodies (`--primary`, `--background`, …) are *runtime*
variables in `:root`/`.dark` that can change with theme switching.

- `@theme inline` inlines the value into utilities →
  `.bg-primary { background-color: hsl(var(--primary)) }`. Shortest path,
  matches the v3 `tailwind.config` `colors: hsl(var(--*))` inline style, keeps
  the dist render byte-identical.
- bare `@theme` emits an extra `--color-*` indirection layer into `:root` and
  routes utilities through `var(--color-*)` →
  `.bg-primary { background-color: var(--color-primary) }`. Deviates from the
  shadcn official v4 template and risks future light-theme switching.

**Correct**

```css
@theme inline {
  --color-background: hsl(var(--background));
  --color-primary: hsl(var(--primary));
  /* … */
}
```

**Wrong**

```css
@theme {            /* ❌ missing `inline` */
  --color-background: hsl(var(--background));
}
```

**Verify after build** — in the dist CSS:

```bash
# intermediate --color-* layer must be GONE (expect 0)
grep -oE '\-\-color-[a-z0-9-]+:hsl' dist/assets/index-*.css | wc -l
# color utilities must inline hsl(var(--*)), NOT var(--color-*)
grep -oE '\.bg-background\{[^}]*\}' dist/assets/index-*.css
# -> .bg-background{background-color:hsl(var(--background))}
```

---

## Common Mistake: bare `@theme` passes silently in local/dark-only UI

**Symptom**: Colors look correct in the dark-only UI, so the issue is
invisible to the eye; but dist CSS carries an extra `--color-*` layer and a
future light theme may fail to switch.

**Cause**: Copying shadcn tokens into `@theme` without `inline`. The 2026-05-31
v3→v4 upgrade commit even *claimed* "via @theme inline" in its message but
shipped bare `@theme`.

**Fix**: `@theme {` → `@theme inline {` (one word). Verified 2026-06-01: the
`--color-*` layer disappeared, color/radius render values unchanged, docker
rebuild + playwright login confirmed **zero visual regression**.

**Prevention**: Run the grep check above as part of build verification.

---

## Related conventions (confirmed during the v3→v4 migration)

- **Dark mode** is locked to the class strategy via
  `@custom-variant dark (&:is(.dark *));`. This keeps shadcn's unused `dark:`
  variants inert (v3 behavior) and avoids v4's `prefers-color-scheme` default.
  Do not remove it unless a real `.dark` toggle is introduced.
- **Radius scale**: shadcn radii (`rounded-sm/md/lg`) are overridden in
  `@theme` via `--radius-sm/md/lg`
  (`calc(var(--radius) - 4px | 2px)` / `var(--radius)`), so Tailwind v4's
  default radius-scale rename does **not** apply to them. Keep them in sync
  with the `--radius` base (`0.75rem`).
- **No PostCSS**: build integration is `@tailwindcss/vite` (Lightning CSS
  handles prefixing/lowering). There is no `postcss.config.js` and no
  autoprefixer — do not reintroduce a direct `postcss` dependency.
- **Class codemod**: v4 renames are applied (`shadow-sm→shadow-xs`,
  `outline-none→outline-hidden`, `bg-gradient-to-*→bg-linear-to-*`,
  `ring`→`ring-3` where 3px is intended). Keep new code on v4 names.
