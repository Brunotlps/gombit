import { useMemo, type ReactNode } from "react";
import { CssBaseline, ThemeProvider } from "@mui/material";

import { ApiClientContext, createAppClient } from "../api/client";
import { theme } from "../theme";

export function AppProviders({ children }: { children: ReactNode }) {
  const client = useMemo(() => createAppClient(), []);
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <ApiClientContext.Provider value={client}>{children}</ApiClientContext.Provider>
    </ThemeProvider>
  );
}
