import clsx from 'clsx';
import { formatAmount, formatMoney } from '../lib/money';

// Сумма моноширинным шрифтом; signed - со знаком и цветом зачисления/списания.
export function Amount({
  amount,
  currency,
  signed,
  hidden,
  className,
}: {
  amount: number;
  currency: string;
  signed?: 'in' | 'out';
  hidden?: boolean;
  className?: string;
}) {
  if (hidden) return <span className={clsx('num', className)}>****</span>;
  const text = formatMoney(Math.abs(amount), currency);
  return (
    <span
      className={clsx(
        'num whitespace-nowrap',
        signed === 'in' && 'text-up',
        signed === 'out' && 'text-down',
        className,
      )}
    >
      {signed === 'in' ? '+' : signed === 'out' ? '−' : amount < 0 ? '−' : ''}
      {text}
    </span>
  );
}

// Крупный баланс: число 32px, код валюты мельче и серым.
export function BigAmount({ amount, currency, hidden }: { amount: number; currency: string; hidden?: boolean }) {
  return (
    <div className="flex items-baseline gap-2">
      <span className="num text-[32px] font-semibold leading-none">{hidden ? '****' : formatAmount(amount)}</span>
      <span className="text-sm text-muted">{currency}</span>
    </div>
  );
}
