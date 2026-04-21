import { api } from '@/api/base/useLudaApi'
import { useAuthStore } from '@/stores/authStore'
import type {
  GameCancelBoostResponse,
  GameJoinRoomResponse,
  GameJoinRoomWithSeatRequest,
  GameLeaveRoomResponse,
  GamePurchaseBoostResponse,
  GameRoomStateResponse,
  GameRoundEvent,
  GameRoundStatusResponse,
} from './types'

type RoundEventHandlers = {
  onOpen?: () => void
  onEvent?: (event: GameRoundEvent) => void
  onError?: (error: unknown) => void
  onClose?: () => void
}

function buildApiUrl(path: string) {
  const baseURL = api.defaults.baseURL || ''
  const cleanPath = path.replace(/^\//, '')

  if (/^https?:\/\//i.test(baseURL)) {
    const normalizedBase = baseURL.endsWith('/') ? baseURL : `${baseURL}/`
    return new URL(cleanPath, normalizedBase).toString()
  }

  if (typeof window !== 'undefined') {
    const relativeBase = baseURL ? `${baseURL.replace(/\/$/, '')}/` : '/'
    return new URL(`${relativeBase}${cleanPath}`, window.location.origin).toString()
  }

  return `${baseURL.replace(/\/$/, '')}/${cleanPath}`
}

async function readRoundEventStream(
  url: string,
  token: string | null,
  signal: AbortSignal,
  handlers: RoundEventHandlers,
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
    const error = new Error(`SSE request failed with ${response.status}`)
    ;(error as Error & { status?: number }).status = response.status
    throw error
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
    buffer += (value || '').replace(/\r\n/g, '\n')

    let boundary = buffer.indexOf('\n\n')
    while (boundary >= 0) {
      const rawEvent = buffer.slice(0, boundary)
      buffer = buffer.slice(boundary + 2)

      const data = rawEvent
        .split('\n')
        .filter((line) => line.startsWith('data:'))
        .map((line) => line.slice(5).trimStart())
        .join('\n')

      if (data) {
        handlers.onEvent?.(JSON.parse(data) as GameRoundEvent)
      }

      boundary = buffer.indexOf('\n\n')
    }
  }
}

export const GameApi = {
  joinRoom(roomId: number) {
    return api
      .post<GameJoinRoomResponse>(`/game/rooms/${roomId}/join`)
      .then((response) => response.data)
  },

  joinRoomWithSeat(roomId: number, payload: GameJoinRoomWithSeatRequest) {
    return api
      .post<GameJoinRoomResponse>(`/game/rooms/${roomId}/join-seat`, payload)
      .then((response) => response.data)
  },

  getRoomState(roomId: number) {
    return api
      .get<GameRoomStateResponse>(`/game/rooms/${roomId}`)
      .then((response) => response.data)
  },

  getRoundStatus(roomId: number, roundId: number) {
    return api
      .get<GameRoundStatusResponse>(`/game/rooms/${roomId}/rounds/${roundId}`)
      .then((response) => response.data)
  },

  purchaseBoost(roomId: number, roundParticipantId: number) {
    return api
      .post<GamePurchaseBoostResponse>(`/game/rooms/${roomId}/participants/${roundParticipantId}/boost`)
      .then((response) => response.data)
  },

  cancelBoost(roomId: number, roundParticipantId: number) {
    return api
      .delete<GameCancelBoostResponse>(`/game/rooms/${roomId}/participants/${roundParticipantId}/boost`)
      .then((response) => response.data)
  },

  leaveRoom(roomId: number) {
    return api
      .post<GameLeaveRoomResponse>(`/game/rooms/${roomId}/leave`)
      .then((response) => response.data)
  },

  subscribeRoundEvents(roomId: number, roundId: number, handlers: RoundEventHandlers) {
    const controller = new AbortController()
    const auth = useAuthStore()
    const url = buildApiUrl(`/game/rooms/${roomId}/rounds/${roundId}/events`)

    void readRoundEventStream(url, auth.AccessToken, controller.signal, handlers)
      .catch(async (error: any) => {
        if (controller.signal.aborted || error?.name === 'AbortError') return
        let streamError = error

        if (error?.status === 401) {
          try {
            const token = await auth.refreshToken()
            if (!controller.signal.aborted && token) {
              try {
                await readRoundEventStream(url, token, controller.signal, handlers)
                return
              } catch (retryError) {
                streamError = retryError
              }
            }
          } catch {}
        }

        if (!controller.signal.aborted) {
          handlers.onError?.(streamError)
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          handlers.onClose?.()
        }
      })

    return () => controller.abort()
  },
}
