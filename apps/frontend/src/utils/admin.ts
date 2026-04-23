import type { ConfigUpsertRequest } from '@/api/types'
import { t } from '@/i18n'

export interface ConfigIssue {
  tone: 'error' | 'warning'
  message: string
}

export interface ConfigMetric {
  label: string
  value: string
  hint: string
}

export interface WinnerProjection {
  place: number
  percent: number
  amount: number
}

export interface ConfigProjection {
  metrics: ConfigMetric[]
  winners: WinnerProjection[]
}

const numberFormatter = new Intl.NumberFormat()

export function formatInteger(value: number) {
  return numberFormatter.format(Math.round(value))
}

export function parseDistributionInput(value: string) {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
    .map((item) => Number(item))
    .filter((item) => Number.isFinite(item))
}

export function distributionToInput(values: number[]) {
  return values.join(', ')
}

export function rebalanceDistribution(count: number, current: number[] = []) {
  const safeCount = Math.max(1, Math.floor(count || 1))
  const normalized = current
    .map((value) => Math.max(0, Math.floor(value)))
    .slice(0, safeCount)

  while (normalized.length < safeCount) {
    normalized.push(0)
  }

  const total = normalized.reduce((sum, value) => sum + value, 0)
  if (total === 100) {
    return normalized
  }

  const base = Math.floor(100 / safeCount)
  const remainder = 100 - base * safeCount
  return Array.from({ length: safeCount }, (_, index) => base + (index < remainder ? 1 : 0))
}

export function normalizeConfigDraft(config: ConfigUpsertRequest): ConfigUpsertRequest {
  const numberWinners = Math.max(1, Math.floor(config.number_winners || 1))
  const normalizedDistribution = rebalanceDistribution(numberWinners, config.winning_distribution)

  return {
    game_id: Math.max(1, Math.floor(config.game_id || 1)),
    capacity: Math.max(1, Math.floor(config.capacity || 1)),
    registration_price: Math.max(0, Math.floor(config.registration_price || 0)),
    is_boost: Boolean(config.is_boost),
    boost_price: config.is_boost ? Math.max(0, Math.floor(config.boost_price || 0)) : 0,
    boost_power: config.is_boost ? Math.max(0, Math.floor(config.boost_power || 0)) : 0,
    number_winners: numberWinners,
    winning_distribution: normalizedDistribution,
    commission: Math.max(0, Math.floor(config.commission || 0)),
    time: Math.max(0, Math.floor(config.time || 0)),
    round_time: Math.max(0, Math.floor(config.round_time || 0)),
    next_round_delay: Math.max(0, Math.floor(config.next_round_delay || 0)),
    min_users: Math.max(1, Math.floor(config.min_users || 1)),
  }
}

export function validateConfigDraft(config: ConfigUpsertRequest) {
  const issues: ConfigIssue[] = []
  const distribution = config.winning_distribution ?? []
  const distributionSum = distribution.reduce((sum, value) => sum + value, 0)

  if (config.game_id <= 0) {
    issues.push({ tone: 'error', message: t('admin.configsSection.validation.gameIdPositive') })
  }
  if (config.capacity < 2 || config.capacity > 20) {
    issues.push({ tone: 'error', message: t('admin.configsSection.validation.capacityRange') })
  }
  if (config.capacity > 10) {
    issues.push({
      tone: 'warning',
      message: t('admin.configsSection.validation.capacityWarning'),
    })
  }
  if (config.registration_price < 0) {
    issues.push({
      tone: 'error',
      message: t('admin.configsSection.validation.registrationPriceNegative'),
    })
  }
  if (config.number_winners < 1 || config.number_winners > 20) {
    issues.push({ tone: 'error', message: t('admin.configsSection.validation.winnersRange') })
  }
  if (config.number_winners > config.capacity) {
    issues.push({ tone: 'error', message: t('admin.configsSection.validation.winnersCapacity') })
  }
  if (distribution.length !== config.number_winners) {
    issues.push({
      tone: 'error',
      message: t('admin.configsSection.validation.distributionLength'),
    })
  }
  if (distribution.some((value) => value < 0 || value > 100)) {
    issues.push({
      tone: 'error',
      message: t('admin.configsSection.validation.distributionRange'),
    })
  }
  if (distributionSum !== 100) {
    issues.push({
      tone: 'error',
      message: t('admin.configsSection.validation.distributionSum'),
    })
  }
  if (config.commission < 0 || config.commission > 100) {
    issues.push({ tone: 'error', message: t('admin.configsSection.validation.commissionRange') })
  }
  if (config.commission >= 40) {
    issues.push({
      tone: 'warning',
      message: t('admin.configsSection.validation.commissionWarning'),
    })
  }
  if (config.time <= 0) {
    issues.push({ tone: 'error', message: t('admin.configsSection.validation.timerPositive') })
  }
  if (config.time > 180) {
    issues.push({
      tone: 'warning',
      message: t('admin.configsSection.validation.timerWarning'),
    })
  }
  if (config.round_time <= 0) {
    issues.push({
      tone: 'error',
      message: t('admin.configsSection.validation.roundTimePositive'),
    })
  }
  if (config.round_time > 300) {
    issues.push({
      tone: 'warning',
      message: t('admin.configsSection.validation.roundTimeWarning'),
    })
  }
  if (config.next_round_delay < 0) {
    issues.push({
      tone: 'error',
      message: t('admin.configsSection.validation.nextRoundDelayPositive'),
    })
  }
  if (config.next_round_delay > 60) {
    issues.push({
      tone: 'warning',
      message: t('admin.configsSection.validation.nextRoundDelayWarning'),
    })
  }
  if (config.min_users < 1) {
    issues.push({ tone: 'error', message: t('admin.configsSection.validation.minUsersMin') })
  }
  if (config.min_users > config.capacity) {
    issues.push({
      tone: 'error',
      message: t('admin.configsSection.validation.minUsersCapacity'),
    })
  }
  if (config.min_users > Math.ceil(config.capacity * 0.75)) {
    issues.push({
      tone: 'warning',
      message: t('admin.configsSection.validation.minUsersWarning'),
    })
  }
  if (config.is_boost) {
    if (config.boost_price < 0) {
      issues.push({
        tone: 'error',
        message: t('admin.configsSection.validation.boostPriceNegative'),
      })
    }
    if (config.boost_power < 0 || config.boost_power > 100) {
      issues.push({
        tone: 'error',
        message: t('admin.configsSection.validation.boostPowerRange'),
      })
    }
    if (config.boost_power === 0) {
      issues.push({
        tone: 'warning',
        message: t('admin.configsSection.validation.boostPowerZero'),
      })
    }
    if (config.boost_price > config.registration_price * 2 && config.registration_price > 0) {
      issues.push({
        tone: 'warning',
        message: t('admin.configsSection.validation.boostPriceWarning'),
      })
    }
  } else if (config.boost_price !== 0 || config.boost_power !== 0) {
    issues.push({
      tone: 'error',
      message: t('admin.configsSection.validation.boostDisabled'),
    })
  }

  return issues
}

export function projectConfigEconomics(config: ConfigUpsertRequest): ConfigProjection {
  const grossBank = config.capacity * config.registration_price
  const organizerShare = Math.round((grossBank * config.commission) / 100)
  const prizePool = grossBank - organizerShare
  const startThresholdBank = config.min_users * config.registration_price
  const maxBoostRevenue = config.is_boost ? config.capacity * config.boost_price : 0

  return {
    metrics: [
      {
        label: t('admin.configsSection.metrics.fullRoomBank.label'),
        value: formatInteger(grossBank),
        hint: t('admin.configsSection.metrics.fullRoomBank.hint', {
          capacity: config.capacity,
          price: formatInteger(config.registration_price),
        }),
      },
      {
        label: t('admin.configsSection.metrics.prizePool.label'),
        value: formatInteger(prizePool),
        hint: t('admin.configsSection.metrics.prizePool.hint', {
          percent: 100 - config.commission,
        }),
      },
      {
        label: t('admin.configsSection.metrics.operatorShare.label'),
        value: formatInteger(organizerShare),
        hint: t('admin.configsSection.metrics.operatorShare.hint', {
          commission: config.commission,
        }),
      },
      {
        label: t('admin.configsSection.metrics.startThresholdBank.label'),
        value: formatInteger(startThresholdBank),
        hint: t('admin.configsSection.metrics.startThresholdBank.hint', {
          minUsers: config.min_users,
        }),
      },
      {
        label: t('admin.configsSection.metrics.maxBoostRevenue.label'),
        value: formatInteger(maxBoostRevenue),
        hint: config.is_boost
          ? t('admin.configsSection.metrics.maxBoostRevenue.hint', {
              capacity: config.capacity,
              price: formatInteger(config.boost_price),
            })
          : t('admin.configsSection.metrics.maxBoostRevenue.disabled'),
      },
    ],
    winners: config.winning_distribution.map((percent, index) => ({
      place: index + 1,
      percent,
      amount: Math.round((prizePool * percent) / 100),
    })),
  }
}
