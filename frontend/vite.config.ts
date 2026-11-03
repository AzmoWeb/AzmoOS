import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// En développement, l'API est relayée vers azmod (make run → :8080).
export default defineConfig({
  plugins: [react()],
  build: { outDir: "../web/dist", emptyOutDir: true, chunkSizeWarningLimit: 600 },
  server: { proxy: { "/api": "http://localhost:8080" } },
});
