const API_BASE = '/api/v1'

export class ApiError extends Error {
  constructor({ status = 0, code = 'NETWORK_ERROR', message = '请求失败', fields = null, requestId = '' }) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.fields = fields
    this.requestId = requestId
  }
}

export function buildQuery(params = {}) {
  const search = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === null || value === '') return
    search.append(key, String(value))
  })
  const query = search.toString()
  return query ? `?${query}` : ''
}

async function request(path, { method = 'GET', query, body } = {}) {
  const url = `${API_BASE}${path}${buildQuery(query)}`
  let response
  try {
    response = await fetch(url, {
      method,
      headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch {
    throw new ApiError({
      code: 'NETWORK_ERROR',
      message: '无法连接后端服务，请确认后端已启动（默认 127.0.0.1:8080）',
    })
  }

  let payload = null
  try {
    payload = await response.json()
  } catch {
    payload = null
  }

  if (!payload) {
    throw new ApiError({
      status: response.status,
      code: 'INVALID_RESPONSE',
      message: '后端返回内容不是合法的 JSON',
    })
  }

  if (!response.ok || payload.code !== 'OK') {
    throw new ApiError({
      status: response.status,
      code: payload.code || 'UNKNOWN_ERROR',
      message: payload.message || '请求失败',
      fields: payload.data?.fields || null,
      requestId: payload.request_id || '',
    })
  }

  return payload.data
}

export const http = {
  get: (path, query) => request(path, { query }),
  post: (path, body) => request(path, { method: 'POST', body }),
  put: (path, body) => request(path, { method: 'PUT', body }),
}
