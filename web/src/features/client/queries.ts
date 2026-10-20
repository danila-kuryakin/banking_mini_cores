import { useQueries, useQuery } from '@tanstack/react-query';
import { accountsApi } from '../../api/accounts';
import { customerApi } from '../../api/customer';
import { filesApi } from '../../api/files';
import { isApiError } from '../../api/http';
import { kycApi } from '../../api/kyc';
import type { Account, Application } from '../../api/types';

// Ключи запросов в одном месте, чтобы инвалидация после мутаций не промахивалась.
export const qk = {
  customer: (userId: string) => ['customer', userId] as const,
  status: (userId: string) => ['customer-status', userId] as const,
  kyc: (userId: string) => ['kyc', userId] as const,
  files: (userId: string) => ['files', userId] as const,
  required: (userId: string) => ['files-required', userId] as const,
  accounts: (userId: string) => ['accounts', userId] as const,
  account: (userId: string, accountId: string) => ['account', userId, accountId] as const,
  balance: (userId: string, accountId: string) => ['balance', userId, accountId] as const,
  transactions: (userId: string, accountId: string) => ['transactions', userId, accountId] as const,
  notifications: (userId: string) => ['notifications', userId] as const,
};

export function useCustomerStatus(userId = 'me') {
  return useQuery({ queryKey: qk.status(userId), queryFn: () => customerApi.status(userId) });
}

export function useCustomer(userId = 'me') {
  return useQuery({ queryKey: qk.customer(userId), queryFn: () => customerApi.get(userId) });
}

// 404 - заявок ещё не было, это не ошибка.
export function useApplication(userId = 'me', poll = false) {
  return useQuery<Application | null>({
    queryKey: qk.kyc(userId),
    queryFn: async () => {
      try {
        return await kycApi.getByUser(userId);
      } catch (err) {
        if (isApiError(err, 404)) return null;
        throw err;
      }
    },
    refetchInterval: poll ? 10_000 : false,
  });
}

export function useFiles(userId = 'me') {
  return useQuery({
    queryKey: qk.files(userId),
    queryFn: async () => (await filesApi.list(userId)).files ?? [],
  });
}

// Счетов у клиента немного (до 3 на валюту), поэтому для сводок берём всё одной страницей.
export function useAllAccounts(userId = 'me') {
  return useQuery({
    queryKey: [...qk.accounts(userId), 'all'],
    queryFn: async () => (await accountsApi.list(userId, { limit: 100, offset: 0 })).accounts ?? [],
    enabled: !!userId,
  });
}

export function useBalances(userId: string, accounts: Account[] | undefined) {
  const results = useQueries({
    queries: (accounts ?? []).map((a) => ({
      queryKey: qk.balance(userId, a.id),
      queryFn: () => accountsApi.balance(userId, a.id),
    })),
  });
  const map = new Map<string, number | undefined>();
  (accounts ?? []).forEach((a, i) => map.set(a.id, results[i]?.data?.amount));
  return map;
}
