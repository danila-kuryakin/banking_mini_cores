import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Check, FileQuestion, ShieldCheck, ShieldX, X } from 'lucide-react';
import { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { errorText } from '../../api/errors';
import { isApiError } from '../../api/http';
import { kycApi } from '../../api/kyc';
import type { Application, Check as KycCheck } from '../../api/types';
import { useUser } from '../../auth/AuthContext';
import { Badge, StatusBadge } from '../../components/Badge';
import { Button } from '../../components/Button';
import { Banner, EmptyState, ErrorBox } from '../../components/Feedback';
import { Textarea } from '../../components/Input';
import { Card, IdCell, KeyValue, Timeline } from '../../components/Misc';
import { Modal } from '../../components/Modal';
import { Skeleton } from '../../components/Spinner';
import { useToast } from '../../components/Toast';
import { formatDateTime, formatIsoDate } from '../../lib/dates';
import { CHECK_TYPE_LABELS, FILE_TYPE_LABELS, FILE_TYPES, statusInfo } from '../../lib/statuses';
import { useCustomer, useFiles } from '../client/queries';
import { DocumentsGallery } from '../staff/DocumentsList';

type Decision = 'approve' | 'reject' | 'docs' | null;

export function ApplicationPage() {
  const { applicationId = '' } = useParams();
  const query = useQuery({ queryKey: ['application', applicationId], queryFn: () => kycApi.get(applicationId) });

  if (query.isLoading) return <Skeleton className="h-96 w-full" />;
  if (query.isError) {
    return isApiError(query.error, 404) ? (
      <EmptyState text="Заявка не найдена" action={<Link to="/staff/kyc" className="link">К очереди</Link>} />
    ) : (
      <ErrorBox error={query.error} onRetry={() => void query.refetch()} />
    );
  }
  return <ApplicationView app={query.data!} />;
}

function ApplicationView({ app }: { app: Application }) {
  const user = useUser();
  const customer = useCustomer(app.user_id);
  const files = useFiles(app.user_id);
  const [decision, setDecision] = useState<Decision>(null);
  const p = customer.data?.profile;
  const canDecide = app.status === 'in_review';
  const events = [...(app.events ?? [])].sort((a, b) => a.created_at.localeCompare(b.created_at));

  return (
    <div className="space-y-4">
      <Link to="/staff/kyc" className="inline-flex items-center gap-1 text-sm text-muted hover:text-text">
        <ArrowLeft size={16} /> Очередь
      </Link>
      <div className="flex flex-wrap items-center gap-3">
        <h1 className="text-xl font-semibold">Заявка</h1>
        <IdCell id={app.id} />
        <StatusBadge status={app.status} kind="application" />
        {app.reviewer_id && (
          <span className="text-xs text-muted">
            Рассматривает: {app.reviewer_id === user.id ? <span className="text-accent">вы</span> : <IdCell id={app.reviewer_id} />}
          </span>
        )}
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[1fr_400px]">
        <div className="space-y-4">
          <Card
            title="Анкета клиента"
            actions={
              <Link to={`/staff/customers/${app.user_id}`} className="text-xs link">
                Карточка клиента
              </Link>
            }
          >
            {customer.isLoading && <Skeleton className="h-24 w-full" />}
            {customer.isError && <ErrorBox error={customer.error} />}
            {p && (
              <KeyValue
                items={[
                  ['Имя', p.first_name || '—'],
                  ['Фамилия', p.last_name || '—'],
                  ['Дата рождения', formatIsoDate(p.birth_date)],
                  ['Гражданство', p.citizenship || '—'],
                  ['Телефон', <span className="num">{p.phone || '—'}</span>],
                  ['Клиент', <IdCell id={app.user_id} />],
                ]}
              />
            )}
          </Card>
          <Card title="Документы">
            {files.isError ? <ErrorBox error={files.error} /> : <DocumentsGallery userId={app.user_id} files={files.data} />}
          </Card>
        </div>

        <div className="space-y-4">
          <Card title="Сведения">
            <KeyValue
              items={[
                ['Подана', formatDateTime(app.submitted_at)],
                ['Решение', formatDateTime(app.decided_at)],
                ...(app.reason ? ([['Комментарий / причина', app.reason]] as [string, string][]) : []),
                ...(app.requested_types?.length
                  ? ([['Запрошены', app.requested_types.map((t) => FILE_TYPE_LABELS[t] ?? t).join(', ')]] as [string, string][])
                  : []),
              ]}
            />
          </Card>
          <Card title="Авто-проверки">
            {(app.checks ?? []).length === 0 ? (
              <p className="text-sm text-muted">Проверок ещё не было</p>
            ) : (
              <ul className="space-y-3">{app.checks!.map((c) => <CheckRow key={c.id} check={c} />)}</ul>
            )}
          </Card>
          <Card title="История">
            <Timeline
              items={events.map((e) => ({
                id: e.id,
                title: (
                  <>
                    {e.from_status ? `${statusInfo(e.from_status, 'application').label} → ` : ''}
                    {statusInfo(e.to_status, 'application').label}
                  </>
                ),
                meta: (
                  <>
                    {formatDateTime(e.created_at)} ·{' '}
                    {e.actor_id ? e.actor_id === user.id ? 'вы' : <span className="num">{e.actor_id.slice(0, 8)}</span> : 'Система'}
                  </>
                ),
                body: e.comment,
                tone: statusInfo(e.to_status, 'application').tone,
              }))}
            />
          </Card>

          <div className="card sticky bottom-4 space-y-2 border p-4 shadow-xl">
            {!canDecide && <p className="text-xs text-muted">Решение доступно только для заявки на рассмотрении.</p>}
            <Button variant="success" block disabled={!canDecide} onClick={() => setDecision('approve')}>
              <Check size={16} /> Одобрить
            </Button>
            <div className="grid grid-cols-2 gap-2">
              <Button variant="danger" disabled={!canDecide} onClick={() => setDecision('reject')}>
                <X size={16} /> Отклонить
              </Button>
              <Button variant="secondary" disabled={!canDecide} onClick={() => setDecision('docs')}>
                <FileQuestion size={16} /> Документы
              </Button>
            </div>
          </div>
        </div>
      </div>
      <DecisionModal app={app} decision={decision} onClose={() => setDecision(null)} />
    </div>
  );
}

function CheckRow({ check }: { check: KycCheck }) {
  const passed = check.result === 'passed';
  return (
    <li className="flex items-start gap-3">
      {passed ? <ShieldCheck size={18} className="mt-0.5 text-up" /> : <ShieldX size={18} className="mt-0.5 text-down" />}
      <div className="min-w-0 flex-1">
        <div className="flex items-center justify-between gap-2 text-sm">
          {CHECK_TYPE_LABELS[check.type] ?? check.type}
          <Badge tone={passed ? 'up' : 'down'}>{statusInfo(check.result, 'check').label}</Badge>
        </div>
        {check.details && <p className="mt-0.5 break-words text-xs text-muted">{check.details}</p>}
      </div>
    </li>
  );
}

function DecisionModal({ app, decision, onClose }: { app: Application; decision: Decision; onClose: () => void }) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const [text, setText] = useState('');
  const [types, setTypes] = useState<string[]>([]);
  const [localError, setLocalError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: async () => {
      if (decision === 'approve') return kycApi.approve(app.id, text.trim());
      if (decision === 'reject') return kycApi.reject(app.id, text.trim());
      return kycApi.requestDocuments(app.id, types, text.trim());
    },
    onSuccess: () => {
      // Ответ решения приходит без проверок и журнала, поэтому карточку перечитываем.
      void queryClient.invalidateQueries({ queryKey: ['application', app.id] });
      void queryClient.invalidateQueries({ queryKey: ['applications'] });
      void queryClient.invalidateQueries({ queryKey: ['customer-status', app.user_id] });
      toast.success(
        decision === 'approve' ? 'Заявка одобрена' : decision === 'reject' ? 'Заявка отклонена' : 'Документы запрошены',
      );
      close();
    },
  });

  const close = () => {
    setText('');
    setTypes([]);
    setLocalError(null);
    mutation.reset();
    onClose();
  };

  const submit = () => {
    setLocalError(null);
    if (decision === 'reject' && !text.trim()) return setLocalError('Укажите причину');
    if (text.trim().length > 500) return setLocalError('Не длиннее 500 символов');
    if (decision === 'docs' && types.length === 0) return setLocalError('Выберите хотя бы один документ');
    mutation.mutate();
  };

  const title = decision === 'approve' ? 'Одобрить заявку' : decision === 'reject' ? 'Отклонить заявку' : 'Запросить документы';

  return (
    <Modal
      open={!!decision}
      onOpenChange={(v) => !v && close()}
      title={title}
      footer={
        <>
          <Button variant="secondary" onClick={close}>
            Отмена
          </Button>
          <Button
            variant={decision === 'approve' ? 'success' : decision === 'reject' ? 'danger' : 'primary'}
            loading={mutation.isPending}
            onClick={submit}
          >
            {decision === 'approve' ? 'Одобрить' : decision === 'reject' ? 'Отклонить' : 'Запросить'}
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        {decision === 'reject' && (
          <Banner tone="down" title="Отказ окончательный">
            Клиент не сможет подать новую заявку.
          </Banner>
        )}
        {decision === 'docs' && (
          <fieldset className="space-y-2">
            <legend className="label">Какие документы нужны</legend>
            {FILE_TYPES.map((t) => (
              <label key={t} className="flex cursor-pointer items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  className="accent-[rgb(var(--accent))]"
                  checked={types.includes(t)}
                  onChange={(e) => setTypes((cur) => (e.target.checked ? [...cur, t] : cur.filter((x) => x !== t)))}
                />
                {FILE_TYPE_LABELS[t]}
              </label>
            ))}
          </fieldset>
        )}
        <Textarea
          label={decision === 'reject' ? 'Причина отказа' : 'Комментарий (необязательно)'}
          value={text}
          maxLength={500}
          onChange={(e) => setText(e.target.value)}
          error={localError ?? undefined}
          hint={`${text.length}/500`}
        />
        {mutation.isError && (
          <ErrorBox
            error={mutation.error}
            text={errorText(mutation.error, { 400: 'Заявка уже не на рассмотрении - обновите страницу' })}
          />
        )}
      </div>
    </Modal>
  );
}
