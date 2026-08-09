# Collector durability findings

## Current flow

`CPAClient.PopUsage` originally decoded the whole destructive response directly
into `[]rawQueueItem`. `Collector.pollOnce` then called `toEvent`, skipped rejected
items, and saved normalized `[]model.UsageEvent`. This protected DB failures but
not field-type drift, crashes, parser failures, or rejected items before save.

## Existing safety properties

- `Buffer.Save` uses a same-directory temporary file, file `fsync`, and atomic
  rename.
- Startup recovery replays pending normalized event batches.
- DB insertion uses `ON CONFLICT (request_id, event_ts, total_tokens) DO NOTHING`.
- API key fingerprinting and masking are deterministic pure operations.

## Missing properties

- Raw replay data is not durable before strong field decoding or normalization.
- Directory metadata is not synced after rename.
- `Buffer.Commit` errors are ignored by the collector.
- Items rejected by `toEvent` disappear without a durable diagnostic artifact.
- Buffer format is unversioned.

## Recommended bounded design

Split the CPA response into per-item `json.RawMessage`, remove sensitive fields,
clear the secret-bearing in-memory messages, and write a versioned envelope with
sanitized payload plus precomputed key fingerprint/mask. Strong field decoding
runs only after durable save. Read both the new envelope and legacy
`[]model.UsageEvent` files; write only the new envelope. Use one processing path
for polling and recovery. Preserve at-least-once semantics and rely on the
existing DB key for deduplication.
