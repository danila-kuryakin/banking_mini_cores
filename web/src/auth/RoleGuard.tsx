import type { ReactNode } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import type { Role } from '../api/types';
import { FullScreenLoader } from '../components/Spinner';
import { homeFor, useAuth } from './AuthContext';

// Фронт лишь не показывает чужую зону; права всё равно проверяет гейтвей.
export function RoleGuard({ roles, children }: { roles: Role[]; children: ReactNode }) {
  const { user, ready } = useAuth();
  const location = useLocation();
  if (!ready) return <FullScreenLoader />;
  if (!user) return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />;
  if (!roles.includes(user.role)) return <Navigate to={homeFor(user.role)} replace />;
  return <>{children}</>;
}

// Страницы входа: уже вошедшего отправляем в его зону.
export function GuestOnly({ children }: { children: ReactNode }) {
  const { user, ready } = useAuth();
  if (!ready) return <FullScreenLoader />;
  if (user) return <Navigate to={homeFor(user.role)} replace />;
  return <>{children}</>;
}
