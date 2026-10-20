import { ApiError } from './http';

// Русская фраза по коду и контексту экрана; message бэкенда - только в деталях.
export type ErrorOverrides = Partial<Record<number, string>> & { match?: [string, string][] };

const DEFAULTS: Record<number, string> = {
  0: 'Нет связи с сервером',
  400: 'Некорректный запрос',
  401: 'Требуется вход',
  403: 'Недостаточно прав',
  404: 'Не найдено',
  409: 'Данные изменились, обновите страницу',
  429: 'Слишком много запросов. Попробуйте позже',
  501: 'Действие пока не поддерживается',
};

export function errorText(err: unknown, overrides: ErrorOverrides = {}): string {
  if (!(err instanceof ApiError)) return 'Что-то пошло не так';
  for (const [needle, text] of overrides.match ?? []) {
    if (err.message.toLowerCase().includes(needle.toLowerCase())) return text;
  }
  const own = overrides[err.status];
  if (own) return own;
  if (err.status >= 500 || err.status === 504) return 'Сервис временно недоступен';
  return DEFAULTS[err.status] ?? 'Что-то пошло не так';
}
