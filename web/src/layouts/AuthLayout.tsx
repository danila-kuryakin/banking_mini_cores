import { Outlet } from 'react-router-dom';
import { Logo, ThemeToggle } from './Header';

export function AuthLayout() {
  return (
    <div className="flex min-h-screen">
      <div className="relative hidden flex-1 overflow-hidden bg-elevated lg:block">
        <div className="absolute -left-40 -top-40 h-[520px] w-[520px] rounded-full bg-accent/20 blur-[120px]" />
        <div className="absolute -bottom-40 right-0 h-[420px] w-[420px] rounded-full bg-info/20 blur-[120px]" />
        <div className="relative flex h-full flex-col justify-between p-12">
          <Logo />
          <div>
            <h2 className="max-w-md text-4xl font-semibold leading-tight">
              Счета, переводы и верификация <span className="text-accent">в одном месте</span>
            </h2>
            <p className="mt-4 max-w-md text-muted">
              Пройдите KYC за три шага, откройте счёт в рублях, долларах или евро и переводите деньги мгновенно.
            </p>
            <div className="mt-10 grid max-w-md grid-cols-3 gap-4">
              {[
                ['3', 'валюты'],
                ['24/7', 'переводы'],
                ['0 ₽', 'за открытие'],
              ].map(([v, l]) => (
                <div key={l} className="rounded-xl border border-border bg-bg/40 p-4">
                  <div className="num text-xl font-semibold text-accent">{v}</div>
                  <div className="text-xs text-muted">{l}</div>
                </div>
              ))}
            </div>
          </div>
          <p className="text-xs text-muted">Учебный стенд Banking Mini Cores</p>
        </div>
      </div>
      <div className="flex flex-1 flex-col">
        <div className="flex items-center justify-between p-4 lg:justify-end">
          <div className="lg:hidden">
            <Logo />
          </div>
          <ThemeToggle />
        </div>
        <div className="flex flex-1 items-center justify-center px-4 pb-16">
          <div className="w-full max-w-[400px]">
            <Outlet />
          </div>
        </div>
      </div>
    </div>
  );
}
