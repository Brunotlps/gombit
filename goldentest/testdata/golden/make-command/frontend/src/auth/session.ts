/**
 * In-memory access token. Never persist it in web storage.
 * Login, refresh, and protected routes are M5-2; getAccessToken may be undefined.
 */
let accessToken: string | undefined;

export function getAccessToken(): string | undefined {
  return accessToken;
}

export function setAccessToken(token: string | undefined): void {
  accessToken = token;
}
