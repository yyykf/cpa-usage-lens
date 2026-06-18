# CPA Usage Lens

[![Release](https://img.shields.io/github/v/release/yyykf/cpa-usage-lens?style=flat-square&color=0d1117)](https://github.com/yyykf/cpa-usage-lens/releases)
[![CI](https://img.shields.io/github/actions/workflow/status/yyykf/cpa-usage-lens/ci.yml?branch=main&style=flat-square&label=CI)](https://github.com/yyykf/cpa-usage-lens/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-MIT-0d1117?style=flat-square)](./LICENSE)

[![Go](https://img.shields.io/badge/Go-0d1117?style=flat-square&logo=go&logoColor=00ADD8)](https://go.dev)
[![React](https://img.shields.io/badge/React%2018-0d1117?style=flat-square&logo=react&logoColor=61DAFB)](https://react.dev)
[![TypeScript](https://img.shields.io/badge/TypeScript-0d1117?style=flat-square&logo=typescript&logoColor=3178C6)](https://www.typescriptlang.org)
[![Vite](https://img.shields.io/badge/Vite-0d1117?style=flat-square&logo=vite&logoColor=646CFF)](https://vite.dev)
[![Tailwind CSS](https://img.shields.io/badge/Tailwind%20CSS%204-0d1117?style=flat-square&logo=tailwindcss&logoColor=38BDF8)](https://tailwindcss.com)
[![Supabase](https://img.shields.io/badge/Supabase-0d1117?style=flat-square&logo=supabase&logoColor=3FCF8E)](https://supabase.com)
[![Docker](https://img.shields.io/badge/Docker-0d1117?style=flat-square&logo=docker&logoColor=2496ED)](https://www.docker.com)

**English** · [简体中文](./README.zh-CN.md)

Account-level usage analytics for self-hosted **CLIProxyAPI (CPA)** users, with a **near-zero local footprint**. An external collector drains CPA's usage queue, writes slimmed-down data to **Supabase (cloud Postgres)**, and serves a **polished dark dashboard** showing per-account request counts, token usage, and estimated cost over any period.

> **What's different:** the data lives in Supabase cloud and the local footprint is near-zero — most similar tools keep everything in a local SQLite file.

![CPA Usage Lens product overview](docs/assets/product-intro.png)

> Concept preview: this generated image shows the intended visual direction, not an exact screenshot of the current UI. The redacted screenshot below is the current product.

## Product Preview

![CPA Usage Lens real dashboard screenshot with account details redacted](docs/assets/dashboard-screenshot.jpg)

## Features

- 📊 **Dark Bento dashboard** — period overview · per-account leaderboard · per-API-key leaderboard (masked) · daily trend · collector health
- ☁️ **Cloud-backed** — data sits in Supabase; locally you run only two lightweight containers (backend + frontend)
- 💰 **Query-time cost estimation** via the LiteLLM price table — stores only the models you actually used, marks missing prices as *unknown*, and reflects price changes automatically (no backfill)
- 🔒 **Single-user auth** — password login (bcrypt + JWT); every data API is authenticated
- 🛡️ **Loss-resistant ingestion** — popped-but-not-yet-persisted batches are buffered to disk and deleted only after a confirmed write; the collector auto-recovers on restart
- ♻️ **Bounded storage** — short-term hot detail (default 7 days, configurable) plus long-term daily rollups; always rolls up *before* cleaning up, so nothing is deleted prematurely
- 🔑 **Sensitive fields stripped** before persistence — the **plaintext `api_key` is never written** (only an irreversible `sha256` fingerprint + a `sk-…last4` mask are kept, to break usage down per key); `response_headers` and `fail.body` are dropped entirely

## Architecture

```
CPA  GET /usage-queue  --poll & pop-->  Collector (strip secrets / dedup by (request_id, event_ts, total_tokens) / disk buffer)
                                           │
                                           ▼
   request_events_hot (hot detail, kept N days) --rollup--> daily_account_usage (account + model + day, long-term)
                                                                      │
   backend  (single Go process) = collector loop + rollup/cleanup + price refresh + HTTP API + auth
   frontend (React)             = nginx static hosting + reverse proxy for /api
   database (Supabase cloud)    = not part of docker compose
```

## Quick Start

Full guide: **[docs/deployment.md](docs/deployment.md)**. In three steps:

1. **Create the tables** in Supabase (`supabase db push`, or run `supabase/migrations/` in the SQL Editor)
2. **Copy `.env.example` to `.env`** and fill it in (CPA URL/key, Supabase connection string, dashboard password)
3. **Use the latest release tag** and start the pre-built images → open `http://<server>:8088`

```bash
export CUL_VERSION=<latest-release-tag>
docker compose -f docker-compose.prod.yml up -d
```

Use the latest tag from [GitHub Releases](https://github.com/yyykf/cpa-usage-lens/releases). See [deployment](docs/deployment.md) for the no-source-checkout path and the optional backend debug override.

> ⚠️ CPA must have `usage-statistics-enabled: true`, and **only one** collector may run against a given CPA queue. See [Important constraints](#important-constraints).

## Tech Stack

| Layer | Technology |
|-------|------------|
| **Backend** | Go (`pgx` direct to Supabase Postgres, stdlib `net/http`, `bcrypt`, `golang-jwt`) |
| **Frontend** | React 18 + Vite + TypeScript + Tailwind CSS + Recharts + lucide-react |
| **Database** | Supabase (Postgres) |
| **Deployment** | Docker Compose (backend + frontend) |

## Project Structure

```
backend/    Go backend: cmd/server + internal/{config,db,model,collector,rollup,pricing,api,timeutil}
frontend/   React frontend: src/{components,pages,lib}
supabase/   migrations (table-creation SQL)
docs/       deployment & operations guide
```

## Important Constraints

> Read this before deploying — it affects data integrity.

- **One collector per CPA queue.** The queue is pop-to-delete; multiple instances would steal records from each other.
- **Pop is not replayable.** Requests produced while the collector is down *longer than* CPA's `redis-usage-queue-retention-seconds` are **lost permanently** — CPA's queue is in-memory only and is cleared on expiry.
- **CPA must enable the queue.** Set `usage-statistics-enabled: true` (despite the stale official comment that describes it as an in-memory-aggregation switch).

Full operational detail — the read-only-instance toggle (`COLLECTOR_ENABLED`), capacity assumptions, and recovery behavior — is in **[docs/deployment.md](docs/deployment.md)**.

## Friendly Links

- [LINUX DO - 新的理想型社区](https://linux.do/)

## License

[MIT](./LICENSE) © 2026 KaiFan Yu
