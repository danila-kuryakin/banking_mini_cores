import type { Role } from '../api/types';

export interface AccessClaims {
  sub: string;
  role: Role | null;
  exp: number;
}

// Роль в токене - "client", в ответе /auth/login - "ROLE_CLIENT": принимаем обе формы, как normalizeRole в гейтвее.
export function normalizeRole(raw: string | undefined | null): Role | null {
  const name = (raw ?? '').trim().toLowerCase().replace(/^role_/, '');
  return name === 'client' || name === 'officer' || name === 'admin' ? name : null;
}

export function decodeAccess(token: string): AccessClaims | null {
  const part = token.split('.')[1];
  if (!part) return null;
  try {
    const json = atob(part.replace(/-/g, '+').replace(/_/g, '/').padEnd(Math.ceil(part.length / 4) * 4, '='));
    const data = JSON.parse(json) as { sub?: string; role?: string; exp?: number };
    if (!data.sub) return null;
    return { sub: data.sub, role: normalizeRole(data.role), exp: data.exp ?? 0 };
  } catch {
    return null;
  }
}
