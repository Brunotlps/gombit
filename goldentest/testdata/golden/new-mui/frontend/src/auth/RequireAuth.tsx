import { Navigate, Outlet } from "react-router";

import { getAccessToken } from "./session";

/**
 * Redirect unauthenticated users to /login. Tokens live in memory, so a
 * full page reload always returns here until the user logs in again.
 */
export function RequireAuth() {
  if (!getAccessToken()) {
    return <Navigate to="/login" replace />;
  }
  return <Outlet />;
}
