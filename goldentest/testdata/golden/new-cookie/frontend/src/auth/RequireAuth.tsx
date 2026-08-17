import { useEffect, useState } from "react";
import { Navigate, Outlet } from "react-router";

import { useApiClient } from "../api/client";
import { isAuthenticated, setAuthenticated } from "./session";

/**
 * Cookie-mode session check (M5-3). Access/refresh tokens live in HttpOnly
 * cookies this code cannot read, so authentication is confirmed by calling
 * GET /me once per mount instead of checking an in-memory token.
 */
export function RequireAuth() {
  const client = useApiClient();
  const [status, setStatus] = useState<"checking" | "ok" | "unauthenticated">(
    isAuthenticated() ? "ok" : "checking",
  );

  useEffect(() => {
    if (status !== "checking") {
      return;
    }
    let active = true;
    client
      .GET("/api/v1/me", {})
      .then((result) => {
        if (!active) {
          return;
        }
        if (result.response.ok) {
          setAuthenticated(true);
          setStatus("ok");
        } else {
          setStatus("unauthenticated");
        }
      })
      .catch(() => {
        if (active) {
          setStatus("unauthenticated");
        }
      });
    return () => {
      active = false;
    };
  }, [client, status]);

  if (status === "checking") {
    return null;
  }
  if (status === "unauthenticated") {
    return <Navigate to="/login" replace />;
  }
  return <Outlet />;
}
