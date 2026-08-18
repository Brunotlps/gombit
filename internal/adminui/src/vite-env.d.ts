/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Public API origin only. Empty means same-origin. Do not bake API.Prefix here. */
  readonly VITE_API_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

interface Window {
  /**
   * Runtime API prefix injected when Gin serves index.html
   * (`__GOMBIT_API_PREFIX__`). Default `/api/v1` when unset.
   */
  __GOMBIT_API_PREFIX__?: string;
}
