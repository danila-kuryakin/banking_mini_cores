import { request } from './http';
import type { Customer, CustomerStatus, Page, Profile } from './types';

// userId "me" гейтвей подменяет идентификатором из токена.
export const customerApi = {
  get: (userId: string) => request<Customer>(`/customer/${userId}`),
  status: (userId: string) => request<CustomerStatus>(`/customer/${userId}/status`),
  updateProfile: (userId: string, profile: Profile) =>
    request<Customer>(`/customer/${userId}`, { method: 'POST', body: profile }),
  list: (page: Page) => request<{ customers: Customer[] | null }>('/customer', { query: { ...page } }),
};
