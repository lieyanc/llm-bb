import path from "node:path"
import { defineConfig } from "vite"
import react from "@vitejs/plugin-react-swc"

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: "internal/web/static",
    emptyOutDir: true,
    rollupOptions: {
      input: {
        public: path.resolve(__dirname, "frontend/src/public/main.tsx"),
        admin: path.resolve(__dirname, "frontend/src/admin/main.tsx"),
      },
      output: {
        entryFileNames: "[name].js",
        chunkFileNames: "chunks/[name]-[hash].js",
        assetFileNames: "[name][extname]",
      },
    },
  },
})
