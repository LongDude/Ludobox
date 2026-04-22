import type { UserRank } from '@/api/types'

const KNOWN_RANKS = new Set(['bronze', 'silver', 'gold', 'platinum', 'diamond'])

export function normalizeRank(rank?: UserRank | null) {
  const normalized = String(rank ?? '').trim().toLowerCase()
  return KNOWN_RANKS.has(normalized) ? normalized : 'unranked'
}

export function rankFrameClass(rank?: UserRank | null) {
  return `rank-frame--${normalizeRank(rank)}`
}
