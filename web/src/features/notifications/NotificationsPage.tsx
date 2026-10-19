import { useQuery } from '@tanstack/react-query';
import { Bell } from 'lucide-react';
import { notificationsApi } from '../../api/notifications';
import type { Notification } from '../../api/types';
import { EmptyState, ErrorBox } from '../../components/Feedback';
import { PageHeader } from '../../components/Misc';
import { Pagination, usePage } from '../../components/Pagination';
import { Table, type Column } from '../../components/Table';
import { formatDateTime } from '../../lib/dates';
import { qk } from '../client/queries';

const columns: Column<Notification>[] = [
  { key: 'date', header: 'Время', render: (n) => <span className="num text-xs">{formatDateTime(n.created_at)}</span> },
  { key: 'text', header: 'Уведомление', render: (n) => n.text },
];

// Список уведомлений клиента; у сотрудника тот же компонент на вкладке карточки клиента.
export function NotificationsList({ userId, pageKey = '' }: { userId: string; pageKey?: string }) {
  const { limit, offset, setOffset } = usePage(pageKey);
  const query = useQuery({
    queryKey: [...qk.notifications(userId), limit, offset],
    queryFn: async () => (await notificationsApi.list(userId, { limit, offset })).notifications ?? [],
    placeholderData: (prev) => prev,
  });
  if (query.isError) return <ErrorBox error={query.error} onRetry={() => void query.refetch()} />;
  return (
    <>
      <Table
        columns={columns}
        rows={query.data}
        rowKey={(n) => n.id}
        loading={query.isLoading}
        empty={<EmptyState icon={<Bell size={36} strokeWidth={1.5} />} text="Уведомлений пока нет" />}
      />
      <Pagination limit={limit} offset={offset} count={query.data?.length} onChange={setOffset} />
    </>
  );
}

export function NotificationsPage() {
  return (
    <>
      <PageHeader title="Уведомления" />
      <div className="card">
        <NotificationsList userId="me" />
      </div>
    </>
  );
}
