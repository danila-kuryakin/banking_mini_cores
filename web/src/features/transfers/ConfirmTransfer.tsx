import { ArrowDown } from 'lucide-react';
import { useRef } from 'react';
import { Amount } from '../../components/Amount';
import { Button } from '../../components/Button';

export interface TransferDraft {
  fromLabel: string;
  toLabel: string;
  amount: number;
  currency: string;
}

// idempotency_key нет: повторный клик провёл бы перевод дважды, поэтому отправка строго одна.
export function ConfirmTransfer({
  draft,
  pending,
  onConfirm,
  onBack,
}: {
  draft: TransferDraft;
  pending: boolean;
  onConfirm: () => void;
  onBack: () => void;
}) {
  const sent = useRef(false);
  const confirm = () => {
    if (sent.current || pending) return;
    sent.current = true;
    onConfirm();
  };

  return (
    <div className="card mx-auto max-w-md p-6">
      <h2 className="mb-6 text-lg font-semibold">Подтвердите перевод</h2>
      <div className="space-y-2 rounded-lg bg-input p-4 text-sm">
        <div className="text-xs text-muted">Откуда</div>
        <div className="num">{draft.fromLabel}</div>
        <ArrowDown size={16} className="my-1 text-muted" />
        <div className="text-xs text-muted">Куда</div>
        <div className="num break-all">{draft.toLabel}</div>
      </div>
      <div className="mt-6 text-center">
        <div className="text-xs text-muted">Сумма</div>
        <Amount amount={draft.amount} currency={draft.currency} className="text-3xl font-semibold" />
      </div>
      <div className="mt-8 grid grid-cols-2 gap-2">
        <Button variant="secondary" onClick={onBack} disabled={pending}>
          Назад
        </Button>
        <Button onClick={confirm} loading={pending}>
          Подтвердить
        </Button>
      </div>
    </div>
  );
}
