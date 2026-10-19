import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { accountsApi } from '../../api/accounts';
import { errorText } from '../../api/errors';
import { Button } from '../../components/Button';
import { ErrorDetails } from '../../components/Feedback';
import { Select } from '../../components/Input';
import { Modal } from '../../components/Modal';
import { useToast } from '../../components/Toast';
import { CURRENCIES, DEFAULT_CURRENCY } from '../../lib/currencies';
import { qk } from '../client/queries';

export function OpenAccountModal({ open, onOpenChange }: { open: boolean; onOpenChange: (v: boolean) => void }) {
  const [currency, setCurrency] = useState(DEFAULT_CURRENCY);
  const queryClient = useQueryClient();
  const toast = useToast();
  const mutation = useMutation({
    mutationFn: () => accountsApi.open('me', currency),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: qk.accounts('me') });
      toast.success(`Счёт в ${currency} открыт`);
      onOpenChange(false);
    },
  });

  return (
    <Modal
      open={open}
      onOpenChange={(v) => {
        if (!v) mutation.reset();
        onOpenChange(v);
      }}
      title="Открыть счёт"
      description="В каждой валюте можно держать до трёх счетов."
      footer={
        <>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            Отмена
          </Button>
          <Button loading={mutation.isPending} onClick={() => mutation.mutate()}>
            Открыть
          </Button>
        </>
      }
    >
      <Select
        label="Валюта"
        value={currency}
        onChange={(e) => setCurrency(e.target.value)}
        error={
          mutation.isError
            ? errorText(mutation.error, {
                match: [
                  ['maximum number of accounts', 'В этой валюте уже 3 счёта'],
                  ['currency is not allowed', 'Валюта недоступна'],
                ],
              })
            : undefined
        }
      >
        {CURRENCIES.map((c) => (
          <option key={c} value={c}>
            {c}
          </option>
        ))}
      </Select>
      {mutation.isError && <ErrorDetails error={mutation.error} />}
    </Modal>
  );
}
