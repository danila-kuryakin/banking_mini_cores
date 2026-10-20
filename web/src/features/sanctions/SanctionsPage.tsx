import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Plus, ShieldBan } from 'lucide-react';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { errorText } from '../../api/errors';
import { kycApi } from '../../api/kyc';
import type { SanctionEntry } from '../../api/types';
import { Button } from '../../components/Button';
import { Banner, EmptyState, ErrorBox } from '../../components/Feedback';
import { Input } from '../../components/Input';
import { PageHeader } from '../../components/Misc';
import { Modal } from '../../components/Modal';
import { Pagination, usePage } from '../../components/Pagination';
import { Table, type Column } from '../../components/Table';
import { useToast } from '../../components/Toast';
import { formatIsoDate } from '../../lib/dates';
import { birthDateError } from '../../lib/validation';

const schema = z.object({
  first_name: z.string().trim().min(1, 'Укажите имя').max(100, 'Не длиннее 100 символов'),
  last_name: z.string().trim().min(1, 'Укажите фамилию').max(100, 'Не длиннее 100 символов'),
  birth_date: z
    .string()
    .min(1, 'Укажите дату рождения')
    .superRefine((v, ctx) => {
      const err = birthDateError(v);
      if (err) ctx.addIssue({ code: z.ZodIssueCode.custom, message: err });
    }),
  source: z.string().trim().max(200, 'Не длиннее 200 символов'),
});
type Form = z.infer<typeof schema>;

const columns: Column<SanctionEntry>[] = [
  { key: 'last', header: 'Фамилия', render: (s) => <span className="font-medium">{s.last_name}</span> },
  { key: 'first', header: 'Имя', render: (s) => s.first_name },
  { key: 'birth', header: 'Дата рождения', render: (s) => <span className="num text-xs">{formatIsoDate(s.birth_date)}</span> },
  { key: 'source', header: 'Источник', render: (s) => s.source || <span className="text-muted">—</span> },
];

export function SanctionsPage() {
  const { limit, offset, setOffset } = usePage();
  const [open, setOpen] = useState(false);
  const query = useQuery({
    queryKey: ['sanctions', limit, offset],
    queryFn: async () => (await kycApi.sanctions({ limit, offset })).entries ?? [],
    placeholderData: (prev) => prev,
  });

  return (
    <>
      <PageHeader
        title="Санкционный список"
        subtitle="Новая запись влияет только на будущие авто-проверки заявок"
        actions={
          <Button onClick={() => setOpen(true)}>
            <Plus size={16} /> Добавить
          </Button>
        }
      />
      <div className="card">
        {query.isError ? (
          <div className="p-4">
            <ErrorBox error={query.error} onRetry={() => void query.refetch()} />
          </div>
        ) : (
          <Table
            columns={columns}
            rows={query.data}
            rowKey={(s) => s.id}
            loading={query.isLoading}
            empty={<EmptyState icon={<ShieldBan size={36} strokeWidth={1.5} />} text="Список пуст" />}
          />
        )}
        <Pagination limit={limit} offset={offset} count={query.data?.length} onChange={setOffset} />
      </div>
      <AddSanctionModal open={open} onOpenChange={setOpen} />
    </>
  );
}

function AddSanctionModal({ open, onOpenChange }: { open: boolean; onOpenChange: (v: boolean) => void }) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const { register, handleSubmit, formState, reset } = useForm<Form>({
    resolver: zodResolver(schema),
    defaultValues: { first_name: '', last_name: '', birth_date: '', source: '' },
  });
  const mutation = useMutation({
    mutationFn: (v: Form) => kycApi.addSanction(v),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['sanctions'] });
      toast.success('Запись добавлена');
      close();
    },
  });
  const close = () => {
    reset();
    mutation.reset();
    onOpenChange(false);
  };
  const errs = formState.errors;

  return (
    <Modal
      open={open}
      onOpenChange={(v) => !v && close()}
      title="Добавить в санкционный список"
      footer={
        <>
          <Button variant="secondary" onClick={close}>
            Отмена
          </Button>
          <Button type="submit" form="sanction-form" loading={mutation.isPending}>
            Добавить
          </Button>
        </>
      }
    >
      <form id="sanction-form" noValidate className="space-y-4" onSubmit={handleSubmit((v) => mutation.mutate(v))}>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Input label="Имя" error={errs.first_name?.message} {...register('first_name')} />
          <Input label="Фамилия" error={errs.last_name?.message} {...register('last_name')} />
        </div>
        <Input label="Дата рождения" type="date" error={errs.birth_date?.message} {...register('birth_date')} />
        <Input label="Источник" placeholder="OFAC SDN" error={errs.source?.message} {...register('source')} />
        <Banner tone="info" title="Удаления записей нет">
          Ошибочную запись придётся убирать через базу.
        </Banner>
        {mutation.isError && (
          <ErrorBox error={mutation.error} text={errorText(mutation.error, { 409: 'Такой человек уже в списке' })} />
        )}
      </form>
    </Modal>
  );
}
