import { useQuery } from '@tanstack/react-query';
import { Plus, Wallet } from 'lucide-react';
import { useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { accountsApi } from '../../api/accounts';
import type { Account } from '../../api/types';
import { Button } from '../../components/Button';
import { EmptyState, ErrorBox } from '../../components/Feedback';
import { PageHeader } from '../../components/Misc';
import { Pagination, usePage } from '../../components/Pagination';
import { qk, useBalances } from '../client/queries';
import { AccountsTable, ICONS } from './AccountsTable';
import { CloseAccountModal } from './CloseAccountModal';
import { OpenAccountModal } from './OpenAccountModal';

export function AccountsPage() {
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const showClosed = params.get('closed') === '1';
  const { limit, offset, setOffset } = usePage();
  const [openModal, setOpenModal] = useState(false);
  const [closing, setClosing] = useState<Account | null>(null);

  const query = useQuery({
    queryKey: [...qk.accounts('me'), limit, offset],
    queryFn: async () => (await accountsApi.list('me', { limit, offset })).accounts ?? [],
    placeholderData: (prev) => prev,
  });
  const visible = query.data?.filter((a) => showClosed || a.status !== 'closed');
  const balances = useBalances('me', visible);

  return (
    <>
      <PageHeader
        title="Счета"
        actions={
          <Button onClick={() => setOpenModal(true)}>
            <Plus size={16} /> Открыть счёт
          </Button>
        }
      />
      <div className="card">
        <div className="flex items-center justify-end border-b px-4 py-3">
          <label className="flex cursor-pointer items-center gap-2 text-xs text-muted">
            <input
              type="checkbox"
              className="accent-[rgb(var(--accent))]"
              checked={showClosed}
              onChange={(e) =>
                setParams(
                  (p) => {
                    const c = new URLSearchParams(p);
                    if (e.target.checked) c.set('closed', '1');
                    else c.delete('closed');
                    return c;
                  },
                  { replace: true },
                )
              }
            />
            Показать закрытые
          </label>
        </div>
        {query.isError ? (
          <div className="p-4">
            <ErrorBox error={query.error} onRetry={() => void query.refetch()} />
          </div>
        ) : (
          <AccountsTable
            accounts={visible}
            balances={balances}
            loading={query.isLoading}
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
            actions={(a) => [
              { label: 'Открыть', icon: ICONS.view, onSelect: () => navigate(`/app/accounts/${a.id}`) },
              {
                label: 'Перевести',
                icon: ICONS.transfer,
                disabled: a.status !== 'active',
                onSelect: () => navigate(`/app/transfer?from=${a.id}`),
              },
              {
                label: 'Закрыть',
                icon: ICONS.close,
                danger: true,
                disabled: a.status !== 'active',
                onSelect: () => setClosing(a),
              },
            ]}
          />
        )}
        <Pagination limit={limit} offset={offset} count={query.data?.length} onChange={setOffset} />
      </div>
      <OpenAccountModal open={openModal} onOpenChange={setOpenModal} />
      <CloseAccountModal account={closing} onClose={() => setClosing(null)} />
    </>
  );
}
