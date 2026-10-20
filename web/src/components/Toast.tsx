import clsx from 'clsx';
import { CheckCircle2, X, XCircle } from 'lucide-react';
import { createContext, useCallback, useContext, useMemo, useRef, useState } from 'react';
import type { ReactNode } from 'react';

type Kind = 'success' | 'error';

interface Item {
  id: number;
  kind: Kind;
  text: string;
}

interface ToastApi {
  success: (text: string) => void;
  error: (text: string) => void;
}

const ToastContext = createContext<ToastApi | null>(null);

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<Item[]>([]);
  const seq = useRef(0);

  const remove = useCallback((id: number) => setItems((list) => list.filter((t) => t.id !== id)), []);

  const push = useCallback(
    (kind: Kind, text: string) => {
      const id = ++seq.current;
      setItems((list) => [...list.slice(-3), { id, kind, text }]);
      setTimeout(() => remove(id), 5000);
    },
    [remove],
  );

  const api = useMemo<ToastApi>(
    () => ({ success: (t) => push('success', t), error: (t) => push('error', t) }),
    [push],
  );

  return (
    <ToastContext.Provider value={api}>
      {children}
      <div className="fixed right-4 top-4 z-[100] flex w-[min(360px,calc(100vw-32px))] flex-col gap-2" aria-live="polite">
        {items.map((t) => (
          <div
            key={t.id}
            role="status"
            className={clsx(
              'flex items-start gap-3 rounded-lg border-l-4 bg-elevated px-4 py-3 text-sm shadow-xl shadow-black/30',
              t.kind === 'success' ? 'border-up' : 'border-down',
            )}
          >
            {t.kind === 'success' ? (
              <CheckCircle2 size={18} className="mt-0.5 shrink-0 text-up" />
            ) : (
              <XCircle size={18} className="mt-0.5 shrink-0 text-down" />
            )}
            <span className="flex-1">{t.text}</span>
            <button className="text-muted hover:text-text" onClick={() => remove(t.id)} aria-label="Закрыть">
              <X size={16} />
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast(): ToastApi {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error('useToast outside ToastProvider');
  return ctx;
}
