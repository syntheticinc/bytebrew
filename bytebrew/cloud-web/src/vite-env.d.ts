/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_SHOW_EE_PRICING?: string;
  readonly VITE_SHOW_CODE_SITE?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
