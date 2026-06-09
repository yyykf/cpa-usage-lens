# Improve deployment docs and README visuals

## Goal

Make CPA Usage Lens easier to deploy and evaluate from the public repository. The work should reduce copy/paste friction for users who do not want to build from source, make production defaults safer, add post-deploy verification guidance, and improve the README's product presentation with a real screenshot plus a generated introduction image.

## Requirements

- Update README and deployment examples so users are instructed to use the latest release tag instead of a hard-coded stale version such as `v0.1.0`.
- Add a no-source-checkout deployment path with direct commands to create a deploy directory, download `docker-compose.prod.yml` and `.env.example`, copy `.env`, edit values, and start the stack.
- Add deployment verification and troubleshooting guidance covering container status, backend health, frontend access, backend logs, collector health, and common misconfigurations.
- Make the production Compose default safer by not exposing the backend debug port by default.
- Add a debug override Compose file for users who intentionally want to expose the backend port during troubleshooting.
- Add README product visuals:
  - Use the project dashboard screenshot (`docs/assets/dashboard-screenshot.jpg`) as the real, redacted product screenshot if inspection shows it is suitable.
  - Generate one product introduction image without real account data and store it in the repository.
  - Clearly label the generated product introduction image as a concept / visual direction rather than an exact current UI screenshot.
  - Reference both visuals from the README where they help users understand the product quickly.
- Keep release assets unchanged for now; do not modify the GitHub Release workflow to upload compose/env artifacts in this task.
- Work from a branch created from `main`.

## Acceptance Criteria

- [x] `README.md` and `README.zh-CN.md` no longer tell users to deploy a specific old tag as the recommended command.
- [x] `docs/deployment.md` includes a copy/paste-friendly no-source-checkout deployment path.
- [x] `docs/deployment.md` includes post-deploy checks and troubleshooting guidance.
- [x] `docker-compose.prod.yml` does not publish backend `8080` by default.
- [x] A debug override file exists for intentionally publishing backend `8080`.
- [x] README references a real redacted screenshot and one generated product introduction image stored under the repository.
- [x] README labels the generated product introduction image as concept / visual direction to avoid implying exact current UI parity.
- [x] Docs explain why the backend port is not exposed by default and how to opt into it for debugging.
- [x] Quality checks appropriate for docs/config changes pass.

## Definition of Done

- Docs/config changes are committed on a branch based on `main`.
- Any generated or copied assets are stored in the repository and referenced with relative paths.
- No secrets from local `.env` are written into tracked files.
- `docker compose -f docker-compose.prod.yml config` and the debug override equivalent are valid.
- A concise execution summary is written under `.project_context/execution/docs/`.

## Technical Approach

- Treat pre-built images as the recommended deployment path because it avoids requiring Go, Node, npm, and local multi-stage Docker builds on the user's server.
- Keep source-build deployment documented as a secondary option for contributors or users who intentionally want local builds.
- Use a separate debug override Compose file instead of keeping the backend port exposed by default. This preserves easy troubleshooting while reducing the default network surface.
- Store README assets under `docs/assets/` so public documentation references stable repository-relative paths.

## Decision (ADR-lite)

**Context**: The current deployment path is functional, but new users still need to infer how to deploy without cloning the source and how to verify whether the collector is working. The production Compose also exposes the backend debug port by default.

**Decision**: Make the GHCR image path the primary path, add direct download commands for the two required files, add smoke tests/troubleshooting, move backend `8080` exposure into an opt-in debug override, and add product visuals.

**Consequences**: First-time deployment becomes easier and safer. Debugging requires an extra override file when backend direct access is needed, but the default production path is cleaner.

## Out of Scope

- Do not change the release workflow to upload release assets yet.
- Do not change backend or frontend runtime behavior.
- Do not rotate or modify local credentials.
- Do not add a new landing page or product website.

## Technical Notes

- Branch: `codex/docs-deployment-friendly-assets`, created from `main`.
- Current latest release verified during planning: `v0.1.1`.
- Relevant files: `README.md`, `README.zh-CN.md`, `docs/deployment.md`, `docker-compose.prod.yml`, `.env.example`, `frontend/nginx.conf`.
- Project dashboard screenshot: `docs/assets/dashboard-screenshot.jpg`, `2370x2294` JPEG.
