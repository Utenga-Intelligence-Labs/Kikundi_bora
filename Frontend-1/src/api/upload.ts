import { api } from "./client";

export interface UploadResponse {
  message: string;
  url: string;
}

export const uploadApi = {
  avatar: (file: File): Promise<UploadResponse> => {
    const form = new FormData();
    form.append("file", file);
    return api.request<UploadResponse>("/upload/avatar", {
      method: "POST",
      body: form,
      headers: { "Content-Type": undefined as unknown as string },
    });
  },

  doc: (file: File, category: "docs" | "reports" = "docs"): Promise<UploadResponse> => {
    const form = new FormData();
    form.append("file", file);
    form.append("category", category);
    return api.request<UploadResponse>("/upload/doc", {
      method: "POST",
      body: form,
      headers: { "Content-Type": undefined as unknown as string },
    });
  },
};
