# Product Spec: CPA Usage Lens

## Objective

Build a lightweight usage analytics project for CLIProxyAPI deployments that have limited local disk and need account-level usage visibility without storing long-term request detail on the CPA server.

## Target User

The initial user operates CLIProxyAPI on a small server and wants to understand per-account consumption without relying on large local SQLite history.

The user wants to see:

- Request counts per account
- Token usage per account and model
- Estimated cost
- Failure counts and recent failure detail
- Short-term request-level detail for troubleshooting
- Long-term daily usage trends

## Problem

Existing monitoring options rely on SQLite persistence. That is convenient, but it can be uncomfortable on small servers when request-level history grows indefinitely.

Supabase Free has a 500 MB database-size quota, so it is also not suitable for indefinite full detail storage. However, it may be viable if request-level detail is retained only briefly and long-term data is stored as compact aggregate rows.

## Non-Goals

- Do not store raw prompts or model responses.
- Do not store full raw request/response payloads.
- Do not store full raw failure bodies by default.
- Do not replace CLIProxyAPI routing or authentication behavior.
- Do not consume the same CPA usage queue from multiple collectors.
- Do not build a high-scale enterprise observability platform in the first version.

## Initial Requirements

1. The system must consume CLIProxyAPI usage events from one CPA instance.
2. The system must write lean request detail rows to Supabase.
3. The system must retain request-level detail for a configurable short window, initially 7 days.
4. The system must aggregate daily usage by account and model.
5. The system must keep aggregate rows after detail rows are deleted.
6. The system must estimate costs using a model price table.
7. The system must deduplicate retries using request ID, event hash, or another stable event identity.
8. The system must avoid storing secrets, raw tokens, raw prompts, or raw responses.
9. The system must expose enough state to know whether the collector is healthy.
10. The system must document Supabase capacity risks and retention assumptions clearly.

## Data Retention Policy

Default policy:

- Hot request detail: 7 days
- Daily account/model aggregate: long-term
- Collector logs: implementation-dependent, should be bounded

Future versions may support configurable retention windows.

## Suggested Tables

### request_events_hot

Short-lived request-level usage detail.

Required fields:

- `ts`
- `event_date`
- `request_id`
- `event_hash`
- `account_key`
- `account_label`
- `provider`
- `model`
- `requested_model`
- `resolved_model`
- `endpoint`
- `failed`
- `status_code`
- `input_tokens`
- `output_tokens`
- `reasoning_tokens`
- `cache_read_tokens`
- `cache_creation_tokens`
- `total_tokens`
- `latency_ms`
- `cost_usd`
- `created_at`

### daily_account_usage

Long-term aggregate table.

Required fields:

- `day`
- `account_key`
- `account_label`
- `provider`
- `model`
- `endpoint`
- `request_count`
- `failed_count`
- token totals
- `cost_usd`

### model_prices

Model pricing used to estimate cost.

### accounts

Optional display metadata for stable account identifiers.

### collector_state

Collector health, lag, cursor, last rollup time, and last error.

## Design Constraints

- Keep row size small.
- Keep index count minimal.
- Prefer deterministic idempotent rollups.
- Recompute recent aggregate days to handle delayed events.
- Avoid relying on Supabase Free as a long-term full-detail audit database.

## Success Criteria

- At roughly 1,000 CPA requests per day, the system remains comfortably below Supabase Free storage limits when detail retention is 7 days and aggregate rows are compact.
- The user can answer "which account used how many requests/tokens/cost today and over time".
- The user can inspect recent failed or expensive requests within the hot-detail retention window.
- The system can delete old detail rows without losing long-term aggregate metrics.
