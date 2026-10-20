import { CopyButton } from '../../components/Misc';
import { CURRENCY_SIGNS } from '../../lib/currencies';
import { formatAccountNumber } from '../../lib/money';

export function CurrencyIcon({ currency }: { currency: string }) {
  return (
    <span className="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-accent/15 text-sm font-semibold text-accent">
      {CURRENCY_SIGNS[currency] ?? currency.slice(0, 1)}
    </span>
  );
}

export function AccountNumber({ number, copy = true }: { number: string; copy?: boolean }) {
  return (
    <span className="inline-flex items-center gap-1">
      <span className="num whitespace-nowrap text-[13px]">{formatAccountNumber(number)}</span>
      {copy && <CopyButton value={number} label="Скопировать номер" />}
    </span>
  );
}
