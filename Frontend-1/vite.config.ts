import { defineConfig } from "@lovable.dev/vite-tanstack-config";
import { VitePWA } from "vite-plugin-pwa";

export default defineConfig({
  tanstackStart: {
    server: { entry: "server" },
    router: {
      // Ignore colocated tests — TanStack matches this against file/dir names.
      routeFileIgnorePattern: "__tests__|\\.test\\.",
    },
    // Static hosting (nginx/docker): disable SSR hydration entirely so the
    // client bundle boots from a prerendered shell instead of expecting
    // server-injected window.__TSS data (which caused "Invariant failed").
    spa: { enabled: true, maskPath: "/" },
  },
  vite: {
    server: {
      // Backend Go API uses :8080 — keep Vite on :8081.
      port: 8081,
    },
    plugins: [
      VitePWA({
        registerType: "autoUpdate",
        injectRegister: null,
        devOptions: { enabled: false },
        filename: "sw.js",
        manifest: false,
        workbox: {
          navigateFallback: "/",
          navigateFallbackDenylist: [/^\/~oauth/, /^\/api\//],
          globPatterns: ["**/*.{js,css,html,svg,png,ico,webmanifest,woff,woff2}"],
          runtimeCaching: [
            {
              urlPattern: ({ request }) => request.mode === "navigate",
              handler: "NetworkFirst",
              options: {
                cacheName: "Money Seeking-html",
                networkTimeoutSeconds: 4,
              },
            },
            {
              urlPattern: ({ url, sameOrigin }) =>
                sameOrigin && /\.(?:js|css|woff2?|png|svg|jpg|jpeg|webp|ico)$/i.test(url.pathname),
              handler: "CacheFirst",
              options: {
                cacheName: "Money Seeking-assets",
                expiration: { maxEntries: 200, maxAgeSeconds: 60 * 60 * 24 * 30 },
              },
            },
          ],
        },
      }),
    ],
  },
});
