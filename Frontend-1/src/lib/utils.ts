import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import { tokenStorage } from "./auth-storage";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function initials(name: string) {
  return name
    .split(/\s+/)
    .slice(0, 2)
    .map((s) => s[0]?.toUpperCase() ?? "")
    .join("") || "?";
}

export function decodeToken(token: string): Record<string, unknown> | null {
  try {
    const payload = token.split(".")[1];
    return JSON.parse(atob(payload));
  } catch {
    return null;
  }
}

export function getTokenRole(): string | null {
  return tokenStorage.getRole();
}

export function isTokenExpired(token: string): boolean {
  const payload = decodeToken(token);
  if (!payload) return true; // undecodable token — treat as expired
  if (!payload.exp) return false; // no expiry claim — cannot be expired
  const now = Math.floor(Date.now() / 1000);
  return (payload.exp as number) < now;
}

export function getTokenExpiry(token: string): number | null {
  const payload = decodeToken(token);
  return (payload?.exp as number) ?? null;
}
