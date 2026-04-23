<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import LeftTab from '@/components/LeftTab.vue'
import UpTab from '@/components/UpTab.vue'
import FooterTab from '@/components/FooterTab.vue'
import { useAuthStore } from '@/stores/authStore'
import { useMatchSessionStore } from '@/stores/matchSessionStore'
import { MatchmakingApi } from '@/api/useMatchmakingApi'
import type { MatchmakingFilters, Pagination, RoomRecommendationResponse } from '@/api/types'
import { useI18n } from '@/i18n'
import { useLayoutInset } from '@/composables/useLayoutInset'
import {
  DEFAULT_MATCHMAKING_PAGE,
  DEFAULT_MATCHMAKING_PAGE_SIZE,
  createMatchmakingDraft,
  filtersToQuery,
  normalizeMatchmakingDraft,
  queryToMatchmakingFilters,
  type MatchmakingFilterDraft,
} from '@/utils/matchmaking'

const auth = useAuthStore()
const session = useMatchSessionStore()
const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const { LeftTabHidden: leftHidden, layoutInset } = useLayoutInset()
const ROOM_VIEW_MODE_KEY = 'matchmaking.rooms.view-mode'

type RoomViewMode = 'grid' | 'list'

function loadRoomViewMode(): RoomViewMode {
  try {
    return localStorage.getItem(ROOM_VIEW_MODE_KEY) === 'list' ? 'list' : 'grid'
  } catch {
    return 'grid'
  }
}

function normalizeRequestFilters(filters: MatchmakingFilters): MatchmakingFilters {
  return {
    ...filters,
    page: filters.page ?? DEFAULT_MATCHMAKING_PAGE,
    page_size: filters.page_size ?? DEFAULT_MATCHMAKING_PAGE_SIZE,
  }
}

function mergeRouteFilters() {
  const storedFilters = session.filters ?? {}
  const routeFilters = queryToMatchmakingFilters(route.query)
  return normalizeRequestFilters({
    ...storedFilters,
    ...routeFilters,
  })
}

const draft = reactive<MatchmakingFilterDraft>(createMatchmakingDraft(mergeRouteFilters()))
const rooms = ref<RoomRecommendationResponse[]>([])
const pagination = ref<Pagination | null>(null)
const loading = ref(false)
const cached = ref(false)
const errorMsg = ref('')
const viewMode = ref<RoomViewMode>(loadRoomViewMode())

let lastLoadId = 0

const validationResult = computed(() => normalizeMatchmakingDraft(draft))

const requestFilters = computed(() => mergeRouteFilters())

const sortedRooms = computed(() =>
  [...rooms.value].sort((left, right) => right.score - left.score || left.room_id - right.room_id),
)

const totalPages = computed(() => {
  if (!pagination.value) return 1
  return Math.max(1, Math.ceil(pagination.value.total / pagination.value.page_size))
})

watch(
  () => route.query,
  () => {
    const filters = requestFilters.value
    Object.assign(draft, createMatchmakingDraft(filters))
    session.setFilters(filters)
    void loadRooms()
  },
  { immediate: true, deep: true },
)

watch(
  () => auth.isAuthenticated,
  (isAuthenticated) => {
    if (!isAuthenticated) {
      rooms.value = []
      pagination.value = null
      cached.value = false
      loading.value = false
      return
    }

    void loadRooms()
  },
)

function getFieldError(field: keyof MatchmakingFilterDraft) {
  const code = validationResult.value.fieldErrors[field]
  if (!code) return ''

  if (field === 'minRegistrationPrice' || field === 'maxRegistrationPrice') {
    return code === 'range'
      ? t('matchmaking.validation.entryRange')
      : t('matchmaking.validation.nonNegativeInteger')
  }

  if (field === 'minCapacity' || field === 'maxCapacity') {
    return code === 'range'
      ? t('matchmaking.validation.capacityRange')
      : t('matchmaking.validation.positiveInteger')
  }

  return t('matchmaking.validation.nonNegativeInteger')
}

function formatBoost(room: RoomRecommendationResponse) {
  if (!room.is_boost) return t('common.off')
  return t('matchmaking.results.boostValue', { value: room.boost_power })
}

function formatScore(room: RoomRecommendationResponse) {
  return room.score.toFixed(2)
}

function setViewMode(next: RoomViewMode) {
  viewMode.value = next
  try {
    localStorage.setItem(ROOM_VIEW_MODE_KEY, next)
  } catch {}
}

function redirectToAuth() {
  router.push({
    path: '/auth',
    query: {
      redirect: route.fullPath,
    },
  })
}

async function loadRooms() {
  if (!auth.isAuthenticated) {
    return
  }

  const loadId = ++lastLoadId
  loading.value = true
  errorMsg.value = ''

  try {
    const response = await MatchmakingApi.recommendRooms(requestFilters.value)

    if (loadId !== lastLoadId) return

    rooms.value = response.items
    pagination.value = response.pagination
    cached.value = response.cached
  } catch (error: any) {
    if (loadId !== lastLoadId) return

    rooms.value = []
    pagination.value = null
    cached.value = false
    errorMsg.value = error?.message || t('matchmaking.errors.recommendations')
  } finally {
    if (loadId === lastLoadId) {
      loading.value = false
    }
  }
}

function applyFilters() {
  if (Object.keys(validationResult.value.fieldErrors).length) {
    errorMsg.value = t('matchmaking.messages.fixFilters')
    return
  }

  errorMsg.value = ''
  const nextFilters = normalizeRequestFilters({
    ...validationResult.value.filters,
    page: DEFAULT_MATCHMAKING_PAGE,
    page_size: requestFilters.value.page_size ?? DEFAULT_MATCHMAKING_PAGE_SIZE,
  })
  session.setFilters(nextFilters)

  router.push({
    path: '/rooms',
    query: filtersToQuery(nextFilters),
  })
}

function resetFilters() {
  errorMsg.value = ''
  session.setFilters(null)
  router.push({ path: '/rooms' })
}

function changePage(nextPage: number) {
  if (nextPage < 1 || (pagination.value && nextPage > totalPages.value)) return

  const nextFilters = {
    ...requestFilters.value,
    page: nextPage,
  }
  session.setFilters(nextFilters)
  router.push({
    path: '/rooms',
    query: filtersToQuery(nextFilters),
  })
}

function enterRoom(room: RoomRecommendationResponse) {
  session.setRecommendedRoomSession(room, requestFilters.value)
  router.push(`/play/${room.room_id}`)
}
</script>

<template>
  <UpTab :show-menu="false" :show-upload="false" />
  <LeftTab />

  <div class="rooms-area" :class="{ collapsed: leftHidden }" :style="{ '--layout-inset': layoutInset }">
    <section class="panel-card filters-panel">
      <div class="card-head">
        <div>
          <p class="eyebrow">{{ t('matchmaking.results.eyebrow') }}</p>
          <h1>{{ t('matchmaking.results.title') }}</h1>
          <p class="description">{{ t('matchmaking.results.description') }}</p>
        </div>
      </div>

      <form class="filters-form" @submit.prevent="applyFilters">
        <label>
          <span>{{ t('matchmaking.filters.minEntry') }}</span>
          <input
            v-model="draft.minRegistrationPrice"
            inputmode="numeric"
            type="number"
            min="0"
            step="1"
            :placeholder="t('matchmaking.filters.placeholder.any')"
          />
          <small v-if="getFieldError('minRegistrationPrice')" class="field-error">
            {{ getFieldError('minRegistrationPrice') }}
          </small>
        </label>

        <label>
          <span>{{ t('matchmaking.filters.maxEntry') }}</span>
          <input
            v-model="draft.maxRegistrationPrice"
            inputmode="numeric"
            type="number"
            min="0"
            step="1"
            :placeholder="t('matchmaking.filters.placeholder.any')"
          />
          <small v-if="getFieldError('maxRegistrationPrice')" class="field-error">
            {{ getFieldError('maxRegistrationPrice') }}
          </small>
        </label>

        <label>
          <span>{{ t('matchmaking.filters.minCapacity') }}</span>
          <input
            v-model="draft.minCapacity"
            inputmode="numeric"
            type="number"
            min="1"
            step="1"
            :placeholder="t('matchmaking.filters.placeholder.any')"
          />
          <small v-if="getFieldError('minCapacity')" class="field-error">
            {{ getFieldError('minCapacity') }}
          </small>
        </label>

        <label>
          <span>{{ t('matchmaking.filters.maxCapacity') }}</span>
          <input
            v-model="draft.maxCapacity"
            inputmode="numeric"
            type="number"
            min="1"
            step="1"
            :placeholder="t('matchmaking.filters.placeholder.any')"
          />
          <small v-if="getFieldError('maxCapacity')" class="field-error">
            {{ getFieldError('maxCapacity') }}
          </small>
        </label>

        <label>
          <span>{{ t('matchmaking.filters.boost') }}</span>
          <select v-model="draft.boostMode">
            <option value="any">{{ t('matchmaking.filters.boostAny') }}</option>
            <option value="true">{{ t('matchmaking.filters.boostOnly') }}</option>
            <option value="false">{{ t('matchmaking.filters.noBoost') }}</option>
          </select>
        </label>

        <label>
          <span>{{ t('matchmaking.filters.minBoostPower') }}</span>
          <input
            v-model="draft.minBoostPower"
            inputmode="numeric"
            type="number"
            min="0"
            step="1"
            :placeholder="t('matchmaking.filters.placeholder.any')"
          />
          <small v-if="getFieldError('minBoostPower')" class="field-error">
            {{ getFieldError('minBoostPower') }}
          </small>
        </label>

        <div class="form-actions">
          <button class="btn" type="button" @click="resetFilters">
            {{ t('common.reset') }}
          </button>
          <button class="btn btn--primary" type="submit">
            {{ t('matchmaking.filters.apply') }}
          </button>
        </div>
      </form>
    </section>

    <section class="panel-card results-panel">
      <div class="card-head">
        <div>
          <p class="eyebrow accent">{{ t('matchmaking.results.listEyebrow') }}</p>
          <h2>{{ t('matchmaking.results.listTitle') }}</h2>
          <p class="description">
            {{
              pagination
                ? t('common.pageSummary', {
                    page: pagination.page,
                    pages: totalPages,
                    total: pagination.total,
                    entity: t('matchmaking.results.roomsEntity'),
                  })
                : t('matchmaking.results.listDescription')
            }}
          </p>
        </div>
        <div class="view-toggle" :aria-label="t('matchmaking.results.viewAria')">
          <button
            class="view-toggle__button"
            :class="{ active: viewMode === 'grid' }"
            type="button"
            @click="setViewMode('grid')"
          >
            {{ t('matchmaking.results.viewGrid') }}
          </button>
          <button
            class="view-toggle__button"
            :class="{ active: viewMode === 'list' }"
            type="button"
            @click="setViewMode('list')"
          >
            {{ t('matchmaking.results.viewList') }}
          </button>
        </div>
      </div>

      <div v-if="!auth.isAuthenticated" class="empty-state">
        <p>{{ t('matchmaking.auth.resultsRequired') }}</p>
        <button class="btn btn--primary" type="button" @click="redirectToAuth">
          {{ t('auth.login') }}
        </button>
      </div>

      <template v-else>
        <p v-if="loading" class="muted-copy">{{ t('matchmaking.results.loading') }}</p>
        <p v-else-if="errorMsg" class="feedback feedback--error">{{ errorMsg }}</p>
        <p v-else-if="!sortedRooms.length" class="muted-copy">{{ t('matchmaking.results.empty') }}</p>

        <div v-else :class="viewMode === 'grid' ? 'room-grid' : 'room-list'">
          <article
            v-for="room in sortedRooms"
            :key="room.room_id"
            class="room-card"
            :class="{ 'room-card--list': viewMode === 'list' }"
          >
            <div class="room-head">
              <div>
                <p class="room-kicker">{{ t('matchmaking.results.roomLabel', { id: room.room_id, game_name: room.game_name }) }}</p>
                <h3>{{ t('matchmaking.results.roomTitle', { players: room.current_players, seats: room.capacity }) }}</h3>
              </div>
              <span class="score-pill">
                {{ t('matchmaking.results.score', { value: formatScore(room) }) }}
              </span>
            </div>

            <dl class="room-meta">
              <div class="room-meta__item">
                <dt>{{ t('matchmaking.results.meta.entry') }}</dt>
                <dd>{{ room.registration_price }}</dd>
              </div>
              <div class="room-meta__item">
                <dt>{{ t('matchmaking.results.meta.capacity') }}</dt>
                <dd>{{ room.capacity }}</dd>
              </div>
              <div class="room-meta__item">
                <dt>{{ t('matchmaking.results.meta.players') }}</dt>
                <dd>{{ room.current_players }}</dd>
              </div>
              <div class="room-meta__item">
                <dt>{{ t('matchmaking.results.meta.minimumUsers') }}</dt>
                <dd>{{ room.min_users }}</dd>
              </div>
              <div class="room-meta__item">
                <dt>{{ t('matchmaking.results.meta.boost') }}</dt>
                <dd>{{ formatBoost(room) }}</dd>
              </div>
            </dl>

            <div class="room-actions">
              <button class="btn btn--primary" type="button" @click="enterRoom(room)">
                {{ t('matchmaking.results.enterRoom') }}
              </button>
            </div>
          </article>
        </div>

        <div v-if="pagination && pagination.total > pagination.page_size" class="pager">
          <button class="btn" type="button" :disabled="pagination.page <= 1" @click="changePage(pagination.page - 1)">
            {{ t('common.prev') }}
          </button>
          <span class="muted-copy">
            {{ t('admin.pager.pageOf', { page: pagination.page, pages: totalPages }) }}
          </span>
          <button
            class="btn"
            type="button"
            :disabled="pagination.page >= totalPages"
            @click="changePage(pagination.page + 1)"
          >
            {{ t('common.next') }}
          </button>
        </div>
      </template>
    </section>
  </div>

  <FooterTab />
</template>

<style scoped>
.rooms-area {
  position: fixed;
  inset: var(--layout-inset, 92px 20px 20px 304px);
  display: grid;
  gap: 1rem;
  overflow: auto;
  align-content: start;
  transition: all var(--transition-slow) ease;
}

.rooms-area.collapsed {
  --layout-inset: 92px 20px 20px 120px;
}

.panel-card {
  padding: 1.35rem;
  border-radius: 1.6rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
  background:
    radial-gradient(circle at top left, color-mix(in oklab, #0ea5e9, white 88%), transparent 28%),
    linear-gradient(180deg, color-mix(in oklab, var(--color-surface), white 14%), var(--color-surface));
  box-shadow: var(--shadow-md);
}

.card-head,
.filters-form,
.room-grid,
.room-list,
.room-card,
.room-head,
.room-actions,
.pager {
  display: grid;
  gap: 0.85rem;
}

.card-head {
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: start;
}

.eyebrow,
.room-kicker {
  margin: 0;
  font-size: 0.72rem;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: #0369a1;
}

.eyebrow.accent {
  color: #b45309;
}

h1,
h2,
h3,
p {
  margin: 0;
}

.description,
.muted-copy,
dt {
  color: var(--color-muted);
}

.status-pill,
.score-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  padding: 0.55rem 0.85rem;
  background: color-mix(in oklab, var(--color-surface), white 10%);
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
  font-weight: 600;
}

.view-toggle {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  padding: 0.35rem;
  border-radius: 999px;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  background: color-mix(in oklab, var(--color-surface), white 10%);
}

.view-toggle__button {
  appearance: none;
  border: 0;
  background: transparent;
  color: var(--color-muted);
  border-radius: 999px;
  padding: 0.55rem 0.85rem;
  font-weight: 600;
  cursor: pointer;
}

.view-toggle__button.active {
  background: linear-gradient(135deg, #0f766e, #0284c7);
  color: #f0fdfa;
}

.filters-form {
  margin-top: 1rem;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

label {
  display: grid;
  gap: 0.35rem;
}

label span,
.field-error {
  font-size: 0.9rem;
}

label span {
  color: var(--color-muted);
}

input,
select {
  width: 100%;
  padding: 0.8rem 0.95rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
  border-radius: 0.9rem;
  background: color-mix(in oklab, var(--color-surface), white 14%);
  color: var(--color-text);
}

.field-error,
.feedback--error {
  color: var(--color-danger);
}

.form-actions {
  grid-column: 1 / -1;
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.room-grid {
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
}

.room-list {
  grid-template-columns: 1fr;
}

.room-card {
  padding: 1rem;
  border-radius: 1.2rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  background: color-mix(in oklab, var(--color-surface), white 10%);
}

.room-card--list {
  grid-template-columns: minmax(0, 1.25fr) minmax(0, 1.6fr) auto;
  align-items: center;
  gap: 0.75rem 1rem;
}

.room-head {
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: start;
}

.room-meta {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
}

.room-meta__item {
  display: grid;
  gap: 0.2rem;
}

.room-card--list .room-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem 1rem;
}

.room-card--list .room-meta__item {
  display: inline-flex;
  align-items: baseline;
  gap: 0.35rem;
  min-width: 0;
}

.room-card--list .room-meta dt,
.room-card--list .room-meta dd {
  white-space: nowrap;
}

dd {
  margin: 0;
  font-weight: 600;
}

.room-actions {
  justify-content: stretch;
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

.btn--primary {
  border-color: transparent;
  background: linear-gradient(135deg, #0f766e, #0284c7);
  color: #f0fdfa;
}

.empty-state {
  display: grid;
  gap: 0.75rem;
  justify-items: start;
}

.pager {
  margin-top: 1rem;
  grid-template-columns: auto auto auto;
  justify-content: space-between;
  align-items: center;
}

@media (max-width: 1100px) {
  .filters-form {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 960px) {
  .rooms-area,
  .rooms-area.collapsed {
    position: static;
    inset: auto;
    margin: calc(76px + 0.75rem) 1rem 5.75rem;
  }
}

@media (max-width: 720px) {
  .card-head,
  .room-head,
  .pager {
    grid-template-columns: 1fr;
  }

  .filters-form,
  .room-meta {
    grid-template-columns: 1fr;
  }

  .room-card--list {
    grid-template-columns: 1fr;
  }

  .form-actions,
  .pager {
    justify-content: stretch;
  }

  .form-actions .btn,
  .room-actions .btn,
  .pager .btn {
    width: 100%;
  }
}
</style>
