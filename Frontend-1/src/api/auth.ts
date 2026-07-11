import { api } from "./client";
import type {
  LoginRequest,
  RegisterRequest,
  AuthResponse,
  User,
  MessageResponse,
  ChangePasswordRequest,
  ResetPasswordRequest,
  UpdateProfileRequest,
  FirstLoginSetupRequest,
} from "./types";

export const authApi = {
  login: (data: LoginRequest) => api.post<AuthResponse>("/auth/login", data),
  register: (data: RegisterRequest) =>
    api.post<AuthResponse>("/auth/register", data),
  logout: () => api.post<MessageResponse>("/auth/logout"),
  me: () => api.get<User>("/me"),
  updateProfile: (data: UpdateProfileRequest) =>
    api.put<{ message: string; data: User }>("/me", data),
  changePassword: (data: ChangePasswordRequest) =>
    api.post<MessageResponse>("/auth/change-password", data),
  resetPassword: (data: ResetPasswordRequest) =>
    api.post<MessageResponse>("/auth/reset-password", data),
  firstLoginSetup: (data: FirstLoginSetupRequest) =>
    api.post<AuthResponse>("/auth/first-login-setup", data),
};
