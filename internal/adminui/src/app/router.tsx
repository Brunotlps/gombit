import { BrowserRouter, Route, Routes, useParams } from "react-router";

import { RequireAuth } from "../auth/RequireAuth";
import { AdminLayout } from "../layouts/AdminLayout";
import { CatalogPage } from "../pages/CatalogPage";
import { ForbiddenPage } from "../pages/ForbiddenPage";
import { LoginPage } from "../pages/LoginPage";
import { ResourceDetailPage } from "../pages/ResourceDetailPage";
import { ResourceFormPage } from "../pages/ResourceFormPage";
import { ResourceListPage } from "../pages/ResourceListPage";
import { AdminGate } from "./providers";

// React Router reuses one :slug match across models. key={slug} remounts the
// list so page/search/ordering/filters do not leak onto the next resource.
function ResourceListRoute() {
  const { slug = "" } = useParams();
  return <ResourceListPage key={slug} />;
}

// Create (`:slug/new`) and edit (`:slug/:id/edit`) reuse one match the same
// way. key remounts so overlapping field names (name, booleans) do not leak
// onto the next model or row. useForm defaultValues only apply on first mount.
function ResourceFormRoute({ mode }: { mode: "create" | "edit" }) {
  const { slug = "", id = "" } = useParams();
  return <ResourceFormPage key={`${slug}-${id || "new"}`} mode={mode} />;
}

export function AppRouter() {
  return (
    <BrowserRouter basename="/admin">
      <Routes>
        <Route path="login" element={<LoginPage />} />
        <Route element={<RequireAuth />}>
          <Route path="forbidden" element={<ForbiddenPage />} />
          <Route element={<AdminGate />}>
            <Route element={<AdminLayout />}>
              <Route index element={<CatalogPage />} />
              <Route path=":slug/new" element={<ResourceFormRoute mode="create" />} />
              <Route path=":slug/:id/edit" element={<ResourceFormRoute mode="edit" />} />
              <Route path=":slug/:id" element={<ResourceDetailPage />} />
              <Route path=":slug" element={<ResourceListRoute />} />
            </Route>
          </Route>
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
