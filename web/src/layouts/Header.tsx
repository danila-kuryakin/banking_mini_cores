import clsx from 'clsx';
import { LogOut, Menu, Moon, Sun, X } from 'lucide-react';
import type { ReactNode } from 'react';
import { Link, NavLink } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';
import { ROLE_LABELS } from '../lib/statuses';
import { useTheme } from '../lib/theme';

export interface NavItem {
  to: string;
  label: string;
  icon?: ReactNode;
  end?: boolean;
}

export function Logo({ to = '/' }: { to?: string }) {
  return (
    <Link to={to} className="flex items-center gap-2 font-semibold">
      <img src="/favicon.svg" alt="" className="h-7 w-7" />
      <span className="hidden sm:inline">
        Mini<span className="text-accent">Bank</span>
      </span>
    </Link>
  );
}

export function ThemeToggle() {
  const { theme, toggle } = useTheme();
  return (
    <button
      onClick={toggle}
      className="rounded-lg p-2 text-muted hover:bg-input hover:text-text"
      aria-label={theme === 'dark' ? 'Светлая тема' : 'Тёмная тема'}
      title={theme === 'dark' ? 'Светлая тема' : 'Тёмная тема'}
    >
      {theme === 'dark' ? <Sun size={18} /> : <Moon size={18} />}
    </button>
  );
}

export function navClass({ isActive }: { isActive: boolean }) {
  return clsx(
    'flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
    isActive ? 'text-accent' : 'text-muted hover:text-text',
  );
}

export function Header({
  homeTo,
  nav,
  right,
  menuOpen,
  onMenuToggle,
}: {
  homeTo: string;
  nav?: NavItem[];
  right?: ReactNode;
  menuOpen?: boolean;
  onMenuToggle?: () => void;
}) {
  const { user, logout } = useAuth();
  return (
    <header className="sticky top-0 z-40 flex h-14 items-center gap-4 border-b bg-elevated px-4">
      {onMenuToggle && (
        <button
          className="rounded-lg p-2 text-muted hover:bg-input md:hidden"
          onClick={onMenuToggle}
          aria-label="Меню"
          aria-expanded={menuOpen}
        >
          {menuOpen ? <X size={20} /> : <Menu size={20} />}
        </button>
      )}
      <Logo to={homeTo} />
      {nav && (
        <nav className="ml-4 hidden items-center gap-1 md:flex">
          {nav.map((n) => (
            <NavLink key={n.to} to={n.to} end={n.end} className={navClass}>
              {n.label}
            </NavLink>
          ))}
        </nav>
      )}
      <div className="ml-auto flex items-center gap-1">
        {right}
        <ThemeToggle />
        {user && (
          <div className="ml-2 hidden items-center gap-2 border-l pl-3 sm:flex">
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-accent/20 text-xs font-semibold uppercase text-accent">
              {(user.email || '?').slice(0, 1)}
            </div>
            <div className="leading-tight">
              <div className="max-w-[180px] truncate text-xs">{user.email || 'Пользователь'}</div>
              <div className="text-[11px] text-muted">{ROLE_LABELS[user.role]}</div>
            </div>
          </div>
        )}
        <button
          onClick={() => void logout()}
          className="ml-1 rounded-lg p-2 text-muted hover:bg-input hover:text-text"
          aria-label="Выйти"
          title="Выйти"
        >
          <LogOut size={18} />
        </button>
      </div>
    </header>
  );
}
