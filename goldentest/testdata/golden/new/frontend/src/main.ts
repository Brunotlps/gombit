import { mountIfResource } from "./resources";

const app = document.querySelector("#app");
if (!(app instanceof HTMLElement)) {
  throw new Error("missing #app");
}

if (mountIfResource(app)) {
  // generated resource pages own #app
} else {
  const apiBase = import.meta.env.VITE_API_URL || "/api/v1";

  app.textContent = "Loading products…";

  void fetch(`${apiBase}/products`)
    .then(async (response) => {
      const body: unknown = await response.json();
      app.textContent = JSON.stringify(body, null, 2);
    })
    .catch((err: unknown) => {
      const message = err instanceof Error ? err.message : "request failed";
      app.textContent = message;
    });
}
