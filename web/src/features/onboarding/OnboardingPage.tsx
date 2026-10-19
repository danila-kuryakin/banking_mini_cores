import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import clsx from 'clsx';
import { Ban, Check, CheckCircle2, Clock, ShieldX } from 'lucide-react';
import { useState } from 'react';
import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { errorText } from '../../api/errors';
import { isApiError } from '../../api/http';
import { filesApi } from '../../api/files';
import { kycApi } from '../../api/kyc';
import type { Application, FileRecord } from '../../api/types';
import { StatusBadge } from '../../components/Badge';
import { Button } from '../../components/Button';
import { Banner, ErrorBox } from '../../components/Feedback';
import { Timeline } from '../../components/Misc';
import { Skeleton } from '../../components/Spinner';
import { formatDateTime } from '../../lib/dates';
import { FILE_TYPE_LABELS, FILE_TYPES, statusInfo, TERMINAL_APPLICATION_STATUSES } from '../../lib/statuses';
import { qk, useApplication, useCustomer, useCustomerStatus, useFiles } from '../client/queries';
import { FileSlot } from './FileSlot';
import { ProfileForm } from './ProfileForm';

type Screen = 'profile' | 'documents' | 'resubmit' | 'review' | 'approved' | 'rejected' | 'blocked';

// Экран выводится из статуса клиента и последней заявки (таблица §4.1 спецификации).
function screenFor(status: string, app: Application | null): Screen {
  switch (status) {
    case 'new':
      return 'profile';
    case 'profile_filled':
      return 'documents';
    case 'on_kyc':
      return app?.status === 'docs_requested' ? 'resubmit' : 'review';
    case 'active':
      return 'approved';
    case 'rejected':
      return 'rejected';
    case 'blocked':
      return 'blocked';
    default:
      return 'profile';
  }
}

const STEPS = ['Личные данные', 'Документы', 'Проверка'];

function stepIndex(screen: Screen): number {
  if (screen === 'profile') return 0;
  if (screen === 'documents' || screen === 'resubmit') return 1;
  if (screen === 'review') return 2;
  return 3;
}

function Stepper({ current }: { current: number }) {
  return (
    <ol className="mb-8 flex items-center gap-2 sm:gap-4">
      {STEPS.map((label, i) => {
        const done = i < current;
        const active = i === current;
        return (
          <li key={label} className="flex flex-1 items-center gap-2 sm:gap-3">
            <span
              className={clsx(
                'flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-sm font-semibold',
                done && 'bg-up text-white',
                active && 'bg-accent text-accent-text',
                !done && !active && 'bg-input text-muted',
              )}
            >
              {done ? <Check size={16} /> : i + 1}
            </span>
            <span className={clsx('hidden text-sm sm:inline', active ? 'font-medium' : 'text-muted')}>{label}</span>
            {i < STEPS.length - 1 && <span className={clsx('h-px flex-1', done ? 'bg-up' : 'bg-border')} />}
          </li>
        );
      })}
    </ol>
  );
}

export function OnboardingPage() {
  const status = useCustomerStatus();
  const screenGuess = status.data ? screenFor(status.data.status, null) : null;
  const app = useApplication('me', screenGuess === 'review' || screenGuess === 'resubmit');

  if (status.isLoading || app.isLoading) {
    return (
      <div className="mx-auto max-w-3xl space-y-4">
        <Skeleton className="h-8 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }
  if (status.isError) return <ErrorBox error={status.error} onRetry={() => void status.refetch()} />;

  const screen = screenFor(status.data!.status, app.data ?? null);
  return (
    <div className="mx-auto max-w-3xl">
      <h1 className="mb-2 text-xl font-semibold">Верификация личности</h1>
      <p className="mb-6 text-sm text-muted">Пройдите три шага, чтобы открыть счёт и делать переводы.</p>
      {stepIndex(screen) < 3 && <Stepper current={stepIndex(screen)} />}
      {screen === 'profile' && <ProfileStep />}
      {(screen === 'documents' || screen === 'resubmit') && (
        <DocumentsStep application={app.data ?? null} resubmit={screen === 'resubmit'} />
      )}
      {screen === 'review' && app.data && <ReviewStep application={app.data} />}
      {screen === 'review' && !app.data && <ReviewPlaceholder />}
      {screen === 'approved' && (
        <Final
          icon={<CheckCircle2 size={56} className="text-up" />}
          title="Верификация пройдена"
          text="Теперь можно открыть счёт и делать переводы."
          action={
            <Link to="/app/accounts">
              <Button>Перейти к счетам</Button>
            </Link>
          }
        />
      )}
      {screen === 'rejected' && (
        <Final
          icon={<ShieldX size={56} className="text-down" />}
          title="В верификации отказано"
          text={app.data?.reason ? `Причина: ${app.data.reason}` : 'Решение окончательное.'}
        />
      )}
      {screen === 'blocked' && (
        <Final
          icon={<Ban size={56} className="text-down" />}
          title="Аккаунт заблокирован"
          text="Обратитесь в поддержку банка."
        />
      )}
    </div>
  );
}

function Final({ icon, title, text, action }: { icon: ReactNode; title: string; text: string; action?: ReactNode }) {
  return (
    <div className="card flex flex-col items-center gap-4 px-6 py-14 text-center">
      {icon}
      <h2 className="text-xl font-semibold">{title}</h2>
      <p className="max-w-md text-sm text-muted">{text}</p>
      {action}
    </div>
  );
}

function ProfileStep() {
  const customer = useCustomer();
  if (customer.isLoading) return <Skeleton className="h-64 w-full" />;
  if (customer.isError) return <ErrorBox error={customer.error} onRetry={() => void customer.refetch()} />;
  return (
    <div className="card p-6">
      <h2 className="mb-1 font-semibold">Шаг 1. Личные данные</h2>
      <p className="mb-6 text-sm text-muted">Данные должны совпадать с паспортом.</p>
      <ProfileForm profile={customer.data!.profile} submitLabel="Сохранить и продолжить" />
    </div>
  );
}

function confirmedByType(files: FileRecord[] | undefined): Map<string, FileRecord> {
  const map = new Map<string, FileRecord>();
  (files ?? []).filter((f) => f.status === 'confirmed').forEach((f) => map.set(f.type, f));
  return map;
}

function DocumentsStep({ application, resubmit }: { application: Application | null; resubmit: boolean }) {
  const queryClient = useQueryClient();
  const files = useFiles();
  const required = useQuery({ queryKey: qk.required('me'), queryFn: () => filesApi.required('me') });
  const [editProfile, setEditProfile] = useState(false);
  const customer = useCustomer();

  const requested = resubmit ? (application?.requested_types ?? []) : [];
  const confirmed = confirmedByType(files.data);
  const requestedMissing = requested.filter((t) => !confirmed.has(t));
  const ready = !!required.data?.ok && requestedMissing.length === 0;

  const submit = useMutation({
    mutationFn: () => kycApi.submit('me'),
    onSuccess: (app) => {
      // Ответ подачи приходит без журнала событий, поэтому заявку перечитываем.
      queryClient.setQueryData(qk.kyc('me'), app);
      void queryClient.invalidateQueries({ queryKey: qk.kyc('me') });
      void queryClient.invalidateQueries({ queryKey: qk.status('me') });
    },
    onError: (err) => {
      // 409 - активная заявка уже есть, перечитываем статус.
      if (isApiError(err, 409)) {
        void queryClient.invalidateQueries({ queryKey: qk.status('me') });
        void queryClient.invalidateQueries({ queryKey: qk.kyc('me') });
      }
    },
  });

  const autoRejected = !resubmit && application?.status === 'auto_rejected';

  return (
    <div className="space-y-4">
      {autoRejected && (
        <Banner
          tone="down"
          title="Автоматическая проверка не пройдена"
          action={
            !editProfile && (
              <Button size="sm" variant="secondary" onClick={() => setEditProfile(true)}>
                Исправить анкету
              </Button>
            )
          }
        >
          {application?.reason || 'Проверьте данные анкеты и подайте заявку снова.'}
        </Banner>
      )}
      {resubmit && (
        <Banner tone="warn" title="Сотрудник запросил документы">
          {application?.reason && <p className="mb-1">{application.reason}</p>}
          <p>Нужно загрузить: {requested.map((t) => FILE_TYPE_LABELS[t] ?? t).join(', ') || '—'}</p>
        </Banner>
      )}
      {editProfile && customer.data && (
        <div className="card p-6">
          <h2 className="mb-4 font-semibold">Анкета</h2>
          <ProfileForm profile={customer.data.profile} onSaved={() => setEditProfile(false)} />
        </div>
      )}

      <div className="card p-6">
        <h2 className="mb-1 font-semibold">Шаг 2. Документы</h2>
        <p className="mb-6 text-sm text-muted">Паспорт и селфи с паспортом обязательны. Файл jpg, png или pdf до 10 МБ.</p>
        {files.isError && <ErrorBox error={files.error} onRetry={() => void files.refetch()} />}
        <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
          {FILE_TYPES.map((t) => (
            <FileSlot
              key={t}
              type={t}
              confirmed={confirmed.get(t)}
              required={t !== 'proof_of_address' || requested.includes(t)}
              requested={requested.includes(t)}
            />
          ))}
        </div>
      </div>

      {submit.isError && (
        <ErrorBox
          error={submit.error}
          text={errorText(submit.error, {
            409: 'У вас уже есть активная заявка',
            match: [
              ['documents are not confirmed', 'Загрузите все обязательные документы'],
              ['profile must be filled', 'Сначала заполните анкету полностью'],
            ],
          })}
        />
      )}
      <div className="flex flex-col items-start gap-2 sm:flex-row sm:items-center sm:justify-end">
        {!ready && (
          <span className="text-xs text-muted">
            Не хватает:{' '}
            {[...(required.data?.missing ?? []), ...requestedMissing]
              .filter((t, i, a) => a.indexOf(t) === i)
              .map((t) => FILE_TYPE_LABELS[t] ?? t)
              .join(', ') || '…'}
          </span>
        )}
        <Button disabled={!ready} loading={submit.isPending} onClick={() => submit.mutate()}>
          Отправить на проверку
        </Button>
      </div>
    </div>
  );
}

function ReviewPlaceholder() {
  return (
    <div className="card flex flex-col items-center gap-3 px-6 py-14 text-center">
      <Clock size={48} className="text-warn" />
      <h2 className="text-lg font-semibold">Заявка на проверке</h2>
    </div>
  );
}

function ReviewStep({ application }: { application: Application }) {
  const terminal = TERMINAL_APPLICATION_STATUSES.has(application.status);
  const events = [...(application.events ?? [])].sort((a, b) => a.created_at.localeCompare(b.created_at));
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-[1fr_320px]">
      <div className="card flex flex-col items-center gap-4 px-6 py-12 text-center">
        <div className="relative">
          <div className="absolute inset-0 animate-ping rounded-full bg-warn/20" />
          <div className="relative flex h-20 w-20 items-center justify-center rounded-full bg-warn/15">
            <Clock size={40} className="text-warn" />
          </div>
        </div>
        <h2 className="text-lg font-semibold">Шаг 3. Заявка на проверке</h2>
        <StatusBadge status={application.status} kind="application" />
        <p className="max-w-sm text-sm text-muted">
          Сначала проходят автоматические проверки, затем заявку рассматривает сотрудник. Страница обновится сама.
        </p>
        {!terminal && <p className="text-xs text-muted">Обновление каждые 10 секунд</p>}
      </div>
      <div className="card p-4">
        <h3 className="mb-4 text-sm font-semibold">Ход проверки</h3>
        <Timeline
          items={events.map((e) => ({
            id: e.id,
            title: statusInfo(e.to_status, 'application').label,
            meta: formatDateTime(e.created_at),
            body: e.comment,
            tone: statusInfo(e.to_status, 'application').tone,
          }))}
        />
      </div>
    </div>
  );
}
