import { Link, Outlet, useNavigate } from "react-router";

import { AppBar, Box, Button, Toolbar, Typography } from "@mui/material";

import { useApiClient } from "../api/client";
import { clearSession } from "../auth/session";

export function AdminLayout() {
  const client = useApiClient();
  const navigate = useNavigate();

  async function onLogout() {
    try {
      await client.logout();
    } finally {
      clearSession();
      navigate("/login", { replace: true });
    }
  }

  return (
    <Box>
      <AppBar position="static">
        <Toolbar>
          <Typography variant="h6" component={Link} to="/" color="inherit" sx={{ flexGrow: 1, textDecoration: "none" }}>
            Admin
          </Typography>
          <Button color="inherit" component={Link} to="/">
            Models
          </Button>
          <Button color="inherit" onClick={() => void onLogout()}>
            Log out
          </Button>
        </Toolbar>
      </AppBar>
      <Box component="main" sx={{ p: 3 }}>
        <Outlet />
      </Box>
    </Box>
  );
}
