import { request } from './http';
import type { Rule, RuleParams } from './types';

export const antifraudApi = {
  rules: () => request<{ rules: Rule[] | null }>('/antifraud/rules'),
  // Переданные params заменяют пороги правила целиком.
  update: (code: string, patch: { enabled?: boolean; params?: RuleParams }) =>
    request<Rule>(`/antifraud/rules/${code}`, { method: 'PATCH', body: patch }),
};
