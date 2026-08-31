import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  Outlet,
  Link,
  createRootRouteWithContext,
  useRouter,
  HeadContent,
  Scripts,
} from "@tanstack/react-router";
import { useEffect, type ReactNode } from "react";

import "@fontsource/plus-jakarta-sans/400.css";
import "@fontsource/plus-jakarta-sans/600.css";
import "@fontsource/plus-jakarta-sans/700.css";
import "@fontsource/plus-jakarta-sans/800.css";
import "@fontsource/inter/400.css";
import "@fontsource/inter/500.css";
import "@fontsource/inter/600.css";

import appCss from "../styles.css?url";
import { reportLovableError } from "../lib/lovable-error-reporting";
import { registerPWA } from "../lib/pwa-register";
import { AuthProvider } from "../lib/auth-provider";
import { AppModalProvider } from "@/components/AppModal";
import type { RouterContext } from "../router";

function NotFoundComponent() {
  return (
    <div className="flex min-h-dvh items-center justify-center bg-background px-4">
      <div className="max-w-md text-center">
        <h1 className="font-display text-7xl font-bold text-primary">404</h1>
        <h2 className="mt-4 font-display text-xl font-semibold">Ukurasa haupatikani</h2>
        <p className="mt-2 text-sm text-muted-foreground">
          Tafadhali rudi nyumbani na ujaribu tena.
        </p>
        <Link
          to="/"
          className="mt-6 inline-flex items-center justify-center rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-primary-foreground"
        >
          Nenda Mwanzo
        </Link>
      </div>
    </div>
  );
}

function ErrorComponent({ error, reset }: { error: Error; reset: () => void }) {
  console.error(error);
  const router = useRouter();
  useEffect(() => {
    reportLovableError(error, { boundary: "tanstack_root_error_component" });
  }, [error]);

  return (
    <div className="flex min-h-dvh items-center justify-center bg-background px-4">
      <div className="max-w-md text-center">
        <h1 className="font-display text-xl font-semibold">Ukurasa huu haukufunguka</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Hitilafu imejitokeza. Tafadhali jaribu tena.
        </p>
        <button
          onClick={() => { router.invalidate(); reset(); }}
          className="mt-6 inline-flex items-center justify-center rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-primary-foreground"
        >
          Jaribu Tena
        </button>
      </div>
    </div>
  );
}

export const Route = createRootRouteWithContext<RouterContext>()({
  head: () => ({
    meta: [
      { charSet: "utf-8" },
      { name: "viewport", content: "width=device-width, initial-scale=1, viewport-fit=cover" },
      { title: "Money Seeking — Mfumo wa Money Seeking na Mikopo" },
      { name: "description", content: "Mfumo wa kidijitali wa kusimamia kikundi cha Money Seeking na mikopo: wanachama, michango, mikopo na marejesho." },
      { name: "theme-color", content: "#ffffff" },
      { name: "apple-mobile-web-app-capable", content: "yes" },
      { name: "apple-mobile-web-app-status-bar-style", content: "black-translucent" },
      { name: "apple-mobile-web-app-title", content: "Money Seeking" },
      { name: "mobile-web-app-capable", content: "yes" },
      { name: "application-name", content: "Money Seeking" },
      { property: "og:title", content: "Money Seeking — Mfumo wa Money Seeking na Mikopo" },
      { property: "og:description", content: "Sajili wanachama, kumbuka michango, toa mikopo na fuatilia marejesho — yote sehemu moja." },
      { property: "og:type", content: "website" },
      { property: "og:image", content: "/icon.png" },
      { name: "twitter:card", content: "summary" },
    ],
    links: [
      { rel: "stylesheet", href: appCss },
      { rel: "manifest", href: "/manifest.webmanifest" },
      { rel: "icon", type: "image/png", href: "/icon.png" },
      { rel: "apple-touch-icon", href: "/icon.png" },
    ],
  }),
  shellComponent: RootShell,
  component: RootComponent,
  notFoundComponent: NotFoundComponent,
  errorComponent: ErrorComponent,
});

function RootShell({ children }: { children: ReactNode }) {
  return (
    <html lang="sw">
      <head>
        <HeadContent />
      </head>
      <body>
        {children}
        <Scripts />
      </body>
    </html>
  );
}

function RootComponent() {
  const { queryClient } = Route.useRouteContext();
  useEffect(() => { registerPWA(); }, []);
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <AppModalProvider>
          <Outlet />
        </AppModalProvider>
      </AuthProvider>
    </QueryClientProvider>
  );
}
