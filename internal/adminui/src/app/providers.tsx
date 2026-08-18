import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { Navigate, Outlet } from "react-router";

import { Alert, Box, CssBaseline, ThemeProvider } from "@mui/material";

import { ApiClientContext, bootstrapCSRF, createAdminClient, useApiClient } from "../api/client";
import { ContractError } from "../api/error";
import type { Catalog, ModelMeta } from "../api/types";
import { LoadingMessage } from "../components/Loading";
import { theme } from "../theme";

export function AppProviders({ children }: { children: ReactNode }) {
  const client = useMemo(() => createAdminClient(), []);

  useEffect(() => {
    void bootstrapCSRF();
  }, []);

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <ApiClientContext.Provider value={client}>{children}</ApiClientContext.Provider>
    </ThemeProvider>
  );
}

type CatalogState = {
  catalog: Catalog;
  bySlug: Map<string, ModelMeta>;
};

const CatalogContext = createContext<CatalogState | null>(null);

export function useCatalog(): CatalogState {
  const value = useContext(CatalogContext);
  if (value === null) {
    throw new Error("useCatalog must be used within AdminGate");
  }
  return value;
}

/** Loads GET {API.Prefix}/admin/meta. 401 → login; 403 → forbidden; other errors stay here. */
export function AdminGate() {
  const client = useApiClient();
  const [state, setState] = useState<CatalogState | null>(null);
  const [error, setError] = useState<ContractError | null>(null);

  useEffect(() => {
    let active = true;
    client
      .catalog()
      .then((envelope) => {
        if (!active) {
          return;
        }
        const models = envelope.data.models ?? [];
        const bySlug = new Map<string, ModelMeta>();
        for (const model of models) {
          bySlug.set(model.slug, model);
        }
        setState({ catalog: { models }, bySlug });
      })
      .catch((err: unknown) => {
        if (!active) {
          return;
        }
        if (err instanceof ContractError) {
          setError(err);
          return;
        }
        setError(new ContractError("error", "failed to load admin catalog", 500));
      });
    return () => {
      active = false;
    };
  }, [client]);

  if (error?.status === 401) {
    return <Navigate to="/login" replace />;
  }
  if (error?.status === 403) {
    return <Navigate to="/forbidden" replace />;
  }
  if (error) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">{error.message}</Alert>
      </Box>
    );
  }
  if (state === null) {
    return <LoadingMessage />;
  }
  return (
    <CatalogContext.Provider value={state}>
      <Outlet />
    </CatalogContext.Provider>
  );
}
