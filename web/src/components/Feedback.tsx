import { AlertTriangle, ChevronDown, Inbox, RotateCw } from 'lucide-react';
import { useState } from 'react';
import type { ReactNode } from 'react';
import { errorText, type ErrorOverrides } from '../api/errors';
import { ApiError } from '../api/http';
import { Button } from './Button';

export function EmptyState({ icon, text, action }: { icon?: ReactNode; text: ReactNode; action?: ReactNode }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 px-4 py-12 text-center text-sm text-muted">
      <div className="text-muted/70">{icon ?? <Inbox size={36} strokeWidth={1.5} />}</div>
      <p>{text}</p>
      {action}
    </div>
  );
}

// Технические детали ошибки: текст бэкенда и X-Request-Id для поиска в логах.
export function ErrorDetails({ error }: { error: unknown }) {
  const [open, setOpen] = useState(false);
  if (!(error instanceof ApiError)) return null;
  return (
    <div className="mt-2 text-xs text-muted">
      <button type="button" className="inline-flex items-center gap-1 hover:text-text" onClick={() => setOpen((v) => !v)}>
        Подробнее <ChevronDown size={12} className={open ? 'rotate-180' : undefined} />
      </button>
      {open && (
        <pre className="mt-1 whitespace-pre-wrap break-all rounded bg-input p-2 font-mono text-[11px]">
          HTTP {error.status}: {error.message}
          {error.requestId ? `\nrequest_id: ${error.requestId}` : ''}
        </pre>
      )}
    </div>
  );
}

export function ErrorBox({
  error,
  overrides,
  onRetry,
  text,
}: {
  error: unknown;
  overrides?: ErrorOverrides;
  onRetry?: () => void;
  text?: string;
}) {
  return (
    <div className="rounded-lg border border-down/40 bg-down/10 px-4 py-3 text-sm">
      <div className="flex items-start gap-3">
        <AlertTriangle size={18} className="mt-0.5 shrink-0 text-down" />
        <div className="flex-1">
          <p>{text ?? errorText(error, overrides)}</p>
          <ErrorDetails error={error} />
        </div>
        {onRetry && (
          <Button size="sm" variant="secondary" onClick={onRetry}>
            <RotateCw size={14} /> Повторить
          </Button>
        )}
      </div>
    </div>
  );
}

export function Banner({
  tone = 'warn',
  title,
  children,
  action,
}: {
  tone?: 'warn' | 'down' | 'info' | 'up';
  title?: ReactNode;
  children?: ReactNode;
  action?: ReactNode;
}) {
  const cls = {
    warn: 'border-warn/40 bg-warn/10',
    down: 'border-down/40 bg-down/10',
    info: 'border-info/40 bg-info/10',
    up: 'border-up/40 bg-up/10',
  }[tone];
  return (
    <div className={`flex flex-col gap-3 rounded-lg border px-4 py-3 text-sm sm:flex-row sm:items-center ${cls}`}>
      <div className="flex-1">
        {title && <p className="font-medium">{title}</p>}
        {children && <div className="mt-1 text-muted">{children}</div>}
      </div>
      {action}
    </div>
  );
}
