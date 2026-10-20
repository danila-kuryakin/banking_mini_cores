import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, Wallet } from 'lucide-react';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { accountsApi } from '../../api/accounts';
import { isApiError } from '../../api/http';
import { StatusBadge } from '../../components/Badge';
import { EmptyState, ErrorBox } from '../../components/Feedback';
import { Card, IdCell, KeyValue, Timeline } from '../../components/Misc';
import { Pagination, usePage } from '../../components/Pagination';
import { Skeleton } from '../../components/Spinner';
import { Tabs } from '../../components/Tabs';
import { formatDateTime, formatIsoDate } from '../../lib/dates';
import { FILE_TYPE_LABELS, statusInfo } from '../../lib/statuses';
import { AccountsTable } from '../accounts/AccountsTable';
import { qk, useApplication, useBalances, useCustomer, useCustomerStatus, useFiles } from '../client/queries';
import { NotificationsList } from '../notifications/NotificationsPage';
import { DocumentsList } from '../staff/DocumentsList';
import { fullName } from './CustomersPage';

type TabKey = 'profile' | 'docs' | 'kyc' | 'accounts' | 'notifications';

const TABS: { key: TabKey; label: string }[] = [
  { key: 'profile', label: 'Анкета' },
  { key: 'docs', label: 'Документы' },
  { key: 'kyc', label: 'KYC' },
  { key: 'accounts', label: 'Счета' },
  { key: 'notifications', label: 'Уведомления' },
];

export function CustomerPage() {
  const { userId = '' } = useParams();
  const [params, setParams] = useSearchParams();
  const tab = (TABS.find((t) => t.key === params.get('tab'))?.key ?? 'profile') as TabKey;
  const customer = useCustomer(userId);

  if (customer.isLoading) return <Skeleton className="h-64 w-full" />;
  if (customer.isError) {
    return isApiError(customer.error, 404) ? (
      <EmptyState text="Клиент не найден" action={<Link to="/staff/customers" className="link">К списку</Link>} />
    ) : (
      <ErrorBox error={customer.error} onRetry={() => void customer.refetch()} />
    );
  }

  return (
    <div className="space-y-4">
      <Link to="/staff/customers" className="inline-flex items-center gap-1 text-sm text-muted hover:text-text">
        <ArrowLeft size={16} /> Клиенты
      </Link>
      <div className="flex flex-wrap items-center gap-3">
        <h1 className="text-xl font-semibold">{fullName(customer.data)}</h1>
        <StatusBadge status={customer.data!.status} kind="customer" />
        <IdCell id={userId} />
      </div>
      <div className="card">
        <div className="px-4 pt-3">
          <Tabs tabs={TABS} value={tab} onChange={(k) => setParams({ tab: k }, { replace: true })} />
        </div>
        <div className={tab === 'profile' || tab === 'kyc' ? 'p-4' : undefined}>
          {tab === 'profile' && <ProfileTab userId={userId} />}
          {tab === 'docs' && <DocsTab userId={userId} />}
          {tab === 'kyc' && <KycTab userId={userId} />}
          {tab === 'accounts' && <AccountsTab userId={userId} />}
          {tab === 'notifications' && <NotificationsList userId={userId} />}
        </div>
      </div>
    </div>
  );
}

function ProfileTab({ userId }: { userId: string }) {
  const customer = useCustomer(userId);
  const status = useCustomerStatus(userId);
  const p = customer.data!.profile;
  return (
    <div className="space-y-6">
      <KeyValue
        items={[
          ['Имя', p.first_name || '—'],
          ['Фамилия', p.last_name || '—'],
          ['Дата рождения', formatIsoDate(p.birth_date)],
          ['Гражданство', p.citizenship || '—'],
          ['Телефон', <span className="num">{p.phone || '—'}</span>],
          ['Карточка создана', formatDateTime(customer.data!.created_at)],
          ['Статус', status.data ? <StatusBadge status={status.data.status} kind="customer" /> : '…'],
          ['Статус изменён', formatDateTime(status.data?.changed_at)],
        ]}
      />
      <p className="text-xs text-muted">История смены статусов клиента пока не отдаётся API.</p>
    </div>
  );
}

function DocsTab({ userId }: { userId: string }) {
  const files = useFiles(userId);
  if (files.isError) return <ErrorBox error={files.error} onRetry={() => void files.refetch()} />;
  return <DocumentsList userId={userId} files={files.data} loading={files.isLoading} />;
}

function KycTab({ userId }: { userId: string }) {
  const app = useApplication(userId);
  if (app.isLoading) return <Skeleton className="h-32 w-full" />;
  if (app.isError) return <ErrorBox error={app.error} onRetry={() => void app.refetch()} />;
  if (!app.data) return <EmptyState text="Клиент ещё не подавал заявку" />;
  const a = app.data;
  return (
    <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
      <div className="space-y-4">
        <KeyValue
          items={[
            ['Заявка', <IdCell id={a.id} />],
            ['Статус', <StatusBadge status={a.status} kind="application" />],
            ['Подана', formatDateTime(a.submitted_at)],
            ['Решение', formatDateTime(a.decided_at)],
            ['Причина / комментарий', a.reason || '—'],
            ['Запрошены', a.requested_types?.map((t) => FILE_TYPE_LABELS[t] ?? t).join(', ') || '—'],
          ]}
        />
        <Link to={`/staff/kyc/${a.id}`} className="inline-block text-sm link">
          Открыть заявку
        </Link>
      </div>
      <Card title="История" className="bg-input/40">
        <Timeline
          items={[...(a.events ?? [])]
            .sort((x, y) => x.created_at.localeCompare(y.created_at))
            .map((e) => ({
              id: e.id,
              title: statusInfo(e.to_status, 'application').label,
              meta: `${formatDateTime(e.created_at)} · ${e.actor_id ? e.actor_id.slice(0, 8) : 'Система'}`,
              body: e.comment,
              tone: statusInfo(e.to_status, 'application').tone,
            }))}
        />
      </Card>
    </div>
  );
}

function AccountsTab({ userId }: { userId: string }) {
  const navigate = useNavigate();
  const { limit, offset, setOffset } = usePage('acc_');
  const query = useQuery({
    queryKey: [...qk.accounts(userId), limit, offset],
    queryFn: async () => (await accountsApi.list(userId, { limit, offset })).accounts ?? [],
    placeholderData: (prev) => prev,
  });
  const balances = useBalances(userId, query.data);
  if (query.isError) return <ErrorBox error={query.error} onRetry={() => void query.refetch()} />;
  return (
    <>
      <AccountsTable
        accounts={query.data}
        balances={balances}
        loading={query.isLoading}
        onOpen={(a) => navigate(`/staff/accounts/${a.id}?user=${userId}`)}
        empty={<EmptyState icon={<Wallet size={36} strokeWidth={1.5} />} text="Счетов нет" />}
      />
      <Pagination limit={limit} offset={offset} count={query.data?.length} onChange={setOffset} />
    </>
  );
}
