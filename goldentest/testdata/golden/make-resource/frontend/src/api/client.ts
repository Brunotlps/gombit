import { createContext, useContext } from "react";

import {
  applyTokenPair,
  clearSession,
  getAccessToken,
  getRefreshToken,
} from "../auth/session";
import { createGombitClient, unwrap } from "./generated/client";

export type ApiClient = ReturnType<typeof createGombitClient>;

export const ApiClientContext = createContext<ApiClient | null>(null);

/**
 * Wire the generated openapi-fetch client. VITE_API_URL is public; empty
 * means same-origin so the Vite `/api` proxy used by `gombit dev` works.
 *
 * On 401, rotate the in-memory refresh token once and retry. Concurrent
 * 401s share that refresh instead of returning stale failures. Tokens are
 * never written to web storage.
 */
export function createAppClient(): ApiClient {
  const baseUrl = import.meta.env.VITE_API_URL ?? "";
  const client = createGombitClient({
    baseUrl,
    getAccessToken,
  });

  let refreshInFlight: Promise<boolean> | null = null;

  async function refreshSession(): Promise<boolean> {
    if (refreshInFlight) {
      return refreshInFlight;
    }
    const currentRefresh = getRefreshToken();
    if (!currentRefresh) {
      return false;
    }
    refreshInFlight = (async () => {
      try {
        const rotated = await unwrap(
          await client.POST("/api/v1/auth/refresh", {
            body: { refresh_token: currentRefresh },
          }),
        );
        applyTokenPair(rotated.data);
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
    async onResponse({ request, response }) {
      if (response.status !== 401 || isAuthURL(request.url)) {
        return response;
      }
      const ok = await refreshSession();
      if (!ok) {
        return response;
      }
      const retry = new Request(request, {
        headers: new Headers(request.headers),
      });
      const access = getAccessToken();
      if (access) {
        retry.headers.set("Authorization", `Bearer ${access}`);
      }
      return fetch(retry);
    },
  });
  return client;
}

export function useApiClient(): ApiClient {
  const client = useContext(ApiClientContext);
  if (client === null) {
    throw new Error("useApiClient must be used within AppProviders");
  }
  return client;
}

function isAuthURL(url: string): boolean {
  try {
    const path = new URL(url, "http://gombit.invalid").pathname;
    return (
      path.endsWith("/auth/login") ||
      path.endsWith("/auth/refresh") ||
      path.endsWith("/auth/logout") ||
      path.endsWith("/auth/register")
    );
  } catch {
    return url.includes("/auth/");
  }
}
