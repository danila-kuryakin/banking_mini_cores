import clsx from 'clsx';
import type { ReactNode } from 'react';
import { statusInfo, type StatusKind, type Tone } from '../lib/statuses';

const TONES: Record<Tone, string> = {
  up: 'bg-up/15 text-up',
  down: 'bg-down/15 text-down',
  warn: 'bg-warn/15 text-warn',
  info: 'bg-info/15 text-info',
  muted: 'bg-muted/15 text-muted',
};

export function Badge({ tone = 'muted', children, className }: { tone?: Tone; children: ReactNode; className?: string }) {
  return (
    <span
      className={clsx(
        'inline-flex h-6 items-center whitespace-nowrap rounded-md px-2 text-xs font-medium',
        TONES[tone],
        className,
      )}
    >
      {children}
    </span>
  );
}

export function StatusBadge({ status, kind }: { status: string; kind?: StatusKind }) {
  const info = statusInfo(status, kind);
  return <Badge tone={info.tone}>{info.label}</Badge>;
}
