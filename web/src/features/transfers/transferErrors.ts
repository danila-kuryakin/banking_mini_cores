import { ApiError } from '../../api/http';

export type TransferFailure =
  | { kind: 'field'; text: string }
  | { kind: 'declined'; text: string }
  | { kind: 'fatal'; text: string; retry: boolean };

// Разбор ответа перевода по таблице §4.5 спецификации; id транзакции из текста не разбираем.
export function classifyTransferError(err: unknown): TransferFailure {
  if (!(err instanceof ApiError)) return { kind: 'fatal', text: 'Что-то пошло не так', retry: true };
  const msg = err.message.toLowerCase();
  if (err.status === 400) {
    if (msg.includes('declined: insufficient funds')) return { kind: 'declined', text: 'Недостаточно средств' };
    if (msg.includes('declined: denied by antifraud'))
      return { kind: 'declined', text: 'Перевод отклонён системой безопасности' };
    if (msg.includes('both accounts must be active'))
      return { kind: 'field', text: 'Один из счетов заблокирован или закрыт' };
    if (msg.includes('currency does not match')) return { kind: 'field', text: 'Валюта счетов не совпадает' };
    if (msg.includes('same account')) return { kind: 'field', text: 'Выберите другой счёт' };
    if (msg.includes('currency is not allowed')) return { kind: 'fatal', text: 'Валюта недоступна', retry: false };
    return { kind: 'fatal', text: 'Перевод не выполнен: проверьте данные', retry: false };
  }
  if (err.status === 403) return { kind: 'fatal', text: 'Счёт списания вам не принадлежит', retry: false };
  if (err.status === 404) return { kind: 'field', text: 'Счёт получателя не найден' };
  if (err.status === 503)
    return { kind: 'fatal', text: 'Сервис проверки переводов недоступен, попробуйте позже', retry: true };
  if (err.status === 0) return { kind: 'fatal', text: 'Нет связи с сервером', retry: true };
  return { kind: 'fatal', text: 'Не удалось выполнить перевод', retry: true };
}
