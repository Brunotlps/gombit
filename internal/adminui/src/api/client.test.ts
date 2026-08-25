import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { clearSession } from "../auth/session";
import { bootstrapCSRF, createAdminClient } from "./client";

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function header(init: RequestInit | undefined, name: string): string | null {
  if (!init?.headers) {
    return null;
  }
  const headers = new Headers(init.headers);
  return headers.get(name);
}

describe("admin client silent refresh", () => {
  beforeEach(() => {
    clearSession();
  });

  afterEach(() => {
    clearSession();
    vi.unstubAllGlobals();
  });

  it("awaits CSRF bootstrap before POST /auth/refresh on 401", async () => {
    let releaseCSRF: () => void = () => undefined;
    const csrfHeld = new Promise<void>((resolve) => {
      releaseCSRF = resolve;
    });
    let meHits = 0;
    const refreshTokens: Array<string | null> = [];

    vi.stubGlobal(
      "fetch",
      async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
        const url = String(input);
        if (url.includes("/auth/csrf")) {
          await csrfHeld;
          return jsonResponse(200, { data: { csrf_token: "csrf-1" } });
        }
        if (url.includes("/auth/refresh")) {
          const token = header(init, "X-CSRF-Token");
          refreshTokens.push(token);
          if (token !== "csrf-1") {
            return jsonResponse(403, {
              error: { code: "authorization", message: "csrf token missing or invalid" },
            });
          }
          return jsonResponse(200, { data: { ok: true } });
        }
        if (url.includes("/me")) {
          meHits += 1;
          if (meHits === 1) {
            return jsonResponse(401, {
              error: { code: "authentication", message: "unauthorized" },
            });
          }
          return jsonResponse(200, { data: { id: 1, email: "a@b.c" } });
        }
        return jsonResponse(404, { error: { code: "not_found", message: url } });
      },
    );

    const client = createAdminClient();
    void bootstrapCSRF();
    const me = client.me();
    await Promise.resolve();
    await Promise.resolve();
    releaseCSRF();
    const envelope = await me;
    expect(envelope.data.email).toBe("a@b.c");
    expect(meHits).toBe(2);
    expect(refreshTokens).toEqual(["csrf-1"]);
  });

  it("starts CSRF bootstrap from refresh when providers have not", async () => {
    let meHits = 0;
    vi.stubGlobal(
      "fetch",
      async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
        const url = String(input);
        if (url.includes("/auth/csrf")) {
          return jsonResponse(200, { data: { csrf_token: "csrf-2" } });
        }
        if (url.includes("/auth/refresh")) {
          const token = header(init, "X-CSRF-Token");
          if (token !== "csrf-2") {
            return jsonResponse(403, {
              error: { code: "authorization", message: "csrf token missing or invalid" },
            });
          }
          return jsonResponse(200, { data: { ok: true } });
        }
        if (url.includes("/me")) {
          meHits += 1;
          if (meHits === 1) {
            return jsonResponse(401, {
              error: { code: "authentication", message: "unauthorized" },
            });
          }
          return jsonResponse(200, { data: { id: 1, email: "a@b.c" } });
        }
        return jsonResponse(404, { error: { code: "not_found", message: url } });
      },
    );

    const client = createAdminClient();
    const envelope = await client.me();
    expect(envelope.data.email).toBe("a@b.c");
    expect(meHits).toBe(2);
  });
});
