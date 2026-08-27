# BENCH-1 methodology

This is the full write-up behind the numbers in the root `README.md`
`## Performance` block. Those numbers are generated from
`benchmarks/results/latest/` by `make benchmark-report`; this document explains
what was measured and, just as importantly, what the numbers **do not** mean.

> **Read [How not to interpret these results](#how-not-to-interpret-these-results)
> first.** It is not boilerplate — the topology below makes several intuitive
> readings wrong.

## What is measured

One **canonical application** — the same `/api/projects` CRUD API
([schema](../docs/schema.md)) — is implemented six times and measured
identically:

| Stack | Framework | Runtime |
| --- | --- | --- |
| Go control | Gin + GORM | Go |
| Gombit | Gombit (Huma + GORM + Atlas) | Go |
| Python | Django + DRF | CPython |
| Ruby | Rails + ActiveRecord | Ruby |
| PHP | Laravel + Eloquent (nginx + PHP-FPM) | PHP |
| Node | NestJS + TypeORM | Node |

The headline comparison the suite exists for is **Gin+GORM vs Gombit**: same
language, runtime, and ORM family, so the delta isolates Gombit's incremental
cost. The four ecosystem apps are *context*, not a language leaderboard (see
below).

Two families of measurement:

1. **Throughput / latency** — `GET /api/projects?page=1&limit=20` under a
   closed-loop `constant-vus` load (concurrency sweep 1/10/100/500/1000), 5
   trials × 30 s each with a discarded warm-up. `summary.md` carries every
   concurrency level; the README's headline table reports a **single
   concurrency (100 clients)** with median requests/sec **and** median p50/p95/p99
   latency, and a ⚠ flag on any group whose trials disagree by more than 5 %
   (coefficient of variation). `make benchmark-crud-all`.
   (The abstraction-cost microbenchmark — net/http → Gin → Huma → Gombit,
   `go test -bench=BenchmarkFrameworkTax` — is the first README table; persisting
   its rows into the generated report is a follow-up, so that section is a
   placeholder for now.)
2. **Operational footprint** — container-start → first `/livez` 200 cold start
   (≥20 restarts, median/p95), idle memory, memory + CPU under the same load,
   and image size. `make benchmark-footprint`.

## Fairness controls (issue #141 §7/§16/§18)

- **Identical schema and seed.** All six use the same `users`/`projects` tables
  and the same deterministic seed (1,000 users, 100,000 projects, round-robin
  ownership), so every app answers the same query against the same rows.
- **Fixed resource limits.** Each app container is pinned to **2 vCPU / 1 GiB**
  and PostgreSQL to **2 vCPU / 2 GiB** (compose `deploy.resources.limits`).
  Because a compose file only *requests* limits, the suite reads the *applied*
  cgroup limit off the live container (`inspect-limits`) and records that honest
  classification (`enforced` / `partial` / `not applied`) in
  `metadata.resource_limits` — it never assumes the ceiling held.
- **Connection pooling** capped at 20 total per app (pre-fork servers divide it
  across workers so the *total* matches).
- **Production configuration.** gunicorn / Puma (clustered) / nginx+PHP-FPM /
  compiled Node — never a framework dev server. Per-request access logging is
  off.
- **Pinned versions.** Every language, framework, database, and the load
  generator are pinned exactly (`benchmarks/config/versions.env`, each app's
  manifest); the recorded framework/runtime versions are derived from those
  sources, not hand-copied.

## Topology

The load generator (k6) runs in a container **on the same host** as the app —
the issue's "another container on the same host". It is not a separate
load-generation machine. At high concurrency k6 therefore contends with the app
for CPU; that is the recorded topology, not an error.

The load is **closed-loop** (`constant-vus`: N clients, each sending the next
request only after the previous response). This matches the issue's
concurrency-as-client-count axis, but it means the numbers are subject to
**coordinated omission**: when the app slows, a client simply waits before its
next request, so reported tail latency *understates* true client-observed wait.

## Reproducing

```sh
make benchmark-crud-all      # throughput -> results.json
make benchmark-footprint     # footprint  -> footprint.json
make benchmark-summary       # results.json -> summary.md
make benchmark-report        # README ## Performance block (+ summary.md)
```

The `metadata.json` beside the results records the host (CPU, cores, RAM),
commit + dirty state, OS/kernel/arch, pinned versions, and the applied resource
limits, so a published table always carries the conditions it was produced
under.

## How not to interpret these results

- **This is not a language or framework leaderboard.** The apps differ in
  language, runtime, ORM, and process model all at once. "Framework X does N
  rps" tells you about *this app, this query, this host, this day* — nothing
  about X in general. The only apples-to-apples comparison here is Gin+GORM vs
  Gombit (same everything but the framework); treat the rest as ecosystem
  context.
- **Tail latency is optimistic.** Closed-loop load hides queueing via
  coordinated omission (above). Do not read p95/p99 here as a service-level
  objective.
- **Same-host contention is included, not controlled out.** At 500/1000 VUs the
  load generator competes with the app for the same cores. High-concurrency
  rows measure "app + k6 sharing 2 vCPU", which is deliberately conservative but
  not the same as "app alone on 2 vCPU".
- **It is one machine, one run.** A single snapshot on one host. The committed
  `metadata.json` names that host; numbers from different hardware are not
  comparable. Re-run on your own hardware before drawing operational
  conclusions.
- **Memory is the container working set**, from `docker stats` (cgroup usage
  minus reclaimable cache) — a deployment-footprint proxy for RSS, not a precise
  heap measurement.
- **Cold start is container-start → ready**, not process fork time or
  serverless cold start; it excludes image pull (the image is pre-pulled).
- **Higher rps is not "better" in isolation.** Read it next to the footprint
  table: a stack's throughput, its memory, and its cold start are the same
  trade-off surface.

If a number here is going to influence a decision, reproduce it on your target
hardware with your workload first.
