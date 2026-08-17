import { createContext, useContext } from "react";

import { getAccessToken } from "../auth/session";
import { createGombitClient } from "./generated/client";

export type ApiClient = ReturnType<typeof createGombitClient>;

export const ApiClientContext = createContext<ApiClient | null>(null);

/**
 * Wire the generated openapi-fetch client. VITE_API_URL is public; empty
 * means same-origin so the Vite `/api` proxy used by `gombit dev` works.
 */
export function createAppClient(): ApiClient {
  const baseUrl = import.meta.env.VITE_API_URL ?? "";
  return createGombitClient({
    baseUrl,
    getAccessToken,
  });
}

export function useApiClient(): ApiClient {
  const client = useContext(ApiClientContext);
  if (client === null) {
    throw new Error("useApiClient must be used within AppProviders");
  }
  return client;
}
