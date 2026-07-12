# CLIProxyAPI usage contract compatibility

## Versions inspected

Repository: <https://github.com/router-for-me/CLIProxyAPI>

- `v7.2.55`: no OpenAI `cache_write_tokens` extraction.
- `v7.2.56`: began mapping OpenAI `input_tokens_details.cache_write_tokens`
  into the existing queue field `cache_creation_tokens`.
- `v7.2.67`: expanded alias normalization and, for OpenAI-style usage, copies
  the same cache-read count into both `cached_tokens` and
  `cache_read_tokens`.

Release references:

- <https://github.com/router-for-me/CLIProxyAPI/releases/tag/v7.2.56>
- <https://github.com/router-for-me/CLIProxyAPI/releases/tag/v7.2.67>
- <https://github.com/router-for-me/CLIProxyAPI/compare/v7.2.66...v7.2.67>

## v7.2.67 Codex OAuth shape

The upstream parser has a test with this source usage:

```json
{
  "input_tokens": 100,
  "output_tokens": 20,
  "total_tokens": 120,
  "input_tokens_details": {
    "cached_tokens": 30,
    "cache_write_tokens": 40
  }
}
```

Its queue token record becomes:

```text
input_tokens=100
output_tokens=20
cached_tokens=30
cache_read_tokens=30
cache_creation_tokens=40
total_tokens=120
```

Consequences for Lens:

- `cached_tokens` and `cache_read_tokens` are aliases for the same 30 tokens in
  this OpenAI/Codex shape and must not be added together.
- `cache_write_tokens` needs no new Lens queue or database column; CPA already
  normalizes it into `cache_creation_tokens`.
- Both cache read and cache write can be non-zero in one request.
- OpenAI `input_tokens` includes both cache categories, so the baseline-price
  input is `100 - 30 - 40 = 30` in the example.

## Older CPA compatibility

The queue JSON fields are optional from Lens's point of view: a missing field
decodes to zero. The compatibility behavior should therefore be:

- CPA before `v7.2.56`: cache-write usage is unavailable and remains zero;
  Lens must continue ingesting the record without error.
- CPA `v7.2.56` through `v7.2.66`: cache write is available as
  `cache_creation_tokens`, while OpenAI cache read may appear only as
  `cached_tokens`; Lens must recognize it once.
- CPA `v7.2.67` and later: cache read may appear in both alias fields; Lens must
  canonicalize it once, and use `cache_creation_tokens` for writes.

This task does not attempt to reconstruct cache writes that an old CPA never
reported.

## Cross-provider constraint

CPA's Claude parser reports input tokens separately from cache-read and
cache-creation tokens, whereas OpenAI's input total includes cache reads and
writes. The pricing fix must preserve this distinction. A provider-agnostic
rule that always subtracts cache tokens from `input_tokens` would undercount
Claude.
