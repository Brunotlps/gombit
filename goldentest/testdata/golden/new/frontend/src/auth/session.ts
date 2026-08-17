/**
 * In-memory access and refresh tokens. Never persist them in web storage.
 * Refresh tokens are returned once in the login/refresh JSON body and held
 * here until logout or rotation. Cookie/session mode is M5-3.
 */
let accessToken: string | undefined;
let refreshToken: string | undefined;

export function getAccessToken(): string | undefined {
  return accessToken;
}

export function setAccessToken(token: string | undefined): void {
  accessToken = token;
}

export function getRefreshToken(): string | undefined {
  return refreshToken;
}

export function setRefreshToken(token: string | undefined): void {
  refreshToken = token;
}

export function clearSession(): void {
  accessToken = undefined;
  refreshToken = undefined;
}

export function applyTokenPair(pair: unknown): void {
  if (typeof pair !== "object" || pair === null) {
    return;
  }
  const record = pair as { access_token?: unknown; refresh_token?: unknown };
  if (typeof record.access_token === "string" && record.access_token !== "") {
    accessToken = record.access_token;
  }
  if (typeof record.refresh_token === "string" && record.refresh_token !== "") {
    refreshToken = record.refresh_token;
  }
}
