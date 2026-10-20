import { request } from './http';
import type { Account, Balance, Page, StatusChange, Transaction } from './types';

export const accountsApi = {
  list: (userId: string, page: Page) =>
    request<{ accounts: Account[] | null }>(`/customer/${userId}/accounts`, { query: { ...page } }),
  get: (userId: string, accountId: string) => request<Account>(`/customer/${userId}/accounts/${accountId}`),
  balance: (userId: string, accountId: string) =>
    request<Balance>(`/customer/${userId}/accounts/${accountId}/balance`),
  open: (userId: string, currency: string) =>
    request<Account>(`/customer/${userId}/accounts`, { method: 'POST', body: { currency } }),
  close: (userId: string, accountId: string) =>
    request<Account>(`/customer/${userId}/accounts/${accountId}/close`, { method: 'POST' }),
  transactions: (userId: string, accountId: string, page: Page) =>
    request<{ transactions: Transaction[] | null }>(`/customer/${userId}/accounts/${accountId}/transactions`, {
      query: { ...page },
    }),

  // Ветка офицера, без пользователя в пути.
  block: (accountId: string, reason: string) =>
    request<Account>(`/accounts/${accountId}/block`, { method: 'POST', body: { reason } }),
  unblock: (accountId: string, comment: string) =>
    request<Account>(`/accounts/${accountId}/unblock`, { method: 'POST', body: { comment } }),
  deposit: (accountId: string, amount: number, currency: string) =>
    request<Transaction>(`/accounts/${accountId}/deposit`, { method: 'POST', body: { amount, currency } }),
  history: (accountId: string, page: Page) =>
    request<{ changes: StatusChange[] | null }>(`/accounts/${accountId}/history`, { query: { ...page } }),
};
