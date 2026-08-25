/**
 * Cookie-mode session state (M5-3). The access/refresh tokens live in
 * HttpOnly cookies the browser manages automatically; this module only
 * tracks whether a session looks active (confirmed via GET /me or a
 * successful login/refresh) and the non-HttpOnly CSRF token the SPA must
 * double-submit on state-changing requests. Nothing here is ever written
 * to web storage.
 */
let authenticated = false;
let csrfToken: string | undefined;

export function isAuthenticated(): boolean {
  return authenticated;
}

export function setAuthenticated(value: boolean): void {
  authenticated = value;
}

export function getCSRFToken(): string | undefined {
  return csrfToken;
}

export function setCSRFToken(token: string | undefined): void {
  csrfToken = token;
}

export function clearSession(): void {
  authenticated = false;
  csrfToken = undefined;
}
