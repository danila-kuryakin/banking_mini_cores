// Единственный словарь статус -> подпись и цвет для всех бейджей.
export type Tone = 'up' | 'down' | 'warn' | 'info' | 'muted';

interface StatusInfo {
  label: string;
  tone: Tone;
}

const STATUSES: Record<string, StatusInfo> = {
  // клиент
  new: { label: 'Новый', tone: 'info' },
  profile_filled: { label: 'Анкета заполнена', tone: 'info' },
  on_kyc: { label: 'На проверке', tone: 'warn' },
  active: { label: 'Активен', tone: 'up' },
  rejected: { label: 'Отклонён', tone: 'down' },
  blocked: { label: 'Заблокирован', tone: 'down' },
  // заявка KYC
  submitted: { label: 'Подана', tone: 'info' },
  auto_checks: { label: 'Авто-проверки', tone: 'info' },
  auto_rejected: { label: 'Авто-отказ', tone: 'down' },
  in_review: { label: 'На рассмотрении', tone: 'warn' },
  docs_requested: { label: 'Запрошены документы', tone: 'warn' },
  approved: { label: 'Одобрена', tone: 'up' },
  // счёт
  closed: { label: 'Закрыт', tone: 'muted' },
  // транзакция
  completed: { label: 'Проведена', tone: 'up' },
  declined: { label: 'Отклонена', tone: 'down' },
  // проверка и файл
  passed: { label: 'Пройдена', tone: 'up' },
  failed: { label: 'Не пройдена', tone: 'down' },
  uploaded: { label: 'Не подтверждён', tone: 'muted' },
  confirmed: { label: 'Принят', tone: 'up' },
};

// Подписи там, где один статус означает разное в разных сущностях.
const OVERRIDES: Partial<Record<StatusKind, Record<string, string>>> = {
  application: { rejected: 'Отклонена', approved: 'Одобрена' },
  customer: { approved: 'Одобрен' },
};

export type StatusKind = 'customer' | 'application' | 'account' | 'transaction' | 'check' | 'file';

export function statusInfo(status: string, kind?: StatusKind): StatusInfo {
  const base = STATUSES[status];
  if (!base) return { label: status || '—', tone: 'muted' };
  const label = (kind && OVERRIDES[kind]?.[status]) || base.label;
  return { label, tone: base.tone };
}

export const APPLICATION_STATUSES = [
  'submitted',
  'auto_checks',
  'auto_rejected',
  'in_review',
  'docs_requested',
  'approved',
  'rejected',
] as const;

export const TERMINAL_APPLICATION_STATUSES = new Set(['approved', 'rejected', 'auto_rejected']);

export const FILE_TYPES = ['passport', 'selfie', 'proof_of_address'] as const;
export type FileType = (typeof FILE_TYPES)[number];

export const FILE_TYPE_LABELS: Record<string, string> = {
  passport: 'Паспорт',
  selfie: 'Селфи с паспортом',
  proof_of_address: 'Подтверждение адреса',
};

export const CHECK_TYPE_LABELS: Record<string, string> = {
  sanctions: 'Санкционный список',
  age: 'Возраст 18+',
};

export const TX_TYPE_LABELS: Record<string, string> = {
  transfer: 'Перевод',
  deposit: 'Пополнение',
};

export const ROLE_LABELS: Record<string, string> = {
  client: 'Клиент',
  officer: 'Сотрудник',
  admin: 'Администратор',
};

export const RULE_LABELS: Record<string, string> = {
  single_amount_limit: 'Лимит одного перевода',
  daily_amount_limit: 'Дневной лимит',
  velocity: 'Частота переводов',
  same_account: 'Перевод на тот же счёт',
};
