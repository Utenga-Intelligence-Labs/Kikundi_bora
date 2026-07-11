// Guarded service worker registration. Never registers in dev / Lovable preview.
export function registerPWA() {
  if (typeof window === "undefined") return;
  const host = window.location.hostname;
  const search = window.location.search;
  const inIframe = window.self !== window.top;
  const isPreview =
    host.startsWith("id-preview--") ||
    host.startsWith("preview--") ||
    host === "lovableproject.com" ||
    host.endsWith(".lovableproject.com") ||
    host === "lovableproject-dev.com" ||
    host.endsWith(".lovableproject-dev.com") ||
    host === "beta.lovable.dev" ||
    host.endsWith(".beta.lovable.dev");
  const killSwitch = search.includes("sw=off");
  const isProd = import.meta.env.PROD;

  const refuse = !isProd || inIframe || isPreview || killSwitch;

  if (refuse) {
    if ("serviceWorker" in navigator) {
      navigator.serviceWorker.getRegistrations().then((regs) => {
        regs.forEach((r) => {
          if (r.active?.scriptURL.endsWith("/sw.js")) r.unregister();
        });
      });
    }
    return;
  }

  if ("serviceWorker" in navigator) {
    window.addEventListener("load", () => {
      navigator.serviceWorker.register("/sw.js").catch(() => {});
    });
  }
}
