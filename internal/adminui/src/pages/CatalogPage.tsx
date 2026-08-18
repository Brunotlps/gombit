import { Link } from "react-router";

import {
  Alert,
  List,
  ListItemButton,
  ListItemText,
  Paper,
  Typography,
} from "@mui/material";

import { useCatalog } from "../app/providers";

export function CatalogPage() {
  const { catalog } = useCatalog();
  const models = catalog.models;

  return (
    <>
      <Typography variant="h4" component="h1" sx={{ mb: 2 }}>
        Models
      </Typography>
      {models.length === 0 ? (
        <Alert severity="info">
          No models registered. Call admin.Register from a feature package.
        </Alert>
      ) : (
        <Paper>
          <List>
            {models.map((model) => (
              <ListItemButton key={model.slug} component={Link} to={`/${model.slug}`}>
                <ListItemText primary={model.plural} secondary={model.slug} />
              </ListItemButton>
            ))}
          </List>
        </Paper>
      )}
    </>
  );
}
