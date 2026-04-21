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
    gameHistory.value = response.items ?? []
    gameHistoryTotal.value = response.total ?? 0
    gameHistoryPage.value = response.page ?? gameHistoryPage.value
    gameHistoryPageSize.value = response.page_size ?? gameHistoryPageSize.value
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
            <p class="eyebrow accent">{{ t('profile.history.eyebrow') }}</p>
            <h2>{{ t('profile.history.title') }}</h2>
            <p class="section-copy">{{ t('profile.history.description') }}</p>
          </div>
          <button class="btn" :disabled="historyLoading" @click="loadGameHistory">
            {{ t('common.refresh') }}
          </button>
        </div>

        <p v-if="historyLoading && gameHistory.length === 0" class="state-copy">
          {{ t('profile.history.loading') }}
        </p>
        <p v-if="historyErrorMsg" class="state-copy error">{{ historyErrorMsg }}</p>

        <div v-if="!historyLoading || gameHistory.length > 0" class="history-list">
          <article v-for="item in gameHistory" :key="item.participant_id" class="history-card">
            <div class="history-topline">
              <div>
                <strong>{{ item.game_name || t('profile.history.gameFallback', { id: item.game_id }) }}</strong>
                <p class="muted">
                  {{
                    t('profile.history.roundSummary', {
                      round: item.round_id,
                      room: item.room_id,
                      seat: item.seat_number,
                    })
                  }}
                </p>
              </div>
              <span class="result-pill" :class="resultTone(item.result)">
                {{ resultLabel(item.result) }}
              </span>
            </div>

            <dl class="history-meta">
              <div>
                <dt>{{ t('profile.history.meta.joined') }}</dt>
                <dd>{{ formatDateTime(item.joined_at) }}</dd>
              </div>
              <div>
                <dt>{{ t('profile.history.meta.finished') }}</dt>
                <dd>{{ formatDateTime(item.finished_at) }}</dd>
              </div>
              <div>
                <dt>{{ t('profile.history.meta.entry') }}</dt>
                <dd>{{ formatMoney(item.entry_fee) }}</dd>
              </div>
              <div>
                <dt>{{ t('profile.history.meta.boost') }}</dt>
                <dd>{{ formatMoney(item.boost_fee) }}</dd>
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
  transition: all var(--transition-slow) ease;
}

.history-area.collapsed {
  --layout-inset: 60px 20px 20px 80px;
}

.container {
  max-width: 1100px;
  margin: auto;
  width: 100%;
}

.panel-card {
  display: grid;
  gap: 1rem;
  padding: 1.35rem;
  border-radius: 1.5rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
  background:
    radial-gradient(circle at top left, color-mix(in oklab, #0ea5e9, white 88%), transparent 26%),
    linear-gradient(180deg, color-mix(in oklab, var(--color-surface), white 16%), var(--color-surface));
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
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  border-radius: 1.1rem;
  background: color-mix(in oklab, var(--color-surface), white 10%);
}

.history-topline {
  align-items: flex-start;
}

.history-topline strong {
  display: block;
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
