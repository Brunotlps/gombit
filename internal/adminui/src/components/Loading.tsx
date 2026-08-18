import { Typography } from "@mui/material";

/** Shown instead of a blank screen while catalog / session checks run. */
export function LoadingMessage({ label = "Loading…" }: { label?: string }) {
  return (
    <Typography sx={{ p: 3 }} color="text.secondary" role="status">
      {label}
    </Typography>
  );
}
