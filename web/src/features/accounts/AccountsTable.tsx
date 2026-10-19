import * as Dropdown from '@radix-ui/react-dropdown-menu';
import clsx from 'clsx';
import { ArrowLeftRight, Eye, MoreHorizontal, XCircle } from 'lucide-react';
import type { ReactNode } from 'react';
import type { Account } from '../../api/types';
import { Amount } from '../../components/Amount';
import { StatusBadge } from '../../components/Badge';
import { Skeleton } from '../../components/Spinner';
import { Table, type Column } from '../../components/Table';
import { formatDate } from '../../lib/dates';
import { AccountNumber, CurrencyIcon } from './AccountNumber';

export interface RowAction {
  label: string;
  icon: ReactNode;
  onSelect: () => void;
  disabled?: boolean;
  danger?: boolean;
}

export function AccountsTable({
  accounts,
  balances,
  loading,
  hidden,
  empty,
  onOpen,
  actions,
}: {
  accounts: Account[] | undefined;
  balances: Map<string, number | undefined>;
  loading?: boolean;
  hidden?: boolean;
  empty: ReactNode;
  onOpen: (a: Account) => void;
  actions?: (a: Account) => RowAction[];
}) {
  const columns: Column<Account>[] = [
    {
      key: 'currency',
      header: 'Валюта',
      render: (a) => (
        <span className="inline-flex items-center gap-2 font-medium">
          <CurrencyIcon currency={a.currency} />
          {a.currency}
        </span>
      ),
    },
    { key: 'number', header: 'Номер счёта', render: (a) => <AccountNumber number={a.number} /> },
    {
      key: 'balance',
      header: 'Баланс',
      align: 'right',
      render: (a) => {
        const b = balances.get(a.id);
        return b === undefined ? <Skeleton className="ml-auto h-4 w-20" /> : <Amount amount={b} currency={a.currency} hidden={hidden} />;
      },
    },
    { key: 'status', header: 'Статус', render: (a) => <StatusBadge status={a.status} kind="account" /> },
    {
      key: 'opened',
      header: 'Открыт',
      hideOnMobile: true,
      render: (a) => <span className="num text-xs text-muted">{formatDate(a.created_at)}</span>,
    },
  ];
  if (actions) {
    columns.push({
      key: 'actions',
      header: '',
      align: 'right',
      hideOnMobile: true,
      render: (a) => <RowMenu items={actions(a)} />,
    });
  }

  return (
    <Table
      columns={columns}
      rows={accounts}
      rowKey={(a) => a.id}
      loading={loading}
      empty={empty}
      onRowClick={onOpen}
      rowClassName={(a) => clsx(a.status !== 'active' && 'opacity-60')}
    />
  );
}

function RowMenu({ items }: { items: RowAction[] }) {
  return (
    <Dropdown.Root>
      <Dropdown.Trigger asChild>
        <button
          className="rounded p-1.5 text-muted hover:bg-bg hover:text-text"
          aria-label="Действия"
          onClick={(e) => e.stopPropagation()}
        >
          <MoreHorizontal size={16} />
        </button>
      </Dropdown.Trigger>
      <Dropdown.Portal>
        <Dropdown.Content
          align="end"
          sideOffset={4}
          className="z-50 min-w-[180px] rounded-lg border border-border bg-elevated p-1 shadow-xl"
          onClick={(e) => e.stopPropagation()}
        >
          {items.map((it) => (
            <Dropdown.Item
              key={it.label}
              disabled={it.disabled}
              onSelect={it.onSelect}
              className={clsx(
                'flex cursor-pointer items-center gap-2 rounded-md px-3 py-2 text-sm outline-none data-[disabled]:cursor-not-allowed data-[highlighted]:bg-input data-[disabled]:opacity-40',
                it.danger && 'text-down',
              )}
            >
              {it.icon}
              {it.label}
            </Dropdown.Item>
          ))}
        </Dropdown.Content>
      </Dropdown.Portal>
    </Dropdown.Root>
  );
}

export const ICONS = {
  view: <Eye size={14} />,
  transfer: <ArrowLeftRight size={14} />,
  close: <XCircle size={14} />,
};
