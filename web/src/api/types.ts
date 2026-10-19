// DTO гейтвея один в один (services/api-gateway/internal/adapters/rest/handler/dto).

export type Role = 'client' | 'officer' | 'admin';

export interface User {
  id: string;
  email: string;
  role: string;
  created_at: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  access_expires_at: string;
  refresh_expires_at: string;
}

export interface LoginResponse {
  user: User;
  tokens: TokenPair;
}

export interface RegisterResponse {
  user_id: string;
  email: string;
  role: string;
  created_at: string;
}

export interface Profile {
  first_name: string;
  last_name: string;
  birth_date: string;
  citizenship: string;
  phone: string;
}

export interface Customer {
  id: string;
  user_id: string;
  status: string;
  profile: Profile;
  created_at: string;
  updated_at: string;
}

export interface CustomerStatus {
  status: string;
  changed_at: string;
}

export interface Check {
  id: string;
  type: string;
  result: string;
  details: string;
  created_at: string;
}

export interface KycEvent {
  id: string;
  from_status?: string;
  to_status: string;
  actor_id?: string;
  comment?: string;
  created_at: string;
}

export interface Application {
  id: string;
  user_id: string;
  status: string;
  reviewer_id?: string;
  reason?: string;
  requested_types?: string[];
  submitted_at: string;
  decided_at?: string;
  created_at: string;
  updated_at: string;
  checks?: Check[];
  events?: KycEvent[];
}

export interface SanctionEntry {
  id: string;
  first_name: string;
  last_name: string;
  birth_date: string;
  source: string;
}

export interface FileRecord {
  id: string;
  user_id: string;
  type: string;
  status: string;
  filename: string;
  content_type: string;
  size_bytes: number;
  sha256: string;
  created_at: string;
  confirmed_at?: string;
}

export interface UploadTicket {
  file_id: string;
  upload_url: string;
  expires_at: string;
}

export interface DownloadTicket {
  url: string;
  expires_at: string;
}

export interface RequiredFiles {
  ok: boolean;
  missing: string[] | null;
}

export interface Account {
  id: string;
  user_id: string;
  number: string;
  currency: string;
  status: 'active' | 'blocked' | 'closed';
  status_changed_at: string;
  created_at: string;
  updated_at: string;
}

export interface Balance {
  account_id: string;
  amount: number;
  currency: string;
}

export interface StatusChange {
  id: number;
  account_id: string;
  from_status?: string;
  to_status: string;
  actor_id?: string;
  reason?: string;
  created_at: string;
}

export interface Entry {
  id: string;
  account_id: string;
  direction: 'debit' | 'credit';
  amount: number;
  currency: string;
}

export interface Transaction {
  id: string;
  type: 'transfer' | 'deposit';
  status: 'completed' | 'declined';
  from_account_id: string;
  to_account_id: string;
  amount: number;
  currency: string;
  initiator_id?: string;
  decline_reason?: string;
  created_at: string;
  entries?: Entry[];
}

export interface RuleParams {
  limits?: Record<string, number>;
  max_count?: number;
  window_seconds?: number;
}

export interface Rule {
  code: string;
  description: string;
  enabled: boolean;
  params: RuleParams;
  updated_by?: string;
  updated_at: string;
}

export interface Notification {
  id: string;
  event_id: string;
  event_type: string;
  text: string;
  created_at: string;
}

export interface Page {
  limit: number;
  offset: number;
}
