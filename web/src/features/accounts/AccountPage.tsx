import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, ArrowLeftRight, Plus, XCircle } from 'lucide-react';
import { useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { accountsApi } from '../../api/accounts';
import { BigAmount } from '../../components/Amount';
import { StatusBadge } from '../../components/Badge';
import { Button } from '../../components/Button';
import { EmptyState, ErrorBox } from '../../components/Feedback';
import { Card, CopyButton, Tooltip } from '../../components/Misc';
import { Skeleton } from '../../components/Spinner';
import { formatDate } from '../../lib/dates';
import { qk, useAllAccounts } from '../client/queries';
import { AccountNumber, CurrencyIcon } from './AccountNumber';
import { CloseAccountModal } from './CloseAccountModal';
import { Statement } from './Statement';

export function AccountPage() {
  const { accountId = '' } = useParams();
  const navigate = useNavigate();
  const [closing, setClosing] = useState(false);
  const account = useQuery({ queryKey: qk.account('me', accountId), queryFn: () => accountsApi.get('me', accountId) });
  const balance = useQuery({
    queryKey: qk.balance('me', accountId),
    queryFn: () => accountsApi.balance('me', accountId),
  });
  const all = useAllAccounts();

  if (account.isLoading) return <Skeleton className="h-48 w-full" />;
  if (account.isError) {
    return account.error && (account.error as { status?: number }).status === 404 ? (
      <EmptyState text="Счёт не найден" action={<Link to="/app/accounts" className="link">К списку счетов</Link>} />
    ) : (
      <ErrorBox error={account.error} onRetry={() => void account.refetch()} />
    );
  }
  const a = account.data!;
  const active = a.status === 'active';

  return (
    <div className="space-y-4">
      <Link to="/app/accounts" className="inline-flex items-center gap-1 text-sm text-muted hover:text-text">
        <ArrowLeft size={16} /> Счета
      </Link>
      <div className="card p-6">
        <div className="flex flex-col gap-6 md:flex-row md:items-end md:justify-between">
          <div className="space-y-3">
            <div className="flex flex-wrap items-center gap-3">
              <CurrencyIcon currency={a.currency} />
              <AccountNumber number={a.number} />
              <StatusBadge status={a.status} kind="account" />
            </div>
            {balance.data ? (
              <BigAmount amount={balance.data.amount} currency={a.currency} />
            ) : (
              <Skeleton className="h-8 w-48" />
            )}
            <p className="text-xs text-muted">Открыт {formatDate(a.created_at)}</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button disabled={!active} onClick={() => navigate(`/app/transfer?from=${a.id}`)}>
              <ArrowLeftRight size={16} /> Перевести
            </Button>
            <Tooltip content="Пополнение выполняет сотрудник банка">
              <span>
                <Button variant="secondary" disabled>
                  <Plus size={16} /> Пополнить
                </Button>
              </span>
            </Tooltip>
            <Button variant="secondary" disabled={!active} onClick={() => setClosing(true)}>
              <XCircle size={16} /> Закрыть
            </Button>
          </div>
        </div>
        <div className="mt-6 flex flex-wrap items-center gap-2 rounded-lg bg-input px-3 py-2 text-xs text-muted">
          Номер для перевода от другого клиента:
          <span className="num break-all text-text">{a.number}</span>
          <CopyButton value={a.number} label="Скопировать номер для перевода" />
        </div>
      </div>
      <Card title="Выписка" padded={false}>
        <Statement userId="me" account={a} ownAccounts={all.data ?? []} />
      </Card>
      <CloseAccountModal account={closing ? a : null} onClose={() => setClosing(false)} />
    </div>
  );
}
