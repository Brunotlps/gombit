import { createContext, useContext } from "react";

import {
  clearSession,
  getCSRFToken,
  setAuthenticated,
  setCSRFToken,
} from "../auth/session";
import { ContractError } from "./error";
import type { Catalog, CatalogAux, ModelMeta, PageMeta, Row } from "./types";

export type Envelope<T, M = unknown> = {
  data: T;
  meta?: M;
};

const CSRF_PATH = "/api/v1/auth/csrf";
const REFRESH_PATH = "/api/v1/auth/refresh";

function apiOrigin(): string {
  return import.meta.env.VITE_API_URL ?? "";
}

/**
 * Thin same-origin fetch wrapper around D10 `{data, meta, error}`.
 * Cookie session + CSRF double-submit (X-CSRF-Token) on unsafe methods.
 * Concurrent 401s share one refreshInFlight promise. CSRF stays in memory.
 */
export function createAdminClient() {
  const baseUrl = apiOrigin();
  let refreshInFlight: Promise<boolean> | null = null;

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

  async function request<T, M = unknown>(
    method: string,
    path: string,
    options?: { body?: unknown; query?: Record<string, string | number | undefined> },
  ): Promise<Envelope<T, M>> {
    const url = buildURL(baseUrl, path, options?.query);
    const init = (csrf?: string): RequestInit => {
      const headers = new Headers();
      if (options?.body !== undefined) {
        headers.set("Content-Type", "application/json");
      }
      if (isUnsafeMethod(method)) {
        const token = csrf ?? getCSRFToken();
        if (token) {
          headers.set("X-CSRF-Token", token);
        }
      }
      return {
        method,
        credentials: "same-origin",
        headers,
        body: options?.body === undefined ? undefined : JSON.stringify(options.body),
      };
    };

    let response = await fetch(url, init());
    if (response.status === 401 && !isAuthPath(path)) {
      const ok = await refreshSession();
      if (ok) {
        response = await fetch(url, init());
      }
    }

    const parsed: unknown = await readJSON(response);
    if (!response.ok) {
      throw ContractError.fromResponse(response, parsed);
    }
    if (parsed === null || typeof parsed !== "object" || !("data" in parsed)) {
      throw new ContractError("empty_response", "Empty response", response.status);
    }
    const envelope = parsed as Envelope<T, M>;
    return envelope;
  }

  return {
    get: <T, M = unknown>(path: string, query?: Record<string, string | number | undefined>) =>
      request<T, M>("GET", path, { query }),
    post: <T, M = unknown>(path: string, body?: unknown) => request<T, M>("POST", path, { body }),
    patch: <T, M = unknown>(path: string, body?: unknown) => request<T, M>("PATCH", path, { body }),
    delete: <T, M = unknown>(path: string) => request<T, M>("DELETE", path),
    login: (email: string, password: string) =>
      request<{ id: number; email: string }>("POST", "/api/v1/auth/login", {
        body: { email, password },
      }),
    logout: () => request<{ ok: boolean }>("POST", "/api/v1/auth/logout"),
    me: () => request<{ id: number; email: string }>("GET", "/api/v1/me"),
    catalog: () => request<Catalog, CatalogAux>("GET", "/api/v1/admin/meta"),
    model: (slug: string) => request<ModelMeta>("GET", `/api/v1/admin/meta/${slug}`),
    list: (slug: string, query?: Record<string, string | number | undefined>) =>
      request<Row[], PageMeta>("GET", `/api/v1/admin/resources/${slug}`, { query }),
    create: (slug: string, body: Row) =>
      request<Row>("POST", `/api/v1/admin/resources/${slug}`, { body }),
    detail: (slug: string, id: string) =>
      request<Row>("GET", `/api/v1/admin/resources/${slug}/${id}`),
    update: (slug: string, id: string, body: Row) =>
      request<Row>("PATCH", `/api/v1/admin/resources/${slug}/${id}`, { body }),
    remove: (slug: string, id: string) =>
      request<{ ok: boolean }>("DELETE", `/api/v1/admin/resources/${slug}/${id}`),
  };
}

export type AdminClient = ReturnType<typeof createAdminClient>;

export const ApiClientContext = createContext<AdminClient | null>(null);

export function useApiClient(): AdminClient {
  const client = useContext(ApiClientContext);
  if (client === null) {
    throw new Error("useApiClient must be used within AppProviders");
  }
  return client;
}

/**
 * Fetches a fresh CSRF cookie/token pair. Call on app load and on the login
 * page so POST /auth/login and other unsafe methods have a token to
 * double-submit. The token is held in memory only.
 */
export async function bootstrapCSRF(): Promise<void> {
  const baseUrl = apiOrigin();
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

function isUnsafeMethod(method: string): boolean {
  return method !== "GET" && method !== "HEAD" && method !== "OPTIONS";
}

function isAuthPath(path: string): boolean {
  return (
    path.endsWith("/auth/login") ||
    path.endsWith("/auth/refresh") ||
    path.endsWith("/auth/logout") ||
    path.endsWith("/auth/register") ||
    path.endsWith("/auth/csrf")
  );
}

function buildURL(
  baseUrl: string,
  path: string,
  query?: Record<string, string | number | undefined>,
): string {
  const origin = typeof window !== "undefined" ? window.location.origin : "http://127.0.0.1";
  const url = new URL(baseUrl + path, origin);
  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value === undefined || value === "") {
        continue;
      }
      url.searchParams.set(key, String(value));
    }
  }
  return url.toString();
}

async function readJSON(response: Response): Promise<unknown> {
  const text = await response.text();
  if (text === "") {
    return undefined;
  }
  try {
    return JSON.parse(text) as unknown;
  } catch {
    return undefined;
  }
}
