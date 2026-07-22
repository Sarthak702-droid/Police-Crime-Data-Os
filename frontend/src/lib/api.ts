import type { ApiResponse } from '../types'
import { oidcEnabled, refreshOIDCToken } from './oidc'

const API_BASE = import.meta.env.VITE_API_BASE_URL || '/api/v1'

class ApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.status = status
  }
}

function token() {
  return localStorage.getItem('drishti_access_token')
}

async function renewToken(): Promise<boolean> {
  const refreshToken = localStorage.getItem('drishti_refresh_token')
  if (!refreshToken) return false
  if (oidcEnabled()) return refreshOIDCToken(refreshToken)
  const response = await fetch(`${API_BASE}/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: refreshToken }),
  })
  if (!response.ok) return false
  const body = (await response.json()) as ApiResponse<{ token: string; refresh_token: string }>
  localStorage.setItem('drishti_access_token', body.data.token)
  localStorage.setItem('drishti_refresh_token', body.data.refresh_token)
  return true
}

export async function api<T>(path: string, init: RequestInit = {}, retry = true): Promise<T> {
  const headers = new Headers(init.headers)
  if (!(init.body instanceof FormData)) headers.set('Content-Type', 'application/json')
  if (token()) headers.set('Authorization', `Bearer ${token()}`)
  const response = await fetch(`${API_BASE}${path}`, { ...init, headers })
  if (response.status === 401 && retry && await renewToken()) return api<T>(path, init, false)
  const contentType = response.headers.get('content-type') || ''
  if (!contentType.includes('application/json')) {
    if (!response.ok) throw new ApiError(`Request failed (${response.status})`, response.status)
    return response.blob() as Promise<T>
  }
  const body = (await response.json()) as ApiResponse<T>
  if (!response.ok || !body.success) throw new ApiError(body.message || 'Request failed', response.status)
  return body.data
}

export const get = <T>(path: string) => api<T>(path)
export const post = <T>(path: string, data?: unknown) => api<T>(path, { method: 'POST', body: data === undefined ? undefined : JSON.stringify(data) })
export const patch = <T>(path: string, data: unknown) => api<T>(path, { method: 'PATCH', body: JSON.stringify(data) })
export const put = <T>(path: string, data: unknown) => api<T>(path, { method: 'PUT', body: JSON.stringify(data) })
export const del = <T>(path: string) => api<T>(path, { method: 'DELETE' })
export const upload = <T>(path: string, form: FormData) => api<T>(path, { method: 'POST', body: form })

export { ApiError, API_BASE }
