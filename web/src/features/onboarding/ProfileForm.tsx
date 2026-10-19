import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { AlertTriangle } from 'lucide-react';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { customerApi } from '../../api/customer';
import { errorText } from '../../api/errors';
import { isApiError } from '../../api/http';
import type { Profile } from '../../api/types';
import { Button } from '../../components/Button';
import { ErrorBox } from '../../components/Feedback';
import { Input } from '../../components/Input';
import { COUNTRIES } from '../../lib/countries';
import { ageOn, isCalendarDate, profileSchema, type ProfileFormInput, type ProfileFormOutput } from '../../lib/validation';
import { qk } from '../client/queries';

const FIELDS: (keyof Profile)[] = ['first_name', 'last_name', 'birth_date', 'citizenship', 'phone'];

export function ProfileForm({
  profile,
  readOnly,
  onSaved,
  submitLabel = 'Сохранить',
}: {
  profile: Profile;
  readOnly?: boolean;
  onSaved?: (complete: boolean) => void;
  submitLabel?: string;
}) {
  const queryClient = useQueryClient();
  const [serverError, setServerError] = useState<unknown>(null);
  const { register, handleSubmit, formState, watch, setError } = useForm<ProfileFormInput, unknown, ProfileFormOutput>({
    resolver: zodResolver(profileSchema),
    defaultValues: profile,
  });

  const save = useMutation({
    mutationFn: (values: Profile) => customerApi.updateProfile('me', values),
    onSuccess: (customer) => {
      queryClient.setQueryData(qk.customer('me'), customer);
      void queryClient.invalidateQueries({ queryKey: qk.status('me') });
      onSaved?.(FIELDS.every((f) => customer.profile[f]));
    },
    onError: (err, values) => {
      setServerError(err);
      // Заполненную анкету нельзя разукомплектовать - подсвечиваем опустошённые поля.
      if (isApiError(err, 400) && err.message.includes('cannot be emptied')) {
        FIELDS.filter((f) => profile[f] && !values[f]).forEach((f) =>
          setError(f, { message: 'Поле нельзя очистить' }),
        );
      }
    },
  });

  const birth = watch('birth_date');
  const under18 = birth && isCalendarDate(birth) && ageOn(birth) < 18;
  const errs = formState.errors;

  return (
    <form
      noValidate
      onSubmit={handleSubmit((v) => {
        setServerError(null);
        save.mutate(v);
      })}
      className="space-y-4"
    >
      <fieldset disabled={readOnly} className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Input label="Имя" autoComplete="given-name" error={errs.first_name?.message} {...register('first_name')} />
        <Input label="Фамилия" autoComplete="family-name" error={errs.last_name?.message} {...register('last_name')} />
        <Input
          label="Дата рождения"
          type="date"
          max={new Date().toISOString().slice(0, 10)}
          error={errs.birth_date?.message}
          {...register('birth_date')}
        />
        <div>
          <Input
            label="Гражданство"
            list="countries"
            autoComplete="country-name"
            error={errs.citizenship?.message}
            {...register('citizenship')}
          />
          <datalist id="countries">
            {COUNTRIES.map((c) => (
              <option key={c} value={c} />
            ))}
          </datalist>
        </div>
        <Input
          label="Телефон"
          type="tel"
          autoComplete="tel"
          placeholder="+79991234567"
          error={errs.phone?.message}
          {...register('phone')}
        />
      </fieldset>
      {under18 && !readOnly && (
        <p className="flex items-center gap-2 text-xs text-warn">
          <AlertTriangle size={14} /> Верификацию проходят только с 18 лет - авто-проверка отклонит заявку.
        </p>
      )}
      {!!serverError && (
        <ErrorBox
          error={serverError}
          text={errorText(serverError, {
            409: 'Данные изменились, обновите страницу',
            match: [
              ['cannot be emptied', 'Заполненное поле нельзя очистить'],
              ['can only be edited', 'Анкету уже нельзя изменить'],
              ['phone', 'Неверный формат телефона'],
              ['birth_date', 'Неверная дата рождения'],
            ],
          })}
        />
      )}
      {!readOnly && (
        <div className="flex flex-wrap items-center gap-3">
          <Button type="submit" loading={save.isPending}>
            {submitLabel}
          </Button>
          <span className="text-xs text-muted">Можно сохранить частично и вернуться позже</span>
        </div>
      )}
    </form>
  );
}
