// Суммы на бэкенде - int64 в минорных единицах; все валюты из конфига с двумя знаками.
const MINOR_DIGITS = 2;
const MINOR_FACTOR = 100;

const formatters = new Map<string, Intl.NumberFormat>();

function formatter(currency: string): Intl.NumberFormat {
  let f = formatters.get(currency);
  if (!f) {
    try {
      f = new Intl.NumberFormat('ru-RU', {
        style: 'currency',
        currency,
        minimumFractionDigits: MINOR_DIGITS,
        maximumFractionDigits: MINOR_DIGITS,
      });
    } catch {
      f = new Intl.NumberFormat('ru-RU', { minimumFractionDigits: MINOR_DIGITS, maximumFractionDigits: MINOR_DIGITS });
    }
    formatters.set(currency, f);
  }
  return f;
}

export function formatMoney(amount: number, currency: string): string {
  return formatter(currency).format(amount / MINOR_FACTOR);
}

const plain = new Intl.NumberFormat('ru-RU', {
  minimumFractionDigits: MINOR_DIGITS,
  maximumFractionDigits: MINOR_DIGITS,
});

// Сумма без символа валюты - для крупного баланса, где код валюты рисуется отдельно.
export function formatAmount(amount: number): string {
  return plain.format(amount / MINOR_FACTOR);
}

// Разбор ввода пользователя в минорные единицы строкой, без умножения во float.
export function toMinor(input: string): number | null {
  const s = input.replace(/[\s\u00a0]/g, '').replace(',', '.');
  const m = /^(\d+)(?:\.(\d{0,2}))?$/.exec(s);
  if (!m) return null;
  const major = m[1] ?? '0';
  const minor = (m[2] ?? '').padEnd(MINOR_DIGITS, '0');
  const value = Number(major) * MINOR_FACTOR + Number(minor);
  if (!Number.isSafeInteger(value)) return null;
  return value;
}

// Обратная к toMinor: минорные единицы в строку для поля ввода ("1234.50").
export function fromMinor(amount: number): string {
  const sign = amount < 0 ? '-' : '';
  const abs = Math.abs(amount);
  const major = Math.floor(abs / MINOR_FACTOR);
  const minor = String(abs % MINOR_FACTOR).padStart(MINOR_DIGITS, '0');
  return `${sign}${major}.${minor}`;
}

// Номер счёта группами: 40817 810 5 0000 0000001.
export function formatAccountNumber(n: string): string {
  if (!/^\d{20}$/.test(n)) return n;
  return [n.slice(0, 5), n.slice(5, 8), n.slice(8, 9), n.slice(9, 13), n.slice(13)].join(' ');
}

export function shortId(id: string | undefined): string {
  if (!id) return '—';
  return id.length > 8 ? id.slice(0, 8) : id;
}
