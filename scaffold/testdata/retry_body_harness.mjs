/**
 * Runtime contract for #106: buffer a POST/PATCH JSON body, consume the
 * original Request (as fetch does), then retry after a 401 + successful
 * refresh. Request.body is forced to undefined to match Firefox, where the
 * getter is unimplemented but clone()/arrayBuffer() still work.
 *
 * Usage: node --experimental-strip-types harness.mjs /path/to/retry.ts
 */
import { pathToFileURL } from "node:url";

const retryPath = process.argv[2];
if (!retryPath) {
  throw new Error("usage: retry_body_harness.mjs <retry.ts>");
}

const { bufferRetryBody, retryInit } = await import(pathToFileURL(retryPath).href);
if (typeof bufferRetryBody !== "function" || typeof retryInit !== "function") {
  throw new Error("retry.ts must export bufferRetryBody and retryInit");
}

const bodyDesc = Object.getOwnPropertyDescriptor(Request.prototype, "body");
Object.defineProperty(Request.prototype, "body", {
  configurable: true,
  enumerable: bodyDesc?.enumerable ?? false,
  get() {
    return undefined;
  },
});

try {
  await run();
} finally {
  if (bodyDesc) {
    Object.defineProperty(Request.prototype, "body", bodyDesc);
  }
}

async function run() {
  const originalJSON = JSON.stringify({ name: "gadget", price: 12 });
  const productURL = "http://gombit.test/api/v1/products";
  const refreshURL = "http://gombit.test/api/v1/auth/refresh";

  const request = new Request(productURL, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: originalJSON,
  });
  if (request.body != null) {
    throw new Error("Firefox simulation failed: request.body should be undefined");
  }

  const bodies = new WeakMap();
  await bufferRetryBody(request, bodies);
  if (!bodies.has(request)) {
    throw new Error(
      "bufferRetryBody skipped the POST (Request.body gate would do this in Firefox)",
    );
  }

  const fetches = [];
  const realFetch = globalThis.fetch;
  globalThis.fetch = async (input, init) => {
    const url =
      typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
    const method =
      init?.method ?? (input instanceof Request ? input.method : "GET");
    let body = init?.body;
    if (body === undefined && input instanceof Request && method !== "GET" && method !== "HEAD") {
      try {
        body = await input.arrayBuffer();
      } catch {
        body = undefined;
      }
    } else if (input instanceof Request) {
      try {
        await input.arrayBuffer();
      } catch {
        // already consumed
      }
    }
    fetches.push({ url, method, body });
    if (url === refreshURL) {
      return new Response(JSON.stringify({ data: { ok: true } }), {
        status: 200,
        headers: { "content-type": "application/json" },
      });
    }
    const dataCalls = fetches.filter((call) => call.url === productURL);
    if (dataCalls.length === 1) {
      return new Response(
        JSON.stringify({ error: { code: "authentication", message: "expired" } }),
        { status: 401, headers: { "content-type": "application/json" } },
      );
    }
    return new Response(JSON.stringify({ data: { id: 1 } }), {
      status: 200,
      headers: { "content-type": "application/json" },
    });
  };

  try {
    const first = await fetch(request);
    if (first.status !== 401) {
      throw new Error(`first fetch status = ${first.status}, want 401`);
    }

    const refresh = await fetch(refreshURL, { method: "POST" });
    if (!refresh.ok) {
      throw new Error(`refresh status = ${refresh.status}, want 2xx`);
    }

    const retry = await fetch(
      request.url,
      retryInit(request, new Headers(request.headers), bodies.get(request)),
    );
    if (!retry.ok) {
      throw new Error(`retry status = ${retry.status}, want 2xx`);
    }
  } finally {
    globalThis.fetch = realFetch;
  }

  const productCalls = fetches.filter((call) => call.url === productURL);
  if (productCalls.length !== 2) {
    throw new Error(`product fetch count = ${productCalls.length}, want 2 (401 then retry)`);
  }
  const retryBody = productCalls[1].body;
  const retryBytes = bodyBytes(retryBody);
  if (retryBytes !== originalJSON) {
    throw new Error(`retry body = ${JSON.stringify(retryBytes)}, want ${JSON.stringify(originalJSON)}`);
  }

  await assertPatchRetry();
  process.stdout.write("ok\n");
}

async function assertPatchRetry() {
  const originalJSON = JSON.stringify({ name: "renamed" });
  const request = new Request("http://gombit.test/api/v1/products/1", {
    method: "PATCH",
    headers: { "content-type": "application/json" },
    body: originalJSON,
  });
  if (request.body != null) {
    throw new Error("Firefox simulation failed on PATCH: request.body should be undefined");
  }
  const bodies = new WeakMap();
  await bufferRetryBody(request, bodies);
  await request.arrayBuffer();
  const init = retryInit(request, new Headers(request.headers), bodies.get(request));
  if (init.signal !== request.signal) {
    throw new Error("retryInit dropped request.signal");
  }
  const got = bodyBytes(init.body);
  if (got !== originalJSON) {
    throw new Error(`PATCH retry body = ${JSON.stringify(got)}, want ${JSON.stringify(originalJSON)}`);
  }
}

function bodyBytes(body) {
  if (body == null) {
    return "";
  }
  if (typeof body === "string") {
    return body;
  }
  if (body instanceof ArrayBuffer) {
    return Buffer.from(body).toString("utf8");
  }
  if (ArrayBuffer.isView(body)) {
    return Buffer.from(body.buffer, body.byteOffset, body.byteLength).toString("utf8");
  }
  throw new Error(`unexpected retry body type ${Object.prototype.toString.call(body)}`);
}
