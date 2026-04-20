<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import LeftTab from '@/components/LeftTab.vue'
import UpTab from '@/components/UpTab.vue'
import FooterTab from '@/components/FooterTab.vue'
import { useLayoutInset } from '@/composables/useLayoutInset'
import { useI18n } from '@/i18n'
import { useAuthStore } from '@/stores/authStore'
import { useMatchSessionStore } from '@/stores/matchSessionStore'
import { MatchmakingApi } from '@/api/useMatchmakingApi'
import {
  DEFAULT_MATCHMAKING_PAGE,
  DEFAULT_MATCHMAKING_PAGE_SIZE,
  createMatchmakingDraft,
  filtersToQuery,
  normalizeMatchmakingDraft,
  type MatchmakingFilterDraft,
} from '@/utils/matchmaking'

const auth = useAuthStore()
const session = useMatchSessionStore()
const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const { LeftTabHidden: leftHidden, layoutInset } = useLayoutInset()

const draft = reactive<MatchmakingFilterDraft>(createMatchmakingDraft())
const submittingQuickMatch = ref(false)
const formError = ref('')

const validationResult = computed(() => normalizeMatchmakingDraft(draft))

const activeFilterSummary = computed(() => {
  const filters = validationResult.value.filters
  const items: string[] = []

  if (filters.min_registration_price !== undefined || filters.max_registration_price !== undefined) {
    items.push(
      t('matchmaking.summary.entryRange', {
        min: filters.min_registration_price ?? 0,
        max: filters.max_registration_price ?? t('matchmaking.summary.unlimited'),
      }),
    )
  }

  if (filters.min_capacity !== undefined || filters.max_capacity !== undefined) {
    items.push(
      t('matchmaking.summary.capacityRange', {
        min: filters.min_capacity ?? 1,
        max: filters.max_capacity ?? t('matchmaking.summary.unlimited'),
      }),
    )
  }

  if (filters.is_boost !== undefined) {
    items.push(
      filters.is_boost ? t('matchmaking.filters.boostOnly') : t('matchmaking.filters.noBoost'),
    )
  }

  if (filters.min_boost_power !== undefined) {
    items.push(
      t('matchmaking.summary.boostPowerMin', {
        value: filters.min_boost_power,
      }),
    )
  }

  return items
})

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

function resetFilters() {
  formError.value = ''
  Object.assign(draft, createMatchmakingDraft())
}

function redirectToAuth(fullPath: string) {
  router.push({
    path: '/auth',
    query: {
      redirect: fullPath,
    },
  })
}

async function handleQuickMatch() {
  formError.value = ''

  if (Object.keys(validationResult.value.fieldErrors).length) {
    formError.value = t('matchmaking.messages.fixFilters')
    return
  }

  if (!auth.isAuthenticated) {
    redirectToAuth(route.fullPath)
    return
  }

  try {
    submittingQuickMatch.value = true
    session.setLoading(true)

    const response = await MatchmakingApi.quickMatch(validationResult.value.filters)
    session.setQuickMatchSession(response, validationResult.value.filters)

    await router.push(`/play/${response.room.room_id}`)
  } catch (error: any) {
    const message = error?.message || t('matchmaking.errors.quickMatch')
    formError.value = message
    session.setError(message)
  } finally {
    submittingQuickMatch.value = false
    if (session.loading) {
      session.setLoading(false)
    }
  }
}

function handleBrowseRooms() {
  formError.value = ''

  if (Object.keys(validationResult.value.fieldErrors).length) {
    formError.value = t('matchmaking.messages.fixFilters')
    return
  }

  const target = {
    path: '/rooms',
    query: filtersToQuery({
      ...validationResult.value.filters,
      page: DEFAULT_MATCHMAKING_PAGE,
      page_size: DEFAULT_MATCHMAKING_PAGE_SIZE,
    }),
  }

  if (!auth.isAuthenticated) {
    redirectToAuth(router.resolve(target).fullPath)
    return
  }

  router.push(target)
}
</script>
<template>
  <UpTab :show-menu="true" :show-upload="true" />
  <LeftTab />

  <main
    class="home-area"
    :class="{ collapsed: leftHidden }"
    :style="{ '--layout-inset': layoutInset }"
  >
    <section class="hero-card">
      <div class="hero-copy">
        <p class="eyebrow">{{ t('matchmaking.home.eyebrow') }}</p>
        <h1>{{ t('matchmaking.home.title') }}</h1>
        <p class="description">{{ t('matchmaking.home.description') }}</p>
      </div>

      <div class="hero-status">
        <span class="status-pill status-pill--accent">
          {{
            auth.isAuthenticated
              ? t('matchmaking.home.status.ready')
              : t('matchmaking.home.status.authRequired')
          }}
        </span>
        <span v-if="activeFilterSummary.length" class="status-pill">
          {{ t('matchmaking.home.status.filtersReady') }}
        </span>
      </div>
    </section>

    <section class="home-grid">
      <article class="panel-card quick-card">
        <div class="card-head">
          <div>
            <p class="eyebrow accent">{{ t('matchmaking.quick.eyebrow') }}</p>
            <h2>{{ t('matchmaking.quick.title') }}</h2>
            <p class="description">{{ t('matchmaking.quick.description') }}</p>
          </div>
        </div>

        <div class="summary-list">
          <template v-if="activeFilterSummary.length">
            <span v-for="item in activeFilterSummary" :key="item" class="summary-chip">
              {{ item }}
            </span>
          </template>
          <p v-else class="muted-copy">{{ t('matchmaking.home.noFilters') }}</p>
        </div>

        <p v-if="formError" class="feedback feedback--error">{{ formError }}</p>

        <button class="btn btn--primary btn--block" :disabled="submittingQuickMatch" @click="handleQuickMatch">
          {{
            submittingQuickMatch
              ? t('matchmaking.quick.loading')
              : t('matchmaking.quick.cta')
          }}
        </button>
      </article>

      <article class="panel-card filters-card">
        <div class="card-head">
          <div>
            <p class="eyebrow">{{ t('matchmaking.filters.eyebrow') }}</p>
            <h2>{{ t('matchmaking.filters.title') }}</h2>
            <p class="description">{{ t('matchmaking.filters.description') }}</p>
          </div>
        </div>

        <form class="filters-form" @submit.prevent="handleBrowseRooms">
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
              {{ t('matchmaking.filters.findRooms') }}
            </button>
          </div>
        </form>
      </article>
    </section>
  </main>

  <FooterTab />
</template>

<style scoped>
.home-area {
  position: fixed;
  inset: var(--layout-inset, 92px 20px 20px 304px);
  display: grid;
  gap: 1rem;
  overflow: auto;
  align-content: start;
  transition: all var(--transition-slow) ease;
}

.home-area.collapsed {
  --layout-inset: 92px 20px 20px 120px;
}

.hero-card,
.panel-card {
  border-radius: 1.6rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
  box-shadow: var(--shadow-md);
}

.hero-card {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 1rem;
  padding: 1.5rem;
  background:
    radial-gradient(circle at top right, rgba(245, 158, 11, 0.18), transparent 24%),
    linear-gradient(
      135deg,
      color-mix(in oklab, var(--color-bg-secondary), white 18%),
      color-mix(in oklab, var(--color-surface), transparent 6%)
    );
}

.hero-copy,
.card-head,
.filters-form,
.summary-list,
.hero-status {
  display: grid;
  gap: 0.85rem;
}

.home-grid {
  display: grid;
  grid-template-columns: minmax(320px, 0.9fr) minmax(0, 1.1fr);
  gap: 1rem;
}

.panel-card {
  padding: 1.35rem;
  background:
    radial-gradient(circle at top left, color-mix(in oklab, #0ea5e9, white 88%), transparent 28%),
    linear-gradient(180deg, color-mix(in oklab, var(--color-surface), white 14%), var(--color-surface));
}

.card-head {
  align-items: start;
}

.eyebrow {
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
p {
  margin: 0;
}

.description,
.muted-copy {
  color: var(--color-muted);
}

.hero-status {
  align-content: start;
  justify-items: end;
}

.status-pill,
.summary-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  padding: 0.55rem 0.85rem;
  background: color-mix(in oklab, var(--color-surface), white 10%);
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
  font-weight: 600;
}

.status-pill--accent {
  background: color-mix(in oklab, var(--color-primary-secondary), transparent 84%);
}

.summary-list {
  align-content: start;
}

.filters-form {
  grid-template-columns: repeat(2, minmax(0, 1fr));
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

.btn--block {
  width: 100%;
}

@media (max-width: 1180px) {
  .home-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 960px) {
  .home-area,
  .home-area.collapsed {
    position: static;
    inset: auto;
    margin: calc(76px + 0.75rem) 1rem 5.75rem;
  }
}

@media (max-width: 760px) {
  .hero-card {
    grid-template-columns: 1fr;
  }

  .hero-status {
    justify-items: start;
  }

  .filters-form {
    grid-template-columns: 1fr;
  }

  .form-actions {
    justify-content: stretch;
  }

  .form-actions .btn {
    width: 100%;
  }
}
</style>
