import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import type { MatchmakingFilters, QuickMatchResponse, RoomRecommendationResponse } from '@/api/types'
import { useAuthStore } from '@/stores/authStore'

export type MatchSessionSource = 'quick-match' | 'recommendation'

export interface QuickMatchMeta {
  round_id: number
  round_participant_id: number
  seat_number: number
  reused_existing_room: boolean
}

export const useMatchSessionStore = defineStore('matchSession', () => {
  const auth = useAuthStore()

  const source = ref<MatchSessionSource | null>(null)
  const selectedRoom = ref<RoomRecommendationResponse | null>(null)
  const quickMatchMeta = ref<QuickMatchMeta | null>(null)
  const filters = ref<MatchmakingFilters | null>(null)
  const loading = ref(false)
  const error = ref('')

  const activeRoomId = computed(() => selectedRoom.value?.room_id ?? null)

  function reset() {
    source.value = null
    selectedRoom.value = null
    quickMatchMeta.value = null
    filters.value = null
    loading.value = false
    error.value = ''
  }

  function setLoading(next = true) {
    loading.value = next
    if (next) {
      error.value = ''
    }
  }

  function setError(message: string) {
    error.value = message
    loading.value = false
  }

  function setFilters(next: MatchmakingFilters | null | undefined) {
    filters.value = next ? { ...next } : null
  }

  function setQuickMatchSession(response: QuickMatchResponse, requestFilters?: MatchmakingFilters) {
    source.value = 'quick-match'
    selectedRoom.value = response.room
    quickMatchMeta.value = {
      round_id: response.round_id,
      round_participant_id: response.round_participant_id,
      seat_number: response.seat_number,
      reused_existing_room: response.reused_existing_room,
    }
    setFilters(requestFilters)
    loading.value = false
    error.value = ''
  }

  function setRecommendedRoomSession(
    room: RoomRecommendationResponse,
    requestFilters?: MatchmakingFilters,
  ) {
    source.value = 'recommendation'
    selectedRoom.value = room
    quickMatchMeta.value = null
    setFilters(requestFilters)
    loading.value = false
    error.value = ''
  }

  watch(
    () => auth.isAuthenticated,
    (isAuthenticated) => {
      if (!isAuthenticated) {
        reset()
      }
    },
    { immediate: true },
  )

  return {
    source,
    selectedRoom,
    quickMatchMeta,
    filters,
    loading,
    error,
    activeRoomId,
    setLoading,
    setError,
    setFilters,
    setQuickMatchSession,
    setRecommendedRoomSession,
    reset,
  }
})
