import clsx from 'clsx';
import type { ReactNode } from 'react';

interface Tab<K extends string> {
  key: K;
  label: ReactNode;
}

// Подчёркивание жёлтой полосой под активной вкладкой.
export function Tabs<K extends string>({
  tabs,
  value,
  onChange,
  className,
}: {
  tabs: Tab<K>[];
  value: K;
  onChange: (key: K) => void;
  className?: string;
}) {
  return (
    <div role="tablist" className={clsx('flex gap-6 overflow-x-auto overflow-y-hidden border-b', className)}>
      {tabs.map((t) => (
        <button
          key={t.key}
          role="tab"
          aria-selected={t.key === value}
          onClick={() => onChange(t.key)}
          className={clsx(
            '-mb-px whitespace-nowrap border-b-2 pb-3 pt-1 text-sm font-medium transition-colors',
            t.key === value ? 'border-accent text-text' : 'border-transparent text-muted hover:text-text',
          )}
        >
          {t.label}
        </button>
      ))}
    </div>
  );
}
