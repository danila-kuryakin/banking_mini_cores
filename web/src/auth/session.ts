import type { Role, TokenPair } from '../api/types';
import { readStorage, writeStorage } from '../lib/storage';
import { decodeAccess, normalizeRole } from './jwt';

const REFRESH_KEY = 'bmc.refresh';
const EMAIL_KEY = 'bmc.email';
const REFRESH_AHEAD_MS = 60_000;

export interface SessionUser {
  id: string;
  email: string;
  role: Role;
}

type Listener = () => void;
type ExpiredListener = () => void;

// Access живёт только в памяти, refresh - в localStorage: cookie гейтвей не ставит.
class Session {
  private access: string | null = null;
  private accessExp = 0;
  private user: SessionUser | null = null;
  private inflight: Promise<string | null> | null = null;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private listeners = new Set<Listener>();
  private expiredListeners = new Set<ExpiredListener>();

  getUser(): SessionUser | null {
    return this.user;
  }

  hasRefresh(): boolean {
    return !!readStorage(REFRESH_KEY);
  }

  subscribe(fn: Listener): () => void {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  onExpired(fn: ExpiredListener): () => void {
    this.expiredListeners.add(fn);
    return () => this.expiredListeners.delete(fn);
  }

  // Вход: email запоминаем, его не вернёт refresh после перезагрузки.
  start(tokens: TokenPair, user: { id: string; email: string; role: string }): void {
    writeStorage(REFRESH_KEY, tokens.refresh_token);
    writeStorage(EMAIL_KEY, user.email);
    const role = normalizeRole(user.role) ?? decodeAccess(tokens.access_token)?.role ?? 'client';
    this.user = { id: user.id, email: user.email, role };
    this.applyAccess(tokens);
    this.emit();
  }

  async accessToken(): Promise<string | null> {
    if (this.access && Date.now() < this.accessExp - 5_000) return this.access;
    if (!this.hasRefresh()) return null;
    return this.refresh();
  }

  // Один общий промис: повторный refresh-токен бэкенд считает компрометацией и гасит все сессии.
  refresh(): Promise<string | null> {
    if (!this.inflight) {
      this.inflight = this.doRefresh().finally(() => {
        this.inflight = null;
      });
    }
    return this.inflight;
  }

  private async doRefresh(): Promise<string | null> {
    const refreshToken = readStorage(REFRESH_KEY);
    if (!refreshToken) return null;
    let res: Response;
    try {
      res = await fetch('/api/v1/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });
    } catch {
      return null;
    }
    if (res.status === 401 || res.status === 400) {
      this.clear();
      this.expiredListeners.forEach((fn) => fn());
      return null;
    }
    if (!res.ok) return null;
    const tokens = (await res.json()) as TokenPair;
    // Новый refresh сохраняем до того, как кто-то повторит запросы со старым.
    writeStorage(REFRESH_KEY, tokens.refresh_token);
    const claims = decodeAccess(tokens.access_token);
    if (claims?.role) {
      this.user = { id: claims.sub, email: readStorage(EMAIL_KEY) ?? '', role: claims.role };
    }
    this.applyAccess(tokens);
    this.emit();
    return this.access;
  }

  private applyAccess(tokens: TokenPair): void {
    this.access = tokens.access_token;
    const fromBody = Date.parse(tokens.access_expires_at);
    const fromClaims = (decodeAccess(tokens.access_token)?.exp ?? 0) * 1000;
    this.accessExp = Number.isNaN(fromBody) ? fromClaims : fromBody;
    if (this.timer) clearTimeout(this.timer);
    const delay = Math.max(this.accessExp - Date.now() - REFRESH_AHEAD_MS, 5_000);
    this.timer = setTimeout(() => void this.refresh(), delay);
  }

  refreshToken(): string | null {
    return readStorage(REFRESH_KEY);
  }

  clear(): void {
    if (this.timer) clearTimeout(this.timer);
    this.timer = null;
    this.access = null;
    this.accessExp = 0;
    this.user = null;
    writeStorage(REFRESH_KEY, null);
    this.emit();
  }

  private emit(): void {
    this.listeners.forEach((fn) => fn());
  }
}

export const session = new Session();
