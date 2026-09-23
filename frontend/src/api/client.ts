import axios from 'axios';
import type { AxiosRequestConfig } from 'axios';
import type { AuthTokens } from '../types';

const RAW_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';
const API_BASE = RAW_BASE.replace(/\/+$/, '');

const client = axios.create({
  baseURL: API_BASE,
  withCredentials: true,
  timeout: 15000,
});

function tokenExpiresIn(token: string): number | null {
  try {
    const payload = JSON.parse(atob((token.split('.')[1] ?? '').replace(/-/g, '+').replace(/_/g, '/')));
    return typeof payload.exp === 'number' ? payload.exp - Math.floor(Date.now() / 1000) : null;
  } catch {
    return null;
  }
}

client.interceptors.request.use(async (config) => {
  const url = config.url || '';
  const isAuthCall = url.includes('/v1/auth/login') || url.includes('/v1/auth/refresh') || url.includes('/health/');
  let token = localStorage.getItem('access_token');
  if (token && !isAuthCall) {
    const ttl = tokenExpiresIn(token);
    if (ttl !== null && ttl < 60) {
      const rt = localStorage.getItem('refresh_token');
      if (rt) {
        try {
          const tokens = await sharedRefresh(rt);
          token = tokens.access_token;
          localStorage.setItem('access_token', token);
          if (tokens.refresh_token) {
            localStorage.setItem('refresh_token', tokens.refresh_token);
          }
        } catch {
          /* fall through — the 401 handler below will deal with it */
        }
      }
    }
  }
  if (token && config.headers && !config.headers.Authorization) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

export function clearSession() {
  localStorage.removeItem('access_token');
  localStorage.removeItem('refresh_token');
  localStorage.removeItem('user');
  localStorage.removeItem('current_batch_id');
}

// Single-flight refresh: parallel 401s must share ONE refresh call, because the
// backend rotates refresh tokens and treats concurrent reuse as replay,
// revoking the whole session (which kicks the user back to login).
let refreshPromise: Promise<AuthTokens> | null = null;

function sharedRefresh(token: string): Promise<AuthTokens> {
  if (!refreshPromise) {
    refreshPromise = client
      .post('/v1/auth/refresh', { refresh_token: token }, { withCredentials: true })
      .then((res) => res.data as AuthTokens)
      .finally(() => {
        refreshPromise = null;
      });
  }
  return refreshPromise;
}

client.interceptors.response.use(
  (response) => response,
  async (error) => {
    const original = error.config as (typeof error.config & { _retry?: boolean }) | undefined;
    const url: string = original?.url || '';
    const isAuthCall = url.includes('/v1/auth/login') || url.includes('/v1/auth/refresh') || url.includes('/health/');
    // Never auto-refresh the login/refresh calls themselves, and never retry twice:
    // their 401s are real errors the UI must show (otherwise this recurses forever).
    if (error.response?.status !== 401 || !original || original._retry || isAuthCall) {
      return Promise.reject(error);
    }
    const refreshToken = localStorage.getItem('refresh_token');
    if (!refreshToken) {
      clearSession();
      window.location.href = '/login';
      return Promise.reject(error);
    }
    original._retry = true;
    try {
      const tokens = await sharedRefresh(refreshToken);
      localStorage.setItem('access_token', tokens.access_token);
      if (tokens.refresh_token) {
        localStorage.setItem('refresh_token', tokens.refresh_token);
      }
      original.headers.Authorization = `Bearer ${tokens.access_token}`;
      return client(original);
    } catch {
      clearSession();
      window.location.href = '/login';
      return Promise.reject(error);
    }
  },
);

export const api = {
  get: (url: string, params?: object, options?: AxiosRequestConfig) => client.get(url, { ...options, params }),
  post: (url: string, data?: object, options?: object) => client.post(url, data, options),
  patch: (url: string, data?: object) => client.patch(url, data),
  put: (url: string, data?: object, options?: object) => client.put(url, data, options),
  delete: (url: string) => client.delete(url),
};

export async function uploadFile(uploadURL: string, file: File, init?: { signal?: AbortSignal }): Promise<void> {
  const response = await fetch(uploadURL, {
    method: 'PUT',
    body: file,
    signal: init?.signal,
    headers: file.type ? { 'Content-Type': file.type } : undefined,
  });
  if (!response.ok) throw new Error(`File upload failed (${response.status})`);
}

/** Normalize list responses: plain arrays, {items}, or {data} (gallery). */
export function toList<T>(payload: unknown): T[] {
  if (Array.isArray(payload)) return payload as T[];
  if (payload && typeof payload === 'object') {
    const o = payload as Record<string, unknown>;
    if (Array.isArray(o.items)) return o.items as T[];
    if (Array.isArray(o.data)) return o.data as T[];
  }
  return [];
}

export function errMsg(err: unknown, fallback: string): string {
  const data = (err as { response?: { data?: unknown } } | null)?.response?.data;
  if (data && typeof data === 'object') {
    const o = data as { error?: { message?: unknown }; message?: unknown };
    if (typeof o.error?.message === 'string' && o.error.message) return o.error.message;
    if (typeof o.message === 'string' && o.message) return o.message;
  }
  // Network errors / timeouts / aborts have no response — surface them
  // instead of the generic fallback so users see what actually happened.
  if (typeof err === 'object' && err !== null && 'code' in err && (err as { code?: unknown }).code === 'ECONNABORTED') {
    return 'Request timed out. Check your connection and try again.';
  }
  if (err instanceof Error && err.message) return err.message;
  return fallback;
}

/** Basic-auth header for the protected /health endpoints (creds kept in session storage). */
export function basicAuthHeader(): Record<string, string> {
  const u = sessionStorage.getItem('basic_user');
  const p = sessionStorage.getItem('basic_pass');
  if (!u || p === null) return {};
  return { Authorization: 'Basic ' + btoa(u + ':' + p) };
}

export default client;
