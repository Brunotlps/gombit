import { useState } from "react";
import { useForm, Controller } from "react-hook-form";
import { Link, useNavigate } from "react-router";

import { Alert, Box, Button, Paper, TextField, Typography } from "@mui/material";

import { useApiClient } from "../api/client";
import { applyContractErrors } from "../api/formErrors";
import { unwrap } from "../api/generated/client";

type ProductFormValues = {
  name: string;
  price: number;
};

export function ProductFormPage() {
  const client = useApiClient();
  const navigate = useNavigate();
  const [status, setStatus] = useState("");
  const {
    control,
    handleSubmit,
    setError,
    formState: { isSubmitting },
  } = useForm<ProductFormValues>({
    defaultValues: { name: "", price: 0 },
  });

  async function onSubmit(values: ProductFormValues) {
    setStatus("");
    try {
      await unwrap(
        await client.POST("/api/v1/products", {
          body: { name: values.name, price: values.price },
        }),
      );
      navigate("/");
    } catch (err: unknown) {
      if (!applyContractErrors(setError, err)) {
        setStatus(err instanceof Error ? err.message : "request failed");
      }
    }
  }

  return (
    <Box>
      <Typography variant="h4" component="h1" sx={{ mb: 1 }}>
        New product
      </Typography>
      <Button component={Link} to="/" sx={{ mb: 2 }}>
        Back to list
      </Button>
      <Paper sx={{ p: 3, maxWidth: 480 }}>
        <Box component="form" onSubmit={handleSubmit(onSubmit)} sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
          <Controller
            name="name"
            control={control}
            rules={{ required: "Name is required" }}
            render={({ field, fieldState }) => (
              <TextField
                {...field}
                label="Name"
                fullWidth
                error={!!fieldState.error}
                helperText={fieldState.error?.message}
                disabled={isSubmitting}
              />
            )}
          />
          <Controller
            name="price"
            control={control}
            render={({ field, fieldState }) => (
              <TextField
                {...field}
                type="number"
                label="Price"
                fullWidth
                error={!!fieldState.error}
                helperText={fieldState.error?.message}
                disabled={isSubmitting}
                onChange={(event) => {
                  const raw = event.target.value;
                  field.onChange(raw === "" ? 0 : Number(raw));
                }}
              />
            )}
          />
          <Button type="submit" variant="contained" disabled={isSubmitting}>
            Create
          </Button>
        </Box>
        {status ? (
          <Alert severity="error" sx={{ mt: 2 }}>
            {status}
          </Alert>
        ) : null}
      </Paper>
    </Box>
  );
}
