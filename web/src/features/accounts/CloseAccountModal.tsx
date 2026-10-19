import { useMutation, useQueryClient } from '@tanstack/react-query';
import { accountsApi } from '../../api/accounts';
import type { Account } from '../../api/types';
import { Button } from '../../components/Button';
import { ErrorBox } from '../../components/Feedback';
import { Modal } from '../../components/Modal';
import { useToast } from '../../components/Toast';
import { formatAccountNumber } from '../../lib/money';
import { qk } from '../client/queries';

export function CloseAccountModal({ account, onClose }: { account: Account | null; onClose: () => void }) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const mutation = useMutation({
    mutationFn: (a: Account) => accountsApi.close('me', a.id),
    onSuccess: (a) => {
      void queryClient.invalidateQueries({ queryKey: qk.accounts('me') });
      queryClient.setQueryData(qk.account('me', a.id), a);
      toast.success('Счёт закрыт');
      onClose();
    },
  });

  return (
    <Modal
      open={!!account}
      onOpenChange={(v) => {
        if (!v) {
          mutation.reset();
          onClose();
        }
      }}
      title="Закрыть счёт?"
      description="Счёт можно закрыть только с нулевым балансом. Закрытый счёт открыть снова нельзя."
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Отмена
          </Button>
          <Button variant="danger" loading={mutation.isPending} onClick={() => account && mutation.mutate(account)}>
            Закрыть счёт
          </Button>
        </>
      }
    >
      {account && (
        <p className="num text-sm">
          {formatAccountNumber(account.number)} · {account.currency}
        </p>
      )}
      {mutation.isError && (
        <div className="mt-4">
          <ErrorBox
            error={mutation.error}
            overrides={{
              match: [
                ['zero balance', 'На счёте есть деньги - сначала переведите их'],
                ['cannot move', 'Этот счёт закрыть нельзя'],
              ],
            }}
          />
        </div>
      )}
    </Modal>
  );
}
