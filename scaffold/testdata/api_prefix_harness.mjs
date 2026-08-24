/**
 * Runtime contract for #109: typed OpenAPI paths stay `/api/v1/...`;
 * rewriteAPIRequest maps that default prefix to the live injected prefix.
 *
 * Usage: node --experimental-strip-types harness.mjs /path/to/apiPrefix.ts
 */
import { pathToFileURL } from "node:url";

const prefixPath = process.argv[2];
if (!prefixPath) {
  throw new Error("usage: api_prefix_harness.mjs <apiPrefix.ts>");
}

globalThis.window = { __GOMBIT_API_PREFIX__: "/svc/v2" };

const mod = await import(pathToFileURL(prefixPath).href);
const { apiPrefix, apiPath, rewriteAPIRequest, DEFAULT_API_PREFIX } = mod;
if (typeof apiPrefix !== "function" || typeof apiPath !== "function" || typeof rewriteAPIRequest !== "function") {
  throw new Error("apiPrefix.ts must export apiPrefix, apiPath, rewriteAPIRequest");
}
if (DEFAULT_API_PREFIX !== "/api/v1") {
  throw new Error(`DEFAULT_API_PREFIX = ${JSON.stringify(DEFAULT_API_PREFIX)}, want /api/v1`);
}

if (apiPrefix() !== "/svc/v2") {
  throw new Error(`apiPrefix() = ${JSON.stringify(apiPrefix())}, want /svc/v2`);
}
if (apiPath("/auth/csrf") !== "/svc/v2/auth/csrf") {
  throw new Error(`apiPath(/auth/csrf) = ${JSON.stringify(apiPath("/auth/csrf"))}`);
}

const original = new Request("http://gombit.test/api/v1/products", {
  method: "POST",
  headers: { "content-type": "application/json" },
  body: JSON.stringify({ name: "gadget" }),
});
const rewritten = rewriteAPIRequest(original);
const rewrittenURL = new URL(rewritten.url);
if (rewrittenURL.pathname !== "/svc/v2/products") {
  throw new Error(`rewritten pathname = ${rewrittenURL.pathname}, want /svc/v2/products`);
}
if (rewritten.method !== "POST") {
  throw new Error(`rewritten method = ${rewritten.method}, want POST`);
}
const body = await rewritten.text();
if (body !== JSON.stringify({ name: "gadget" })) {
  throw new Error(`rewritten body = ${body}`);
}

const v10 = new Request("http://gombit.test/api/v10/products");
if (new URL(rewriteAPIRequest(v10).url).pathname !== "/api/v10/products") {
  throw new Error("must not rewrite /api/v10 when the typed prefix is /api/v1");
}

window.__GOMBIT_API_PREFIX__ = "/api/v1";
const unchanged = rewriteAPIRequest(new Request("http://gombit.test/api/v1/auth/login"));
if (new URL(unchanged.url).pathname !== "/api/v1/auth/login") {
  throw new Error("default prefix must be a no-op rewrite");
}

window.__GOMBIT_API_PREFIX__ = "__GOMBIT_API_PREFIX__";
if (apiPrefix() !== "/api/v1") {
  throw new Error("placeholder must fall back to /api/v1");
}

process.stdout.write("ok\n");
