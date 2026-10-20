import { request } from './http';
import type { Application, Page, SanctionEntry } from './types';

export interface QueueFilter {
  status?: string[];
  reviewer_id?: string;
  unassigned?: boolean;
}

export const kycApi = {
  getByUser: (userId: string) => request<Application>(`/customer/${userId}/kyc`),
  submit: (userId: string) => request<Application>(`/customer/${userId}/kyc`, { method: 'POST' }),

  list: (filter: QueueFilter, page: Page) =>
    request<{ applications: Application[] | null }>('/kyc/applications', {
      query: { status: filter.status, reviewer_id: filter.reviewer_id, unassigned: filter.unassigned || undefined, ...page },
    }),
  get: (applicationId: string) => request<Application>(`/kyc/applications/${applicationId}`),
  claim: (applicationId: string) => request<Application>(`/kyc/applications/${applicationId}/claim`, { method: 'POST' }),
  approve: (applicationId: string, comment: string) =>
    request<Application>(`/kyc/applications/${applicationId}/approve`, { method: 'POST', body: { comment } }),
  reject: (applicationId: string, reason: string) =>
    request<Application>(`/kyc/applications/${applicationId}/reject`, { method: 'POST', body: { reason } }),
  requestDocuments: (applicationId: string, types: string[], comment: string) =>
    request<Application>(`/kyc/applications/${applicationId}/documents`, { method: 'POST', body: { types, comment } }),

  sanctions: (page: Page) => request<{ entries: SanctionEntry[] | null }>('/kyc/sanctions', { query: { ...page } }),
  addSanction: (entry: Omit<SanctionEntry, 'id'>) =>
    request<SanctionEntry>('/kyc/sanctions', { method: 'POST', body: entry }),
};
