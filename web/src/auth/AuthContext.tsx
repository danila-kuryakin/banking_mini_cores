import { useQueryClient } from '@tanstack/react-query';
import { createContext, useCallback, useContext, useEffect, useMemo, useState, useSyncExternalStore } from 'react';
import type { ReactNode } from 'react';
import { useNavigate } from 'react-router-dom';
import { authApi } from '../api/auth';
import type { Role } from '../api/types';
import { useToast } from '../components/Toast';
import { session, type SessionUser } from './session';

interface AuthValue {
  user: SessionUser | null;
  ready: boolean;
  login: (email: string, password: string) => Promise<SessionUser>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthValue | null>(null);

export function homeFor(role: Role): string {
  return role === 'client' ? '/app' : '/staff/kyc';
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const user = useSyncExternalStore(
    (fn) => session.subscribe(fn),
    () => session.getUser(),
  );
  const [ready, setReady] = useState(() => !session.hasRefresh());
  const navigate = useNavigate();
  const toast = useToast();
  const queryClient = useQueryClient();

  // После перезагрузки страницы восстанавливаем сессию по refresh-токену.
  useEffect(() => {
    if (ready) return;
    void session.refresh().finally(() => setReady(true));
  }, [ready]);

  useEffect(
    () =>
      session.onExpired(() => {
        queryClient.clear();
        toast.error('Сессия истекла, войдите снова');
        navigate('/login', { replace: true });
      }),
    [navigate, toast, queryClient],
  );

  const login = useCallback(async (email: string, password: string) => {
    const res = await authApi.login(email, password);
    session.start(res.tokens, res.user);
    return session.getUser()!;
  }, []);

  // Чистим хранилище даже при ошибке сети: logout идемпотентен на бэкенде.
  const logout = useCallback(async () => {
    const token = session.refreshToken();
    session.clear();
    queryClient.clear();
    navigate('/login', { replace: true });
    if (token) {
      try {
        await authApi.logout(token);
      } catch {
        // сессия на клиенте уже забыта
      }
    }
  }, [navigate, queryClient]);

  const value = useMemo(() => ({ user, ready, login, logout }), [user, ready, login, logout]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth outside AuthProvider');
  return ctx;
}

// Для экранов под RoleGuard пользователь гарантирован.
export function useUser(): SessionUser {
  const { user } = useAuth();
  if (!user) throw new Error('no user in protected route');
  return user;
}
