# Group import end-to-end performance audit

## Scope and evidence

This audit reads the production import path on `ci/isolated-enterprise-e2e` at commit `249794a86880de68ff562c5cbef123dc5d0e5718` and the successful GitHub Actions run `42418225415`.

Verified CI measurements:

- `filterImportLinks`: 10k = 10.20 ms, 30k = 29.43 ms, 50k = 48.91 ms. This is local parsing/deduplication only.
- PostgreSQL 16 `persistImportFailures`: 10k = 298 ms, 30k = 992 ms, 50k = 1,633 ms; all inserted-row counts were verified.
- All jobs in that run passed, including Go tests, frontend checks, PostgreSQL integration, and migrations through version 90.

These figures do **not** measure the complete Telegram import pipeline.

## Production path

1. `CreateImportPreviewWithAccount` filters and normalizes uploaded text, then stores a review job. No Telegram request is made at preview time.
2. `ConfirmImportJob` moves the job to `pending`.
3. `processImportJobs` claims one job at a time per Engine invocation with `FOR UPDATE SKIP LOCKED`, a two-minute lease, heartbeat, and fencing token.
4. Each chunk contains at most 30 links. The scheduler processes up to ten chunks (300 links) for that job before yielding unfinished work back to `pending`.
5. `importByLinks` resolves entries sequentially in its loop. Background jobs prefer the shared web resolver for public links; if an account is selected, private invites use MTProto and failed public web lookups can fall back to MTProto.
6. Successful groups are persisted with `UpsertBatch`; failed links are persisted through the batch `UNNEST` UPSERT path.
7. The worker persists progress with lease-token and expected-offset checks, supporting restart/resume and cancellation.

## Main throughput constraint

The shared Telegram web scheduler is configured with:

- Maximum in-flight requests per process: 2.
- Minimum request-start interval: 2 seconds.
- Additional random jitter: 0–1 second.

But the current job execution path does not create parallel resolver calls: `processImportJobs` handles jobs serially within an invocation, and `importByLinks` handles links serially within each chunk. The scheduler's concurrency limit of two is a safety ceiling, not evidence that a single job uses two concurrent requests. Concurrent requests can only arise from other callers/Engine instances in the same process.

For one 50,000-link job with unique public links and no cache hits, the configured 2–3 second start spacing implies roughly 27.8–41.7 hours just for request starts. This is a configuration-derived estimate, **not a measured end-to-end duration**. Actual time may be longer due to network latency, retries, rate limits, and MTProto fallback. MTProto requests are separately paced in PostgreSQL to one request per account per second, including across application replicas.

Do not simply increase concurrency or reduce pacing: Telegram 429/FLOOD_WAIT responses and account restrictions make uncontrolled parallelism unsafe.

## Confirmed strengths

- Preview is separated from execution; users can review normalized candidates first.
- Input filtering deduplicates normalized public usernames and case-sensitive private invite hashes.
- Background jobs use leases, heartbeats, fencing tokens, cancellation checks, and persisted offsets.
- Web requests share a process-level concurrency/rate-limit scheduler and circuit breaker.
- MTProto request pacing is coordinated in PostgreSQL per account, across application replicas.
- Successful group writes and failure writes are batched.
- Positive resolution results are cached for ten minutes, with per-key in-flight deduplication.

## Gaps before claiming end-to-end performance

- No CI benchmark measures a complete worker job from payload through resolver and final group upsert.
- No authorized live Telegram public-link run has measured actual response latency, success rate, 429 frequency, or total completion time.
- No multi-job contention test measures queue wait time or fairness.
- No measured memory peak exists for a 50k job held in the database payload and decoded by the worker.
- The PostgreSQL load test measures failure-row persistence, not successful `system_groups` upserts.

## Safe validation and implementation order

1. Refactor the worker's external effects behind narrow injectable interfaces (resolver and group persistence) without changing production defaults.
2. Add deterministic worker tests using a local HTTP fixture and isolated PostgreSQL. Verify 30-link chunk boundaries, checkpoint offsets, cancellation, lease loss/fencing, retry behavior, failure persistence, and final group/failure counts.
3. Add metrics for per-job elapsed time, links/sec, cache-hit ratio, web/MTProto/fallback counts, 429 wait time, and peak heap.
4. Only after fixture tests pass, run a separately gated live-network smoke test using a small set of explicitly authorized public test targets. Report public web and MTProto scenarios separately.
5. Use those measurements to decide whether safe bounded parallelism is warranted. Preserve global Telegram pacing and per-account MTProto limits; do not assume a concurrency increase improves throughput.

Never run high-volume tests against production accounts, production tenants, or the production group library.
