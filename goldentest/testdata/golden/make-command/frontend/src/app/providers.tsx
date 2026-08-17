import { useMemo, type ReactNode } from "react";

import { ApiClientContext, createAppClient } from "../api/client";

export function AppProviders({ children }: { children: ReactNode }) {
  const client = useMemo(() => createAppClient(), []);
  return <ApiClientContext.Provider value={client}>{children}</ApiClientContext.Provider>;
}
