import { zodResolver } from '@hookform/resolvers/zod';
import clsx from 'clsx';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { Link, useNavigate } from 'react-router-dom';
import { z } from 'zod';
import { authApi } from '../../api/auth';
import { errorText } from '../../api/errors';
import { useAuth } from '../../auth/AuthContext';
import { Button } from '../../components/Button';
import { ErrorBox } from '../../components/Feedback';
import { Input, PasswordInput } from '../../components/Input';
import { email, password, passwordStrength } from '../../lib/validation';

const schema = z
  .object({ email, password, confirm: z.string() })
  .refine((v) => v.password === v.confirm, { path: ['confirm'], message: 'Пароли не совпадают' });
type Form = z.infer<typeof schema>;

const STRENGTH = [
  { label: 'Слабый', cls: 'bg-down' },
  { label: 'Слабый', cls: 'bg-down' },
  { label: 'Средний', cls: 'bg-warn' },
  { label: 'Хороший', cls: 'bg-up' },
  { label: 'Надёжный', cls: 'bg-up' },
];

export function RegisterPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [error, setError] = useState<unknown>(null);
  const { register, handleSubmit, formState, watch } = useForm<Form>({ resolver: zodResolver(schema) });
  const pwd = watch('password') ?? '';
  const strength = passwordStrength(pwd);

  // После регистрации сразу входим теми же данными и ведём в онбординг.
  const onSubmit = handleSubmit(async (values) => {
    setError(null);
    try {
      await authApi.register(values.email, values.password);
      await login(values.email, values.password);
      navigate('/app/onboarding', { replace: true });
    } catch (err) {
      setError(err);
    }
  });

  return (
    <div className="card p-6 sm:p-8">
      <h1 className="text-2xl font-semibold">Создать аккаунт</h1>
      <p className="mt-1 text-sm text-muted">Регистрация займёт минуту</p>
      <form className="mt-6 space-y-4" onSubmit={onSubmit} noValidate>
        <Input label="Email" type="email" autoComplete="email" autoFocus error={formState.errors.email?.message} {...register('email')} />
        <div>
          <PasswordInput
            label="Пароль"
            autoComplete="new-password"
            hint="От 8 до 72 символов"
            error={formState.errors.password?.message}
            {...register('password')}
          />
          {pwd && (
            <div className="mt-2 flex items-center gap-2">
              <div className="flex flex-1 gap-1">
                {[1, 2, 3, 4].map((i) => (
                  <div
                    key={i}
                    className={clsx('h-1 flex-1 rounded-full', i <= Math.max(strength, 1) ? STRENGTH[strength]!.cls : 'bg-input')}
                  />
                ))}
              </div>
              <span className="text-xs text-muted">{STRENGTH[strength]!.label}</span>
            </div>
          )}
        </div>
        <PasswordInput
          label="Повторите пароль"
          autoComplete="new-password"
          error={formState.errors.confirm?.message}
          {...register('confirm')}
        />
        {!!error && (
          <ErrorBox
            error={error}
            text={errorText(error, {
              409: 'Email уже зарегистрирован',
              400: 'Проверьте email и пароль',
              429: 'Сервер перегружен. Попробуйте через минуту',
            })}
          />
        )}
        <Button type="submit" block loading={formState.isSubmitting}>
          Зарегистрироваться
        </Button>
      </form>
      <p className="mt-6 text-center text-sm text-muted">
        Уже есть аккаунт?{' '}
        <Link to="/login" className="link">
          Войти
        </Link>
      </p>
    </div>
  );
}
