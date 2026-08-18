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
          Your account has no view permission for any registered admin model.
          Ask an administrator for a group or direct permission, or sign in
          with a superuser account.
        </Alert>
        <Button component={Link} to="/login" variant="contained">
          Back to login
        </Button>
      </Paper>
    </Box>
  );
}
