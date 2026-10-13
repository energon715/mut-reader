import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vite";

// https://vite.dev/config/
export default defineConfig({
  plugins: [svelte(), {
    name: "mut-offline-policy",
    apply: "build",
    transformIndexHtml() {
      return [{ tag: "meta", attrs: {
        "http-equiv": "Content-Security-Policy",
        content: "default-src 'self'; script-src 'self' 'unsafe-inline' 'wasm-unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data: blob:; connect-src 'self'; worker-src 'self' blob:; object-src 'none'; frame-src 'none'; base-uri 'none'; form-action 'none'",
      }, injectTo: "head-prepend" }];
    },
  }],
  define: { "import.meta.env.VITE_MUT_HOME": JSON.stringify("/home/user") },
});
