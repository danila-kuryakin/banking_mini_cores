import clsx from 'clsx';
import { Banknote, ClipboardCheck, ShieldAlert, ShieldBan, UserPlus, Users } from 'lucide-react';
import { useState } from 'react';
import { NavLink, Outlet } from 'react-router-dom';
import { useUser } from '../auth/AuthContext';
import { Header, navClass, type NavItem } from './Header';

const NAV: (NavItem & { adminOnly?: boolean })[] = [
  { to: '/staff/kyc', label: 'Очередь KYC', icon: <ClipboardCheck size={18} /> },
  { to: '/staff/customers', label: 'Клиенты', icon: <Users size={18} /> },
  { to: '/staff/cash', label: 'Касса', icon: <Banknote size={18} /> },
  { to: '/staff/sanctions', label: 'Санкции', icon: <ShieldBan size={18} /> },
  { to: '/staff/antifraud', label: 'Антифрод', icon: <ShieldAlert size={18} /> },
  { to: '/staff/officers', label: 'Сотрудники', icon: <UserPlus size={18} />, adminOnly: true },
];

export function StaffLayout() {
  const user = useUser();
  const [open, setOpen] = useState(false);
  const items = NAV.filter((n) => !n.adminOnly || user.role === 'admin');

  return (
    <div className="min-h-screen">
      <Header homeTo="/staff/kyc" menuOpen={open} onMenuToggle={() => setOpen((v) => !v)} />
      <div className="flex">
        <aside
          className={clsx(
            'fixed inset-y-14 left-0 z-30 w-[220px] shrink-0 border-r bg-elevated p-3 transition-transform md:sticky md:top-14 md:h-[calc(100vh-56px)] md:translate-x-0',
            open ? 'translate-x-0' : '-translate-x-full',
          )}
          onClick={() => setOpen(false)}
        >
          <nav className="flex flex-col gap-1">
            {items.map((n) => (
              <NavLink
                key={n.to}
                to={n.to}
                className={(s) => clsx(navClass(s), s.isActive && 'bg-input')}
              >
                {n.icon}
                {n.label}
              </NavLink>
            ))}
          </nav>
        </aside>
        {open && <div className="fixed inset-0 top-14 z-20 bg-black/50 md:hidden" onClick={() => setOpen(false)} />}
        <main className="min-w-0 flex-1 px-4 py-6 md:px-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
