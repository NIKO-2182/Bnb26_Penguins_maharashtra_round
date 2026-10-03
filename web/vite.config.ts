/**
 * File: web/vite.config.ts
 * Purpose: Vite build + dev server. Dev proxies mirror nginx.conf so the
 *          same relative /api and /sim paths work locally and in Docker.
 */
import path from "node:path"
import { defineConfig } from "vite"
import react from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"

const API_TARGET = process.env.FAIRDROP_API_URL ?? "http://localhost:8080"
const SIM_TARGET = process.env.FAIRDROP_SIM_URL ?? "http://localhost:8090"

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { "@": path.resolve(__dirname, "./src") },
  },
  server: {
    host: "0.0.0.0",
    port: 3000,
    allowedHosts: true,
    proxy: {
      "/api": {
        target: API_TARGET,
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/api/, ""),
      },
      "/sim": {
        target: SIM_TARGET,
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/sim/, ""),
      },
    },
  },
  worker: { format: "es" },
})
