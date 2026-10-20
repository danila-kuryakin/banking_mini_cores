import { request } from './http';
import type { LoginResponse, RegisterResponse, User } from './types';

export const authApi = {
  login: (email: string, password: string) =>
    request<LoginResponse>('/auth/login', { method: 'POST', body: { email, password }, auth: false }),
  register: (email: string, password: string) =>
    request<RegisterResponse>('/auth/register', { method: 'POST', body: { email, password }, auth: false }),
  logout: (refreshToken: string) =>
    request<void>('/auth/logout', { method: 'POST', body: { refresh_token: refreshToken }, auth: false }),
  createOfficer: (email: string, password: string) =>
    request<User>('/auth/officers', { method: 'POST', body: { email, password } }),
};
