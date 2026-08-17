import { BrowserRouter, Navigate, Route, Routes } from "react-router";

import { AppLayout } from "../layouts/AppLayout";
import { ProductFormPage } from "../pages/ProductFormPage";
import { ProductListPage } from "../pages/ProductListPage";
import { generatedResourceRoutes } from "../resources";

export function AppRouter() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<ProductListPage />} />
          <Route path="products/new" element={<ProductFormPage />} />
          {generatedResourceRoutes.map((route) => (
            <Route key={String(route.path)} path={route.path} element={route.element} />
          ))}
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
