// Список должен совпадать с currencies в конфиге account-service; RUB - по умолчанию.
export const CURRENCIES: string[] = (import.meta.env.VITE_CURRENCIES || 'RUB,USD,EUR')
  .split(',')
  .map((c: string) => c.trim().toUpperCase())
  .filter(Boolean);

export const DEFAULT_CURRENCY = CURRENCIES.includes('RUB') ? 'RUB' : (CURRENCIES[0] ?? 'RUB');

export const CURRENCY_SIGNS: Record<string, string> = { RUB: '₽', USD: '$', EUR: '€' };
