import { QueryClient } from "@tanstack/react-query";
import { createRouter } from "@tanstack/react-router";
import { routeTree } from "./routeTree.gen";
import type { User } from "@/api/types";

export interface RouterContext {
  queryClient: QueryClient;
  user: User | null;
}

export const getRouter = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 60_000,           // 60s — data considered fresh for 1 minute
        gcTime: 5 * 60_000,         // 5 min garbage collection
        refetchOnWindowFocus: false, // no refetch on tab switch
        retry: 1,                    // single retry on failure
      },
    },
  });

  const router = createRouter({
    routeTree,
    context: { queryClient, user: null },
    scrollRestoration: true,
    defaultPreloadStaleTime: 60_000,
  });

  return router;
};
