import { useQuery } from '@tanstack/react-query';
import { ReceiptText } from 'lucide-react';
import { accountsApi } from '../../api/accounts';
import type { Account, Transaction } from '../../api/types';
import { Amount } from '../../components/Amount';
import { StatusBadge } from '../../components/Badge';
import { EmptyState, ErrorBox } from '../../components/Feedback';
import { Tooltip } from '../../components/Misc';
import { Pagination, usePage } from '../../components/Pagination';
import { Table, type Column } from '../../components/Table';
import { formatDateTime } from '../../lib/dates';
import { formatAccountNumber, shortId } from '../../lib/money';
import { TX_TYPE_LABELS } from '../../lib/statuses';
import { qk } from '../client/queries';

// Контрагент: номер, если счёт свой; иначе сокращённый id; у пополнения - касса.
export function counterparty(tx: Transaction, accountId: string, own: Map<string, Account>): string {
  if (tx.type === 'deposit') return 'Касса';
  const other = tx.from_account_id === accountId ? tx.to_account_id : tx.from_account_id;
  const acc = own.get(other);
  return acc ? formatAccountNumber(acc.number) : `счёт ${shortId(other)}…`;
}

export function direction(tx: Transaction, accountId: string): 'in' | 'out' {
  return tx.to_account_id === accountId ? 'in' : 'out';
}

export function Statement({
  userId,
  account,
  ownAccounts,
  onRowClick,
}: {
  userId: string;
  account: Account;
  ownAccounts: Account[];
  onRowClick?: (tx: Transaction) => void;
}) {
  const { limit, offset, setOffset } = usePage();
  const query = useQuery({
    queryKey: [...qk.transactions(userId, account.id), limit, offset],
    queryFn: async () => (await accountsApi.transactions(userId, account.id, { limit, offset })).transactions ?? [],
    placeholderData: (prev) => prev,
  });
  const own = new Map(ownAccounts.map((a) => [a.id, a]));

  const columns: Column<Transaction>[] = [
    { key: 'date', header: 'Дата', render: (t) => <span className="num text-xs">{formatDateTime(t.created_at)}</span> },
    { key: 'type', header: 'Тип', render: (t) => TX_TYPE_LABELS[t.type] ?? t.type },
    {
      key: 'cp',
      header: 'Контрагент',
      render: (t) => <span className="num text-xs">{counterparty(t, account.id, own)}</span>,
    },
    {
      key: 'amount',
      header: 'Сумма',
      align: 'right',
      render: (t) => (
        <Amount
          amount={t.amount}
          currency={t.currency}
          signed={t.status === 'declined' ? undefined : direction(t, account.id)}
          className={t.status === 'declined' ? 'text-muted line-through' : undefined}
        />
      ),
    },
    {
      key: 'status',
      header: 'Статус',
      align: 'right',
      render: (t) =>
        t.status === 'declined' ? (
          <Tooltip content={t.decline_reason}>
            <span className="cursor-help">
              <StatusBadge status={t.status} kind="transaction" />
            </span>
          </Tooltip>
        ) : (
          <StatusBadge status={t.status} kind="transaction" />
        ),
    },
  ];

  if (query.isError) return <ErrorBox error={query.error} onRetry={() => void query.refetch()} />;
  return (
    <>
      <Table
        columns={columns}
        rows={query.data}
        rowKey={(t) => t.id}
        loading={query.isLoading}
        onRowClick={onRowClick}
        empty={<EmptyState icon={<ReceiptText size={36} strokeWidth={1.5} />} text="Операций пока нет" />}
      />
      <Pagination limit={limit} offset={offset} count={query.data?.length} onChange={setOffset} />
    </>
  );
}
