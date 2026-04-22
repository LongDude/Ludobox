import { api } from '@/api/base/useLudaApi'
import {
  buildApiUrl,
  readJsonSseStream,
  SseRequestError,
  type SseHandlers,
} from '@/api/base/sse'
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

type RoundEventHandlers = SseHandlers<GameRoundEvent>

function delay(ms: number, signal: AbortSignal) {
  if (signal.aborted) return Promise.resolve()

  return new Promise<void>((resolve) => {
    const timeout = setTimeout(resolve, ms)
    signal.addEventListener(
      'abort',
      () => {
        clearTimeout(timeout)
        resolve()
      },
      { once: true },
    )
  })
}

async function runAuthorizedRoundEventStream(
  url: string,
  auth: ReturnType<typeof useAuthStore>,
  signal: AbortSignal,
  handlers: RoundEventHandlers,
) {
  let retryDelay = 1000

  while (!signal.aborted) {
    try {
      await readJsonSseStream<GameRoundEvent>(url, auth.AccessToken, signal, handlers)
      if (!signal.aborted) {
        throw new Error('SSE stream closed')
      }
    } catch (error: any) {
      if (signal.aborted || error?.name === 'AbortError') return

      if (error instanceof SseRequestError && error.status === 401) {
        try {
          await auth.refreshToken()
          retryDelay = 1000
          continue
        } catch (refreshError) {
          handlers.onError?.(refreshError)
        }
      } else {
        handlers.onError?.(error)
      }

      await delay(retryDelay, signal)
      retryDelay = Math.min(retryDelay * 2, 10000)
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

  leaveParticipant(roomId: number, roundParticipantId: number) {
    return api
      .post<GameLeaveRoomResponse>(`/game/rooms/${roomId}/participants/${roundParticipantId}/leave`)
      .then((response) => response.data)
  },

  subscribeRoundEvents(roomId: number, roundId: number, handlers: RoundEventHandlers) {
    const controller = new AbortController()
    const auth = useAuthStore()
    const url = buildApiUrl(api.defaults.baseURL, `/game/rooms/${roomId}/rounds/${roundId}/events`)

    void runAuthorizedRoundEventStream(url, auth, controller.signal, handlers)
      .finally(() => {
        if (!controller.signal.aborted) {
          handlers.onClose?.()
        }
      })

    return () => controller.abort()
  },
}
