// Бэкенд отдаёт UTC в RFC 3339, показываем в локальной зоне dd.MM.yyyy HH:mm.
const pad = (n: number) => String(n).padStart(2, '0');

function parse(value: string | undefined | null): Date | null {
  if (!value) return null;
  const d = new Date(value);
  if (Number.isNaN(d.getTime()) || d.getUTCFullYear() <= 1) return null;
  return d;
}

export function formatDateTime(value: string | undefined | null): string {
  const d = parse(value);
  if (!d) return '—';
  return `${pad(d.getDate())}.${pad(d.getMonth() + 1)}.${d.getFullYear()} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export function formatDate(value: string | undefined | null): string {
  const d = parse(value);
  if (!d) return '—';
  return `${pad(d.getDate())}.${pad(d.getMonth() + 1)}.${d.getFullYear()}`;
}

// Дата без времени (YYYY-MM-DD) календарная, часовой пояс к ней не применяем.
export function formatIsoDate(value: string | undefined | null): string {
  if (!value) return '—';
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  return m ? `${m[3]}.${m[2]}.${m[1]}` : value;
}

export function formatRelative(value: string | undefined | null, now = Date.now()): string {
  const d = parse(value);
  if (!d) return '—';
  const sec = Math.round((now - d.getTime()) / 1000);
  if (sec < 60) return 'только что';
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min} мин назад`;
  const h = Math.floor(min / 60);
  if (h < 24) return `${h} ч назад`;
  const days = Math.floor(h / 24);
  if (days < 7) return `${days} дн назад`;
  return formatDate(value);
}

export function timeOf(value: string | undefined | null): number {
  return parse(value)?.getTime() ?? 0;
}
