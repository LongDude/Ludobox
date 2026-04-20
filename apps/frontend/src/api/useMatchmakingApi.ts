import { api } from '@/api/base/useLudaApi'
import type { MatchmakingFilters, QuickMatchResponse, RecommendRoomsResponse } from './types'

function buildMatchmakingParams(filters?: MatchmakingFilters) {
  if (!filters) return undefined

  return {
    min_registration_price: filters.min_registration_price,
    max_registration_price: filters.max_registration_price,
    min_capacity: filters.min_capacity,
    max_capacity: filters.max_capacity,
    is_boost: filters.is_boost,
    min_boost_power: filters.min_boost_power,
    page: filters.page,
    page_size: filters.page_size,
  }
}

export const MatchmakingApi = {
  quickMatch(filters?: MatchmakingFilters) {
    return api
      .get<QuickMatchResponse>('/matchmaking/rooms/quick-match', {
        params: buildMatchmakingParams(filters),
      })
      .then((response) => response.data)
  },

  recommendRooms(filters?: MatchmakingFilters) {
    return api
      .get<RecommendRoomsResponse>('/matchmaking/rooms/recommendations', {
        params: buildMatchmakingParams(filters),
      })
      .then((response) => response.data)
  },
}
