# Add CLIProxyAPI links to README

## Goal

Make the README relationship to CLIProxyAPI explicit by linking the first CPA mention and adding CLIProxyAPI to the friendly links section. This helps readers identify the upstream ecosystem project without adding a larger marketing section.

## Requirements

- Link the first `CLIProxyAPI (CPA)` mention in the English README to the official CLIProxyAPI repository.
- Apply the same treatment in the Chinese README.
- Add CLIProxyAPI to the existing friendly links section in both READMEs.
- Keep the change documentation-only.

## Acceptance Criteria

- [ ] `README.md` contains a Markdown link to `https://github.com/router-for-me/CLIProxyAPI`.
- [ ] `README.zh-CN.md` contains a Markdown link to `https://github.com/router-for-me/CLIProxyAPI`.
- [ ] The wording remains concise and consistent with the existing README tone.
- [ ] `git diff --check` passes.

## Definition of Done

- Documentation updated.
- Formatting checked.
- No backend, frontend, deployment, or migration behavior changes.

## Technical Approach

Use the existing opening paragraph for the primary contextual link because it explains the product's CPA dependency at the point where readers first need it. Use the existing friendly links section for ecosystem visibility without introducing a new section.

## Decision (ADR-lite)

**Context**: The README already describes CPA as the source system but did not link to the upstream repository.

**Decision**: Add Markdown links in the first mention and in the friendly links section instead of creating a new promotional block.

**Consequences**: The README stays compact, readers can reach the upstream project directly, and future maintenance is limited to a stable GitHub repository URL.

## Out of Scope

- Changing deployment instructions.
- Adding badges or screenshots.
- Adding links to third-party usage dashboard projects.
- Changing CLIProxyAPI upstream documentation.

## Technical Notes

- Inspected `README.md` and `README.zh-CN.md`.
- Existing friendly links section was introduced by `0c82315`.
- Repository context was refreshed with `git fetch origin`; local `main` matched `origin/main`.
