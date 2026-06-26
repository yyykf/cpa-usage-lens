# Add Webpage Logo

## Goal

Add a recognizable CPA Usage Lens brand mark so the app no longer shows the browser's generic favicon in the tab and the in-app brand mark stays consistent.

## Requirements

* Provide a small SVG logo that matches the existing carbon instrument console visual language.
* Use the same visual idea for the browser tab favicon and the in-app brand mark.
* Keep the implementation lightweight: no new runtime dependency, no bitmap-only asset pipeline.
* Preserve existing dashboard and login layout behavior across desktop and mobile.

## Acceptance Criteria

* [ ] `frontend/index.html` declares a favicon for the app.
* [ ] The browser tab can render a CPA Usage Lens-specific icon instead of the generic globe icon.
* [ ] Dashboard header and login page use a shared logo component instead of duplicated inline SVG.
* [ ] `npm run build` passes for the frontend.

## Definition of Done

* Frontend build passes.
* Visual change is scoped to brand/logo surfaces.
* No backend behavior changes.
* No new dependency added.

## Technical Approach

Create `frontend/public/logo.svg` as a standalone favicon-safe SVG and add a `<link rel="icon">` in `frontend/index.html`. Extract the current inline mark into a reusable `BrandLogo` React component so dashboard and login share the same logo semantics and future logo tweaks only need one component change.

## Decision (ADR-lite)

**Context**: The app already has a small inline trend-line mark in the dashboard and login page, but the browser tab still shows the default generic icon.

**Decision**: Use a simple custom SVG mark based on the existing trend-line/data-lens visual metaphor, and wire it as both favicon and reusable in-app logo component.

**Consequences**: This keeps the change cheap and consistent. A future formal brand refresh can replace `logo.svg` and `BrandLogo` without touching dashboard logic or data components.

## Out of Scope

* Full visual redesign.
* App icon generation for every platform size.
* Changing page title text.
* Backend/API changes.

## Technical Notes

* Inspected `frontend/index.html`: title exists, favicon link missing.
* Inspected `frontend/src/pages/Dashboard.tsx` and `frontend/src/pages/Login.tsx`: both duplicate the same inline SVG brand mark.
* Relevant guidelines: `.trellis/spec/frontend/styling-guidelines.md` and `.project_context/explore/frontend/design-system.md`.
