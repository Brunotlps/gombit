import { Link, Outlet } from "react-router";

import { generatedResources } from "../resources";

export function AppLayout() {
  return (
    <div>
      <header>
        <nav>
          <Link to="/">Products</Link>
          {" · "}
          <Link to="/products/new">New product</Link>
          {generatedResources.map((resource) => (
            <span key={resource.slug}>
              {" · "}
              <Link to={resource.listPath}>{resource.title}</Link>
            </span>
          ))}
        </nav>
      </header>
      <main>
        <Outlet />
      </main>
    </div>
  );
}
