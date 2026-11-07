# Group import end-to-end performance audit

## Scope and evidence

This audit reads the production import path on `ci/isolated-enterprise-e2e` at commit `249794a86880de68ff562c5cbef123dc5d0e5718` and the successful GitHub Actions run `42418225415`.

Verified CI measurements:

- `filterImportLinks`: 10k = 10.20 ms, 30k = 29.43 ms, 50k = 48.91 ms. This is local parsing/deduplication only.
- PostgreSQL 16 `persistImportFailures`: 10k = 298 ms, 30k = 992 ms, 50k = 1,633 ms; all inserted-row counts were verified.
- All CI jobs passed, including Go tests, frontend checks, PostgreSQL integration, and migrations through version 90.

These figures do **not** measure the complete Telegram import pipeline.

## Production path

1. `CreateImportPreviewWithAccount` filters and normalizes the uploaded text, then stores a review job. No Telegram request is made at preview time.
2. `ConfirmImportJob` moves the job to `pending`.
3. `processImportJobs` claims jobs with `FOR UPDATE SKIP LOCKED`, a two-minute lease, heartbeat, and fencing token.
4. Each worker chunk contains at most 30 links. One scheduler goroutine per Engine instance processes up to ten chunks (300 links) before yielding unfinished work back to `pending`.
5. `importByLinks` resolves entries one at a time in its loop. Background jobs prefer the shared web resolver for public links; if an account is selected, private invites use MTProto and failed public web lookups can fall back to MTProto.
6. Successful groups are persisted with `UpsertBatch`; failed links are persisted through the batch `UNNEST` UPSERT path.
7. The worker persists progress with lease-token and expected-offset checks, supporting restart/resume and cancellation.

## Main throughput constraint

The shared Telegram web scheduler is configured with:

- Maximum in-flight requests per process: 2.
- Minimum request-start interval: 2 seconds.
- Additional random jitter: 0–1 second.

However, the import loop itself is sequential. A single job on a single Engine instance therefore does not use the scheduler's two-request concurrency; it generally starts at most one public-page lookup every 2–3 seconds, before accounting for network latency, retries, or cache hits.

For 50,000 unique cache-miss public links, the scheduler interval alone implies a theoretical lower bound of about 27.8–41.7 hours per process (2–3 seconds between starts). This is a configuration-derived estimate, **not a measured end-to-end duration**. Real elapsed time can be longer due to response latency, retries, rate limits, and fallback work. MTProto lookups are separately rate-limited in PostgreSQL to one request per account per second.

This conservative pacing protects Telegram and should not be relaxed without controlled tests and explicit rate-limit monitoring.

## Confirmed strengths

- Preview is separated from execution; users can review the normalized candidate list first.
- Input filtering deduplicates normalized public usernames and case-sensitive private invite hashes.
- Background jobs use leases, heartbeats, fencing tokens, cancellation checks, and persisted offsets.
- Web requests share a process-level concurrency/rate-limit scheduler and circuit breaker.
- MTProto request pacing is coordinated in PostgreSQL per account, across application replicas.
- Successful group writes and failure writes are batched.
- Positive resolution results are cached for ten minutes, with per-key in-flight deduplication.

## Gaps before claiming end-to-end performance

- No CI benchmark currently measures a complete worker job from payload through resolver and final group upsert.
- No authorized live Telegram public-link run has measured actual response latency, success rate, 429 frequency, or total completion time.
- No production-like multi-job contention test measures queue wait time or fairness.
- No measured memory peak exists for a 50k job held in the database payload and decoded by the worker.
- The current PostgreSQL load test measures failure-row persistence, not successful `system_groups` upserts.

## Next safe validation

1. Add a separately gated live-network smoke/load scenario using only explicitly authorized public test targets; keep it outside ordinary CI and cap its request count.
2. Record per-job elapsed time, links/sec, cache-hit ratio, web/MTProto/fallback counts, 429 wait time, and peak heap.
3. Add a deterministic worker integration test with a local HTTP fixture and isolated PostgreSQL to verify checkpointing, cancellation, retries, and final row counts. Label this as a fixture-based integration test, never as live Telegram performance.
4. Only after those measurements, decide whether to change the global web pacing, worker parallelism, or per-account MTProto cadence.

Do not run high-volume tests against production accounts, production tenants, or the production group library.
