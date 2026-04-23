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
  queryToMatchmakingFilters,
  type MatchmakingFilterDraft,
} from '@/utils/matchmaking'

const auth = useAuthStore()
const session = useMatchSessionStore()
const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const { LeftTabHidden: leftHidden, layoutInset } = useLayoutInset()

const initialFilters = session.filters ?? queryToMatchmakingFilters(route.query)
const draft = reactive<MatchmakingFilterDraft>(createMatchmakingDraft(initialFilters))
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
  session.setFilters(null)
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

  session.setFilters({
    ...validationResult.value.filters,
    page: DEFAULT_MATCHMAKING_PAGE,
    page_size: DEFAULT_MATCHMAKING_PAGE_SIZE,
  })
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
    <!-- Minimalist hero section -->
    <div class="hero-minimal">
      <h1 class="hero-title">{{ t('matchmaking.home.title') }}</h1>
      <p class="hero-subtitle">{{ t('matchmaking.home.description') }}</p>
    </div>

    <!-- Two clear choices -->
    <div class="choices-container">
      <!-- Choice 1: Start Now -->
      <button class="choice-card choice-primary" @click="handleQuickMatch">
        <div class="choice-icon">⚡</div>
        <h2 class="choice-title">{{ t('matchmaking.quick.title') }}</h2>
        <p class="choice-description">{{ t('matchmaking.quick.description') }}</p>
        <div class="choice-action">
          <span>{{ t('matchmaking.quick.cta') }}</span>
        </div>
      </button>

      <!-- Choice 2: Search Rooms -->
      <div class="choice-card choice-secondary">
        <div class="choice-icon">🔍</div>
        <h2 class="choice-title">{{ t('matchmaking.filters.title') }}</h2>
        <p class="choice-description">{{ t('matchmaking.filters.description') }}</p>
        
        <!-- Minimal filters inline -->
        <div class="inline-filters">
          <select v-model="draft.boostMode" class="filter-select">
            <option value="any">{{ t('matchmaking.filters.boostAny') }}</option>
            <option value="true">{{ t('matchmaking.filters.boostOnly') }}</option>
            <option value="false">{{ t('matchmaking.filters.noBoost') }}</option>
          </select>
          
          <input
            v-model="draft.minRegistrationPrice"
            type="number"
            placeholder="Min entry"
            class="filter-input"
          />
          
          <input
            v-model="draft.maxRegistrationPrice"
            type="number"
            placeholder="Max entry"
            class="filter-input"
          />
        </div>
        
        <button class="choice-action-btn" @click="handleBrowseRooms">
          <span>{{ t('matchmaking.filters.findRooms') }}</span>
          <span class="arrow">→</span>
        </button>
      </div>
    </div>

    <!-- Status indicator (minimal) -->
    <div v-if="auth.isAuthenticated && activeFilterSummary.length" class="status-minimal">
      <span class="status-dot"></span>
      <span>{{ activeFilterSummary[0] }}</span>
    </div>
  </main>

  <FooterTab />
</template>

<style scoped>
.home-area {
  position: fixed;
  inset: var(--layout-inset, 92px 20px 20px 304px);
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  gap: 3rem;
  overflow: auto;
  transition: all var(--transition-slow) ease;
  padding: 2rem;
}

.home-area.collapsed {
  --layout-inset: 92px 20px 20px 120px;
}

/* Hero section - minimal */
.hero-minimal {
  text-align: center;
/*  max-width: 600px;*/
}

.hero-title {
  font-size: 3.5rem;
  font-weight: 700;
  margin: 0 0 1rem 0;
  background: color-mix(in oklab, rgba(245, 158, 11, 1), var(--color-surface) 5%);
  /*color: var(--color-text);*/
  text-shadow: 0 0.5px 1.5px rgba(0, 0, 0, 0.06);

  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  letter-spacing: -0.02em;
}

.hero-subtitle {
  font-size: 1.1rem;
  color: var(--color-muted);
  margin: 0;
  line-height: 1.5;
}

/* Two choice cards */
.choices-container {
  display: flex;
  gap: 2rem;
  max-width: 1000px;
  width: 100%;
  justify-content: center;
  flex-wrap: wrap;
}

.choice-card {
  flex: 1;
  min-width: 280px;
  max-width: 400px;
  padding: 2rem;
  border-radius: 1.5rem;
  background: rgba(255, 255, 255, 0.03);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  text-align: center;
}

.choice-card:hover {
  transform: translateY(-4px);
  border-color: rgba(255, 255, 255, 0.2);
  background: rgba(255, 255, 255, 0.05);
}

.choice-primary {
  background: linear-gradient(135deg, rgba(15, 118, 110, 0.15), rgba(2, 132, 199, 0.15));
  border-color: rgba(15, 118, 110, 1);

  cursor: pointer;
}

.choice-primary:hover {
  background: linear-gradient(135deg, rgba(15, 118, 110, 0.25), rgba(2, 132, 199, 0.25));
  border-color: rgba(15, 118, 110, 0.5);
}

.choice-secondary {
  background: rgba(255, 255, 255, 0.03);
}

.choice-icon {
  font-size: 2.5rem;
  margin-bottom: 1rem;
}

.choice-title {
  font-size: 1.5rem;
  font-weight: 600;
  margin: 0 0 0.5rem 0;
  color: var(--color-text);
}

.choice-description {
  font-size: 0.9rem;
  color: var(--color-muted);
  margin: 0 0 1.5rem 0;
  line-height: 1.4;
}

.choice-action {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 600;
  color: #0ea5e9;
  margin-top: 1rem;
}

.choice-action-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 600;
  color: #0ea5e9;
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  font-size: 1rem;
  margin-top: 1rem;
  transition: gap 0.2s ease;
}

.choice-action-btn:hover .arrow,
.choice-action:hover .arrow {
  transform: translateX(4px);
}

.arrow {
  transition: transform 0.2s ease;
}

/* Inline filters for secondary card */
.inline-filters {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin: 1rem 0;
}

.filter-select,
.filter-input {
  padding: 0.6rem 0.8rem;
  border-radius: 0.5rem;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(0, 0, 0, 0.1);
  color: var(--color-text);
  font-size: 0.85rem;
  transition: all 0.2s ease;
}

.filter-select:focus,
.filter-input:focus {
  outline: none;
  border-color: #0ea5e9;
  background: rgba(0, 0, 0, 0.5);
}

/* Minimal status */
.status-minimal {
  position: fixed;
  bottom: 2rem;
  right: 2rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  border-radius: 999px;
  background: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(10px);
  font-size: 0.8rem;
  color: var(--color-muted);
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #10b981;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

/* Responsive */
@media (max-width: 960px) {
  .home-area,
  .home-area.collapsed {
    position: static;
    inset: auto;
    margin: calc(76px + 0.75rem) 1rem 5.75rem;
    justify-content: flex-start;
    padding: 1rem;
  }

  .hero-title {
    font-size: 2.5rem;
  }

  .choices-container {
    flex-direction: column;
    align-items: center;
  }

  .choice-card {
    width: 100%;
  }

  .status-minimal {
    position: static;
    margin-top: 1rem;
    justify-content: center;
  }
}
</style>