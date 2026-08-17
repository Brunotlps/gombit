import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const backend = process.env.GOMBIT_DEV_BACKEND ?? "http://127.0.0.1:8080";
const host = process.env.GOMBIT_DEV_FRONTEND_HOST ?? "127.0.0.1";
const port = Number(process.env.GOMBIT_DEV_FRONTEND_PORT ?? "5173");

export default defineConfig({
  plugins: [react()],
  server: {
    host,
    port,
    strictPort: true,
    proxy: {
      "/api": { target: backend, changeOrigin: true },
      "/openapi.json": { target: backend, changeOrigin: true },
      "/docs": { target: backend, changeOrigin: true },
    },
  },
});
