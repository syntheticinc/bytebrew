import { api } from './client';

interface AuthResponse {
  access_token: string;
  refresh_token: string;
  user_id: string;
}

export async function login(email: string, password: string): Promise<AuthResponse> {
  return api.request<AuthResponse>('POST', '/api/v1/auth/login', {
    email,
    password,
  });
}

export async function register(email: string, password: string): Promise<AuthResponse> {
  return api.request<AuthResponse>('POST', '/api/v1/auth/register', {
    email,
    password,
  });
}

export async function refreshAccessToken(refreshToken: string): Promise<string> {
  const res = await api.request<{ access_token: string }>(
    'POST',
    '/api/v1/auth/refresh',
    { refresh_token: refreshToken },
  );
  return res.access_token;
}

export async function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  await api.request<{ message: string }>('POST', '/api/v1/auth/change-password', {
    current_password: currentPassword,
    new_password: newPassword,
  });
}

export async function deleteAccount(password: string): Promise<void> {
  await api.request<{ message: string }>('DELETE', '/api/v1/users/me', {
    password,
  });
}

export async function googleLogin(idToken: string): Promise<AuthResponse> {
  return api.request<AuthResponse>('POST', '/api/v1/auth/google', {
    id_token: idToken,
  });
}

export async function forgotPassword(email: string): Promise<void> {
  await api.request<{ message: string }>('POST', '/api/v1/auth/forgot-password', {
    email,
  });
}

export async function resetPassword(token: string, newPassword: string): Promise<void> {
  await api.request<{ message: string }>('POST', '/api/v1/auth/reset-password', {
    token,
    new_password: newPassword,
  });
}

export async function verifyEmail(token: string): Promise<AuthResponse> {
  return api.request<AuthResponse>('POST', '/api/v1/auth/verify-email', { token });
}

export async function resendVerification(email: string): Promise<{ message: string }> {
  return api.request<{ message: string }>('POST', '/api/v1/auth/resend-verification', { email });
}

interface RegisterResponse {
  user_id: string;
  message: string;
}

export async function registerWithVerification(email: string, password: string): Promise<RegisterResponse> {
  return api.request<RegisterResponse>('POST', '/api/v1/auth/register', {
    email,
    password,
  });
}
