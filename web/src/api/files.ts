import { request } from './http';
import type { DownloadTicket, FileRecord, RequiredFiles, UploadTicket } from './types';

export const filesApi = {
  list: (userId: string) =>
    request<{ files: FileRecord[] | null }>(`/customer/${userId}/files`, { query: { limit: 100, offset: 0 } }),
  required: (userId: string) => request<RequiredFiles>(`/customer/${userId}/files/required`),
  init: (userId: string, type: string, filename: string) =>
    request<UploadTicket>(`/customer/${userId}/files`, { method: 'POST', body: { type, filename } }),
  confirm: (userId: string, fileId: string) =>
    request<FileRecord>(`/customer/${userId}/files/${fileId}/confirm`, { method: 'POST' }),
  downloadUrl: (userId: string, fileId: string) => request<DownloadTicket>(`/customer/${userId}/files/${fileId}/url`),
};
