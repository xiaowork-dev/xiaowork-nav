/**
 * Single HTTP entry point for the frontend.
 *
 * Every backend call must go through this module so that base URL handling,
 * credentials, CSRF signalling and error normalisation stay consistent.
 */

const BASE_URL: string = import.meta.env.VITE_API_BASE_URL ?? ''

/** Normalised backend error (`{ error: { code, message, details, request_id } }`). */
export interface ApiErrorDetail {
  field: string
  issue: string
}

export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly details: ApiErrorDetail[]
  readonly requestId?: string

  constructor(status: number, code: string, message: string, details: ApiErrorDetail[] = [], requestId?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.details = details
    this.requestId = requestId
  }
}

export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE'
  body?: unknown
  query?: Record<string, string | number | boolean | undefined>
  signal?: AbortSignal
}

function buildUrl(path: string, query?: RequestOptions['query']): string {
  const url = `${BASE_URL}${path}`
  if (!query) return url
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined && value !== '') params.set(key, String(value))
  }
  const qs = params.toString()
  return qs ? `${url}?${qs}` : url
}

/** Perform a request and return the parsed JSON body. */
export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, query, signal } = options

  const headers: Record<string, string> = { Accept: 'application/json' }
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  // CSRF signal expected by the backend on state-changing requests.
  if (method !== 'GET') headers['X-Requested-With'] = 'XMLHttpRequest'

  const response = await fetch(buildUrl(path, query), {
    method,
    headers,
    credentials: 'include',
    body: body === undefined ? undefined : JSON.stringify(body),
    signal,
  })

  if (response.status === 204) return undefined as T

  const text = await response.text()
  const data = text ? JSON.parse(text) : undefined

  if (!response.ok) {
    const err = data?.error ?? {}
    throw new ApiError(
      response.status,
      err.code ?? 'internal_error',
      err.message ?? `请求失败（HTTP ${response.status}）`,
      err.details ?? [],
      err.request_id,
    )
  }

  return data as T
}

export const api = {
  get: <T>(path: string, options?: Omit<RequestOptions, 'method' | 'body'>) =>
    request<T>(path, { ...options, method: 'GET' }),
  post: <T>(path: string, body?: unknown, options?: Omit<RequestOptions, 'method' | 'body'>) =>
    request<T>(path, { ...options, method: 'POST', body }),
  patch: <T>(path: string, body?: unknown, options?: Omit<RequestOptions, 'method' | 'body'>) =>
    request<T>(path, { ...options, method: 'PATCH', body }),
  put: <T>(path: string, body?: unknown, options?: Omit<RequestOptions, 'method' | 'body'>) =>
    request<T>(path, { ...options, method: 'PUT', body }),
  delete: <T>(path: string, options?: Omit<RequestOptions, 'method' | 'body'>) =>
    request<T>(path, { ...options, method: 'DELETE' }),
}
