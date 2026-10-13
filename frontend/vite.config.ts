import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vite";
import { homedir } from "node:os";

// https://vite.dev/config/
export default defineConfig({
  plugins: [svelte()],
  define: { "import.meta.env.VITE_MUT_HOME": JSON.stringify(homedir()) },
});
