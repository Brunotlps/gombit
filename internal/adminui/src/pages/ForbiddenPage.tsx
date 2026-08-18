import { Link } from "react-router";

import { Alert, Box, Button, Paper, Typography } from "@mui/material";

export function ForbiddenPage() {
  return (
    <Box sx={{ p: 4, maxWidth: 560, mx: "auto" }}>
      <Paper sx={{ p: 3 }}>
        <Typography variant="h5" component="h1" sx={{ mb: 1 }}>
          Forbidden
        </Typography>
        <Alert severity="warning" sx={{ mb: 2 }}>
          This admin requires a superuser. Groups and permissions land in
          ADMIN-3. Sign in with an account created by gombit createsuperuser.
        </Alert>
        <Button component={Link} to="/login" variant="contained">
          Back to login
        </Button>
      </Paper>
    </Box>
  );
}
