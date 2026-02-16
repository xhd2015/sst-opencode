import { defineConfig } from "vite"
import desktopPlugin from "./vite"

export default defineConfig({
  plugins: [desktopPlugin] as any,
  server: {
    host: "0.0.0.0",
    allowedHosts: true,
    port: 3000,
    hmr: {
      host: "port-4444-ae2842d.xhd2015.xyz",
      protocol: "wss",
    },
    proxy: {
      "/api": {
        target: "http://localhost:5096",
        changeOrigin: true,
      },
    },
  },
  build: {
    target: "esnext",
    // sourcemap: true,
  },
})
