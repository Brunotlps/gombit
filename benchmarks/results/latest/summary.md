# Benchmark summary

Generated from `results.json` (`benchmarks/internal/summary`) — do not edit by hand.

**Read with:** the load model is closed-loop `constant-vus` (N concurrent clients), so reported tail latency is subject to **coordinated omission** and understates true client-observed wait; the load generator shares the host with the app. Headline `rps`/`p50`/`p95`/`p99` are the **median** across trials; `rps CoV` is the coefficient of variation of throughput (stddev/mean) across trials; ⚠ marks a group whose trials vary by more than 5%, whose numbers should be distrusted.

## crud-list

| framework | concurrency | trials | rps (median) | rps CoV | p50 ms | p95 ms | p99 ms | errors |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| django | 1 | 5 | 73 | 3.1% | 12.0 | 18.8 | 28.6 | 0 |
| django | 10 | 5 | 136 | 1.3% | 59.7 | 132.5 | 165.3 | 0 |
| django | 100 | 5 | 138 | 0.4% | 689.2 | 1011.2 | 1046.2 | 0 |
| django | 500 | 5 | 130 | 1.8% | 3566.0 | 4640.0 | 5772.3 | 0 |
| django | 1000 | 5 | 114 | 1.7% | 7277.5 | 11560.4 | 12477.2 | 0 |
| gin-gorm | 1 | 5 | 125 | 4.6% | 6.2 | 13.0 | 15.1 | 0 |
| gin-gorm | 10 | 5 | 150 | 0.3% | 86.8 | 97.5 | 101.5 | 0 |
| gin-gorm | 100 | 5 | 138 | 9.9% ⚠ | 613.3 | 1404.1 | 1915.1 | 0 |
| gin-gorm | 500 | 5 | 137 | 1.8% | 3083.8 | 6828.5 | 9091.5 | 0 |
| gin-gorm | 1000 | 5 | 125 | 8.8% ⚠ | 6194.0 | 13906.8 | 18213.9 | 0 |
| gombit | 1 | 5 | 122 | 3.1% | 6.3 | 13.3 | 16.1 | 0 |
| gombit | 10 | 5 | 150 | 0.3% | 86.6 | 97.3 | 101.4 | 0 |
| gombit | 100 | 5 | 140 | 3.4% | 612.7 | 1390.5 | 1875.3 | 0 |
| gombit | 500 | 5 | 137 | 6.3% ⚠ | 3071.6 | 6810.2 | 9462.0 | 0 |
| gombit | 1000 | 5 | 126 | 12.3% ⚠ | 6106.4 | 13570.2 | 18037.0 | 0 |
| laravel | 1 | 5 | 35 | 1.5% | 27.9 | 31.5 | 34.6 | 0 |
| laravel | 10 | 5 | 67 | 0.7% | 143.7 | 199.9 | 216.0 | 0 |
| laravel | 100 | 5 | 64 | 3.9% | 1511.1 | 1692.5 | 2143.4 | 0 |
| laravel | 500 | 5 | 50 | 1.9% | 7502.0 | 13802.1 | 14682.5 | 0 |
| laravel | 1000 | 5 | 34 | 1.0% | 21907.2 | 28709.3 | 29322.3 | 0 |
| nestjs | 1 | 5 | 100 | 6.2% ⚠ | 7.8 | 16.1 | 17.7 | 0 |
| nestjs | 10 | 5 | 137 | 0.2% | 88.3 | 98.5 | 102.7 | 0 |
| nestjs | 100 | 5 | 131 | 0.4% | 764.3 | 820.1 | 890.8 | 0 |
| nestjs | 500 | 5 | 109 | 13.3% ⚠ | 3892.4 | 5549.8 | 5567.3 | 0 |
| nestjs | 1000 | 5 | 100 | 21.1% ⚠ | 9076.6 | 11262.5 | 11394.0 | 0 |
| rails | 1 | 5 | 102 | 6.2% ⚠ | 8.1 | 16.4 | 18.6 | 0 |
| rails | 10 | 5 | 166 | 0.1% | 76.9 | 89.5 | 94.9 | 0 |
| rails | 100 | 5 | 165 | 1.9% | 600.1 | 674.3 | 689.4 | 0 |
| rails | 500 | 5 | 167 | 0.2% | 2984.6 | 3105.6 | 3653.2 | 0 |
| rails | 1000 | 5 | 167 | 0.3% | 5930.6 | 6199.9 | 7784.4 | 0 |
