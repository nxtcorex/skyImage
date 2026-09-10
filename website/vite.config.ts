import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({
  base: "./",
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src")
    },
    // node_modules 里的 react/react-dom 是 pnpm 软链接，若不强制去重，
    // 预构建会把 react.development.js 打成两份（一份相对路径、一份根相对路径），
    // 导致 "Invalid hook call / more than one copy of React"。
    dedupe: ["react", "react-dom", "react/jsx-runtime", "react/jsx-dev-runtime"],
    extensions: [".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".json"]
  },
  server: {
    host: true,
    port: 5174
  }
});
