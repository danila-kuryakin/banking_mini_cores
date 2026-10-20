import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import clsx from 'clsx';
import { ClipboardList } from 'lucide-react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { errorText } from '../../api/errors';
import { isApiError } from '../../api/http';
import { kycApi, type QueueFilter } from '../../api/kyc';
import type { Application } from '../../api/types';
import { useUser } from '../../auth/AuthContext';
import { StatusBadge } from '../../components/Badge';
import { Button } from '../../components/Button';
import { EmptyState, ErrorBox } from '../../components/Feedback';
import { IdCell, PageHeader } from '../../components/Misc';
import { Pagination, usePage } from '../../components/Pagination';
import { Table, type Column } from '../../components/Table';
import { Tabs } from '../../components/Tabs';
import { useToast } from '../../components/Toast';
import { formatDateTime } from '../../lib/dates';
import { APPLICATION_STATUSES, statusInfo } from '../../lib/statuses';

type TabKey = 'free' | 'mine' | 'docs' | 'all';

const TABS: { key: TabKey; label: string }[] = [
  { key: 'free', label: 'Свободные' },
  { key: 'mine', label: 'Мои' },
  { key: 'docs', label: 'Ждут документов' },
  { key: 'all', label: 'Все' },
];

function filterFor(tab: TabKey, statuses: string[]): QueueFilter {
  switch (tab) {
    case 'free':
      return { status: ['in_review'], unassigned: true };
    case 'mine':
      return { status: ['in_review'], reviewer_id: 'me' };
    case 'docs':
      return { status: ['docs_requested'] };
    default:
      return { status: statuses };
  }
}

export function QueuePage() {
  const user = useUser();
  const navigate = useNavigate();
  const toast = useToast();
  const queryClient = useQueryClient();
  const [params, setParams] = useSearchParams();
  const tab = (TABS.find((t) => t.key === params.get('tab'))?.key ?? 'free') as TabKey;
  const statuses = params.getAll('status');
  const { limit, offset, setOffset } = usePage();
  const filter = filterFor(tab, statuses);

  const query = useQuery({
    queryKey: ['applications', filter, limit, offset],
    queryFn: async () => (await kycApi.list(filter, { limit, offset })).applications ?? [],
    placeholderData: (prev) => prev,
    refetchInterval: 30_000,
  });

  const claim = useMutation({
    mutationFn: (app: Application) => kycApi.claim(app.id),
    onSuccess: (app) => {
      toast.success('Заявка взята в работу');
      void queryClient.invalidateQueries({ queryKey: ['applications'] });
      navigate(`/staff/kyc/${app.id}`);
    },
    onError: (err) => {
      toast.error(
        isApiError(err, 400) ? 'Заявку уже взял другой сотрудник или она ушла из очереди' : errorText(err),
      );
      void queryClient.invalidateQueries({ queryKey: ['applications'] });
    },
  });

  const update = (fn: (p: URLSearchParams) => void) =>
    setParams(
      (p) => {
        const c = new URLSearchParams(p);
        c.delete('offset');
        fn(c);
        return c;
      },
      { replace: true },
    );

  const toggleStatus = (s: string) =>
    update((c) => {
      const cur = c.getAll('status');
      c.delete('status');
      (cur.includes(s) ? cur.filter((x) => x !== s) : [...cur, s]).forEach((x) => c.append('status', x));
    });

  const columns: Column<Application>[] = [
    { key: 'id', header: 'Заявка', render: (a) => <IdCell id={a.id} /> },
    { key: 'user', header: 'Клиент', render: (a) => <IdCell id={a.user_id} /> },
    { key: 'status', header: 'Статус', render: (a) => <StatusBadge status={a.status} kind="application" /> },
    { key: 'submitted', header: 'Подана', render: (a) => <span className="num text-xs">{formatDateTime(a.submitted_at)}</span> },
    {
      key: 'reviewer',
      header: 'Назначена',
      render: (a) =>
        a.reviewer_id ? (
          a.reviewer_id === user.id ? <span className="text-accent">Мне</span> : <IdCell id={a.reviewer_id} />
        ) : (
          <span className="text-muted">—</span>
        ),
    },
    {
      key: 'action',
      header: '',
      align: 'right',
      render: (a) =>
        a.status === 'in_review' && !a.reviewer_id ? (
          <Button
            size="sm"
            loading={claim.isPending && claim.variables?.id === a.id}
            disabled={claim.isPending}
            onClick={(e) => {
              e.stopPropagation();
              claim.mutate(a);
            }}
          >
            Взять
          </Button>
        ) : null,
    },
  ];

  return (
    <>
      <PageHeader title="Очередь KYC" subtitle="Заявки на верификацию клиентов" />
      <div className="card">
        <div className="px-4 pt-3">
          <Tabs tabs={TABS} value={tab}             onChange={(k) =>
              update((c) => {
                c.set('tab', k);
                c.delete('status');
              })
            }
          />
        </div>
        {tab === 'all' && (
          <div className="flex flex-wrap gap-2 border-b px-4 py-3">
            {APPLICATION_STATUSES.map((s) => (
              <button
                key={s}
                onClick={() => toggleStatus(s)}
                className={clsx(
                  'rounded-full border px-3 py-1 text-xs transition-colors',
                  statuses.includes(s) ? 'border-accent bg-accent/15 text-accent' : 'text-muted hover:text-text',
                )}
              >
                {statusInfo(s, 'application').label}
              </button>
            ))}
          </div>
        )}
        {query.isError ? (
          <div className="p-4">
            <ErrorBox error={query.error} onRetry={() => void query.refetch()} />
          </div>
        ) : (
          <Table
            columns={columns}
            rows={query.data}
            rowKey={(a) => a.id}
            loading={query.isLoading}
            onRowClick={(a) => navigate(`/staff/kyc/${a.id}`)}
            empty={<EmptyState icon={<ClipboardList size={36} strokeWidth={1.5} />} text="Заявок нет" />}
          />
        )}
        <Pagination limit={limit} offset={offset} count={query.data?.length} onChange={setOffset} />
      </div>
    </>
  );
}
