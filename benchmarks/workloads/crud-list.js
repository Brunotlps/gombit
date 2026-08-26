// Headline BENCH-1 read workload (issue #141 §"Required headline workload"):
// GET /api/projects?page=1&limit=20 against a seeded database. The same script
// runs against every implementation; the orchestrator sets TARGET_URL, VUS
// (concurrency), and DURATION.
//
// Load model — closed-loop constant VUs. The issue's concurrency-and-tail-
// latency axis is "N concurrent clients" (1/10/100/500/1000), which is a
// closed-loop VU concept, so this uses an explicit constant-vus executor at
// VUS concurrent clients for DURATION. Closed-loop load is subject to
// COORDINATED OMISSION: when the app slows, a VU waits before issuing its next
// request, so fewer requests go out and the reported tail latency understates
// true client-observed wait. Issue #141 §7 allows either constant-rate load or
// documenting this limitation; the concurrency-sweep framing makes VUs the
// natural executor, so the limitation is documented here and in
// benchmarks/README.md / methodology rather than hidden. gracefulStop is 0s so
// the measured window is exactly DURATION (no up-to-30s default drain in which
// a slow implementation would get a longer experiment than a fast one); the
// orchestrator records the *actual* elapsed run time from state, not the flag.
//
// Topology — the k6 container runs on the host network on the SAME machine as
// the app under test (issue's "another container on the same host"), not a
// separate load-generation host. At high VUs k6 contends for CPU with the app;
// this is the pinned topology, recorded as such, not hidden.
//
// This is the measured run only; warm-up is a separate short invocation whose
// output the orchestrator discards.
//
// handleSummary dumps k6's RAW metrics and state — no interpretation here. All
// mapping (throughput, percentiles, the http_req_failed Rate whose FAILED
// count is `passes` not `fails`, elapsed duration) happens in the Go parser
// (benchmarks/internal/k6), where it is unit-tested against a captured golden,
// instead of living untested in JavaScript.
import http from 'k6/http';
import { check } from 'k6';

const target = __ENV.TARGET_URL;

export const options = {
  scenarios: {
    crud_list: {
      executor: 'constant-vus',
      vus: Number(__ENV.VUS || 10),
      duration: __ENV.DURATION || '30s',
      gracefulStop: '0s',
    },
  },
  summaryTrendStats: ['p(50)', 'p(95)', 'p(99)'],
};

export default function () {
  const res = http.get(target);
  // A 200 with a full 20-row page; a wrong status or row count is a real
  // error (counted via http_req_failed / checks), not a slow success.
  check(res, {
    'status is 200': (r) => r.status === 200,
    'has 20 rows': (r) => {
      try {
        return JSON.parse(r.body).data.length === 20;
      } catch {
        return false;
      }
    },
  });
}

export function handleSummary(data) {
  const out = __ENV.SUMMARY_OUT || 'stdout';
  return { [out]: JSON.stringify({ metrics: data.metrics, state: data.state }) + '\n' };
}
