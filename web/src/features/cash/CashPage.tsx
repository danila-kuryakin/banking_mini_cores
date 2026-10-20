import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import clsx from 'clsx';
import { Banknote, CheckCircle2, Search, Wallet } from 'lucide-react';
import { useMemo, useRef, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { accountsApi } from '../../api/accounts';
import { customerApi } from '../../api/customer';
import { errorText } from '../../api/errors';
import type { Account, Customer, Transaction } from '../../api/types';
import { Amount } from '../../components/Amount';
import { StatusBadge } from '../../components/Badge';
import { Button } from '../../components/Button';
import { EmptyState, ErrorBox } from '../../components/Feedback';
import { Input } from '../../components/Input';
import { Card, IdCell, PageHeader } from '../../components/Misc';
import { Modal } from '../../components/Modal';
import { Skeleton } from '../../components/Spinner';
import { Table, type Column } from '../../components/Table';
import { useToast } from '../../components/Toast';
import { formatDateTime } from '../../lib/dates';
import { formatAccountNumber, formatMoney, toMinor } from '../../lib/money';
import { AccountNumber, CurrencyIcon } from '../accounts/AccountNumber';
import { qk, useAllAccounts, useBalances } from '../client/queries';
import { fullName } from '../customers/CustomersPage';

const JOURNAL_KEY = 'bmc.cash.journal';
const JOURNAL_SIZE = 50;
const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

interface JournalRow {
  tx: Transaction;
  number: string;
  userId: string;
  name: string;
}

// Журнала кассы в API нет, поэтому операции смены хранятся в sessionStorage вкладки.
function readJournal(): JournalRow[] {
  try {
    return JSON.parse(sessionStorage.getItem(JOURNAL_KEY) ?? '[]') as JournalRow[];
  } catch {
    return [];
  }
}

function writeJournal(rows: JournalRow[]) {
  try {
    sessionStorage.setItem(JOURNAL_KEY, JSON.stringify(rows.slice(0, JOURNAL_SIZE)));
  } catch {
    // хранилище недоступно - журнал живёт до перезагрузки
  }
}

export function CashPage() {
  const [params, setParams] = useSearchParams();
  const userId = params.get('user') ?? '';
  const accountId = params.get('account') ?? '';
  const [journal, setJournal] = useState<JournalRow[]>(readJournal);

  const select = (next: { user?: string; account?: string }) =>
    setParams(
      (p) => {
        const c = new URLSearchParams(p);
        for (const [k, v] of Object.entries(next)) {
          if (v) c.set(k, v);
          else c.delete(k);
        }
        return c;
      },
      { replace: true },
    );

  const onDeposited = (row: JournalRow) => {
    const next = [row, ...journal];
    writeJournal(next);
    setJournal(next);
  };

  return (
    <>
      <PageHeader title="Касса" subtitle="Пополнение счетов клиентов наличными" />
      <div className="grid grid-cols-1 gap-4 xl:grid-cols-[360px_1fr]">
        <CustomerPicker userId={userId} onSelect={(u) => select({ user: u, account: '' })} />
        <div className="space-y-4">
          {userId ? (
            <DepositPanel
              userId={userId}
              accountId={accountId}
              onSelectAccount={(a) => select({ account: a })}
              onDeposited={onDeposited}
            />
          ) : (
            <Card>
              <EmptyState icon={<Banknote size={36} strokeWidth={1.5} />} text="Выберите клиента слева" />
            </Card>
          )}
          <Journal rows={journal} />
        </div>
      </div>
    </>
  );
}

// Поиска клиентов в API нет: берём первую сотню и фильтруем на клиенте, плюс ввод user_id вручную.
function CustomerPicker({ userId, onSelect }: { userId: string; onSelect: (userId: string) => void }) {
  const [filter, setFilter] = useState('');
  const query = useQuery({
    queryKey: ['customers', 100, 0],
    queryFn: async () => (await customerApi.list({ limit: 100, offset: 0 })).customers ?? [],
  });
  const term = filter.trim().toLowerCase();
  const list = useMemo(
    () =>
      (query.data ?? []).filter(
        (c) =>
          c.status === 'active' &&
          (!term || fullName(c).toLowerCase().includes(term) || c.user_id.toLowerCase().startsWith(term)),
      ),
    [query.data, term],
  );
  const exactId = UUID_RE.test(filter.trim()) && !list.some((c) => c.user_id === filter.trim());

  return (
    <Card title="Клиент" padded={false}>
      <div className="border-b p-3">
        <Input
          placeholder="ФИО или user_id"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          suffix={<Search size={16} className="mr-1 text-muted" />}
        />
        {exactId && (
          <Button size="sm" variant="secondary" block className="mt-2" onClick={() => onSelect(filter.trim())}>
            Открыть клиента {filter.trim().slice(0, 8)}…
          </Button>
        )}
      </div>
      {query.isLoading && (
        <div className="space-y-2 p-3">
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
        </div>
      )}
      {query.isError && (
        <div className="p-3">
          <ErrorBox error={query.error} onRetry={() => void query.refetch()} />
        </div>
      )}
      {query.data && list.length === 0 && <EmptyState text="Активных клиентов не найдено" />}
      <ul className="max-h-[480px] overflow-y-auto">
        {list.map((c: Customer) => (
          <li key={c.user_id}>
            <button
              onClick={() => onSelect(c.user_id)}
              className={clsx(
                'flex w-full items-center justify-between gap-2 border-b border-border/60 px-4 py-2.5 text-left text-sm transition-colors hover:bg-input',
                c.user_id === userId && 'bg-input text-accent',
              )}
            >
              <span className="truncate">{fullName(c)}</span>
              <span className="num shrink-0 text-xs text-muted">{c.user_id.slice(0, 8)}</span>
            </button>
          </li>
        ))}
      </ul>
      {(query.data?.length ?? 0) >= 100 && (
        <p className="px-4 py-2 text-[11px] text-muted">Показаны первые 100 клиентов. Остальных - по user_id.</p>
      )}
    </Card>
  );
}

function DepositPanel({
  userId,
  accountId,
  onSelectAccount,
  onDeposited,
}: {
  userId: string;
  accountId: string;
  onSelectAccount: (accountId: string) => void;
  onDeposited: (row: JournalRow) => void;
}) {
  const customer = useQuery({ queryKey: qk.customer(userId), queryFn: () => customerApi.get(userId) });
  const accounts = useAllAccounts(userId);
  const active = (accounts.data ?? []).filter((a) => a.status === 'active');
  const balances = useBalances(userId, active);
  const account = active.find((a) => a.id === accountId);

  if (customer.isError) return <ErrorBox error={customer.error} onRetry={() => void customer.refetch()} />;

  return (
    <Card
      title={
        <span className="flex items-center gap-2">
          {customer.data ? fullName(customer.data) : '…'}
          {customer.data && <StatusBadge status={customer.data.status} kind="customer" />}
        </span>
      }
      actions={
        <Link to={`/staff/customers/${userId}?tab=accounts`} className="text-xs link">
          Карточка клиента
        </Link>
      }
    >
      {accounts.isLoading && <Skeleton className="h-24 w-full" />}
      {accounts.isError && <ErrorBox error={accounts.error} onRetry={() => void accounts.refetch()} />}
      {accounts.data && active.length === 0 && (
        <EmptyState icon={<Wallet size={36} strokeWidth={1.5} />} text="У клиента нет активных счетов" />
      )}
      {active.length > 0 && (
        <>
          <div className="label">Счёт зачисления</div>
          <div className="grid grid-cols-1 gap-2 md:grid-cols-2">
            {active.map((a) => (
              <button
                key={a.id}
                onClick={() => onSelectAccount(a.id)}
                className={clsx(
                  'flex items-center gap-3 rounded-lg border px-3 py-2.5 text-left transition-colors',
                  a.id === accountId ? 'border-accent bg-accent/10' : 'hover:bg-input',
                )}
              >
                <CurrencyIcon currency={a.currency} />
                <span className="min-w-0 flex-1">
                  <AccountNumber number={a.number} copy={false} />
                  <span className="block text-xs text-muted">
                    {balances.get(a.id) === undefined ? '…' : formatMoney(balances.get(a.id)!, a.currency)}
                  </span>
                </span>
              </button>
            ))}
          </div>
          {account && customer.data && (
            <DepositForm key={account.id} account={account} userId={userId} name={fullName(customer.data)} onDeposited={onDeposited} />
          )}
        </>
      )}
    </Card>
  );
}

function DepositForm({
  account,
  userId,
  name,
  onDeposited,
}: {
  account: Account;
  userId: string;
  name: string;
  onDeposited: (row: JournalRow) => void;
}) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const [amountText, setAmountText] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [confirming, setConfirming] = useState(false);
  const [done, setDone] = useState<Transaction | null>(null);
  // Ключа идемпотентности нет: повторная отправка провела бы пополнение дважды.
  const sent = useRef(false);
  const amount = toMinor(amountText);

  const deposit = useMutation({
    mutationFn: () => accountsApi.deposit(account.id, amount!, account.currency),
    onSuccess: (tx) => {
      void queryClient.invalidateQueries({ queryKey: qk.balance(userId, account.id) });
      void queryClient.invalidateQueries({ queryKey: qk.transactions(userId, account.id) });
      onDeposited({ tx, number: account.number, userId, name });
      toast.success(`Зачислено ${formatMoney(tx.amount, tx.currency)}`);
      setDone(tx);
      setConfirming(false);
      setAmountText('');
    },
    onSettled: () => {
      sent.current = false;
    },
  });

  const open = () => {
    setDone(null);
    if (amount === null || amount <= 0) {
      setError('Сумма больше нуля, не больше 2 знаков после запятой');
      return;
    }
    setError(null);
    deposit.reset();
    setConfirming(true);
  };

  const confirm = () => {
    if (sent.current || deposit.isPending) return;
    sent.current = true;
    deposit.mutate();
  };

  return (
    <div className="mt-5 border-t pt-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start">
        <div className="flex-1">
          <Input
            label={`Сумма, ${account.currency}`}
            inputMode="decimal"
            placeholder="0.00"
            className="num text-lg"
            value={amountText}
            onChange={(e) => setAmountText(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && open()}
            error={error ?? undefined}
          />
        </div>
        <Button className="sm:mt-[22px]" onClick={open}>
          <Banknote size={16} /> Пополнить
        </Button>
      </div>
      {done && (
        <p className="mt-3 flex items-center gap-2 text-sm text-up">
          <CheckCircle2 size={16} /> Зачислено {formatMoney(done.amount, done.currency)} · <IdCell id={done.id} />
        </p>
      )}
      <Modal
        open={confirming}
        onOpenChange={(v) => !v && !deposit.isPending && setConfirming(false)}
        title="Подтвердите пополнение"
        footer={
          <>
            <Button variant="secondary" disabled={deposit.isPending} onClick={() => setConfirming(false)}>
              Отмена
            </Button>
            <Button loading={deposit.isPending} onClick={confirm}>
              Зачислить
            </Button>
          </>
        }
      >
        <div className="space-y-2 rounded-lg bg-input p-4 text-sm">
          <div className="text-xs text-muted">Клиент</div>
          <div>{name}</div>
          <div className="text-xs text-muted">Счёт</div>
          <div className="num">{formatAccountNumber(account.number)}</div>
        </div>
        <div className="mt-5 text-center">
          {amount !== null && <Amount amount={amount} currency={account.currency} className="text-3xl font-semibold" />}
        </div>
        {deposit.isError && (
          <div className="mt-4">
            <ErrorBox
              error={deposit.error}
              text={errorText(deposit.error, {
                match: [
                  ['must be active', 'Счёт не активен'],
                  ['currency', 'Валюта недоступна'],
                ],
              })}
            />
          </div>
        )}
      </Modal>
    </div>
  );
}

function Journal({ rows }: { rows: JournalRow[] }) {
  const columns: Column<JournalRow>[] = [
    { key: 'date', header: 'Время', render: (r) => <span className="num text-xs">{formatDateTime(r.tx.created_at)}</span> },
    {
      key: 'client',
      header: 'Клиент',
      render: (r) => (
        <Link to={`/staff/customers/${r.userId}`} className="hover:text-accent">
          {r.name}
        </Link>
      ),
    },
    {
      key: 'account',
      header: 'Счёт',
      hideOnMobile: true,
      render: (r) => (
        <Link to={`/staff/accounts/${r.tx.to_account_id}?user=${r.userId}`} className="num text-xs hover:text-accent">
          {formatAccountNumber(r.number)}
        </Link>
      ),
    },
    { key: 'amount', header: 'Сумма', align: 'right', render: (r) => <Amount amount={r.tx.amount} currency={r.tx.currency} signed="in" /> },
    { key: 'status', header: 'Статус', align: 'right', render: (r) => <StatusBadge status={r.tx.status} kind="transaction" /> },
  ];

  // Итог смены по каждой валюте отдельно.
  const totals = new Map<string, number>();
  rows.filter((r) => r.tx.status === 'completed').forEach((r) => totals.set(r.tx.currency, (totals.get(r.tx.currency) ?? 0) + r.tx.amount));

  return (
    <Card
      title="Операции смены"
      padded={false}
      actions={
        <div className="flex flex-wrap gap-3 text-xs text-muted">
          {[...totals.entries()].map(([c, v]) => (
            <span key={c}>
              Итого: <Amount amount={v} currency={c} className="text-text" />
            </span>
          ))}
        </div>
      }
    >
      <Table
        columns={columns}
        rows={rows}
        rowKey={(r) => r.tx.id}
        empty={<EmptyState text="В этой вкладке пополнений ещё не было" />}
      />
      <p className="border-t px-4 py-2 text-[11px] text-muted">
        Журнал хранится только в этой вкладке браузера: общего журнала кассы API пока не отдаёт.
      </p>
    </Card>
  );
}
