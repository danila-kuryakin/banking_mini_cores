import { useQuery } from '@tanstack/react-query';
import { Users } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { customerApi } from '../../api/customer';
import type { Customer } from '../../api/types';
import { StatusBadge } from '../../components/Badge';
import { EmptyState, ErrorBox } from '../../components/Feedback';
import { IdCell, PageHeader } from '../../components/Misc';
import { Pagination, usePage } from '../../components/Pagination';
import { Table, type Column } from '../../components/Table';
import { formatDateTime } from '../../lib/dates';

export function fullName(c: Customer | undefined): string {
  const n = [c?.profile.last_name, c?.profile.first_name].filter(Boolean).join(' ');
  return n || 'Без имени';
}

const columns: Column<Customer>[] = [
  {
    key: 'name',
    header: 'ФИО',
    render: (c) => <span className={c.profile.first_name ? 'font-medium' : 'text-muted'}>{fullName(c)}</span>,
  },
  { key: 'user', header: 'user_id', render: (c) => <IdCell id={c.user_id} /> },
  { key: 'status', header: 'Статус', render: (c) => <StatusBadge status={c.status} kind="customer" /> },
  { key: 'created', header: 'Создан', render: (c) => <span className="num text-xs">{formatDateTime(c.created_at)}</span> },
];

export function CustomersPage() {
  const navigate = useNavigate();
  const { limit, offset, setOffset } = usePage();
  const query = useQuery({
    queryKey: ['customers', limit, offset],
    queryFn: async () => (await customerApi.list({ limit, offset })).customers ?? [],
    placeholderData: (prev) => prev,
  });
  return (
    <>
      <PageHeader title="Клиенты" />
      <div className="card">
        {query.isError ? (
          <div className="p-4">
            <ErrorBox error={query.error} onRetry={() => void query.refetch()} />
          </div>
        ) : (
          <Table
            columns={columns}
            rows={query.data}
            rowKey={(c) => c.id}
            loading={query.isLoading}
            onRowClick={(c) => navigate(`/staff/customers/${c.user_id}`)}
            empty={<EmptyState icon={<Users size={36} strokeWidth={1.5} />} text="Клиентов нет" />}
          />
        )}
        <Pagination limit={limit} offset={offset} count={query.data?.length} onChange={setOffset} />
      </div>
    </>
  );
}
