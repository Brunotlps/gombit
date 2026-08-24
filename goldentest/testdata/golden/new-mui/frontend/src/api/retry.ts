/**
 * 401 retry helpers for createAppClient.
 *
 * Fetch consumes a Request's body stream. Firefox's Request.body getter is
 * unimplemented (always undefined; MDN / Bugzilla 1387483), so buffering
 * MUST be gated on method, not the Request.body getter. clone() and
 * arrayBuffer() are implemented there and still yield the JSON bytes.
 */

export async function bufferRetryBody(
  request: Request,
  bodies: WeakMap<Request, ArrayBuffer>,
): Promise<void> {
  if (request.method !== "GET" && request.method !== "HEAD") {
    bodies.set(request, await request.clone().arrayBuffer());
  }
}

export function retryInit(
  request: Request,
  headers: Headers,
  body: ArrayBuffer | undefined,
): RequestInit {
  const init: RequestInit = {
    method: request.method,
    headers,
    credentials: request.credentials,
    cache: request.cache,
    redirect: request.redirect,
    referrer: request.referrer,
    referrerPolicy: request.referrerPolicy,
    integrity: request.integrity,
    keepalive: request.keepalive,
    signal: request.signal,
    mode: request.mode,
  };
  if (body !== undefined && request.method !== "GET" && request.method !== "HEAD") {
    init.body = body;
  }
  return init;
}
