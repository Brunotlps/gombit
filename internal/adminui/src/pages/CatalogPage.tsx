import { Link } from "react-router";

import {
  Alert,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  Paper,
  Typography,
} from "@mui/material";

import { useCatalog } from "../app/providers";
import { canList } from "../capabilities";

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
            {models.map((model) =>
              canList(model) ? (
                <ListItemButton key={model.slug} component={Link} to={`/${model.slug}`}>
                  <ListItemText primary={model.plural} secondary={model.slug} />
                </ListItemButton>
              ) : (
                <ListItem key={model.slug}>
                  <ListItemText primary={model.plural} secondary={`${model.slug} (listing disabled)`} />
                </ListItem>
              ),
            )}
          </List>
        </Paper>
      )}
    </>
  );
}
