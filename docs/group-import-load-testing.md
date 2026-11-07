# Group import load checks

## Local deterministic parser benchmark

Run from the repository root:

```sh
go test ./internal/domain/grouplib -run '^$' -bench '^BenchmarkFilterImportLinks$' -benchmem -count=3
```

The benchmark exercises the production `filterImportLinks` parser and
normalizer with deterministic 10k, 30k, and 50k line corpora. Each corpus
contains 50% unique valid public links, 25% duplicates, and 25% invalid lines.
It reports wall time, throughput, allocations, and bytes processed.

**Scope:** this benchmark measures local parsing/deduplication only. It does
not resolve links over the network, exercise Telegram rate limits, write to
PostgreSQL, or represent end-to-end import throughput. Do not compare its
numbers with live Telegram import performance.

## Full import load test requirements

A real end-to-end run must use an isolated PostgreSQL database and explicit
authorized Telegram test fixtures. Public-link network resolution and
MTProto invite resolution must be reported as separate scenarios. Never run
high-volume tests against production accounts or production group libraries.
