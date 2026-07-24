import {
  createContext,
  useContext,
  useEffect,
  type ReactNode,
} from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { authApi } from "@/api/auth";
import { api } from "@/api/client";
import { tokenStorage } from "./auth-storage";
import type { User, LoginRequest, RegisterRequest, AuthResponse } from "@/api/types";

export { initials } from "./utils";

interface AuthContextValue {
  user: User | null;
  isLoading: boolean;
  login: (data: LoginRequest) => Promise<AuthResponse>;
  register: (data: RegisterRequest) => Promise<AuthResponse>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const qc = useQueryClient();

  useEffect(() => {
    api.configure({
      getToken: () => tokenStorage.get(),
      onUnauthorized: () => {
        tokenStorage.clear();
        qc.setQueryData(["auth", "me"], null);
        qc.clear();
        if (typeof window !== "undefined") {
          const path = window.location.pathname;
          if (path !== "/ingia" && path !== "/sajili" && path !== "/sahau") {
            window.location.assign("/ingia");
          }
        }
      },
    });
  }, [qc]);

  const { data: user, isLoading } = useQuery({
    queryKey: ["auth", "me"],
    queryFn: async () => {
      if (!tokenStorage.exists()) return null;
      try {
        return await authApi.me();
      } catch {
        tokenStorage.clear();
        return null;
      }
    },
    staleTime: 5 * 60 * 1000,
    retry: false,
  });

  const loginMutation = useMutation({
    mutationFn: (data: LoginRequest) => authApi.login(data),
    onSuccess: (res) => {
      tokenStorage.set(res.token);
      // Invalidate to force a fresh /me fetch (includes leadership, member_id)
      qc.invalidateQueries({ queryKey: ["auth", "me"] });
    },
  });

  const registerMutation = useMutation({
    mutationFn: (data: RegisterRequest) => authApi.register(data),
    onSuccess: (res) => {
      tokenStorage.set(res.token);
      qc.invalidateQueries({ queryKey: ["auth", "me"] });
    },
  });

  const logoutMutation = useMutation({
    mutationFn: () => authApi.logout(),
    onSettled: () => {
      tokenStorage.clear();
      qc.setQueryData(["auth", "me"], null);
      qc.clear();
    },
  });

  const currentUser = user ?? null;
  const isMember = !!currentUser?.member_id;
  const isLeadership = (currentUser?.leadership?.length ?? 0) > 0;
  const isAdmin = currentUser?.role === "admin";

  const hasLeadershipRole = (...roles: string[]) => {
    if (!currentUser?.leadership) return false;
    return currentUser.leadership.some((r) => roles.includes(r));
  };

  const value: AuthContextValue = {
    user: currentUser,
    isLoading,
    login: async (data) => { return await loginMutation.mutateAsync(data); },
    register: async (data) => { return await registerMutation.mutateAsync(data); },
    logout: async () => { await logoutMutation.mutateAsync(); },
    isMember,
    isLeadership,
    isAdmin,
    hasLeadershipRole,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
