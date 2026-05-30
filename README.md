# CPA Usage Lens

Early workspace for designing a lightweight CLIProxyAPI usage collector and analytics layer.

The initial storage direction is Supabase, but the project name intentionally stays storage-agnostic. The core design is to keep only short-lived request-level details, persist long-term daily aggregates, and avoid storing large raw request/response payloads.

## Local Preview

`local-preview/` contains an isolated Docker Compose stack for comparing existing CPA usage dashboards without touching a real CPA instance.

```bash
cd local-preview
docker compose up -d
docker compose down
```
