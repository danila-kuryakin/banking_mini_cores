import { LogOut } from 'lucide-react';
import { useAuth, useUser } from '../../auth/AuthContext';
import { StatusBadge } from '../../components/Badge';
import { Button } from '../../components/Button';
import { ErrorBox } from '../../components/Feedback';
import { Card, KeyValue } from '../../components/Misc';
import { Skeleton } from '../../components/Spinner';
import { formatDateTime } from '../../lib/dates';
import { ROLE_LABELS } from '../../lib/statuses';
import { useCustomer, useCustomerStatus, useFiles } from '../client/queries';
import { DocumentsList } from '../staff/DocumentsList';
import { ProfileForm } from '../onboarding/ProfileForm';

export function ProfilePage() {
  const user = useUser();
  const { logout } = useAuth();
  const customer = useCustomer();
  const status = useCustomerStatus();
  const files = useFiles();
  // Правка анкеты открыта только до отправки на KYC.
  const editable = status.data?.status === 'new' || status.data?.status === 'profile_filled';

  return (
    <div className="mx-auto max-w-3xl space-y-4">
      <Card
        title="Учётная запись"
        actions={
          <Button size="sm" variant="secondary" onClick={() => void logout()}>
            <LogOut size={14} /> Выйти
          </Button>
        }
      >
        <KeyValue
          items={[
            ['Email', user.email || '—'],
            ['Роль', ROLE_LABELS[user.role]],
            ['Статус', status.data ? <StatusBadge status={status.data.status} kind="customer" /> : '…'],
            ['Статус изменён', formatDateTime(status.data?.changed_at)],
          ]}
        />
      </Card>
      <Card title="Анкета">
        {customer.isLoading && <Skeleton className="h-40 w-full" />}
        {customer.isError && <ErrorBox error={customer.error} onRetry={() => void customer.refetch()} />}
        {customer.data && <ProfileForm profile={customer.data.profile} readOnly={!editable} />}
      </Card>
      <Card title="Документы" padded={false}>
        <DocumentsList userId="me" files={files.data} loading={files.isLoading} />
      </Card>
    </div>
  );
}
