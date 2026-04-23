<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import LeftTab from '@/components/LeftTab.vue'
import UpTab from '@/components/UpTab.vue'
import { UserApi } from '@/api/useUserApi'
import { useAuthStore } from '@/stores/authStore'
import type { UserGameHistoryItem, UserGameHistoryResult } from '@/api/types'
import { useI18n } from '@/i18n'
import { useLayoutInset } from '@/composables/useLayoutInset'

const auth = useAuthStore()
const { locale, t } = useI18n()
const { LeftTabHidden: leftHidden, layoutInset } = useLayoutInset()

const historyLoading = ref(false)
const historyErrorMsg = ref('')
const gameHistory = ref<UserGameHistoryItem[]>([])
const gameHistoryTotal = ref(0)
const gameHistoryPage = ref(1)
const gameHistoryPageSize = ref(8)

const gameHistoryPageSummary = computed(() =>
  t('common.pageSummary', {
    page: gameHistoryPage.value,
    pages: Math.max(1, Math.ceil(gameHistoryTotal.value / gameHistoryPageSize.value)),
    total: gameHistoryTotal.value,
    entity: t('profile.history.entity'),
  }),
)

onMounted(async () => {
  if (auth.isAuthenticated && !auth.User) {
    try {
      await auth.authenticate()
    } catch {}
  }

  if (auth.isAuthenticated) {
    void loadGameHistory().catch(() => {})
  }
})

async function loadGameHistory() {
  if (!auth.isAuthenticated) return

  historyLoading.value = true
  historyErrorMsg.value = ''

  try {
    const response = await UserApi.listCurrentUserGameHistory({
      page: gameHistoryPage.value,
      page_size: gameHistoryPageSize.value,
    })
    const normalizedItems = (response.items ?? []).map(normalizeHistoryItem)
    const aggregatedItems = aggregateHistoryItems(normalizedItems)
    const responsePage = response.page ?? gameHistoryPage.value
    const responsePageSize = response.page_size ?? gameHistoryPageSize.value
    const duplicateCount = Math.max(0, normalizedItems.length - aggregatedItems.length)
    const adjustedTotal = Math.max(aggregatedItems.length, Number(response.total ?? aggregatedItems.length) - duplicateCount)
    const visibleTotalFloor = Math.max(aggregatedItems.length, (Math.max(1, responsePage) - 1) * responsePageSize + aggregatedItems.length)
    gameHistory.value = aggregatedItems
    gameHistoryTotal.value = Math.max(adjustedTotal, visibleTotalFloor)
    gameHistoryPage.value = responsePage
    gameHistoryPageSize.value = responsePageSize
  } catch (error: any) {
    historyErrorMsg.value = error?.message || t('profile.history.error.load')
  } finally {
    historyLoading.value = false
  }
}

function previousHistoryPage() {
  if (gameHistoryPage.value > 1) {
    gameHistoryPage.value -= 1
    void loadGameHistory()
  }
}

function nextHistoryPage() {
  const pages = Math.max(1, Math.ceil(gameHistoryTotal.value / gameHistoryPageSize.value))
  if (gameHistoryPage.value < pages) {
    gameHistoryPage.value += 1
    void loadGameHistory()
  }
}

function formatMoney(value: number) {
  return new Intl.NumberFormat(locale.value === 'ru' ? 'ru-RU' : 'en-US').format(value)
}

function formatDateTime(value?: string | null) {
  if (!value) return '-'

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value

  return new Intl.DateTimeFormat(locale.value === 'ru' ? 'ru-RU' : 'en-US', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)
}

function resultLabel(result: UserGameHistoryResult) {
  if (result === 'won') return t('profile.history.result.won')
  if (result === 'lost') return t('profile.history.result.lost')
  if (result === 'left') return t('profile.history.result.left')
  if (result === 'cancelled') return t('profile.history.result.cancelled')
  if (result === 'waiting') return t('profile.history.result.waiting')
  if (result === 'active') return t('profile.history.result.active')
  if (result === 'finished') return t('profile.history.result.finished')
  return t('profile.history.result.unknown')
}

function resultTone(result: UserGameHistoryResult) {
  if (result === 'won') return 'won'
  if (result === 'lost') return 'lost'
  if (result === 'active' || result === 'waiting') return 'live'
  return 'neutral'
}

function normalizeSeatList(value: unknown, fallbackSeat?: unknown) {
  if (Array.isArray(value)) {
    return value
      .map((seat) => Number(seat))
      .filter((seat) => Number.isFinite(seat) && seat > 0)
  }

  const fallback = Number(fallbackSeat)
  if (Number.isFinite(fallback) && fallback > 0) {
    return [fallback]
  }

  return []
}

function inferWinningSeats(item: any, reservedSeats: number[]) {
  const explicitWinningSeats = normalizeSeatList(item?.winning_seats)
  if (explicitWinningSeats.length > 0) {
    return explicitWinningSeats
  }

  const legacySeat = normalizeSeatList(undefined, item?.seat_number)
  const winningSeatsCount = Number(item?.winning_seats_count ?? 0)
  const winningMoney = Number(item?.winning_money ?? 0)
  const result = String(item?.result ?? '').toLowerCase()
  const isWinningItem = winningMoney > 0 || result === 'won' || winningSeatsCount > 0

  if (legacySeat.length > 0 && isWinningItem) {
    return legacySeat
  }

  if (reservedSeats.length === 1 && isWinningItem) {
    return reservedSeats
  }

  return []
}

function normalizeHistoryItem(item: any): UserGameHistoryItem {
  const reservedSeats = normalizeSeatList(item?.reserved_seats, item?.seat_number)
  const winningSeats = inferWinningSeats(item, reservedSeats)

  return {
    round_id: Number(item?.round_id ?? 0),
    room_id: Number(item?.room_id ?? 0),
    game_id: Number(item?.game_id ?? 0),
    game_name: String(item?.game_name ?? ''),
    round_status: String(item?.round_status ?? ''),
    result: String(item?.result ?? '') as UserGameHistoryResult,
    reserved_seats: reservedSeats,
    winning_seats: winningSeats,
    reserved_seats_count: Number(item?.reserved_seats_count ?? reservedSeats.length ?? 0),
    winning_seats_count: Number(item?.winning_seats_count ?? winningSeats.length ?? 0),
    entry_fee: Number(item?.entry_fee ?? 0),
    boost_fee: Number(item?.boost_fee ?? 0),
    total_spent: Number(item?.total_spent ?? (Number(item?.entry_fee ?? 0) + Number(item?.boost_fee ?? 0))),
    winning_money: Number(item?.winning_money ?? 0),
    net_result: Number(item?.net_result ?? 0),
    joined_at: String(item?.joined_at ?? ''),
    finished_at: item?.finished_at ? String(item.finished_at) : null,
  }
}

function aggregateHistoryItems(items: UserGameHistoryItem[]) {
  const grouped = new Map<string, UserGameHistoryItem>()

  for (const item of items) {
    const key = [item.round_id, item.room_id, item.game_id].join(':')
    const existing = grouped.get(key)
    if (!existing) {
      grouped.set(key, {
        ...item,
        reserved_seats: [...item.reserved_seats].sort((left, right) => left - right),
        winning_seats: [...item.winning_seats].sort((left, right) => left - right),
      })
      continue
    }

    const reservedSeats = [...new Set([...existing.reserved_seats, ...item.reserved_seats])].sort(
      (left, right) => left - right,
    )
    const winningSeats = [...new Set([...existing.winning_seats, ...item.winning_seats])].sort(
      (left, right) => left - right,
    )

    grouped.set(key, {
      ...existing,
      round_status: pickRoundStatus(existing.round_status, item.round_status),
      result: pickRoundResult(existing.result, item.result, winningSeats.length),
      reserved_seats: reservedSeats,
      winning_seats: winningSeats,
      reserved_seats_count: reservedSeats.length,
      winning_seats_count: winningSeats.length,
      entry_fee: existing.entry_fee + item.entry_fee,
      boost_fee: existing.boost_fee + item.boost_fee,
      total_spent: existing.total_spent + item.total_spent,
      winning_money: existing.winning_money + item.winning_money,
      net_result: existing.net_result + item.net_result,
      joined_at: pickEarlierDate(existing.joined_at, item.joined_at),
      finished_at: pickLaterDate(existing.finished_at, item.finished_at),
    })
  }

  return [...grouped.values()].sort((left, right) => {
    const rightTime = Date.parse(right.finished_at ?? right.joined_at ?? '') || 0
    const leftTime = Date.parse(left.finished_at ?? left.joined_at ?? '') || 0
    return rightTime - leftTime
  })
}

function pickRoundStatus(current: string, next: string) {
  const currentValue = String(current || '').toLowerCase()
  const nextValue = String(next || '').toLowerCase()
  if (currentValue === nextValue) return current
  if (['finished', 'finalized', 'completed'].includes(currentValue)) return current
  if (['finished', 'finalized', 'completed'].includes(nextValue)) return next
  if (currentValue === 'active' || nextValue === 'active') return currentValue === 'active' ? current : next
  if (currentValue === 'waiting' || nextValue === 'waiting') return currentValue === 'waiting' ? current : next
  return current || next
}

function pickRoundResult(
  current: UserGameHistoryResult,
  next: UserGameHistoryResult,
  winningSeatsCount: number,
): UserGameHistoryResult {
  if (winningSeatsCount > 0 || current === 'won' || next === 'won') return 'won'

  const priority = ['active', 'waiting', 'lost', 'finished', 'left', 'cancelled']
  for (const value of priority) {
    if (current === value || next === value) return value
  }

  return current || next || 'unknown'
}

function pickEarlierDate(current?: string | null, next?: string | null) {
  if (!current) return next ?? ''
  if (!next) return current

  const currentTime = Date.parse(current)
  const nextTime = Date.parse(next)
  if (Number.isNaN(currentTime)) return current
  if (Number.isNaN(nextTime)) return next
  return currentTime <= nextTime ? current : next
}

function pickLaterDate(current?: string | null, next?: string | null) {
  if (!current) return next ?? null
  if (!next) return current

  const currentTime = Date.parse(current)
  const nextTime = Date.parse(next)
  if (Number.isNaN(currentTime)) return current
  if (Number.isNaN(nextTime)) return next
  return currentTime >= nextTime ? current : next
}

function formatSeats(value?: number[] | null) {
  if (!Array.isArray(value) || value.length === 0) return t('profile.history.seatsNone')
  return value.join(', ')
}
</script>

<template>
  <UpTab :show-menu="false" :show-upload="false" />
  <LeftTab />

  <div
    class="history-area"
    :class="{ collapsed: leftHidden }"
    :style="{ '--layout-inset': layoutInset }"
  >
    <div class="container">
      <section class="panel-card">
        <div class="card-head">
          <div>
            <h2>{{ t('profile.history.title') }}</h2>
            <p class="section-copy">{{ t('profile.history.description') }}</p>
          </div>
        </div>

        <p v-if="historyLoading && gameHistory.length === 0" class="state-copy">
          {{ t('profile.history.loading') }}
        </p>
        <p v-if="historyErrorMsg" class="state-copy error">{{ historyErrorMsg }}</p>

        <div v-if="!historyLoading || gameHistory.length > 0" class="history-list">
          <article v-for="item in gameHistory" :key="item.round_id" class="history-card">
            <div class="history-topline">
              <div>
                <strong>{{ formatDateTime(item.finished_at) }}</strong>
                <p class="muted">
                  {{ t('profile.history.roundSummary', { round: item.round_id, room: item.room_id, game: item.game_name }) }},
                  {{ t('profile.history.seatsWinningSummary', { count: item.winning_seats_count, selected: item.reserved_seats_count}) }}
                </p>
              </div>
              
              <span class="result-pill" :class="resultTone(item.result)">
                {{ resultLabel(item.result) }}
              </span>

            </div>


            <dl class="history-meta">
              <div>
                <dt>{{ t('profile.history.meta.entry') }}</dt>
                <dd>{{ formatMoney(item.entry_fee) }}</dd>
              </div>
              <div>
                <dt>{{ t('profile.history.meta.boost') }}</dt>
                <dd>{{ formatMoney(item.boost_fee) }}</dd>
              </div>
              <div>
                <dt>{{ t('profile.history.meta.totalSpent') }}</dt>
                <dd>{{ formatMoney(item.total_spent) }}</dd>
              </div>
              <div>
                <dt>{{ t('profile.history.meta.winning') }}</dt>
                <dd>{{ formatMoney(item.winning_money) }}</dd>
              </div>
              <div>
                <dt>{{ t('profile.history.meta.net') }}</dt>
                <dd :class="{ positive: item.net_result > 0, negative: item.net_result < 0 }">
                  {{ formatMoney(item.net_result) }}
                </dd>
              </div>
            </dl>
          </article>

          <p v-if="!historyLoading && gameHistory.length === 0" class="state-copy">
            {{ t('profile.history.empty') }}
          </p>
        </div>

        <div v-if="gameHistoryTotal > 0" class="pager">
          <button class="btn" :disabled="gameHistoryPage <= 1 || historyLoading" @click="previousHistoryPage">
            {{ t('common.prev') }}
          </button>
          <span class="pager-copy">{{ gameHistoryPageSummary }}</span>
          <button
            class="btn"
            :disabled="
              gameHistoryPage >= Math.max(1, Math.ceil(gameHistoryTotal / gameHistoryPageSize)) ||
              historyLoading
            "
            @click="nextHistoryPage"
          >
            {{ t('common.next') }}
          </button>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.history-area {
  position: fixed;
  inset: var(--layout-inset, 60px 20px 20px 310px);
  display: grid;
  align-items: start;
  overflow: auto;
  margin-top: 0.5rem;
  transition: all var(--transition-slow) ease;
}

.history-area.collapsed {
  --layout-inset: 60px 20px 20px 80px;
}

.container {
  max-width: none;
  margin: auto;
  width: 100%;
  padding: 0;
}

.panel-card {
  display: grid;
  gap: 1rem;
  padding: 1.35rem;
  border-radius: 1.5rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
  background: 
    radial-gradient(circle at top right, color-mix(in oklab, #0ea5e9, var(--color-surface) 95%), transparent 60%),
    linear-gradient(180deg, color-mix(in oklab, var(--color-surface), white 12%), var(--color-surface));
  box-shadow: var(--shadow-md);
}

.card-head,
.history-topline,
.pager {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: flex-start;
  flex-wrap: wrap;
}

.card-head h2 {
  margin: 0;
}

.eyebrow {
  margin: 0 0 0.35rem;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  font-size: 0.72rem;
  color: #0369a1;
}

.eyebrow.accent {
  color: #15803d;
}

.section-copy,
.muted,
.state-copy,
.pager-copy {
  color: var(--color-muted);
}

.section-copy,
.state-copy {
  margin: 0;
}

.state-copy.error {
  color: var(--color-danger);
}

.history-list {
  display: grid;
  gap: 0.85rem;
}

.history-card {
  display: grid;
  gap: 0.9rem;
  padding: 1rem;
  border: 1px solid color-mix(in oklab, var(--color-border), white 20%);
  border-radius: 1.1rem;
  background: color-mix(in oklab, var(--color-surface), white 2%);
}

.history-topline {
  align-items: flex-start;
}

.history-topline strong {
  display: block;
}

.seat-summary {
  display: grid;
  gap: 0.3rem;
  color: var(--color-muted);
  font-size: 0.88rem;
}

.result-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  white-space: nowrap;
  border-radius: 999px;
  padding: 0.35rem 0.75rem;
  font-size: 0.82rem;
  font-weight: 700;
}

.result-pill.won {
  background: color-mix(in oklab, var(--color-success), white 80%);
  color: #166534;
}

.result-pill.lost {
  background: color-mix(in oklab, var(--color-danger), white 84%);
  color: #991b1b;
}

.result-pill.live {
  background: color-mix(in oklab, #0284c7, white 82%);
  color: #075985;
}

.result-pill.neutral {
  background: color-mix(in oklab, var(--color-border), white 38%);
  color: var(--color-text);
}

.history-meta {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr));
  gap: 0.75rem;
  margin: 0;
}

.history-meta div {
  display: grid;
  gap: 0.15rem;
}

.history-meta dt {
  color: var(--color-muted);
  font-size: 0.8rem;
}

.history-meta dd {
  margin: 0;
  font-weight: 600;
}

.history-meta dd.positive {
  color: var(--color-success);
}

.history-meta dd.negative {
  color: var(--color-danger);
}

.pager {
  align-items: center;
}

.btn {
  appearance: none;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  background: color-mix(in oklab, var(--color-surface), white 8%);
  color: var(--color-text);
  border-radius: 999px;
  padding: 0.8rem 1rem;
  font-weight: 600;
  cursor: pointer;
  transition:
    transform var(--transition-fast) ease,
    border-color var(--transition-fast) ease,
    background var(--transition-fast) ease;
}

.btn:hover {
  transform: translateY(-1px);
}

.btn:disabled {
  cursor: not-allowed;
  opacity: 0.6;
  transform: none;
}

@media (max-width: 960px) {
  .history-area,
  .history-area.collapsed {
    position: static;
    inset: auto;
    margin: calc(76px + 0.75rem) 1rem 5.75rem;
  }
}

@media (max-width: 760px) {
  .panel-card {
    padding: 1rem;
  }
}
</style>
