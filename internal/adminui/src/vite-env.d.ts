/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Public API origin. Empty means same-origin relative `/api/v1`. */
  readonly VITE_API_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
