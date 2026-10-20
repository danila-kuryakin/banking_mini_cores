import * as Switch from '@radix-ui/react-switch';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import clsx from 'clsx';
import { ShieldAlert } from 'lucide-react';
import { useState } from 'react';
import { antifraudApi } from '../../api/antifraud';
import { errorText } from '../../api/errors';
import { isApiError } from '../../api/http';
import type { Rule, RuleParams } from '../../api/types';
import { Button } from '../../components/Button';
import { EmptyState, ErrorBox } from '../../components/Feedback';
import { Input } from '../../components/Input';
import { IdCell, PageHeader } from '../../components/Misc';
import { Skeleton } from '../../components/Spinner';
import { useToast } from '../../components/Toast';
import { CURRENCIES } from '../../lib/currencies';
import { formatDateTime } from '../../lib/dates';
import { fromMinor, toMinor } from '../../lib/money';
import { RULE_LABELS } from '../../lib/statuses';

const RULES_KEY = ['antifraud-rules'];
const LIMIT_RULES = new Set(['single_amount_limit', 'daily_amount_limit']);

export function AntifraudPage() {
  const query = useQuery({ queryKey: RULES_KEY, queryFn: async () => (await antifraudApi.rules()).rules ?? [] });
  return (
    <>
      <PageHeader title="Антифрод" subtitle="Правка действует со следующего перевода" />
      {query.isLoading && (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          {[0, 1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-56 w-full rounded-xl" />
          ))}
        </div>
      )}
      {query.isError && <ErrorBox error={query.error} onRetry={() => void query.refetch()} />}
      {query.data?.length === 0 && <EmptyState icon={<ShieldAlert size={36} strokeWidth={1.5} />} text="Правил нет" />}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {query.data?.map((r) => <RuleCard key={r.code} rule={r} />)}
      </div>
    </>
  );
}

function RuleCard({ rule }: { rule: Rule }) {
  const queryClient = useQueryClient();
  const toast = useToast();

  const onUpdated = (updated: Rule) =>
    queryClient.setQueryData<Rule[]>(RULES_KEY, (list) => list?.map((r) => (r.code === updated.code ? updated : r)));
  const onFailed = (err: unknown) => {
    // 404 - правило пропало, перечитываем список.
    if (isApiError(err, 404)) void queryClient.invalidateQueries({ queryKey: RULES_KEY });
  };

  const toggle = useMutation({
    mutationFn: (enabled: boolean) => antifraudApi.update(rule.code, { enabled }),
    onSuccess: (r) => {
      onUpdated(r);
      toast.success(r.enabled ? 'Правило включено' : 'Правило выключено');
    },
    onError: (err) => {
      onFailed(err);
      toast.error(errorText(err));
    },
  });

  return (
    <section className={clsx('card flex flex-col p-5', !rule.enabled && 'opacity-80')}>
      <div className="flex items-start justify-between gap-4">
        <div>
          <h2 className="font-semibold">{RULE_LABELS[rule.code] ?? rule.code}</h2>
          <p className="num mt-0.5 text-[11px] text-muted">{rule.code}</p>
        </div>
        <Switch.Root
          checked={rule.enabled}
          disabled={toggle.isPending}
          onCheckedChange={(v) => toggle.mutate(v)}
          aria-label="Правило включено"
          className="relative h-6 w-11 shrink-0 rounded-full bg-input transition-colors disabled:opacity-50 data-[state=checked]:bg-accent"
        >
          <Switch.Thumb className="block h-5 w-5 translate-x-0.5 rounded-full bg-white shadow transition-transform data-[state=checked]:translate-x-[22px]" />
        </Switch.Root>
      </div>
      {rule.description && <p className="mt-3 text-sm text-muted">{rule.description}</p>}
      <div className="mt-4 flex-1">
        {LIMIT_RULES.has(rule.code) && <LimitsForm rule={rule} onUpdated={onUpdated} onFailed={onFailed} />}
        {rule.code === 'velocity' && <VelocityForm rule={rule} onUpdated={onUpdated} onFailed={onFailed} />}
        {rule.code === 'same_account' && <p className="text-xs text-muted">Порогов нет - только включение.</p>}
      </div>
      <p className="mt-4 border-t pt-3 text-[11px] text-muted">
        Изменено {formatDateTime(rule.updated_at)}
        {rule.updated_by && (
          <>
            {' · '}
            <IdCell id={rule.updated_by} />
          </>
        )}
      </p>
    </section>
  );
}

interface FormProps {
  rule: Rule;
  onUpdated: (r: Rule) => void;
  onFailed: (err: unknown) => void;
}

function useSaveParams({ rule, onUpdated, onFailed }: FormProps) {
  const toast = useToast();
  return useMutation({
    mutationFn: (params: RuleParams) => antifraudApi.update(rule.code, { params }),
    onSuccess: (r) => {
      onUpdated(r);
      toast.success('Пороги сохранены');
    },
    onError: onFailed,
  });
}

// Лимиты по валютам в major units; пустое поле - валюта без порога, ключ не отправляем.
function LimitsForm(props: FormProps) {
  const { rule } = props;
  const currencies = [...new Set([...CURRENCIES, ...Object.keys(rule.params.limits ?? {})])];
  const [values, setValues] = useState<Record<string, string>>(() =>
    Object.fromEntries(currencies.map((c) => [c, rule.params.limits?.[c] ? fromMinor(rule.params.limits[c]!) : ''])),
  );
  const [errors, setErrors] = useState<Record<string, string>>({});
  const save = useSaveParams(props);

  const submit = () => {
    const limits: Record<string, number> = {};
    const errs: Record<string, string> = {};
    for (const c of currencies) {
      const raw = (values[c] ?? '').trim();
      if (!raw) continue;
      const v = toMinor(raw);
      if (v === null || v <= 0) errs[c] = 'Больше нуля, до 2 знаков';
      else limits[c] = v;
    }
    setErrors(errs);
    if (Object.keys(errs).length === 0) save.mutate({ ...rule.params, limits });
  };

  const currencyError =
    save.isError && isApiError(save.error, 400) && save.error.message.includes('currency is not allowed');

  return (
    <div className="space-y-3">
      {currencies.map((c) => (
        <Input
          key={c}
          label={`Лимит, ${c}`}
          inputMode="decimal"
          placeholder="без порога"
          className="num"
          value={values[c] ?? ''}
          onChange={(e) => setValues((v) => ({ ...v, [c]: e.target.value }))}
          error={errors[c] ?? (currencyError && !CURRENCIES.includes(c) ? 'Валюта недоступна' : undefined)}
        />
      ))}
      {save.isError && !currencyError && <ErrorBox error={save.error} text={errorText(save.error, { 400: 'Неверные пороги' })} />}
      <Button size="sm" loading={save.isPending} onClick={submit}>
        Сохранить
      </Button>
    </div>
  );
}

function VelocityForm(props: FormProps) {
  const { rule } = props;
  const [count, setCount] = useState(String(rule.params.max_count ?? ''));
  const [windowSec, setWindowSec] = useState(String(rule.params.window_seconds ?? ''));
  const [error, setError] = useState<string | null>(null);
  const save = useSaveParams(props);
  const minutes = Number(windowSec) > 0 ? Math.round(Number(windowSec) / 60) : null;

  const submit = () => {
    const c = Number(count);
    const w = Number(windowSec);
    if (!Number.isInteger(c) || c <= 0 || !Number.isInteger(w) || w <= 0) {
      setError('Целые числа больше нуля');
      return;
    }
    setError(null);
    // params уходят целиком: частичные бэкенд считает заменой всех порогов.
    save.mutate({ ...rule.params, max_count: c, window_seconds: w });
  };

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-2 gap-3">
        <Input label="Переводов не больше" inputMode="numeric" className="num" value={count} onChange={(e) => setCount(e.target.value)} />
        <Input
          label="За окно, секунд"
          inputMode="numeric"
          className="num"
          value={windowSec}
          onChange={(e) => setWindowSec(e.target.value)}
          hint={minutes !== null ? `≈ ${minutes} мин` : undefined}
        />
      </div>
      {error && <p className="text-xs text-down">{error}</p>}
      {save.isError && <ErrorBox error={save.error} text={errorText(save.error, { 400: 'Неверные пороги' })} />}
      <Button size="sm" loading={save.isPending} onClick={submit}>
        Сохранить
      </Button>
    </div>
  );
}
