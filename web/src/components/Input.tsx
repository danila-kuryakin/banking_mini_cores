import clsx from 'clsx';
import { Eye, EyeOff } from 'lucide-react';
import { forwardRef, useId, useState } from 'react';
import type { InputHTMLAttributes, ReactNode, SelectHTMLAttributes, TextareaHTMLAttributes } from 'react';

interface FieldProps {
  label?: ReactNode;
  error?: string;
  hint?: ReactNode;
}

function Wrapper({ id, label, error, hint, children }: FieldProps & { id: string; children: ReactNode }) {
  return (
    <div>
      {label && (
        <label htmlFor={id} className="label">
          {label}
        </label>
      )}
      {children}
      {error ? (
        <p className="mt-1 text-xs text-down" role="alert">
          {error}
        </p>
      ) : (
        hint && <p className="mt-1 text-xs text-muted">{hint}</p>
      )}
    </div>
  );
}

type InputProps = FieldProps & InputHTMLAttributes<HTMLInputElement> & { suffix?: ReactNode };

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { label, error, hint, className, id, suffix, ...rest },
  ref,
) {
  const autoId = useId();
  const fieldId = id ?? autoId;
  return (
    <Wrapper id={fieldId} label={label} error={error} hint={hint}>
      <div className="relative">
        <input
          ref={ref}
          id={fieldId}
          aria-invalid={!!error || undefined}
          className={clsx('field', error && 'field-error', suffix ? 'pr-12' : undefined, className)}
          {...rest}
        />
        {suffix && <div className="absolute inset-y-0 right-0 flex items-center pr-2">{suffix}</div>}
      </div>
    </Wrapper>
  );
});

export const PasswordInput = forwardRef<HTMLInputElement, Omit<InputProps, 'type' | 'suffix'>>(
  function PasswordInput(props, ref) {
    const [visible, setVisible] = useState(false);
    return (
      <Input
        ref={ref}
        type={visible ? 'text' : 'password'}
        suffix={
          <button
            type="button"
            className="rounded p-1.5 text-muted hover:text-text"
            onClick={() => setVisible((v) => !v)}
            aria-label={visible ? 'Скрыть пароль' : 'Показать пароль'}
          >
            {visible ? <EyeOff size={16} /> : <Eye size={16} />}
          </button>
        }
        {...props}
      />
    );
  },
);

type SelectProps = FieldProps & SelectHTMLAttributes<HTMLSelectElement>;

export const Select = forwardRef<HTMLSelectElement, SelectProps>(function Select(
  { label, error, hint, className, id, children, ...rest },
  ref,
) {
  const autoId = useId();
  const fieldId = id ?? autoId;
  return (
    <Wrapper id={fieldId} label={label} error={error} hint={hint}>
      <select
        ref={ref}
        id={fieldId}
        aria-invalid={!!error || undefined}
        className={clsx('field cursor-pointer appearance-none bg-no-repeat pr-8', error && 'field-error', className)}
        style={{
          backgroundImage:
            "url(\"data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%2381858C' stroke-width='2'%3E%3Cpath d='m6 9 6 6 6-6'/%3E%3C/svg%3E\")",
          backgroundPosition: 'right 12px center',
        }}
        {...rest}
      >
        {children}
      </select>
    </Wrapper>
  );
});

type TextareaProps = FieldProps & TextareaHTMLAttributes<HTMLTextAreaElement>;

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(function Textarea(
  { label, error, hint, className, id, ...rest },
  ref,
) {
  const autoId = useId();
  const fieldId = id ?? autoId;
  return (
    <Wrapper id={fieldId} label={label} error={error} hint={hint}>
      <textarea
        ref={ref}
        id={fieldId}
        aria-invalid={!!error || undefined}
        className={clsx('field h-auto min-h-[88px] py-2', error && 'field-error', className)}
        {...rest}
      />
    </Wrapper>
  );
});
