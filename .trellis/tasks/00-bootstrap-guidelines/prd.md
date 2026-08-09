# Bootstrap Executable Project Specifications

## Goal

Turn `.trellis/spec/` from mostly empty scaffolding into the project's
normative engineering guardrails. A future implementation or review that loads
the relevant specs should understand the real architecture, preserve data and
business semantics, follow local code patterns, and run the right verification
without relying on conversation history.

## Spec Role

- Specs define current executable constraints: how code must be organized,
  which layer owns a decision, exact cross-layer contracts, forbidden patterns,
  and required verification.
- `.project_context/design/` explains architecture and decision history; specs
  state the current rule and link to design context instead of duplicating it.
- Task PRDs define one change; execution records hold one rollout's evidence.
  Neither temporary progress nor point-in-time production values belong in spec.
- Documentation alone is not a guarantee. Index routing, task context loading,
  automated tests, and Trellis quality checks form the anti-drift loop.

## Scope

### Backend specs

- directory and package ownership
- configuration and composition boundaries
- database queries, migrations, hot-to-daily lineage, and query-time cost
- error propagation and destructive-source failure behavior
- logging and secret exclusion
- code/test quality gates
- preserve and cross-link existing deployment, usage-queue, and cost contracts

### Frontend specs

- directory and component ownership
- component composition and shared primitives
- hooks and auto-refresh behavior
- local/auth/server/URL state boundaries
- Go JSON DTO to TypeScript/default-object alignment
- canonical token-accounting display contract
- frontend quality, null/loading/empty/error states, and production build gate
- preserve existing Tailwind/shadcn styling contract

### Navigation and thinking guides

- make backend/frontend indexes executable routers with change triggers,
  pre-development checklists, and quality checks
- keep guides as short thinking checklists that point to normative specs
- remove repository-irrelevant Trellis template examples from shared guides

## Rules

- Document the current proven architecture, not generic Go/React advice.
- Every important rule needs source, test, config, or project-doc evidence.
- Do not weaken a required safety contract to match a known defect. Mark known
  deviations and link a repair task until implementation catches up.
- Use MUST/MUST NOT for gateable constraints, SHOULD for defaults that may be
  intentionally overridden, and MAY for allowed options.
- Cross-layer/infra contracts keep the seven-section executable format.
  Local conventions use a smaller trigger/pattern/evidence/avoid/verify shape.
- Reference stable file paths and symbol names; avoid line numbers.
- Delete non-applicable headings and all placeholder/template text.
- Do not change application source code in this task.
- Do not create a second architecture source of truth under `.trellis/spec/`.
  Architecture rationale remains in `.project_context/design/`.

## Files

Update or reshape all files under:

```text
.trellis/spec/backend/
.trellis/spec/frontend/
.trellis/spec/guides/
```

Add a dedicated frontend token-accounting display spec because it is a stable
business/cross-layer contract missing from the original scaffold.

## Acceptance Criteria

- [x] Backend index routes change triggers to required specs and contains
      pre-development and quality checklists.
- [x] Frontend index routes change triggers to required specs and contains
      pre-development and quality checklists.
- [x] Backend directory, database, error, logging, and quality specs contain
      project-specific rules with real evidence.
- [x] Frontend directory, component, hook, state, type-safety, quality, and
      token-accounting specs contain project-specific rules with real evidence.
- [x] Existing usage-queue, cost, deployment, and styling contracts remain
      consistent with the new indexes and surrounding specs.
- [x] Shared guides contain project-relevant thinking triggers and no unrelated
      Trellis upstream/template examples.
- [x] No `To be filled`, `To fill`, `TODO: fill`, placeholder instructions, or
      empty template headings remain under `.trellis/spec/`.
- [x] Every relative Markdown link resolves to an existing repository file.
- [x] Every documented verification command is runnable and passes, or its
      external prerequisite is stated precisely.
- [x] `python3 ./.trellis/scripts/get_context.py --mode packages` still discovers
      the backend/frontend layers; the shared guides index exists and remains a
      mandatory read in the Trellis before-development workflow.
- [x] `git diff --check` passes.

## Out of Scope

- Product code, schema, API, frontend behavior, environment variables, or
  deployment changes.
- Rewriting ADRs or execution history.
- Fixing unrelated product defects discovered during source inspection.
- Changing Trellis workflow/runtime implementation.
