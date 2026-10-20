import clsx from 'clsx';
import type { ReactNode } from 'react';
import { Skeleton } from './Spinner';

export interface Column<T> {
  key: string;
  header: ReactNode;
  render: (row: T) => ReactNode;
  align?: 'left' | 'right';
  className?: string;
  // В мобильной карточке колонку не показываем.
  hideOnMobile?: boolean;
}

interface Props<T> {
  columns: Column<T>[];
  rows: T[] | undefined;
  rowKey: (row: T) => string;
  loading?: boolean;
  empty?: ReactNode;
  onRowClick?: (row: T) => void;
  rowClassName?: (row: T) => string | undefined;
}

// Таблица на десктопе, список карточек до 768px.
export function Table<T>({ columns, rows, rowKey, loading, empty, onRowClick, rowClassName }: Props<T>) {
  if (loading && !rows) {
    return (
      <div className="space-y-3 p-4">
        {Array.from({ length: 5 }, (_, i) => (
          <Skeleton key={i} className="h-8 w-full" />
        ))}
      </div>
    );
  }
  if (!rows || rows.length === 0) return <>{empty}</>;

  return (
    <>
      <div className="hidden overflow-x-auto md:block">
        <table className="w-full text-xs">
          <thead>
            <tr className="border-b">
              {columns.map((c) => (
                <th
                  key={c.key}
                  className={clsx(
                    'h-10 whitespace-nowrap px-4 font-medium uppercase tracking-wide text-muted',
                    c.align === 'right' ? 'text-right' : 'text-left',
                  )}
                >
                  {c.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr
                key={rowKey(row)}
                onClick={onRowClick ? () => onRowClick(row) : undefined}
                className={clsx(
                  'h-11 border-b border-border/60 transition-colors last:border-0 hover:bg-input',
                  onRowClick && 'cursor-pointer',
                  rowClassName?.(row),
                )}
              >
                {columns.map((c) => (
                  <td
                    key={c.key}
                    className={clsx('px-4 py-2 text-[13px]', c.align === 'right' && 'text-right', c.className)}
                  >
                    {c.render(row)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <ul className="divide-y divide-border md:hidden">
        {rows.map((row) => (
          <li
            key={rowKey(row)}
            onClick={onRowClick ? () => onRowClick(row) : undefined}
            className={clsx('space-y-1.5 px-4 py-3', onRowClick && 'cursor-pointer active:bg-input', rowClassName?.(row))}
          >
            {columns
              .filter((c) => !c.hideOnMobile)
              .map((c) => (
                <div key={c.key} className="flex items-center justify-between gap-3 text-[13px]">
                  <span className="text-xs text-muted">{c.header}</span>
                  <span className="min-w-0 text-right">{c.render(row)}</span>
                </div>
              ))}
          </li>
        ))}
      </ul>
    </>
  );
}
