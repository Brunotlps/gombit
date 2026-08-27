# Benchmark summary

Generated from `results.json` (`benchmarks/internal/summary`) — do not edit by hand.

**Read with:** the load model is closed-loop `constant-vus` (N concurrent clients), so reported tail latency is subject to **coordinated omission** and understates true client-observed wait; the load generator shares the host with the app. Headline `rps`/`p50`/`p95`/`p99` are the **median** across trials; `rps CoV` is the coefficient of variation of throughput (stddev/mean) across trials; ⚠ marks a group whose trials vary by more than 5%, whose numbers should be distrusted.

## crud-list

| framework | concurrency | trials | rps (median) | rps CoV | p50 ms | p95 ms | p99 ms | errors |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| django | 1 | 3 | 134 | 5.4% ⚠ | 6.8 | 8.3 | 9.5 | 0 |
| django | 10 | 3 | 369 | 2.5% | 25.4 | 38.5 | 60.1 | 0 |
| django | 100 | 3 | 357 | 0.8% | 268.3 | 350.6 | 387.6 | 0 |
| gin-gorm | 1 | 3 | 243 | 8.9% ⚠ | 3.8 | 4.8 | 5.4 | 0 |
| gin-gorm | 10 | 3 | 417 | 1.3% | 6.1 | 82.2 | 87.3 | 0 |
| gin-gorm | 100 | 3 | 363 | 0.1% | 213.0 | 500.2 | 690.6 | 0 |
| gombit | 1 | 3 | 207 | 0.7% | 4.5 | 5.4 | 6.1 | 0 |
| gombit | 10 | 3 | 391 | 8.5% ⚠ | 6.5 | 84.1 | 88.3 | 0 |
| gombit | 100 | 3 | 348 | 2.4% | 291.7 | 503.1 | 695.6 | 0 |
| laravel | 1 | 3 | 63 | 3.6% | 15.1 | 18.1 | 21.7 | 0 |
| laravel | 10 | 3 | 124 | 17.2% ⚠ | 87.8 | 112.1 | 170.8 | 0 |
| laravel | 100 | 3 | 115 | 11.9% ⚠ | 877.5 | 911.3 | 988.9 | 0 |
| nestjs | 1 | 3 | 178 | 14.9% ⚠ | 5.2 | 6.9 | 7.7 | 0 |
| nestjs | 10 | 3 | 382 | 0.6% | 9.1 | 79.9 | 83.6 | 0 |
| nestjs | 100 | 3 | 333 | 5.2% ⚠ | 295.9 | 391.7 | 403.8 | 0 |
| rails | 1 | 3 | 192 | 3.1% | 4.8 | 6.4 | 8.1 | 0 |
| rails | 10 | 3 | 442 | 0.8% | 14.2 | 56.3 | 59.5 | 0 |
| rails | 100 | 3 | 455 | 0.5% | 212.3 | 260.5 | 274.0 | 0 |
