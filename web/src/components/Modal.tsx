import * as Dialog from '@radix-ui/react-dialog';
import clsx from 'clsx';
import { X } from 'lucide-react';
import type { ReactNode } from 'react';

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: ReactNode;
  description?: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
  width?: string;
}

export function Modal({ open, onOpenChange, title, description, children, footer, width = 'max-w-[440px]' }: Props) {
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/70 backdrop-blur-[1px]" />
        <Dialog.Content
          className={clsx(
            'fixed left-1/2 top-1/2 z-50 max-h-[90vh] w-[calc(100vw-32px)] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border border-border bg-elevated p-6 shadow-2xl',
            width,
          )}
          {...(description ? {} : { 'aria-describedby': undefined })}
        >
          <div className="mb-4 flex items-start justify-between gap-4">
            <div>
              <Dialog.Title className="text-lg font-semibold">{title}</Dialog.Title>
              {description && <Dialog.Description className="mt-1 text-sm text-muted">{description}</Dialog.Description>}
            </div>
            <Dialog.Close className="rounded p-1 text-muted hover:bg-input hover:text-text" aria-label="Закрыть">
              <X size={18} />
            </Dialog.Close>
          </div>
          {children}
          {footer && <div className="mt-6 flex justify-end gap-2">{footer}</div>}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

// Боковая панель справа - для деталей транзакции.
export function Drawer({ open, onOpenChange, title, children }: Omit<Props, 'footer' | 'width'>) {
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/60" />
        <Dialog.Content
          className="fixed inset-y-0 right-0 z-50 w-full max-w-[480px] overflow-y-auto border-l border-border bg-elevated p-6 shadow-2xl"
          aria-describedby={undefined}
        >
          <div className="mb-6 flex items-center justify-between">
            <Dialog.Title className="text-lg font-semibold">{title}</Dialog.Title>
            <Dialog.Close className="rounded p-1 text-muted hover:bg-input hover:text-text" aria-label="Закрыть">
              <X size={18} />
            </Dialog.Close>
          </div>
          {children}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
