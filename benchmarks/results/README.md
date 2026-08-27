# benchmarks/results

Generated benchmark output (issue #141 §9). **Nothing here is hand-edited** —
Markdown is never the canonical data source; it is generated from the
structured results.

A full run writes a `latest/` snapshot:

```text
results/
└── latest/
    ├── metadata.json   reproducibility metadata (benchmarks/internal/metadata)
    ├── raw/            per-trial raw load-generator output (k6 JSON, ...)
    ├── results.json    the canonical structured throughput results (benchmarks/internal/result)
    ├── results.csv     flat CSV derived from results.json
    ├── summary.md      human-readable tables derived from results.json
    ├── footprint.json  operational-footprint rows (benchmarks/internal/footprint)
    ├── footprint.csv   flat CSV derived from footprint.json
    └── microbench.json framework-tax ns/op·B/op·allocs/op (benchmarks/internal/microbench)
```

`results.json` / `results.csv` are the machine-readable schema defined in
[`benchmarks/internal/result`](../internal/result); `metadata.json` is the
[`benchmarks/internal/metadata`](../internal/metadata) shape, collected by
[`benchmarks/scripts/collect-host-info`](../scripts/collect-host-info).

`latest/` is committed once the full suite is run on a dedicated machine (Phase 7
of [docs/plans/BENCH-1-benchmark-suite.md](../../docs/plans/BENCH-1-benchmark-suite.md)),
so the published README performance tables are always regenerable from a
committed snapshot. The orchestration that fills `latest/` exists —
`make benchmark-crud-all` runs the CRUD-read workload across all six apps and
`make benchmark-summary` renders `summary.md`; committing a canonical snapshot is
the Phase 7 reporting step.
