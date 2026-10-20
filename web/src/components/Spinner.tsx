import clsx from 'clsx';
import { Loader2 } from 'lucide-react';

export function Spinner({ className }: { className?: string }) {
  return <Loader2 className={clsx('animate-spin', className ?? 'h-5 w-5')} aria-label="Загрузка" />;
}

export function FullScreenLoader() {
  return (
    <div className="flex min-h-screen items-center justify-center text-accent">
      <Spinner className="h-8 w-8" />
    </div>
  );
}

export function Skeleton({ className }: { className?: string }) {
  return <div className={clsx('skeleton', className ?? 'h-4 w-full')} />;
}
