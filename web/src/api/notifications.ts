import { request } from './http';
import type { Notification, Page } from './types';

export const notificationsApi = {
  list: (userId: string, page: Page) =>
    request<{ notifications: Notification[] | null }>(`/customer/${userId}/notifications`, { query: { ...page } }),
};
