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
    ├── results.json    the canonical structured results (benchmarks/internal/result)
    ├── results.csv     flat CSV derived from results.json
    └── summary.md      human-readable tables derived from results.json
```

`results.json` / `results.csv` are the machine-readable schema defined in
[`benchmarks/internal/result`](../internal/result); `metadata.json` is the
[`benchmarks/internal/metadata`](../internal/metadata) shape, collected by
[`benchmarks/scripts/collect-host-info`](../scripts/collect-host-info).

`latest/` is committed once the run pipeline exists (Phase 7 of
[docs/plans/BENCH-1-benchmark-suite.md](../../docs/plans/BENCH-1-benchmark-suite.md)),
so the published README performance tables are always regenerable from a
committed snapshot. It does not exist yet — the schema, collector, and config
pins land first (Phase 1 completion); the orchestration that fills `latest/`
is the next slice.
