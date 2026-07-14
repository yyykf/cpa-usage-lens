# Simplify Codex Usage Pricing Changes

## Goal

Review the Codex cache and long-context pricing implementation introduced by
commit `d87dc46` for reuse, maintainability, and efficiency, then apply only
small local cleanups that preserve the committed behavior exactly.

## Requirements

- Review the complete business-code diff `e962ed8...d87dc46` with three
  independent perspectives: code reuse, code quality, and code efficiency.
- Require actionable findings to identify exact files and lines and explain why
  the proposed cleanup preserves behavior.
- Deduplicate overlapping findings and fix only concrete issues whose value
  exceeds their added complexity.
- Preserve all existing provider-aware token semantics, long-context pricing
  semantics, migration behavior, report DTO contracts, and frontend display
  behavior.
- Keep unrelated historical Trellis tasks unchanged.

## Acceptance Criteria

- [x] All three simplify reviewers complete their full-diff review.
- [x] Every accepted finding is fixed with a small, local change.
- [x] Rejected findings are recorded with a concise reason.
- [x] Backend formatting, build, vet, and race-enabled tests pass.
- [x] Frontend production build passes when frontend or DTO code is touched.
- [x] The PostgreSQL long-context integration test passes when database,
      migration, or rollup code is touched.
- [x] The final diff introduces no pricing, API, schema, or UI behavior change.

## Definition of Done

- The simplify review is complete and findings are resolved or explicitly
  rejected.
- Relevant quality gates pass.
- Task artifacts and session history are updated.
- Any cleanup changes are committed with a Conventional Commit and `[#AI]`
  trailer.

## Technical Approach

Use `git diff e962ed8...d87dc46` as the immutable review boundary. Run three
read-only reviewers in parallel against that same diff. Aggregate findings in
the main session, inspect the cited code and tests, and apply the narrowest
behavior-preserving patch. Re-run validation in proportion to the touched
layers.

## Decision (ADR-lite)

**Context**: The feature is already implemented, tested, and committed. This is
a cleanup pass requested through the Spellbook `simplify` workflow.

**Decision**: Freeze product behavior and accept only local reuse, quality, or
efficiency improvements supported by repository evidence.

**Consequences**: Suggestions that redesign the schema, alter pricing rules,
expand provider support, or introduce speculative abstractions are out of scope,
even if they could be reasonable future work.

## Expansion Sweep

- Future evolution: preserve the existing per-model LiteLLM metadata boundary;
  do not add new tier types or provider contracts in this cleanup.
- Related scenarios: verify the full source-to-report-to-frontend token lineage,
  but do not change its public shape.
- Failure and edge cases: retain old-CPA zero-value compatibility, alias
  canonicalization, threshold-boundary behavior, and retained-window rollup
  rebuilding.

## Out of Scope

- New billing features or providers.
- Changes to cache, reasoning, service-tier, or long-context pricing semantics.
- Schema redesign or zero-downtime compatibility changes.
- Recalculation of historical data outside the existing retained-window flow.
- Push, pull request creation, or release publication.

## Technical Notes

- Current branch: `codex/fix-codex-usage-pricing`.
- Base commit: `e962ed8` (`origin/main`, tag `v0.4.1`).
- Business implementation commit: `d87dc46`.
- The later commit `44edd02` only archives the original Trellis task and is not
  part of the business-code review boundary.
- Primary contract: `.trellis/spec/backend/cost-calculation.md`.
- Shared review guides: `.trellis/spec/guides/code-reuse-thinking-guide.md` and
  `.trellis/spec/guides/cross-layer-thinking-guide.md`.

## Simplify Review Decisions

Accepted:

- Extract one backend `TokenBreakdown` DTO, anonymously embed it in the three
  report DTOs, and centralize aggregate-to-DTO conversion while preserving the
  current flat JSON response.
- Add canonical account/key lineage assertions and a flat-JSON regression test.
- Fill and assert the complete base/high `ModelPrice` PostgreSQL round trip.
- Express cache-read alias canonicalization with the built-in `max` formula.
- Reuse one LiteLLM long-context field suffix for all four tier prices.
- Accumulate account/key cost during the grouping pass and remove retained
  `DailyUsage` copies and the second group scan, preserving unknown-cost rules.

Rejected for this cleanup:

- Replacing the atomic rollup delete/rebuild with staging/upsert/stale-delete.
  The write-amplification concern is credible, but the change adds complexity
  to the most correctness-sensitive SQL without workload measurements proving
  it is currently a bottleneck. Keep the already integration-tested atomic
  rebuild and treat this as a separate measured optimization if needed.
- Generic account/key grouping frameworks, pricing strategy layers, custom
  LiteLLM unmarshalling, and cross-migration SQL helpers. Their added
  abstraction outweighs the local duplication in the current two-tier MVP.

## Verification Results

- Three parallel Spellbook reviewers completed reuse, quality, and efficiency
  review of the full `e962ed8...d87dc46` business diff; none found a blocker.
- An independent Trellis check found and fixed two remaining local issues:
  duplicate grouped-cost accumulation and incomplete flat/zero-value JSON
  characterization coverage.
- Passed `gofmt -l .`, `git diff --check HEAD`, `go build ./...`,
  `go vet ./...`, and `go test ./... -race`.
- Passed `npm run build`; the existing Vite bundle-size warning remains
  non-blocking.
- Passed `TestLongContextMigrationAndRollup` against a disposable PostgreSQL 17
  container, which was removed after the test.
- No code-spec update is needed: the cleanup adds no new billing, API, schema,
  deployment, or cross-layer contract.
