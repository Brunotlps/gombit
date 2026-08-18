import { BrowserRouter, Route, Routes } from "react-router";

import { RequireAuth } from "../auth/RequireAuth";
import { AdminLayout } from "../layouts/AdminLayout";
import { CatalogPage } from "../pages/CatalogPage";
import { ForbiddenPage } from "../pages/ForbiddenPage";
import { LoginPage } from "../pages/LoginPage";
import { ResourceDetailPage } from "../pages/ResourceDetailPage";
import { ResourceFormPage } from "../pages/ResourceFormPage";
import { ResourceListPage } from "../pages/ResourceListPage";
import { AdminGate } from "./providers";

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
              <Route path=":slug/new" element={<ResourceFormPage mode="create" />} />
              <Route path=":slug/:id/edit" element={<ResourceFormPage mode="edit" />} />
              <Route path=":slug/:id" element={<ResourceDetailPage />} />
              <Route path=":slug" element={<ResourceListPage />} />
            </Route>
          </Route>
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
