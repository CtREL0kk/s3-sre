import type { Grant, GraphEdge, GraphNode, GraphTreeEdge, ObjectItem, Permission, User, Visibility } from './types'

const BASE = '/api/v1'

const ACCESS_KEY = 'mys3.access'
const REFRESH_KEY = 'mys3.refresh'

export class ApiError extends Error {
  status: number
  code: string
  constructor(status: number, code: string, message: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

let accessToken: string | null = localStorage.getItem(ACCESS_KEY)
let refreshToken: string | null = localStorage.getItem(REFRESH_KEY)

export function setTokens(access: string | null, refresh: string | null) {
  accessToken = access
  refreshToken = refresh
  if (access) localStorage.setItem(ACCESS_KEY, access)
  else localStorage.removeItem(ACCESS_KEY)
  if (refresh) localStorage.setItem(REFRESH_KEY, refresh)
  else localStorage.removeItem(REFRESH_KEY)
}

export function isAuthed(): boolean {
  return Boolean(accessToken)
}

async function refresh(): Promise<boolean> {
  if (!refreshToken) return false
  try {
    const res = await fetch(`${BASE}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
    if (!res.ok) {
      setTokens(null, null)
      return false
    }
    const data = await res.json()
    setTokens(data.access_token, data.refresh_token)
    return true
  } catch {
    return false
  }
}

interface RequestOptions {
  method?: string
  body?: unknown
}

async function request<T = unknown>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers()
  let body: BodyInit | undefined
  if (options.body instanceof FormData) {
    body = options.body
  } else if (options.body !== undefined) {
    headers.set('Content-Type', 'application/json')
    body = JSON.stringify(options.body)
  }
  if (accessToken) headers.set('Authorization', `Bearer ${accessToken}`)

  const doFetch = () => fetch(BASE + path, { method: options.method, headers, body })

  let res = await doFetch()
  if (res.status === 401 && accessToken && !path.startsWith('/auth/')) {
    const ok = await refresh()
    if (ok) res = await doFetch()
  }

  if (!res.ok) {
    let code = 'error'
    let message = res.statusText
    try {
      const data = await res.json()
      code = data?.error?.code ?? code
      message = data?.error?.message ?? message
    } catch {
    }
    throw new ApiError(res.status, code, message)
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

export const api = {
  register: (username: string, email: string, password: string) =>
    request<User>('/auth/register', { method: 'POST', body: { username, email, password } }),

  login: (login: string, password: string) =>
    request<{ access_token: string; refresh_token: string }>('/auth/login', {
      method: 'POST',
      body: { login, password },
    }),

  logout: () => request('/auth/logout', { method: 'POST', body: { refresh_token: refreshToken } }),

  me: () => request<User>('/me'),

  getConfig: () => request<{ storage_mode: string }>('/config'),

  listRoot: (limit = 500) => request<{ items: ObjectItem[]; total: number }>(`/objects?limit=${limit}`),

  listChildren: (id: string, limit = 500) =>
    request<{ items: ObjectItem[]; total: number }>(`/objects/${id}/children?limit=${limit}`),

  getObject: (id: string) => request<ObjectItem>(`/objects/${id}`),

  getGraph: (folderId: string | null) =>
    request<{ nodes: GraphNode[]; edges: GraphEdge[]; tree: GraphTreeEdge[] }>(
      `/objects/graph${folderId ? `?folder=${folderId}` : ''}`,
    ),

  createFolder: (parentId: string | null, name: string) =>
    request<ObjectItem>('/objects', { method: 'POST', body: { parent_id: parentId, name } }),

  uploadFile: (parentId: string, file: File) => {
    const fd = new FormData()
    fd.append('file', file)
    return request<ObjectItem>(`/objects/${parentId}/files`, { method: 'POST', body: fd })
  },

  update: (id: string, patch: { name?: string; parent_id?: string | null }) =>
    request<ObjectItem>(`/objects/${id}`, { method: 'PATCH', body: patch }),

  remove: (id: string) => request(`/objects/${id}`, { method: 'DELETE' }),

  setVisibility: (id: string, visibility: Visibility) =>
    request<ObjectItem>(`/objects/${id}/visibility`, { method: 'PATCH', body: { visibility } }),

  listGrants: (id: string) => request<{ items: Grant[] }>(`/objects/${id}/grants`),

  setGrant: (id: string, granteeId: string, permission: Permission) =>
    request<Grant>(`/objects/${id}/grants`, { method: 'PUT', body: { grantee_id: granteeId, permission } }),

  deleteGrant: (id: string, granteeId: string) =>
    request(`/objects/${id}/grants/${granteeId}`, { method: 'DELETE' }),
}

export async function getContentBlob(id: string): Promise<Blob> {
  const headers = new Headers()
  if (accessToken) headers.set('Authorization', `Bearer ${accessToken}`)
  const res = await fetch(`${BASE}/objects/${id}/content`, { headers })
  if (!res.ok) throw new ApiError(res.status, 'preview_error', 'не удалось получить содержимое')
  return res.blob()
}

export async function downloadObject(id: string, filename: string): Promise<void> {
  const blob = await getContentBlob(id)
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}