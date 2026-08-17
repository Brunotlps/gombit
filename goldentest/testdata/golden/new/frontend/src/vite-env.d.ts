/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Public API origin. Empty means same-origin (Vite `/api` proxy). */
  readonly VITE_API_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
