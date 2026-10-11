export class ApiError extends Error {
  status: number
  errors?: Record<string, string[]>

  constructor(status: number, message: string, errors?: Record<string, string[]>) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.errors = errors
  }
}

const mutatingMethods = new Set(['POST', 'PUT', 'PATCH', 'DELETE'])

function csrfToken(): string {
  const match = document.cookie.match(/(?:^|;\s*)ebox_csrf=([^;]+)/)
  return match ? decodeURIComponent(match[1]) : ''
}

export interface ApiFetchOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  signal?: AbortSignal
}

export async function apiFetch<T>(path: string, options: ApiFetchOptions = {}): Promise<T> {
  const method = options.method ?? 'GET'
  const headers: Record<string, string> = { Accept: 'application/json' }

  if (options.body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }

  if (mutatingMethods.has(method)) {
    const token = csrfToken()
    if (token) {
      headers['X-CSRF-Token'] = token
    }
  }

  const response = await fetch(`/api/v1${path}`, {
    method,
    headers,
    credentials: 'same-origin',
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
    signal: options.signal,
  })

  if (response.status === 204) {
    return undefined as T
  }

  const text = await response.text()
  let payload: unknown = null
  if (text) {
    try {
      payload = JSON.parse(text)
    } catch {
      payload = text
    }
  }

  if (!response.ok) {
    let message = `Request failed with status ${response.status}`
    let errors: Record<string, string[]> | undefined

    if (payload && typeof payload === 'object') {
      const record = payload as Record<string, unknown>
      if (typeof record.message === 'string' && record.message) {
        message = record.message
      }
      if (record.errors && typeof record.errors === 'object') {
        errors = record.errors as Record<string, string[]>
      }
    }

    throw new ApiError(response.status, message, errors)
  }

  return payload as T
}
