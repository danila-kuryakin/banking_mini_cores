import { useQueries } from '@tanstack/react-query';
import { ArrowLeftRight, ArrowRight, Eye, EyeOff, Plus, ReceiptText, Wallet } from 'lucide-react';
import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { accountsApi } from '../../api/accounts';
import type { Account, Transaction } from '../../api/types';
import { Amount, BigAmount } from '../../components/Amount';
import { StatusBadge } from '../../components/Badge';
import { Button } from '../../components/Button';
import { EmptyState, ErrorBox } from '../../components/Feedback';
import { Card } from '../../components/Misc';
import { Skeleton } from '../../components/Spinner';
import { Table, type Column } from '../../components/Table';
import { DEFAULT_CURRENCY } from '../../lib/currencies';
import { formatDateTime, timeOf } from '../../lib/dates';
import { readStorage, writeStorage } from '../../lib/storage';
import { TX_TYPE_LABELS } from '../../lib/statuses';
import { AccountsTable } from '../accounts/AccountsTable';
import { OpenAccountModal } from '../accounts/OpenAccountModal';
import { counterparty, direction } from '../accounts/Statement';
import { qk, useAllAccounts, useBalances } from '../client/queries';

const HIDE_KEY = 'bmc.hideBalances';

interface Row {
  tx: Transaction;
  account: Account;
}

export function DashboardPage() {
  const navigate = useNavigate();
  const [hidden, setHidden] = useState(() => readStorage(HIDE_KEY) === '1');
  const [openModal, setOpenModal] = useState(false);
  const accounts = useAllAccounts();
  const live = accounts.data?.filter((a) => a.status !== 'closed');
  const balances = useBalances('me', live);

  // Последние операции: выписки всех счетов параллельно, слить и отсортировать.
  const statements = useQueries({
    queries: (live ?? []).map((a) => ({
      queryKey: [...qk.transactions('me', a.id), 10, 0],
      queryFn: async () => (await accountsApi.transactions('me', a.id, { limit: 10, offset: 0 })).transactions ?? [],
    })),
  });
  const seen = new Set<string>();
  const recent: Row[] = [];
  statements.forEach((s, i) =>
    (s.data ?? []).forEach((tx) => {
      if (seen.has(tx.id)) return;
      seen.add(tx.id);
      recent.push({ tx, account: live![i]! });
    }),
  );
  recent.sort((a, b) => timeOf(b.tx.created_at) - timeOf(a.tx.created_at));
  const own = new Map((accounts.data ?? []).map((a) => [a.id, a]));

  // Сумма по каждой валюте отдельно, конвертации нет.
  const totals = new Map<string, number>();
  (live ?? []).forEach((a) => totals.set(a.currency, (totals.get(a.currency) ?? 0) + (balances.get(a.id) ?? 0)));
  const mainCurrency = totals.has(DEFAULT_CURRENCY) ? DEFAULT_CURRENCY : ([...totals.keys()][0] ?? DEFAULT_CURRENCY);
  const balancesLoading = (live ?? []).some((a) => balances.get(a.id) === undefined);

  const toggleHidden = () => {
    writeStorage(HIDE_KEY, hidden ? null : '1');
    setHidden(!hidden);
  };

  const columns: Column<Row>[] = [
    { key: 'date', header: 'Дата', render: (r) => <span className="num text-xs">{formatDateTime(r.tx.created_at)}</span> },
    { key: 'type', header: 'Тип', render: (r) => TX_TYPE_LABELS[r.tx.type] ?? r.tx.type },
    {
      key: 'cp',
      header: 'Контрагент',
      hideOnMobile: true,
      render: (r) => <span className="num text-xs">{counterparty(r.tx, r.account.id, own)}</span>,
    },
    {
      key: 'amount',
      header: 'Сумма',
      align: 'right',
      render: (r) => {
        // Перевод между своими счетами - без знака.
        const internal = r.tx.type === 'transfer' && own.has(r.tx.from_account_id) && own.has(r.tx.to_account_id);
        return (
          <Amount
            amount={r.tx.amount}
            currency={r.tx.currency}
            hidden={hidden}
            signed={r.tx.status === 'declined' || internal ? undefined : direction(r.tx, r.account.id)}
          />
        );
      },
    },
    { key: 'status', header: 'Статус', align: 'right', render: (r) => <StatusBadge status={r.tx.status} kind="transaction" /> },
  ];

  if (accounts.isError) return <ErrorBox error={accounts.error} onRetry={() => void accounts.refetch()} />;

  return (
    <div className="space-y-4">
      <div className="card p-6">
        <div className="flex flex-col gap-6 md:flex-row md:items-end md:justify-between">
          <div>
            <div className="mb-3 flex items-center gap-2 text-sm text-muted">
              Общий баланс
              <button onClick={toggleHidden} className="rounded p-1 hover:bg-input hover:text-text" aria-label="Скрыть балансы">
                {hidden ? <EyeOff size={16} /> : <Eye size={16} />}
              </button>
            </div>
            {accounts.isLoading || balancesLoading ? (
              <Skeleton className="h-8 w-56" />
            ) : (
              <BigAmount amount={totals.get(mainCurrency) ?? 0} currency={mainCurrency} hidden={hidden} />
            )}
            <div className="mt-2 flex flex-wrap gap-4 text-sm text-muted">
              {[...totals.entries()]
                .filter(([c]) => c !== mainCurrency)
                .map(([c, v]) => (
                  <span key={c} className="num">
                    {hidden ? '****' : <Amount amount={v} currency={c} />}
                  </span>
                ))}
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button onClick={() => navigate('/app/transfer')} disabled={!live?.some((a) => a.status === 'active')}>
              <ArrowLeftRight size={16} /> Перевести
            </Button>
            <Button variant="secondary" onClick={() => setOpenModal(true)}>
              <Plus size={16} /> Открыть счёт
            </Button>
          </div>
        </div>
      </div>

      <Card
        title="Счета"
        padded={false}
        actions={
          <Link to="/app/accounts" className="inline-flex items-center gap-1 text-xs text-muted hover:text-text">
            Все счета <ArrowRight size={14} />
          </Link>
        }
      >
        <AccountsTable
          accounts={live?.slice(0, 5)}
          balances={balances}
          loading={accounts.isLoading}
          hidden={hidden}
          onOpen={(a) => navigate(`/app/accounts/${a.id}`)}
          empty={
            <EmptyState
              icon={<Wallet size={36} strokeWidth={1.5} />}
              text="Счетов пока нет"
              action={
                <Button size="sm" onClick={() => setOpenModal(true)}>
                  Открыть счёт
                </Button>
              }
            />
          }
        />
      </Card>

      <Card title="Последние операции" padded={false}>
        <Table
          columns={columns}
          rows={recent.slice(0, 10)}
          rowKey={(r) => r.tx.id}
          loading={accounts.isLoading || statements.some((s) => s.isLoading)}
          onRowClick={(r) => navigate(`/app/accounts/${r.account.id}`)}
          empty={<EmptyState icon={<ReceiptText size={36} strokeWidth={1.5} />} text="Операций пока нет" />}
        />
      </Card>
      <OpenAccountModal open={openModal} onOpenChange={setOpenModal} />
    </div>
  );
}
