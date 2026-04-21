import { api } from '@/api/base/useLudaApi'
import type {
  GameCancelBoostResponse,
  GameJoinRoomResponse,
  GameJoinRoomWithSeatRequest,
  GameLeaveRoomResponse,
  GamePurchaseBoostRequest,
  GamePurchaseBoostResponse,
  GameRoundStatusResponse,
} from './types'

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

  getRoundStatus(roomId: number, roundId: number) {
    return api
      .get<GameRoundStatusResponse>(`/game/rooms/${roomId}/rounds/${roundId}`)
      .then((response) => response.data)
  },

  purchaseBoost(roomId: number, roundParticipantId: number, payload: GamePurchaseBoostRequest) {
    return api
      .post<GamePurchaseBoostResponse>(
        `/game/rooms/${roomId}/participants/${roundParticipantId}/boost`,
        payload,
      )
      .then((response) => response.data)
  },

  cancelBoost(roomId: number, roundParticipantId: number) {
    return api
      .delete<GameCancelBoostResponse>(`/game/rooms/${roomId}/participants/${roundParticipantId}/boost`)
      .then((response) => response.data)
  },

  leaveRoom(roomId: number, roundParticipantId: number) {
    return api
      .post<GameLeaveRoomResponse>(`/game/rooms/${roomId}/participants/${roundParticipantId}/leave`)
      .then((response) => response.data)
  },
}
