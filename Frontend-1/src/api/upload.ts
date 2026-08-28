import { api } from "./client";
import { tokenStorage } from "@/lib/auth-storage";

export interface UploadResponse {
  message: string;
  url: string;
}

/**
 * Uploaded files are served from the auth-protected /uploads route, but
 * <img> tags cannot send an Authorization header — so append the token
 * as a query parameter (backend AuthRequired accepts ?token=<jwt>).
 */
export function withUploadToken(url?: string | null): string | undefined {
  if (!url) return undefined;
  if (!url.includes("/uploads/")) return url; // external/data URLs pass through
  const token = tokenStorage.get();
  if (!token) return url;
  return url + (url.includes("?") ? "&" : "?") + "token=" + encodeURIComponent(token);
}

export const uploadApi = {
  avatar: (file: File): Promise<UploadResponse> => {
    const form = new FormData();
    form.append("file", file);
    return api.request<UploadResponse>("/upload/avatar", {
      method: "POST",
      body: form,
    });
  },

  doc: (
    file: File,
    category: "docs" | "reports" | "contributions" = "docs"
  ): Promise<UploadResponse> => {
    const form = new FormData();
    form.append("file", file);
    form.append("category", category);
    return api.request<UploadResponse>("/upload/doc", {
      method: "POST",
      body: form,
    });
  },
};
