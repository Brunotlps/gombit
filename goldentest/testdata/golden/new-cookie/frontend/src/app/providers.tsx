import { useEffect, useMemo, type ReactNode } from "react";

import { bootstrapCSRF, ApiClientContext, createAppClient } from "../api/client";

export function AppProviders({ children }: { children: ReactNode }) {
  const client = useMemo(() => createAppClient(), []);
  useEffect(() => {
    void bootstrapCSRF();
  }, []);
  return <ApiClientContext.Provider value={client}>{children}</ApiClientContext.Provider>;
}
