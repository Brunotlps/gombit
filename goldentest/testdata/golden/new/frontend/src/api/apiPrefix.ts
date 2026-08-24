/**
 * Runtime API prefix (D8 / GOMBIT_API_PREFIX). Read from the served
 * index (`window.__GOMBIT_API_PREFIX__` or
 * `<meta name="gombit-api-prefix">`), not from VITE_* (that would freeze
 * the prefix in the Vite bundle). Default remains `/api/v1`.
 *
 * openapi-fetch types paths as `/api/v1/...` from the placeholder OpenAPI
 * document. Callers keep those typed keys; rewriteAPIRequest maps them to
 * the live prefix on the way out.
 */

export const DEFAULT_API_PREFIX = "/api/v1";
const PREFIX_PLACEHOLDER = "__GOMBIT_API_PREFIX__";

export function apiPrefix(): string {
  const injected = readInjectedAPIPrefix();
  const raw = (injected ?? "").trim();
  if (raw === "" || raw === PREFIX_PLACEHOLDER) {
    return DEFAULT_API_PREFIX;
  }
  return raw.replace(/\/+$/, "") || DEFAULT_API_PREFIX;
}

export function apiPath(path: string): string {
  const suffix = path.startsWith("/") ? path : `/${path}`;
  return apiPrefix() + suffix;
}

/**
 * Rewrite an openapi-fetch Request whose path still starts with the
 * typed `/api/v1` prefix so it hits config.API.Prefix. No-op when the
 * live prefix is the default. Runs in onRequest, before fetch consumes
 * the body, so cloning here is safe (unlike the 401 retry).
 */
export function rewriteAPIRequest(request: Request): Request {
  const live = apiPrefix();
  if (live === DEFAULT_API_PREFIX) {
    return request;
  }
  const url = new URL(request.url, "http://gombit.invalid");
  const from = DEFAULT_API_PREFIX;
  if (url.pathname !== from && !url.pathname.startsWith(from + "/")) {
    return request;
  }
  url.pathname = live + url.pathname.slice(from.length);
  return new Request(url, request);
}

function readInjectedAPIPrefix(): string | undefined {
  if (typeof window !== "undefined" && typeof window.__GOMBIT_API_PREFIX__ === "string") {
    return window.__GOMBIT_API_PREFIX__;
  }
  if (typeof document !== "undefined") {
    const content = document.querySelector('meta[name="gombit-api-prefix"]')?.getAttribute("content");
    if (content) {
      return content;
    }
  }
  return undefined;
}
