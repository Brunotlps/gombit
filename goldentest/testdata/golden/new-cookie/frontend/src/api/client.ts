import { createContext, useContext } from "react";

import {
  clearSession,
  getCSRFToken,
  setAuthenticated,
  setCSRFToken,
} from "../auth/session";
import { createGombitClient } from "./generated/client";
import { bufferRetryBody, retryInit } from "./retry";

export type ApiClient = ReturnType<typeof createGombitClient>;

export const ApiClientContext = createContext<ApiClient | null>(null);

const CSRF_PATH = "/api/v1/auth/csrf";
const REFRESH_PATH = "/api/v1/auth/refresh";

/**
 * Wire the generated openapi-fetch client for cookie-mode session auth
 * (M5-3). Session cookies (HttpOnly) and the CSRF cookie are managed by the
 * browser on same-origin requests; this wiring adds the X-CSRF-Token
 * double-submit header on state-changing requests and retries once after a
 * silent cookie refresh on 401. The retry rebuilds fetch() from buffered
 * body bytes so POST/PATCH JSON survives that refresh. See docs/auth-cookie.md.
 *
 * The CSRF bootstrap and refresh calls below use fetch directly instead of
 * the typed client: they are session infrastructure, not part of the
 * generated application contract that `gombit client generate` produces.
 */
export function createAppClient(): ApiClient {
  const baseUrl = import.meta.env.VITE_API_URL ?? "";
  const client = createGombitClient({ baseUrl });

  client.use({
    onRequest({ request }) {
      if (isUnsafeMethod(request.method)) {
        const token = getCSRFToken();
        if (token) {
          request.headers.set("X-CSRF-Token", token);
        }
      }
      return request;
    },
  });

  let refreshInFlight: Promise<boolean> | null = null;
  const retryBodies = new WeakMap<Request, ArrayBuffer>();

  async function refreshSession(): Promise<boolean> {
    if (refreshInFlight) {
      return refreshInFlight;
    }
    refreshInFlight = (async () => {
      try {
        const response = await fetch(baseUrl + REFRESH_PATH, {
          method: "POST",
          credentials: "same-origin",
          headers: csrfRequestHeaders(),
        });
        if (!response.ok) {
          throw new Error(`refresh failed: ${response.status}`);
        }
        setAuthenticated(true);
        return true;
      } catch {
        clearSession();
        return false;
      } finally {
        refreshInFlight = null;
      }
    })();
    return refreshInFlight;
  }

  client.use({
    async onRequest({ request }) {
      await bufferRetryBody(request, retryBodies);
      return request;
    },
    async onResponse({ request, response }) {
      if (response.status !== 401 || isAuthURL(request.url)) {
        return response;
      }
      const ok = await refreshSession();
      if (!ok) {
        return response;
      }
      const headers = new Headers(request.headers);
      const token = getCSRFToken();
      if (token) {
        headers.set("X-CSRF-Token", token);
      }
      return fetch(request.url, retryInit(request, headers, retryBodies.get(request)));
    },
  });
  return client;
}

/**
 * Fetches a fresh CSRF cookie/token pair. Call once on app load (or right
 * before the first mutating request, e.g. on the login page) so
 * POST /auth/login and friends have a token to double-submit.
 */
export async function bootstrapCSRF(): Promise<void> {
  const baseUrl = import.meta.env.VITE_API_URL ?? "";
  const response = await fetch(baseUrl + CSRF_PATH, { credentials: "same-origin" });
  if (!response.ok) {
    return;
  }
  const body = (await response.json()) as { data?: { csrf_token?: string } };
  if (body.data?.csrf_token) {
    setCSRFToken(body.data.csrf_token);
  }
}

function csrfRequestHeaders(): HeadersInit {
  const token = getCSRFToken();
  return token ? { "X-CSRF-Token": token } : {};
}

export function useApiClient(): ApiClient {
  const client = useContext(ApiClientContext);
  if (client === null) {
    throw new Error("useApiClient must be used within AppProviders");
  }
  return client;
}

function isUnsafeMethod(method: string): boolean {
  return method !== "GET" && method !== "HEAD" && method !== "OPTIONS";
}

function isAuthURL(url: string): boolean {
  try {
    const path = new URL(url, "http://gombit.invalid").pathname;
    return (
      path.endsWith("/auth/login") ||
      path.endsWith("/auth/refresh") ||
      path.endsWith("/auth/logout") ||
      path.endsWith("/auth/register") ||
      path.endsWith("/auth/csrf")
    );
  } catch {
    return url.includes("/auth/");
  }
}
