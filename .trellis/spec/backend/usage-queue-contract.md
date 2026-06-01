# CPA Usage-Queue Consumption Contract

> Hard constraints for consuming CLIProxyAPI's usage-queue. Read this BEFORE touching anything under `internal/collector/`.

---

## The pop-on-read rule (non-negotiable)

`GET /v0/management/usage-queue` is a **destructive read**: every item returned is
**popped off the queue and can never be fetched again**. There is no cursor, no
acknowledgement step, no replay.

This single fact drives the entire collector design:

- **Never process popped data in memory only.** Any failure after the pop
  (network blip, DB error, panic) = permanent, silent data loss.
- **Mandatory flow**: `pop → persist to disk buffer → write to DB → confirm → delete buffer`.
  The disk buffer (`internal/collector/buffer.go`) is the only thing between a
  transient error and lost data. On startup, replay leftover buffer files
  (`recoverPending`) before polling.
- **Short retention**: items expire from the queue in ~60s by default (max 3600s).
  If the collector is down longer than retention, that window of data is gone —
  source limitation, not a bug.
- **Single collector only.** Two instances polling the same queue split the stream
  and each loses half. Do NOT run the collector with multiple replicas / in parallel.

## Forbidden patterns

- ❌ Calling usage-queue for "health checks" or debugging — it consumes real data.
  Use a non-destructive endpoint (e.g. HEAD on the management page) to test reachability.
- ❌ Popping a batch, then doing expensive transforms before persisting it.
- ❌ Retrying a failed pop assuming the data is still queued — it is not.

## Why this matters

The disk-buffer + startup-recovery machinery looks like over-engineering until you
remember pop is irreversible. Every shortcut here trades a few saved lines for
silent data loss. When reviewing collector changes, the first question is always:
"if this step fails, is any already-popped data unrecoverable?"
