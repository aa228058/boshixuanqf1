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

The pull-request CI workflow runs the three local parser benchmark sizes after database migrations and archives the output as `group-import-parser-benchmark`.

## Real PostgreSQL batch-persistence load test

The build CI job applies all migrations to its isolated PostgreSQL 16 service,
then runs the opt-in `importbench` test tag:

```sh
DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=postgres \
DB_NAME=tg_sender_test DB_SSLMODE=disable \
go test -tags=importbench ./internal/domain/grouplib \
  -run '^TestPersistImportFailuresLoad$' -v -count=1
```

This test creates an isolated tenant and import jobs, sends 10k/30k/50k
failure records through the production `persistImportFailures` implementation,
and verifies the exact number of rows persisted in PostgreSQL. Its report
includes measured DB-call duration, rows/second, and Go heap/allocation deltas.
The CI artifact combines this output with the parser benchmark.

This measures the batch failure-write path against real PostgreSQL, not
Telegram link resolution or the entire worker pipeline.
