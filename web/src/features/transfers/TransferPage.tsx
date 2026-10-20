import { useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowDown, CheckCircle2, XCircle } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { ledgerApi } from '../../api/ledger';
import type { Account, Transaction } from '../../api/types';
import { Amount } from '../../components/Amount';
import { Button } from '../../components/Button';
import { Banner, EmptyState, ErrorDetails } from '../../components/Feedback';
import { Input, Select } from '../../components/Input';
import { PageHeader } from '../../components/Misc';
import { Skeleton } from '../../components/Spinner';
import { Tabs } from '../../components/Tabs';
import { formatAccountNumber, formatMoney, fromMinor, toMinor } from '../../lib/money';
import { qk, useAllAccounts, useBalances } from '../client/queries';
import { ConfirmTransfer, type TransferDraft } from './ConfirmTransfer';
import { classifyTransferError } from './transferErrors';

type Mode = 'own' | 'other';
type Stage = 'form' | 'confirm' | 'done';

const ACCOUNT_NUMBER_RE = /^\d{20}$/;

// Номер вводят группами или с пробелами, на бэкенд уходят только цифры.
const digitsOnly = (s: string) => s.replace(/[\s-]/g, '');

const label = (a: Account) => `${formatAccountNumber(a.number)} · ${a.currency}`;

export function TransferPage() {
  const [params] = useSearchParams();
  const queryClient = useQueryClient();
  const accounts = useAllAccounts();
  const active = useMemo(() => (accounts.data ?? []).filter((a) => a.status === 'active'), [accounts.data]);
  const balances = useBalances('me', active);

  const [fromId, setFromId] = useState(params.get('from') ?? '');
  const [mode, setMode] = useState<Mode>('own');
  const [toOwn, setToOwn] = useState('');
  const [toOther, setToOther] = useState('');
  const [amountText, setAmountText] = useState('');
  const [stage, setStage] = useState<Stage>('form');
  const [toError, setToError] = useState<string | null>(null);
  const [amountError, setAmountError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  useEffect(() => {
    if (!fromId && active[0]) setFromId(active[0].id);
  }, [active, fromId]);

  const from = active.find((a) => a.id === fromId);
  const ownTargets = active.filter((a) => a.id !== fromId && a.currency === from?.currency);
  const available = from ? balances.get(from.id) : undefined;
  const toNumber = digitsOnly(toOther);
  const toId = mode === 'own' ? toOwn : '';

  const transfer = useMutation({
    mutationFn: (input: { from: Account; toId: string; toNumber: string; amount: number }) =>
      ledgerApi.transfer('me', {
        from_account_id: input.from.id,
        ...(input.toId ? { to_account_id: input.toId } : { to_account_number: input.toNumber }),
        amount: input.amount,
        currency: input.from.currency,
      }),
    onSettled: (data, _err, input) => {
      // Отказ - тоже транзакция в выписке, поэтому перечитываем и после ошибки.
      for (const id of [input.from.id, input.toId || data?.to_account_id].filter(Boolean) as string[]) {
        void queryClient.invalidateQueries({ queryKey: qk.balance('me', id) });
        void queryClient.invalidateQueries({ queryKey: qk.transactions('me', id) });
      }
    },
    onSuccess: () => setStage('done'),
    onError: (err) => {
      const f = classifyTransferError(err);
      if (f.kind === 'field') {
        setToError(f.text);
        setStage('form');
      } else {
        setStage('done');
      }
    },
  });

  const draft: TransferDraft | null = useMemo(() => {
    const amount = toMinor(amountText);
    if (!from || amount === null) return null;
    const own = active.find((a) => a.id === toId);
    return {
      fromLabel: label(from),
      toLabel: own ? label(own) : formatAccountNumber(toNumber),
      amount,
      currency: from.currency,
    };
  }, [from, amountText, active, toId, toNumber]);

  const goConfirm = () => {
    setToError(null);
    setAmountError(null);
    setNotice(null);
    const amount = toMinor(amountText);
    let ok = true;
    if (mode === 'own' && !toId) {
      setToError('Выберите счёт');
      ok = false;
    } else if (mode === 'other' && !ACCOUNT_NUMBER_RE.test(toNumber)) {
      setToError('Номер счёта получателя - 20 цифр');
      ok = false;
    } else if (toId === fromId || (mode === 'other' && toNumber === from?.number)) {
      setToError('Выберите другой счёт');
      ok = false;
    }
    if (amount === null || amount <= 0) {
      setAmountError('Введите сумму больше нуля, не больше 2 знаков после запятой');
      ok = false;
    } else if (available !== undefined && amount > available) {
      setAmountError(`Доступно ${from ? formatMoney(available, from.currency) : ''}`);
      ok = false;
    }
    if (ok) setStage('confirm');
  };

  const reset = () => {
    transfer.reset();
    setAmountText('');
    setStage('form');
  };

  if (accounts.isLoading) return <Skeleton className="mx-auto h-96 max-w-xl" />;
  if (active.length === 0) {
    return (
      <EmptyState
        text="Нет активных счетов для перевода"
        action={
          <Link to="/app/accounts" className="link">
            Открыть счёт
          </Link>
        }
      />
    );
  }

  if (stage === 'confirm' && draft && from) {
    return (
      <ConfirmTransfer
        draft={draft}
        pending={transfer.isPending}
        onBack={() => setStage('form')}
        onConfirm={() => transfer.mutate({ from, toId, toNumber, amount: draft.amount })}
      />
    );
  }

  if (stage === 'done') {
    return (
      <TransferResult
        tx={transfer.data}
        error={transfer.error}
        onDone={reset}
        onRetry={() => {
          transfer.reset();
          setNotice('Перед повтором проверьте выписку: перевод мог пройти.');
          setStage('form');
        }}
      />
    );
  }

  return (
    <div className="mx-auto max-w-xl">
      <PageHeader title="Перевод" />
      {notice && (
        <div className="mb-4">
          <Banner tone="warn" title={notice} />
        </div>
      )}
      <div className="card p-5">
        <div className="text-xs font-medium uppercase tracking-wide text-muted">Откуда</div>
        <div className="mt-3">
          <Select value={fromId} onChange={(e) => setFromId(e.target.value)} aria-label="Счёт списания">
            {active.map((a) => (
              <option key={a.id} value={a.id}>
                {label(a)}
              </option>
            ))}
          </Select>
          <p className="mt-2 text-xs text-muted">
            Доступно:{' '}
            {from && available !== undefined ? <Amount amount={available} currency={from.currency} className="text-text" /> : '…'}
          </p>
        </div>
      </div>

      <div className="relative z-10 -my-3 flex justify-center">
        <span className="flex h-9 w-9 items-center justify-center rounded-full border-4 border-bg bg-accent text-accent-text">
          <ArrowDown size={16} />
        </span>
      </div>

      <div className="card p-5">
        <div className="text-xs font-medium uppercase tracking-wide text-muted">Куда</div>
        <Tabs
          className="mt-2"
          tabs={[
            { key: 'own', label: 'Между своими' },
            { key: 'other', label: 'Другому клиенту' },
          ]}
          value={mode}
          onChange={(m) => {
            setMode(m);
            setToError(null);
          }}
        />
        <div className="mt-4">
          {mode === 'own' ? (
            <Select
              value={toOwn}
              onChange={(e) => setToOwn(e.target.value)}
              error={toError ?? undefined}
              aria-label="Счёт зачисления"
            >
              <option value="">{ownTargets.length ? 'Выберите счёт' : `Нет других счетов в ${from?.currency ?? ''}`}</option>
              {ownTargets.map((a) => (
                <option key={a.id} value={a.id}>
                  {label(a)}
                </option>
              ))}
            </Select>
          ) : (
            <Input
              placeholder="40817 810 0 0000 0000000"
              inputMode="numeric"
              aria-label="Номер счёта получателя"
              value={toOther}
              onChange={(e) => setToOther(e.target.value)}
              onBlur={() => ACCOUNT_NUMBER_RE.test(toNumber) && setToOther(formatAccountNumber(toNumber))}
              error={toError ?? undefined}
              hint="20 цифр, можно с пробелами"
              className="num"
            />
          )}
        </div>
      </div>

      <div className="card mt-3 p-5">
        <Input
          label="Сумма"
          inputMode="decimal"
          placeholder="0.00"
          value={amountText}
          onChange={(e) => setAmountText(e.target.value)}
          error={amountError ?? undefined}
          className="num text-lg"
          suffix={
            <button
              type="button"
              className="rounded px-2 py-1 text-xs font-semibold text-accent hover:bg-bg"
              disabled={available === undefined}
              onClick={() => available !== undefined && setAmountText(fromMinor(available))}
            >
              Макс
            </button>
          }
        />
        <Button block className="mt-5" onClick={goConfirm}>
          Продолжить
        </Button>
      </div>
    </div>
  );
}

function TransferResult({
  tx,
  error,
  onDone,
  onRetry,
}: {
  tx: Transaction | undefined;
  error: unknown;
  onDone: () => void;
  onRetry: () => void;
}) {
  if (tx && tx.status === 'completed') {
    return (
      <div className="card mx-auto flex max-w-md flex-col items-center gap-3 px-6 py-12 text-center">
        <CheckCircle2 size={56} className="text-up" />
        <h2 className="text-lg font-semibold">Перевод выполнен</h2>
        <Amount amount={tx.amount} currency={tx.currency} className="text-3xl font-semibold" />
        <p className="num text-xs text-muted">ID транзакции: {tx.id}</p>
        <Button className="mt-4" onClick={onDone}>
          Готово
        </Button>
      </div>
    );
  }
  const failure = classifyTransferError(error);
  return (
    <div className="card mx-auto flex max-w-md flex-col items-center gap-3 px-6 py-12 text-center">
      <XCircle size={56} className="text-down" />
      <h2 className="text-lg font-semibold">{failure.text}</h2>
      {failure.kind === 'declined' && <p className="text-sm text-muted">Отказ сохранён в выписке счёта списания.</p>}
      <ErrorDetails error={error} />
      <div className="mt-4 flex gap-2">
        <Button variant="secondary" onClick={onDone}>
          Готово
        </Button>
        {failure.kind === 'fatal' && failure.retry && <Button onClick={onRetry}>Повторить</Button>}
      </div>
    </div>
  );
}
