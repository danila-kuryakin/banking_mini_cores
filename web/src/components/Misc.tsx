import * as RadixTooltip from '@radix-ui/react-tooltip';
import clsx from 'clsx';
import { Check, Copy } from 'lucide-react';
import { useState } from 'react';
import type { ReactNode } from 'react';
import { shortId } from '../lib/money';

export function CopyButton({ value, label = 'Скопировать' }: { value: string; label?: string }) {
  const [done, setDone] = useState(false);
  return (
    <button
      type="button"
      title={label}
      aria-label={label}
      className="inline-flex rounded p-1 text-muted hover:bg-input hover:text-text"
      onClick={(e) => {
        e.stopPropagation();
        void navigator.clipboard?.writeText(value).then(() => {
          setDone(true);
          setTimeout(() => setDone(false), 1500);
        });
      }}
    >
      {done ? <Check size={14} className="text-up" /> : <Copy size={14} />}
    </button>
  );
}

// Сокращённый id моноширинным шрифтом с копированием полного значения.
export function IdCell({ id }: { id: string | undefined }) {
  if (!id) return <span className="text-muted">—</span>;
  return (
    <span className="inline-flex items-center gap-1">
      <span className="num text-xs" title={id}>
        {shortId(id)}
      </span>
      <CopyButton value={id} label="Скопировать id" />
    </span>
  );
}

export function Tooltip({ content, children }: { content: ReactNode; children: ReactNode }) {
  if (!content) return <>{children}</>;
  return (
    <RadixTooltip.Root delayDuration={150}>
      <RadixTooltip.Trigger asChild>{children}</RadixTooltip.Trigger>
      <RadixTooltip.Portal>
        <RadixTooltip.Content
          sideOffset={6}
          className="z-[60] max-w-xs rounded-md border border-border bg-elevated px-3 py-2 text-xs shadow-xl"
        >
          {content}
        </RadixTooltip.Content>
      </RadixTooltip.Portal>
    </RadixTooltip.Root>
  );
}

export function Card({ title, actions, children, className, padded = true }: {
  title?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
  padded?: boolean;
}) {
  return (
    <section className={clsx('card', className)}>
      {(title || actions) && (
        <header className="flex items-center justify-between gap-3 border-b px-4 py-3">
          <h2 className="text-sm font-semibold">{title}</h2>
          {actions}
        </header>
      )}
      <div className={padded ? 'p-4' : undefined}>{children}</div>
    </section>
  );
}

export function PageHeader({ title, subtitle, actions }: { title: ReactNode; subtitle?: ReactNode; actions?: ReactNode }) {
  return (
    <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <h1 className="text-xl font-semibold">{title}</h1>
        {subtitle && <p className="mt-1 text-sm text-muted">{subtitle}</p>}
      </div>
      {actions && <div className="flex flex-wrap gap-2">{actions}</div>}
    </div>
  );
}

export function KeyValue({ items }: { items: [ReactNode, ReactNode][] }) {
  return (
    <dl className="grid grid-cols-1 gap-x-6 gap-y-3 text-sm sm:grid-cols-2">
      {items.map(([k, v], i) => (
        <div key={i} className="min-w-0">
          <dt className="text-xs text-muted">{k}</dt>
          <dd className="mt-0.5 break-words">{v}</dd>
        </div>
      ))}
    </dl>
  );
}

export interface TimelineItem {
  id: string;
  title: ReactNode;
  meta: ReactNode;
  body?: ReactNode;
  tone?: 'up' | 'down' | 'warn' | 'info' | 'muted';
}

export function Timeline({ items }: { items: TimelineItem[] }) {
  const dot = { up: 'bg-up', down: 'bg-down', warn: 'bg-warn', info: 'bg-info', muted: 'bg-muted' };
  return (
    <ol className="relative space-y-4 border-l border-border pl-5">
      {items.map((it) => (
        <li key={it.id} className="relative">
          <span className={clsx('absolute -left-[25px] top-1.5 h-2.5 w-2.5 rounded-full ring-4 ring-elevated', dot[it.tone ?? 'muted'])} />
          <div className="text-sm">{it.title}</div>
          <div className="text-xs text-muted">{it.meta}</div>
          {it.body && <div className="mt-1 rounded-md bg-input px-3 py-2 text-xs">{it.body}</div>}
        </li>
      ))}
    </ol>
  );
}
