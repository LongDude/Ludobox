import type { MatchmakingFilters } from '@/api/types'
import type { LocationQuery, LocationQueryRaw } from 'vue-router'

export const DEFAULT_MATCHMAKING_PAGE = 1
export const DEFAULT_MATCHMAKING_PAGE_SIZE = 12

export type BoostFilterMode = 'any' | 'true' | 'false'

export interface MatchmakingFilterDraft {
  minRegistrationPrice: string
  maxRegistrationPrice: string
  minCapacity: string
  maxCapacity: string
  boostMode: BoostFilterMode
  minBoostPower: string
}

type MatchmakingDraftField = keyof MatchmakingFilterDraft

export interface MatchmakingValidationResult {
  filters: MatchmakingFilters
  fieldErrors: Partial<Record<MatchmakingDraftField, string>>
}

function firstQueryValue(value: string | string[] | null | undefined) {
  if (Array.isArray(value)) return value[0] ?? ''
  return value ?? ''
}

function parseInteger(input: string | number | null | undefined) {
  if (input === null || input === undefined) return undefined

  const trimmed = String(input).trim()
  if (!trimmed) return undefined

  const numeric = Number(trimmed)
  if (!Number.isInteger(numeric)) return Number.NaN
  return numeric
}

function parseBooleanQuery(input: string) {
  if (input === 'true') return true
  if (input === 'false') return false
  return undefined
}

export function createMatchmakingDraft(filters?: MatchmakingFilters): MatchmakingFilterDraft {
  return {
    minRegistrationPrice:
      filters?.min_registration_price === undefined ? '' : String(filters.min_registration_price),
    maxRegistrationPrice:
      filters?.max_registration_price === undefined ? '' : String(filters.max_registration_price),
    minCapacity: filters?.min_capacity === undefined ? '' : String(filters.min_capacity),
    maxCapacity: filters?.max_capacity === undefined ? '' : String(filters.max_capacity),
    boostMode:
      filters?.is_boost === undefined ? 'any' : filters.is_boost ? 'true' : 'false',
    minBoostPower: filters?.min_boost_power === undefined ? '' : String(filters.min_boost_power),
  }
}

export function normalizeMatchmakingDraft(draft: MatchmakingFilterDraft): MatchmakingValidationResult {
  const filters: MatchmakingFilters = {}
  const fieldErrors: Partial<Record<MatchmakingDraftField, string>> = {}

  const minRegistrationPrice = parseInteger(draft.minRegistrationPrice)
  const maxRegistrationPrice = parseInteger(draft.maxRegistrationPrice)
  const minCapacity = parseInteger(draft.minCapacity)
  const maxCapacity = parseInteger(draft.maxCapacity)
  const minBoostPower = parseInteger(draft.minBoostPower)

  if (Number.isNaN(minRegistrationPrice) || (minRegistrationPrice !== undefined && minRegistrationPrice < 0)) {
    fieldErrors.minRegistrationPrice = 'integer'
  } else if (minRegistrationPrice !== undefined) {
    filters.min_registration_price = minRegistrationPrice
  }

  if (Number.isNaN(maxRegistrationPrice) || (maxRegistrationPrice !== undefined && maxRegistrationPrice < 0)) {
    fieldErrors.maxRegistrationPrice = 'integer'
  } else if (maxRegistrationPrice !== undefined) {
    filters.max_registration_price = maxRegistrationPrice
  }

  if (Number.isNaN(minCapacity) || (minCapacity !== undefined && minCapacity <= 0)) {
    fieldErrors.minCapacity = 'positive'
  } else if (minCapacity !== undefined) {
    filters.min_capacity = minCapacity
  }

  if (Number.isNaN(maxCapacity) || (maxCapacity !== undefined && maxCapacity <= 0)) {
    fieldErrors.maxCapacity = 'positive'
  } else if (maxCapacity !== undefined) {
    filters.max_capacity = maxCapacity
  }

  if (draft.boostMode !== 'any') {
    filters.is_boost = draft.boostMode === 'true'
  }

  if (Number.isNaN(minBoostPower) || (minBoostPower !== undefined && minBoostPower < 0)) {
    fieldErrors.minBoostPower = 'integer'
  } else if (minBoostPower !== undefined) {
    filters.min_boost_power = minBoostPower
  }

  if (
    filters.min_registration_price !== undefined &&
    filters.max_registration_price !== undefined &&
    filters.min_registration_price > filters.max_registration_price
  ) {
    fieldErrors.maxRegistrationPrice = 'range'
  }

  if (
    filters.min_capacity !== undefined &&
    filters.max_capacity !== undefined &&
    filters.min_capacity > filters.max_capacity
  ) {
    fieldErrors.maxCapacity = 'range'
  }

  return { filters, fieldErrors }
}

export function filtersToQuery(filters: MatchmakingFilters): LocationQueryRaw {
  const query: LocationQueryRaw = {}

  if (filters.min_registration_price !== undefined) {
    query.min_registration_price = String(filters.min_registration_price)
  }
  if (filters.max_registration_price !== undefined) {
    query.max_registration_price = String(filters.max_registration_price)
  }
  if (filters.min_capacity !== undefined) {
    query.min_capacity = String(filters.min_capacity)
  }
  if (filters.max_capacity !== undefined) {
    query.max_capacity = String(filters.max_capacity)
  }
  if (filters.is_boost !== undefined) {
    query.is_boost = filters.is_boost ? 'true' : 'false'
  }
  if (filters.min_boost_power !== undefined) {
    query.min_boost_power = String(filters.min_boost_power)
  }
  if (filters.page !== undefined && filters.page > 1) {
    query.page = String(filters.page)
  }
  if (
    filters.page_size !== undefined &&
    filters.page_size > 0 &&
    filters.page_size !== DEFAULT_MATCHMAKING_PAGE_SIZE
  ) {
    query.page_size = String(filters.page_size)
  }

  return query
}

export function queryToMatchmakingFilters(query: LocationQuery): MatchmakingFilters {
  const filters: MatchmakingFilters = {}

  const minRegistrationPrice = parseInteger(firstQueryValue(query.min_registration_price as any))
  const maxRegistrationPrice = parseInteger(firstQueryValue(query.max_registration_price as any))
  const minCapacity = parseInteger(firstQueryValue(query.min_capacity as any))
  const maxCapacity = parseInteger(firstQueryValue(query.max_capacity as any))
  const minBoostPower = parseInteger(firstQueryValue(query.min_boost_power as any))
  const page = parseInteger(firstQueryValue(query.page as any))
  const pageSize = parseInteger(firstQueryValue(query.page_size as any))
  const isBoost = parseBooleanQuery(firstQueryValue(query.is_boost as any))

  if (minRegistrationPrice !== undefined && !Number.isNaN(minRegistrationPrice) && minRegistrationPrice >= 0) {
    filters.min_registration_price = minRegistrationPrice
  }
  if (maxRegistrationPrice !== undefined && !Number.isNaN(maxRegistrationPrice) && maxRegistrationPrice >= 0) {
    filters.max_registration_price = maxRegistrationPrice
  }
  if (minCapacity !== undefined && !Number.isNaN(minCapacity) && minCapacity > 0) {
    filters.min_capacity = minCapacity
  }
  if (maxCapacity !== undefined && !Number.isNaN(maxCapacity) && maxCapacity > 0) {
    filters.max_capacity = maxCapacity
  }
  if (isBoost !== undefined) {
    filters.is_boost = isBoost
  }
  if (minBoostPower !== undefined && !Number.isNaN(minBoostPower) && minBoostPower >= 0) {
    filters.min_boost_power = minBoostPower
  }
  if (page !== undefined && !Number.isNaN(page) && page > 0) {
    filters.page = page
  }
  if (pageSize !== undefined && !Number.isNaN(pageSize) && pageSize > 0) {
    filters.page_size = pageSize
  }

  return filters
}
