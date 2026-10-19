import { request } from './http';
import type { Transaction } from './types';

export interface TransferInput {
  from_account_id: string;
  to_account_id: string;
  amount: number;
  currency: string;
}

export const ledgerApi = {
  transfer: (userId: string, input: TransferInput) =>
    request<Transaction>(`/customer/${userId}/transfers`, { method: 'POST', body: input }),
  get: (transactionId: string) => request<Transaction>(`/transactions/${transactionId}`),
};
