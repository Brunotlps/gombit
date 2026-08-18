/**
 * Cookie-mode session state. Access/refresh tokens live in HttpOnly cookies
 * the browser manages; this module only tracks whether a session looks
 * active and the CSRF token the SPA double-submits on unsafe methods.
 * Nothing here is ever written to web storage.
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
