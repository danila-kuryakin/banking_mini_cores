import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { z } from 'zod';
import { errorText } from '../../api/errors';
import { isApiError } from '../../api/http';
import { homeFor, useAuth } from '../../auth/AuthContext';
import { Button } from '../../components/Button';
import { ErrorBox } from '../../components/Feedback';
import { Input, PasswordInput } from '../../components/Input';
import { email } from '../../lib/validation';

// На входе длину пароля не проверяем: старые пароли могли заводиться по другим правилам.
const schema = z.object({ email, password: z.string().min(1, 'Введите пароль') });
type Form = z.infer<typeof schema>;

const LOCK_SECONDS = 30;

export function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [error, setError] = useState<unknown>(null);
  const [lockLeft, setLockLeft] = useState(0);
  const { register, handleSubmit, formState } = useForm<Form>({ resolver: zodResolver(schema) });

  useEffect(() => {
    if (lockLeft <= 0) return;
    const t = setTimeout(() => setLockLeft((s) => s - 1), 1000);
    return () => clearTimeout(t);
  }, [lockLeft]);

  const onSubmit = handleSubmit(async (values) => {
    setError(null);
    try {
      const user = await login(values.email, values.password);
      const from = (location.state as { from?: string } | null)?.from;
      const home = homeFor(user.role);
      navigate(from && from.startsWith(home) ? from : home, { replace: true });
    } catch (err) {
      setError(err);
      if (isApiError(err, 429)) setLockLeft(LOCK_SECONDS);
    }
  });

  return (
    <div className="card p-6 sm:p-8">
      <h1 className="text-2xl font-semibold">Вход</h1>
      <p className="mt-1 text-sm text-muted">Рады видеть вас снова</p>
      <form className="mt-6 space-y-4" onSubmit={onSubmit} noValidate>
        <Input label="Email" type="email" autoComplete="email" autoFocus error={formState.errors.email?.message} {...register('email')} />
        <PasswordInput label="Пароль" autoComplete="current-password" error={formState.errors.password?.message} {...register('password')} />
        {!!error && (
          <ErrorBox
            error={error}
            text={errorText(error, {
              401: 'Неверный email или пароль',
              429: 'Слишком много попыток. Попробуйте позже',
              400: 'Неверный email или пароль',
            })}
          />
        )}
        <Button type="submit" block loading={formState.isSubmitting} disabled={lockLeft > 0}>
          {lockLeft > 0 ? `Войти (${lockLeft} с)` : 'Войти'}
        </Button>
      </form>
      <p className="mt-6 text-center text-sm text-muted">
        Нет аккаунта?{' '}
        <Link to="/register" className="link">
          Создать аккаунт
        </Link>
      </p>
    </div>
  );
}
