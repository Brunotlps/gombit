// Headline BENCH-1 read workload (issue #141 §"Required headline workload"):
// GET /api/projects?page=1&limit=20 against a seeded database. The same script
// runs against every implementation; the orchestrator sets TARGET_URL, VUS
// (concurrency), and DURATION, and reads the machine-readable summary this
// writes via handleSummary.
//
// This is the *measured* run only — a constant VUs load for DURATION. Warm-up
// is a separate short invocation of this same script whose summary the
// orchestrator discards, so no warm-up traffic pollutes the reported metrics.
import http from 'k6/http';
import { check } from 'k6';

const target = __ENV.TARGET_URL;

export const options = {
  vus: Number(__ENV.VUS || 10),
  duration: __ENV.DURATION || '30s',
  // Percentiles the result schema records (issue §9 latency_ms).
  summaryTrendStats: ['p(50)', 'p(95)', 'p(99)'],
  // The load generator itself must not be the thing that fails the run on a
  // slow tail; thresholds are evaluated but not abort-on-fail here.
  discardResponseBodies: false,
};

export default function () {
  const res = http.get(target);
  // A response that isn't 200 with a full 20-row page is a real error, not a
  // slow success — count it so errors surfaces in the summary.
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
  const duration = data.metrics.http_req_duration.values;
  const reqs = data.metrics.http_reqs.values;
  // http_req_failed is a Rate metric where a request is "added" as true when
  // it failed (non-2xx/3xx). In k6's summary a Rate's `passes` is the count of
  // true adds and `fails` the count of false adds — so the number of FAILED
  // requests is `passes`, not `fails` (verified against a live all-200 run:
  // passes=0, fails=<total>). Using `fails` here would report every
  // successful request as an error.
  const failed = data.metrics.http_req_failed
    ? data.metrics.http_req_failed.values
    : { passes: 0 };

  // Failed content checks (a 200 with the wrong row count) don't show up in
  // http_req_failed, so surface them separately — the orchestrator fails the
  // run on either, so a silently-wrong app can't be recorded as valid.
  const checks = data.metrics.checks ? data.metrics.checks.values : { fails: 0 };

  const summary = {
    requests: reqs.count,
    requests_per_second: reqs.rate,
    latency_ms: {
      p50: duration['p(50)'],
      p95: duration['p(95)'],
      p99: duration['p(99)'],
    },
    errors: failed.passes || 0,
    checks_failed: checks.fails || 0,
  };

  const out = __ENV.SUMMARY_OUT || 'stdout';
  return { [out]: JSON.stringify(summary) + '\n' };
}
