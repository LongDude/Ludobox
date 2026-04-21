export type SseHandlers<T> = {
  onOpen?: () => void
  onEvent?: (event: T) => void
  onError?: (error: unknown) => void
  onClose?: () => void
}

export class SseRequestError extends Error {
  status?: number

  constructor(message: string, status?: number) {
    super(message)
    this.status = status
  }
}

export function buildApiUrl(baseURL: string | undefined, path: string) {
  const cleanPath = path.replace(/^\//, '')

  if (baseURL && /^https?:\/\//i.test(baseURL)) {
    const normalizedBase = baseURL.endsWith('/') ? baseURL : `${baseURL}/`
    return new URL(cleanPath, normalizedBase).toString()
  }

  if (typeof window !== 'undefined') {
    const relativeBase = baseURL ? `${baseURL.replace(/\/$/, '')}/` : '/'
    return new URL(`${relativeBase}${cleanPath}`, window.location.origin).toString()
  }

  return `${(baseURL || '').replace(/\/$/, '')}/${cleanPath}`
}

export function extractSseDataMessages(buffer: string) {
  const messages: string[] = []
  let remaining = buffer.replace(/\r\n/g, '\n')

  let boundary = remaining.indexOf('\n\n')
  while (boundary >= 0) {
    const rawEvent = remaining.slice(0, boundary)
    remaining = remaining.slice(boundary + 2)

    const data = rawEvent
      .split('\n')
      .filter((line) => line.startsWith('data:'))
      .map((line) => line.slice(5).trimStart())
      .join('\n')

    if (data) {
      messages.push(data)
    }

    boundary = remaining.indexOf('\n\n')
  }

  return { messages, remaining }
}

export async function readJsonSseStream<T>(
  url: string,
  token: string | null,
  signal: AbortSignal,
  handlers: SseHandlers<T>,
) {
  const headers: Record<string, string> = {
    Accept: 'text/event-stream',
  }
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }

  const response = await fetch(url, {
    headers,
    signal,
    credentials: 'include',
  })

  if (!response.ok) {
    throw new SseRequestError(`SSE request failed with ${response.status}`, response.status)
  }
  if (!response.body) {
    throw new Error('SSE stream is not available')
  }

  handlers.onOpen?.()

  const reader = response.body.pipeThrough(new TextDecoderStream()).getReader()
  let buffer = ''

  while (!signal.aborted) {
    const { value, done } = await reader.read()
    if (done) break

    const parsed = extractSseDataMessages(buffer + (value || ''))
    buffer = parsed.remaining

    for (const message of parsed.messages) {
      handlers.onEvent?.(JSON.parse(message) as T)
    }
  }
}
