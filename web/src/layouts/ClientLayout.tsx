import { useState } from 'react';
import { Navigate, NavLink, Outlet, useLocation } from 'react-router-dom';
import { ErrorBox } from '../components/Feedback';
import { FullScreenLoader } from '../components/Spinner';
import { useCustomerStatus } from '../features/client/queries';
import { Header, navClass, type NavItem } from './Header';
import { NotificationsBell } from './NotificationsBell';

const ACTIVE_NAV: NavItem[] = [
  { to: '/app', label: 'Обзор', end: true },
  { to: '/app/accounts', label: 'Счета' },
  { to: '/app/transfer', label: 'Перевод' },
  { to: '/app/profile', label: 'Профиль' },
];

const ONBOARDING_NAV: NavItem[] = [
  { to: '/app/onboarding', label: 'Верификация' },
  { to: '/app/profile', label: 'Профиль' },
];

// Пока клиент не active, доступны только онбординг, профиль и уведомления.
const OPEN_FOR_ALL = ['/app/onboarding', '/app/profile', '/app/notifications'];

export function ClientLayout() {
  const status = useCustomerStatus();
  const location = useLocation();
  const [menuOpen, setMenuOpen] = useState(false);

  const active = status.data?.status === 'active';
  const nav = active ? ACTIVE_NAV : ONBOARDING_NAV;

  let content = <Outlet />;
  if (status.isLoading) content = <FullScreenLoader />;
  else if (status.isError) content = <ErrorBox error={status.error} onRetry={() => void status.refetch()} />;
  else if (!active && !OPEN_FOR_ALL.some((p) => location.pathname.startsWith(p)))
    content = <Navigate to="/app/onboarding" replace />;

  return (
    <div className="min-h-screen">
      <Header
        homeTo="/app"
        nav={nav}
        right={<NotificationsBell />}
        menuOpen={menuOpen}
        onMenuToggle={() => setMenuOpen((v) => !v)}
      />
      {menuOpen && (
        <nav className="border-b bg-elevated p-2 md:hidden" onClick={() => setMenuOpen(false)}>
          {nav.map((n) => (
            <NavLink key={n.to} to={n.to} end={n.end} className={navClass}>
              {n.label}
            </NavLink>
          ))}
        </nav>
      )}
      <main className="mx-auto w-full max-w-[1200px] px-4 py-6 md:py-8">{content}</main>
    </div>
  );
}
