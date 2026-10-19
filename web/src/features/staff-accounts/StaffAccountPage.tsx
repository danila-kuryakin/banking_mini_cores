import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Lock, LockOpen, Plus } from 'lucide-react';
import { useState } from 'react';
import { Link, useParams, useSearchParams } from 'react-router-dom';
import { accountsApi } from '../../api/accounts';
import { errorText } from '../../api/errors';
import { ledgerApi } from '../../api/ledger';
import type { Account, Transaction } from '../../api/types';
import { Amount, BigAmount } from '../../components/Amount';
import { Badge, StatusBadge } from '../../components/Badge';
import { Button } from '../../components/Button';
import { EmptyState, ErrorBox } from '../../components/Feedback';
import { Input, Textarea } from '../../components/Input';
import { Card, IdCell, KeyValue, Timeline } from '../../components/Misc';
import { Drawer, Modal } from '../../components/Modal';
import { Pagination, usePage } from '../../components/Pagination';
import { Skeleton } from '../../components/Spinner';
import { useToast } from '../../components/Toast';
import { formatDateTime } from '../../lib/dates';
import { toMinor } from '../../lib/money';
import { statusInfo, TX_TYPE_LABELS } from '../../lib/statuses';
import { AccountNumber, CurrencyIcon } from '../accounts/AccountNumber';
import { Statement } from '../accounts/Statement';
import { qk, useAllAccounts } from '../client/queries';

type Action = 'deposit' | 'block' | 'unblock' | null;

// У офицера нет ручки «счёт по id», поэтому владелец приезжает в query (?user=...) из карточки клиента.
export function StaffAccountPage() {
  const { accountId = '' } = useParams();
  const [params] = useSearchParams();
  const userId = params.get('user') ?? '';
  const [action, setAction] = useState<Action>(null);
  const [txId, setTxId] = useState<string | null>(null);

  const account = useQuery({
    queryKey: qk.account(userId, accountId),
    queryFn: () => accountsApi.get(userId, accountId),
    enabled: !!userId,
  });
  const balance = useQuery({
    queryKey: qk.balance(userId, accountId),
    queryFn: () => accountsApi.balance(userId, accountId),
    enabled: !!userId,
  });
  const own = useAllAccounts(userId);

  if (!userId) {
    return <EmptyState text="Откройте счёт из карточки клиента" action={<Link to="/staff/customers" className="link">Клиенты</Link>} />;
  }
  if (account.isLoading) return <Skeleton className="h-64 w-full" />;
  if (account.isError) return <ErrorBox error={account.error} onRetry={() => void account.refetch()} />;
  const a = account.data!;

  return (
    <div className="space-y-4">
      <Link to={`/staff/customers/${userId}?tab=accounts`} className="inline-flex items-center gap-1 text-sm text-muted hover:text-text">
        <ArrowLeft size={16} /> Карточка клиента
      </Link>
      <div className="card p-6">
        <div className="flex flex-col gap-6 md:flex-row md:items-end md:justify-between">
          <div className="space-y-3">
            <div className="flex flex-wrap items-center gap-3">
              <CurrencyIcon currency={a.currency} />
              <AccountNumber number={a.number} />
              <StatusBadge status={a.status} kind="account" />
            </div>
            {balance.data ? <BigAmount amount={balance.data.amount} currency={a.currency} /> : <Skeleton className="h-8 w-48" />}
            <div className="flex flex-wrap gap-x-4 text-xs text-muted">
              <span>
                Владелец: <IdCell id={a.user_id} />
              </span>
              <span>
                Счёт: <IdCell id={a.id} />
              </span>
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button disabled={a.status !== 'active'} onClick={() => setAction('deposit')}>
              <Plus size={16} /> Пополнить
            </Button>
            {a.status === 'blocked' ? (
              <Button variant="secondary" onClick={() => setAction('unblock')}>
                <LockOpen size={16} /> Разблокировать
              </Button>
            ) : (
              <Button variant="danger" disabled={a.status !== 'active'} onClick={() => setAction('block')}>
                <Lock size={16} /> Заблокировать
              </Button>
            )}
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 xl:grid-cols-[1fr_360px]">
        <Card title="Выписка" padded={false}>
          <Statement userId={userId} account={a} ownAccounts={own.data ?? []} onRowClick={(t) => setTxId(t.id)} />
        </Card>
        <Card title="История статусов">
          <StatusHistory accountId={a.id} />
        </Card>
      </div>

      <AccountActionModal account={a} userId={userId} action={action} onClose={() => setAction(null)} />
      <TransactionDrawer txId={txId} account={a} onClose={() => setTxId(null)} />
    </div>
  );
}

function StatusHistory({ accountId }: { accountId: string }) {
  const { limit, offset, setOffset } = usePage('h_');
  const query = useQuery({
    queryKey: ['account-history', accountId, limit, offset],
    queryFn: async () => (await accountsApi.history(accountId, { limit, offset })).changes ?? [],
    placeholderData: (prev) => prev,
  });
  if (query.isLoading) return <Skeleton className="h-24 w-full" />;
  if (query.isError) return <ErrorBox error={query.error} />;
  return (
    <>
      <Timeline
        items={(query.data ?? []).map((c) => ({
          id: String(c.id),
          title: c.from_status
            ? `${statusInfo(c.from_status, 'account').label} → ${statusInfo(c.to_status, 'account').label}`
            : 'Счёт открыт',
          meta: `${formatDateTime(c.created_at)} · ${c.actor_id ? c.actor_id.slice(0, 8) : 'Система'}`,
          body: c.reason,
          tone: statusInfo(c.to_status, 'account').tone,
        }))}
      />
      <div className="-mx-4 -mb-4 mt-4">
        <Pagination limit={limit} offset={offset} count={query.data?.length} onChange={setOffset} />
      </div>
    </>
  );
}

function AccountActionModal({
  account,
  userId,
  action,
  onClose,
}: {
  account: Account;
  userId: string;
  action: Action;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const [text, setText] = useState('');
  const [localError, setLocalError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: async (): Promise<unknown> => {
      if (action === 'deposit') return accountsApi.deposit(account.id, toMinor(text)!, account.currency);
      if (action === 'block') return accountsApi.block(account.id, text.trim());
      return accountsApi.unblock(account.id, text.trim());
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: qk.account(userId, account.id) });
      void queryClient.invalidateQueries({ queryKey: qk.balance(userId, account.id) });
      void queryClient.invalidateQueries({ queryKey: qk.transactions(userId, account.id) });
      void queryClient.invalidateQueries({ queryKey: qk.accounts(userId) });
      void queryClient.invalidateQueries({ queryKey: ['account-history', account.id] });
      toast.success(action === 'deposit' ? 'Счёт пополнен' : action === 'block' ? 'Счёт заблокирован' : 'Счёт разблокирован');
      close();
    },
  });

  const close = () => {
    setText('');
    setLocalError(null);
    mutation.reset();
    onClose();
  };

  const submit = () => {
    setLocalError(null);
    if (action === 'deposit') {
      const amount = toMinor(text);
      if (amount === null || amount <= 0) return setLocalError('Сумма больше нуля, не больше 2 знаков после запятой');
    } else if (action === 'block' && !text.trim()) {
      return setLocalError('Укажите причину');
    }
    if (text.length > 500) return setLocalError('Не длиннее 500 символов');
    mutation.mutate();
  };

  const title = action === 'deposit' ? 'Пополнить счёт' : action === 'block' ? 'Заблокировать счёт' : 'Разблокировать счёт';
  return (
    <Modal
      open={!!action}
      onOpenChange={(v) => !v && close()}
      title={title}
      footer={
        <>
          <Button variant="secondary" onClick={close}>
            Отмена
          </Button>
          <Button variant={action === 'block' ? 'danger' : 'primary'} loading={mutation.isPending} onClick={submit}>
            {action === 'deposit' ? 'Пополнить' : action === 'block' ? 'Заблокировать' : 'Разблокировать'}
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        {action === 'deposit' ? (
          <Input
            label={`Сумма, ${account.currency}`}
            inputMode="decimal"
            placeholder="0.00"
            className="num"
            value={text}
            autoFocus
            onChange={(e) => setText(e.target.value)}
            error={localError ?? undefined}
          />
        ) : (
          <Textarea
            label={action === 'block' ? 'Причина' : 'Комментарий (необязательно)'}
            value={text}
            maxLength={500}
            onChange={(e) => setText(e.target.value)}
            error={localError ?? undefined}
          />
        )}
        {mutation.isError && (
          <ErrorBox
            error={mutation.error}
            text={errorText(mutation.error, {
              match: [
                ['cannot move', 'Из текущего статуса так нельзя'],
                ['must be active', 'Счёт не активен'],
                ['currency', 'Валюта недоступна'],
              ],
            })}
          />
        )}
      </div>
    </Modal>
  );
}

function TransactionDrawer({ txId, account, onClose }: { txId: string | null; account: Account; onClose: () => void }) {
  const query = useQuery({
    queryKey: ['transaction', txId],
    queryFn: () => ledgerApi.get(txId!),
    enabled: !!txId,
  });
  const tx: Transaction | undefined = query.data;
  return (
    <Drawer open={!!txId} onOpenChange={(v) => !v && onClose()} title="Транзакция">
      {query.isLoading && <Skeleton className="h-40 w-full" />}
      {query.isError && <ErrorBox error={query.error} />}
      {tx && (
        <div className="space-y-6">
          <div>
            <Amount amount={tx.amount} currency={tx.currency} className="text-2xl font-semibold" />
            <div className="mt-2 flex gap-2">
              <Badge tone="info">{TX_TYPE_LABELS[tx.type] ?? tx.type}</Badge>
              <StatusBadge status={tx.status} kind="transaction" />
            </div>
          </div>
          <KeyValue
            items={[
              ['ID', <IdCell id={tx.id} />],
              ['Время', formatDateTime(tx.created_at)],
              ['Со счёта', tx.from_account_id ? <IdCell id={tx.from_account_id} /> : 'Касса'],
              ['На счёт', <IdCell id={tx.to_account_id} />],
              ['Инициатор', <IdCell id={tx.initiator_id} />],
              ...(tx.decline_reason ? ([['Причина отказа', tx.decline_reason]] as [string, string][]) : []),
            ]}
          />
          <div>
            <h3 className="mb-2 text-sm font-semibold">Проводки</h3>
            {(tx.entries ?? []).length === 0 ? (
              <p className="text-sm text-muted">Проводок нет</p>
            ) : (
              <table className="w-full text-xs">
                <thead>
                  <tr className="border-b text-left uppercase text-muted">
                    <th className="py-2 font-medium">Счёт</th>
                    <th className="py-2 font-medium">Сторона</th>
                    <th className="py-2 text-right font-medium">Сумма</th>
                  </tr>
                </thead>
                <tbody>
                  {tx.entries!.map((e) => (
                    <tr key={e.id} className="border-b border-border/60 last:border-0">
                      <td className="py-2">
                        <span className={e.account_id === account.id ? 'text-accent' : undefined}>
                          <IdCell id={e.account_id} />
                        </span>
                      </td>
                      <td className="py-2">{e.direction === 'debit' ? 'Дебет' : 'Кредит'}</td>
                      <td className="py-2 text-right">
                        <Amount amount={e.amount} currency={e.currency} signed={e.direction === 'credit' ? 'in' : 'out'} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      )}
    </Drawer>
  );
}
