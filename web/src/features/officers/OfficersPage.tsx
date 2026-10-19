import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { authApi } from '../../api/auth';
import { errorText } from '../../api/errors';
import { Button } from '../../components/Button';
import { ErrorBox } from '../../components/Feedback';
import { Input, PasswordInput } from '../../components/Input';
import { PageHeader } from '../../components/Misc';
import { useToast } from '../../components/Toast';
import { email, password } from '../../lib/validation';

const schema = z.object({ email, password });
type Form = z.infer<typeof schema>;

// Списка сотрудников в API нет - только форма создания.
export function OfficersPage() {
  const toast = useToast();
  const { register, handleSubmit, formState, reset } = useForm<Form>({ resolver: zodResolver(schema) });
  const mutation = useMutation({
    mutationFn: (v: Form) => authApi.createOfficer(v.email, v.password),
    onSuccess: (user) => {
      toast.success(`Сотрудник ${user.email} создан`);
      reset();
    },
  });

  return (
    <>
      <PageHeader title="Сотрудники" subtitle="Списка сотрудников API пока не отдаёт" />
      <div className="card max-w-md p-6">
        <h2 className="mb-4 font-semibold">Новый сотрудник</h2>
        <form className="space-y-4" noValidate onSubmit={handleSubmit((v) => mutation.mutate(v))}>
          <Input label="Email" type="email" autoComplete="off" error={formState.errors.email?.message} {...register('email')} />
          <PasswordInput
            label="Пароль"
            autoComplete="new-password"
            hint="От 8 до 72 символов"
            error={formState.errors.password?.message}
            {...register('password')}
          />
          {mutation.isError && (
            <ErrorBox error={mutation.error} text={errorText(mutation.error, { 409: 'Email уже занят' })} />
          )}
          <Button type="submit" loading={mutation.isPending}>
            Создать
          </Button>
        </form>
      </div>
    </>
  );
}
