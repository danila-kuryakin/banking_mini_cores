import { session } from '../auth/session';

export const API_BASE = '/api/v1';

// Ошибка гейтвея: тело всегда {"message": "..."} на английском, X-Request-Id - для поиска в логах.
export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
    readonly requestId?: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

export function isApiError(err: unknown, status?: number): err is ApiError {
  return err instanceof ApiError && (status === undefined || err.status === status);
}

interface RequestOptions {
  method?: string;
  body?: unknown;
  query?: Record<string, string | number | boolean | string[] | undefined | null>;
  auth?: boolean;
  signal?: AbortSignal;
}

function buildUrl(path: string, query?: RequestOptions['query']): string {
  const params = new URLSearchParams();
  for (const [k, v] of Object.entries(query ?? {})) {
    if (v === undefined || v === null || v === '') continue;
    if (Array.isArray(v)) v.forEach((item) => params.append(k, item));
    else params.append(k, String(v));
  }
  const qs = params.toString();
  return `${API_BASE}${path}${qs ? `?${qs}` : ''}`;
}

async function send(path: string, opts: RequestOptions, token: string | null): Promise<Response> {
  const headers: Record<string, string> = { Accept: 'application/json' };
  if (opts.body !== undefined) headers['Content-Type'] = 'application/json';
  if (token) headers.Authorization = `Bearer ${token}`;
  try {
    return await fetch(buildUrl(path, opts.query), {
      method: opts.method ?? 'GET',
      headers,
      body: opts.body === undefined ? undefined : JSON.stringify(opts.body),
      signal: opts.signal,
    });
  } catch (err) {
    if ((err as Error).name === 'AbortError') throw err;
    throw new ApiError(0, 'network error');
  }
}

async function toError(res: Response): Promise<ApiError> {
  let message = res.statusText || `HTTP ${res.status}`;
  try {
    const data = (await res.json()) as { message?: string };
    if (data?.message) message = data.message;
  } catch {
    // тело не JSON - оставляем статус
  }
  const err = new ApiError(res.status, message, res.headers.get('X-Request-Id') ?? undefined);
  console.warn(`[api] ${res.status} ${res.url}: ${message}`, err.requestId ? `request_id=${err.requestId}` : '');
  return err;
}

// Защищённые запросы: свежий access перед отправкой, на 401 один refresh и один повтор.
export async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const auth = opts.auth ?? true;
  let token = auth ? await session.accessToken() : null;
  let res = await send(path, opts, token);

  if (res.status === 401 && auth && session.hasRefresh()) {
    token = await session.refresh();
    if (token) res = await send(path, opts, token);
  }

  if (!res.ok) throw await toError(res);
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

// PUT по presigned-ссылке прямо в MinIO, с прогрессом - fetch его не даёт.
export function putToStorage(url: string, file: File, onProgress: (pct: number) => void): Promise<void> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open('PUT', url);
    if (file.type) xhr.setRequestHeader('Content-Type', file.type);
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress(Math.round((e.loaded / e.total) * 100));
    };
    xhr.onload = () =>
      xhr.status >= 200 && xhr.status < 300
        ? resolve()
        : reject(new ApiError(xhr.status, `storage upload failed: ${xhr.status}`));
    xhr.onerror = () => reject(new ApiError(0, 'storage upload network error'));
    xhr.send(file);
  });
}
