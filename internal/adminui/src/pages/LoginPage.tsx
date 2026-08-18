import { useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { useNavigate } from "react-router";

import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Paper,
  TextField,
  Typography,
} from "@mui/material";

import { bootstrapCSRF, useApiClient } from "../api/client";
import { applyContractErrors } from "../api/formErrors";
import { setAuthenticated } from "../auth/session";

type LoginValues = {
  email: string;
  password: string;
};

export function LoginPage() {
  const client = useApiClient();
  const navigate = useNavigate();
  const [status, setStatus] = useState("");
  const {
    control,
    handleSubmit,
    setError,
    formState: { isSubmitting },
  } = useForm<LoginValues>({
    defaultValues: { email: "", password: "" },
  });

  async function onLogin(values: LoginValues) {
    setStatus("");
    try {
      // Providers start CSRF on app load; wait for that in-flight pair (or
      // mint one after clearSession) before POST /auth/login.
      await bootstrapCSRF();
      await client.login(values.email, values.password);
      setAuthenticated(true);
      navigate("/", { replace: true });
    } catch (err: unknown) {
      if (!applyContractErrors(setError, err)) {
        setStatus(err instanceof Error ? err.message : "login failed");
      }
    }
  }

  return (
    <Box
      sx={{
        minHeight: "100vh",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        p: 2,
      }}
    >
      <Paper sx={{ p: 4, width: "100%", maxWidth: 420 }}>
        <Typography component="h1" variant="h5" sx={{ mb: 1 }}>
          Admin sign in
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
          Cookie session. The CSRF token stays in memory. Admin access requires
          a model permission or a superuser account.
        </Typography>
        {status ? (
          <Alert severity="error" sx={{ mb: 2 }}>
            {status}
          </Alert>
        ) : null}
        <Box
          component="form"
          onSubmit={handleSubmit(onLogin)}
          sx={{ display: "flex", flexDirection: "column", gap: 2 }}
        >
          <Controller
            name="email"
            control={control}
            rules={{ required: true }}
            render={({ field, fieldState }) => (
              <TextField
                {...field}
                type="email"
                label="Email"
                autoComplete="username"
                fullWidth
                error={!!fieldState.error}
                helperText={fieldState.error?.message}
                disabled={isSubmitting}
              />
            )}
          />
          <Controller
            name="password"
            control={control}
            rules={{ required: true }}
            render={({ field, fieldState }) => (
              <TextField
                {...field}
                type="password"
                label="Password"
                autoComplete="current-password"
                fullWidth
                error={!!fieldState.error}
                helperText={fieldState.error?.message}
                disabled={isSubmitting}
              />
            )}
          />
          <Button type="submit" variant="contained" disabled={isSubmitting}>
            {isSubmitting ? <CircularProgress size={24} color="inherit" /> : "Log in"}
          </Button>
        </Box>
      </Paper>
    </Box>
  );
}
