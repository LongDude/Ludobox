<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import LeftTab from '@/components/LeftTab.vue'
import UpTab from '@/components/UpTab.vue'
import { UserApi } from '@/api/useUserApi'
import type { UserRatingHistoryRequest, UserRatingHistoryResponse, UserRank, UserUpdateRequest } from '@/api/types'
import { useLayoutInset } from '@/composables/useLayoutInset'
import { useI18n } from '@/i18n'
import { useAuthStore } from '@/stores/authStore'
import { useUserCabinetStore } from '@/stores/userCabinetStore'
import { normalizeRank, rankFrameClass, rankTextClass } from '@/utils/rankFrame'

type RatingPeriod = '7d' | '30d' | '90d' | 'all'
type ProfileTab = 'identity' | 'game' | 'rating'

type ChartPoint = {
  item: UserRatingHistoryResponse['items'][number]
  x: number
  y: number
}

const auth = useAuthStore()
const cabinet = useUserCabinetStore()
const router = useRouter()
const { locale, t } = useI18n()
const { LeftTabHidden: leftHidden, layoutInset } = useLayoutInset()

const editing = ref(false)
const saving = ref(false)
const identitySuccessMsg = ref('')
const identityErrorMsg = ref('')
const nicknameSaving = ref(false)
const balanceSaving = ref(false)
const gameSuccessMsg = ref('')
const gameErrorMsg = ref('')
const ratingHistory = ref<UserRatingHistoryResponse | null>(null)
const ratingHistoryLoading = ref(false)
const ratingHistoryErrorMsg = ref('')
const selectedRatingPeriod = ref<RatingPeriod>('30d')
const activeProfileTab = ref<ProfileTab>('identity')

const identityForm = reactive({
  email: '',
  first_name: '',
  last_name: '',
  locale_type: '',
  password: '',
})

const gameForm = reactive({
  nickname: '',
  delta: '' as string | number,
})

const periodOptions: Array<{ value: RatingPeriod; labelKey: string }> = [
  { value: '7d', labelKey: 'profile.game.period.7d' },
  { value: '30d', labelKey: 'profile.game.period.30d' },
  { value: '90d', labelKey: 'profile.game.period.90d' },
  { value: 'all', labelKey: 'profile.game.period.all' },
]

const profileTabs = computed<Array<{ key: ProfileTab; label: string; description: string }>>(() => [
  {
    key: 'identity',
    label: t('profile.tabs.identity'),
    description: t('profile.tabs.identityDescription'),
  },
  {
    key: 'game',
    label: t('profile.tabs.game'),
    description: t('profile.tabs.gameDescription'),
  },
  {
    key: 'rating',
    label: t('profile.tabs.rating'),
    description: t('profile.tabs.ratingDescription'),
  },
])

onMounted(async () => {
  if (auth.isAuthenticated && !auth.User) {
    try {
      await auth.authenticate()
    } catch {}
  }

  if (auth.isAuthenticated) {
    void cabinet.ensureLoaded().catch(() => {})
    void loadRatingHistory(false)
  }
})

const fullName = computed(() => {
  const user = auth.User
  if (!user) return ''
  return [user.first_name, user.last_name].filter(Boolean).join(' ')
})

const avatarUrl = computed(() => auth.User?.photo || '')
const avatarRankClass = computed(() => rankFrameClass(cabinet.profile?.rank))

const avatarLetter = computed(() => {
  const user = auth.User
  const name = [user?.first_name, user?.last_name].filter(Boolean).join(' ') || user?.email || ''
  const trimmed = name.trim()
  return trimmed ? trimmed[0].toUpperCase() : 'U'
})

const formattedBalance = computed(() => {
  const profile = cabinet.profile
  if (!profile) return '-'

  return new Intl.NumberFormat(locale.value === 'ru' ? 'ru-RU' : 'en-US').format(profile.balance)
})

const formattedRating = computed(() => {
  const profile = cabinet.profile
  if (!profile) return '-'

  return new Intl.NumberFormat(locale.value === 'ru' ? 'ru-RU' : 'en-US').format(profile.rating)
})

const ratingRankLabel = computed(() => translateRank(cabinet.profile?.rank))
const ratingRankTextClass = computed(() => rankTextClass(cabinet.profile?.rank))
const ratingRankTooltip = computed(() => translateRankTooltip(cabinet.profile?.rank))

const periodGainLabel = computed(() => formatSignedNumber(ratingHistory.value?.period_change ?? 0))

const latestRatingUpdate = computed(() => {
  const items = ratingHistory.value?.items ?? []
  return items.length > 0 ? items[items.length - 1] : null
})

const chartPoints = computed<ChartPoint[]>(() => {
  const items = ratingHistory.value?.items ?? []
  if (items.length === 0) return []

  const width = 100
  const height = 100
  const minRating = Math.min(...items.map((item) => item.rating_after))
  const maxRating = Math.max(...items.map((item) => item.rating_after))
  const range = Math.max(maxRating - minRating, 1)

  return items.map((item, index) => {
    const x = items.length === 1 ? width / 2 : (index / (items.length - 1)) * width
    const y = height - ((item.rating_after - minRating) / range) * height
    return { item, x, y }
  })
})

const chartPolyline = computed(() => chartPoints.value.map((point) => `${point.x},${point.y}`).join(' '))

const chartArea = computed(() => {
  const points = chartPoints.value
  if (points.length === 0) return ''

  const first = points[0]
  const last = points[points.length - 1]
  return `0,100 ${first.x},${first.y} ${points.map((point) => `${point.x},${point.y}`).join(' ')} ${last.x},100`
})

const chartStartLabel = computed(() => {
  const first = ratingHistory.value?.items?.[0]
  return first ? formatRatingDate(first.created_at) : ''
})

const chartEndLabel = computed(() => {
  const last = latestRatingUpdate.value
  return last ? formatRatingDate(last.created_at) : ''
})

const activeGameError = computed(() => gameErrorMsg.value || cabinet.error || '')

watch(
  () => auth.User,
  (user) => {
    if (!user) return
    identityForm.email = user.email || ''
    identityForm.first_name = user.first_name || ''
    identityForm.last_name = user.last_name || ''
    identityForm.locale_type = user.locale_type || ''
    identityForm.password = ''
  },
  { immediate: true },
)

watch(
  () => cabinet.profile,
  (profile) => {
    if (!profile) return
    gameForm.nickname = profile.nickname || ''
  },
  { immediate: true },
)

watch(selectedRatingPeriod, () => {
  if (!auth.isAuthenticated) return
  void loadRatingHistory(false)
})

watch(
  () => cabinet.profile?.rating,
  (next, previous) => {
    if (!auth.isAuthenticated || next === undefined || next === previous) return
    void loadRatingHistory(false)
  },
)

function displayRole(role: string) {
  const normalized = role.toLowerCase()
  if (normalized === 'admin') return t('roles.admin')
  if (normalized === 'user') return t('roles.user')
  return role
}

function translateRank(rank?: UserRank) {
  if (!rank) return '-'

  const normalized = rank.toLowerCase()
  if (normalized === 'bronze') return t('profile.game.rank.bronze')
  if (normalized === 'silver') return t('profile.game.rank.silver')
  if (normalized === 'gold') return t('profile.game.rank.gold')
  if (normalized === 'platinum') return t('profile.game.rank.platinum')
  if (normalized === 'diamond') return t('profile.game.rank.diamond')
  return rank
}

function translateRankTooltip(rank?: UserRank) {
  return t(`profile.game.rankTooltip.${normalizeRank(rank)}`)
}

function formatSignedNumber(value: number) {
  const formatter = new Intl.NumberFormat(locale.value === 'ru' ? 'ru-RU' : 'en-US')
  if (value > 0) return `+${formatter.format(value)}`
  return formatter.format(value)
}

function formatRatingDate(value: string) {
  const date = new Date(value)
  return new Intl.DateTimeFormat(locale.value === 'ru' ? 'ru-RU' : 'en-US', {
    day: '2-digit',
    month: 'short',
  }).format(date)
}

function formatRatingDateTime(value: string) {
  const date = new Date(value)
  return new Intl.DateTimeFormat(locale.value === 'ru' ? 'ru-RU' : 'en-US', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)
}

function buildRatingHistoryRequest(period: RatingPeriod): UserRatingHistoryRequest {
  const now = new Date()
  const end = now.toISOString()

  if (period === 'all') {
    return { date_to: end }
  }

  const start = new Date(now)
  const days = period === '7d' ? 7 : period === '30d' ? 30 : 90
  start.setDate(start.getDate() - days)

  return {
    date_from: start.toISOString(),
    date_to: end,
  }
}

function resetIdentityForm() {
  const user = auth.User
  if (!user) return
  identityForm.email = user.email || ''
  identityForm.first_name = user.first_name || ''
  identityForm.last_name = user.last_name || ''
  identityForm.locale_type = user.locale_type || ''
  identityForm.password = ''
}

function startEditing() {
  identitySuccessMsg.value = ''
  identityErrorMsg.value = ''
  resetIdentityForm()
  editing.value = true
}

function cancelEditing() {
  identitySuccessMsg.value = ''
  identityErrorMsg.value = ''
  resetIdentityForm()
  editing.value = false
}

function buildIdentityPayload(): UserUpdateRequest {
  const user = auth.User
  const payload: UserUpdateRequest = {}

  if (user) {
    if (identityForm.email && identityForm.email !== user.email) payload.email = identityForm.email
    if (identityForm.first_name !== user.first_name) payload.first_name = identityForm.first_name
    if (identityForm.last_name !== user.last_name) payload.last_name = identityForm.last_name
    if ((identityForm.locale_type || '') !== (user.locale_type || '')) {
      payload.locale_type = identityForm.locale_type || undefined
    }
  } else {
    if (identityForm.email) payload.email = identityForm.email
    if (identityForm.first_name) payload.first_name = identityForm.first_name
    if (identityForm.last_name) payload.last_name = identityForm.last_name
    if (identityForm.locale_type) payload.locale_type = identityForm.locale_type
  }

  if (identityForm.password) payload.password = identityForm.password

  return payload
}

async function saveIdentityProfile() {
  identitySuccessMsg.value = ''
  identityErrorMsg.value = ''

  const payload = buildIdentityPayload()
  if (!Object.keys(payload).length) {
    identitySuccessMsg.value = t('profile.msg.nothing')
    return
  }

  try {
    saving.value = true
    await auth.updateUser(payload)
    identitySuccessMsg.value = t('profile.msg.updated')
  } catch (error: any) {
    identityErrorMsg.value = error?.message || t('profile.msg.failed')
  } finally {
    saving.value = false
  }
}

async function refreshGameProfile() {
  gameSuccessMsg.value = ''
  gameErrorMsg.value = ''

  try {
    await Promise.all([cabinet.refresh(), loadRatingHistory(false)])
  } catch (error: any) {
    gameErrorMsg.value = error?.message || t('profile.game.msg.failedLoad')
  }
}

async function loadRatingHistory(showError = true) {
  if (!auth.isAuthenticated) {
    ratingHistory.value = null
    ratingHistoryErrorMsg.value = ''
    return
  }

  ratingHistoryLoading.value = true
  if (showError) {
    ratingHistoryErrorMsg.value = ''
  }

  try {
    ratingHistory.value = await UserApi.getCurrentUserRatingHistory(
      buildRatingHistoryRequest(selectedRatingPeriod.value),
    )
    ratingHistoryErrorMsg.value = ''
  } catch (error: any) {
    if (showError) {
      ratingHistoryErrorMsg.value = error?.message || t('profile.game.ratingHistory.error')
    }
  } finally {
    ratingHistoryLoading.value = false
  }
}

async function saveNickname() {
  gameSuccessMsg.value = ''
  gameErrorMsg.value = ''

  const nickname = gameForm.nickname.trim()
  if (!nickname) {
    gameErrorMsg.value = t('profile.game.msg.nicknameRequired')
    return
  }

  try {
    nicknameSaving.value = true
    await cabinet.updateNickname(nickname)
    gameSuccessMsg.value = t('profile.game.msg.nicknameUpdated')
  } catch (error: any) {
    gameErrorMsg.value = error?.message || t('profile.game.msg.failedSaveNickname')
  } finally {
    nicknameSaving.value = false
  }
}

async function applyBalanceDelta() {
  gameSuccessMsg.value = ''
  gameErrorMsg.value = ''

  const rawDelta = String(gameForm.delta ?? '').trim()
  const delta = Number(rawDelta)
  if (!rawDelta || !Number.isInteger(delta) || delta === 0) {
    gameErrorMsg.value = t('profile.game.msg.deltaRequired')
    return
  }

  try {
    balanceSaving.value = true
    await cabinet.applyBalanceDelta(delta)
    gameForm.delta = ''
    gameSuccessMsg.value = t('profile.game.msg.balanceUpdated')
    void loadRatingHistory(false)
  } catch (error: any) {
    gameErrorMsg.value = error?.message || t('profile.game.msg.failedBalance')
  } finally {
    balanceSaving.value = false
  }
}

async function logout() {
  await auth.logout()
  cabinet.reset()
  ratingHistory.value = null
  router.replace('/auth')
}
</script>

<template>
  <UpTab :show-menu="false" :show-upload="false" />
  <LeftTab :hidden="true" />

  <div
    class="profile-area"
    :class="{ collapsed: leftHidden }"
    :style="{ '--layout-inset': layoutInset }"
  >
    <div class="container">
      <div class="profile-shell">
        <nav class="profile-tabs" :aria-label="t('profile.tabs.label')">
          <button
            v-for="tab in profileTabs"
            :key="tab.key"
            class="profile-tab"
            :class="{ active: activeProfileTab === tab.key }"
            type="button"
            :aria-selected="activeProfileTab === tab.key"
            @click="activeProfileTab = tab.key"
          >
            <strong>{{ tab.label }}</strong>
            <span>{{ tab.description }}</span>
          </button>
        </nav>

        <section v-if="activeProfileTab === 'identity'" class="panel-card identity-card">
          <div class="card-head">
            <div>
              <p class="eyebrow">{{ t('profile.identityEyebrow') }}</p>
              <h2>{{ t('profile.identityTitle') }}</h2>
              <p class="section-copy">{{ t('profile.identityDescription') }}</p>
            </div>
          </div>

          <div class="profile-header">
            <div class="avatar rank-frame" :class="avatarRankClass">
              <img
                v-if="avatarUrl"
                :src="avatarUrl"
                :alt="fullName || auth.User?.email || t('profile.avatarAlt')"
              />
              <span v-else>{{ avatarLetter }}</span>
            </div>
            <div class="identity">
              <h3 class="name">{{ fullName || t('profile.title') }}</h3>
              <p class="muted">{{ auth.User?.email || '-' }}</p>
            </div>
          </div>

          <div v-if="!editing" class="meta">
            <div class="row">
              <span class="label">{{ t('auth.email') }}</span>
              <span class="value">{{ auth.User?.email || '-' }}</span>
            </div>
            <div class="row">
              <span class="label">{{ t('profile.emailConfirmed') }}</span>
              <span
                class="value"
                :class="{ ok: auth.User?.email_confirmed, warn: !auth.User?.email_confirmed }"
              >
                {{ auth.User?.email_confirmed ? t('common.yes') : t('common.no') }}
              </span>
            </div>
            <div class="row" v-if="auth.User?.locale_type">
              <span class="label">{{ t('profile.locale') }}</span>
              <span class="value">{{ auth.User?.locale_type }}</span>
            </div>
            <div class="row" v-if="auth.User?.roles?.length">
              <span class="label">{{ t('profile.roles') }}</span>
              <span class="value roles">
                <span v-for="role in auth.User?.roles" :key="role" class="chip">
                  {{ displayRole(role) }}
                </span>
              </span>
            </div>
          </div>

          <div v-else class="editor">
            <h3>{{ t('profile.editTitle') }}</h3>
            <div class="grid">
              <label>
                <span>{{ t('auth.email') }}</span>
                <input v-model="identityForm.email" type="email" :placeholder="t('auth.email')" />
              </label>
              <label>
                <span>{{ t('profile.form.firstName') }}</span>
                <input
                  v-model="identityForm.first_name"
                  type="text"
                  :placeholder="t('profile.form.firstName')"
                />
              </label>
              <label>
                <span>{{ t('profile.form.lastName') }}</span>
                <input
                  v-model="identityForm.last_name"
                  type="text"
                  :placeholder="t('profile.form.lastName')"
                />
              </label>
              <label>
                <span>{{ t('profile.form.locale') }}</span>
                <input
                  v-model="identityForm.locale_type"
                  type="text"
                  :placeholder="t('profile.form.locale')"
                />
              </label>
              <label class="wide">
                <span>{{ t('profile.form.newPassword') }}</span>
                <input
                  v-model="identityForm.password"
                  type="password"
                  :placeholder="t('profile.form.keepBlank')"
                />
              </label>
            </div>
          </div>

          <div class="feedback">
            <span v-if="identitySuccessMsg" class="ok">{{ identitySuccessMsg }}</span>
            <span v-if="identityErrorMsg" class="err">{{ identityErrorMsg }}</span>
          </div>

          <div class="actions">
            <template v-if="editing">
              <button class="btn" :disabled="saving" @click="cancelEditing">
                {{ t('profile.btn.cancel') }}
              </button>
              <button class="btn btn--primary" :disabled="saving" @click="saveIdentityProfile">
                {{ saving ? t('profile.saving') : t('profile.btn.save') }}
              </button>
            </template>
            <template v-else>
              <button class="btn" @click="startEditing">{{ t('profile.btn.edit') }}</button>
              <button class="btn btn--primary" @click="logout">{{ t('profile.btn.logout') }}</button>
            </template>
          </div>
        </section>

        <section v-else-if="activeProfileTab === 'game'" class="panel-card game-card">
          <div class="card-head">
            <div>
              <p class="eyebrow accent">{{ t('profile.game.eyebrow') }}</p>
              <h2>{{ t('profile.game.title') }}</h2>
              <p class="section-copy">{{ t('profile.game.description') }}</p>
            </div>
            <button class="btn" :disabled="cabinet.loading" @click="refreshGameProfile">
              {{ t('common.refresh') }}
            </button>
          </div>

          <p v-if="cabinet.loading && !cabinet.profile" class="state-copy">
            {{ t('profile.game.loading') }}
          </p>

          <template v-else-if="cabinet.profile">
            <div class="meta">
              <div class="row">
                <span class="label">{{ t('profile.game.nickname') }}</span>
                <span class="value">{{ cabinet.profile.nickname }}</span>
              </div>
              <div class="row">
                <span class="label">{{ t('profile.game.balance') }}</span>
                <span class="value balance">{{ formattedBalance }}</span>
              </div>
            </div>

            <div class="rating-summary">
              <article class="summary-card">
                <span class="summary-label">{{ t('profile.game.rating') }}</span>
                <strong>{{ formattedRating }}</strong>
              </article>
              <article class="summary-card">
                <span class="summary-label">{{ t('profile.game.rank') }}</span>
                <span
                  class="rank-with-tooltip"
                  :class="ratingRankTextClass"
                  tabindex="0"
                  :aria-label="ratingRankTooltip"
                >
                  <strong>{{ ratingRankLabel }}</strong>
                  <span class="rank-tooltip" role="tooltip">{{ ratingRankTooltip }}</span>
                </span>
              </article>
            </div>

            <div class="feedback">
              <span v-if="gameSuccessMsg" class="ok">{{ gameSuccessMsg }}</span>
              <span v-if="activeGameError" class="err">{{ activeGameError }}</span>
            </div>

            <form class="game-editor" @submit.prevent="saveNickname">
              <label>
                <span>{{ t('profile.game.nickname') }}</span>
                <input
                  v-model="gameForm.nickname"
                  type="text"
                  :placeholder="t('profile.game.nicknamePlaceholder')"
                />
              </label>
              <div class="inline-actions">
                <button class="btn btn--primary" type="submit" :disabled="nicknameSaving">
                  {{ nicknameSaving ? t('common.saving') : t('profile.game.saveNickname') }}
                </button>
              </div>
            </form>

            <form class="game-editor" @submit.prevent="applyBalanceDelta">
              <label>
                <span>{{ t('profile.game.delta') }}</span>
                <input
                  v-model="gameForm.delta"
                  type="number"
                  step="1"
                  :placeholder="t('profile.game.deltaPlaceholder')"
                />
              </label>
              <p class="helper">{{ t('profile.game.deltaHelp') }}</p>
              <div class="inline-actions">
                <button class="btn btn--primary" type="submit" :disabled="balanceSaving">
                  {{ balanceSaving ? t('common.saving') : t('profile.game.applyDelta') }}
                </button>
              </div>
            </form>
          </template>

          <div v-else class="state-block">
            <p class="state-copy error">
              {{ activeGameError || t('profile.game.msg.failedLoad') }}
            </p>
          </div>
        </section>

        <section v-else class="panel-card rating-card">
          <div class="card-head">
            <div>
              <p class="eyebrow accent">{{ t('profile.rating.eyebrow') }}</p>
              <h2>{{ t('profile.game.ratingHistoryTitle') }}</h2>
              <p class="section-copy">{{ t('profile.game.ratingHistoryDescription') }}</p>
            </div>
            <button class="btn" :disabled="ratingHistoryLoading" @click="loadRatingHistory()">
              {{ t('common.refresh') }}
            </button>
          </div>

          <p v-if="cabinet.loading && !cabinet.profile" class="state-copy">
            {{ t('profile.game.loading') }}
          </p>

          <template v-else-if="cabinet.profile">
            <div class="rating-summary">
              <article class="summary-card">
                <span class="summary-label">{{ t('profile.game.rating') }}</span>
                <strong>{{ formattedRating }}</strong>
              </article>
              <article class="summary-card">
                <span class="summary-label">{{ t('profile.game.rank') }}</span>
                <span
                  class="rank-with-tooltip"
                  :class="ratingRankTextClass"
                  tabindex="0"
                  :aria-label="ratingRankTooltip"
                >
                  <strong>{{ ratingRankLabel }}</strong>
                  <span class="rank-tooltip" role="tooltip">{{ ratingRankTooltip }}</span>
                </span>
              </article>
              <article class="summary-card">
                <span class="summary-label">{{ t('profile.game.periodGain') }}</span>
                <strong :class="{ positive: (ratingHistory?.period_change ?? 0) > 0 }">
                  {{ periodGainLabel }}
                </strong>
              </article>
            </div>

            <section class="rating-history">
              <div class="rating-head">
                <div class="period-switch">
                  <button
                    v-for="option in periodOptions"
                    :key="option.value"
                    class="period-chip"
                    :class="{ active: selectedRatingPeriod === option.value }"
                    :disabled="ratingHistoryLoading"
                    @click="selectedRatingPeriod = option.value"
                  >
                    {{ t(option.labelKey) }}
                  </button>
                </div>
              </div>

              <div v-if="ratingHistoryLoading" class="chart-empty">
                {{ t('profile.game.ratingHistory.loading') }}
              </div>
              <div v-else-if="ratingHistoryErrorMsg" class="chart-empty error">
                {{ ratingHistoryErrorMsg }}
              </div>
              <div v-else-if="chartPoints.length === 0" class="chart-empty">
                {{ t('profile.game.ratingHistory.empty') }}
              </div>
              <div v-else class="chart-shell">
                <svg
                  class="rating-chart"
                  viewBox="0 0 100 100"
                  preserveAspectRatio="none"
                  role="img"
                  :aria-label="t('profile.game.ratingHistoryTitle')"
                >
                  <defs>
                    <linearGradient id="ratingArea" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stop-color="rgba(8, 145, 178, 0.38)" />
                      <stop offset="100%" stop-color="rgba(8, 145, 178, 0.02)" />
                    </linearGradient>
                  </defs>
                  <polygon :points="chartArea" fill="url(#ratingArea)" />
                  <polyline class="chart-line" :points="chartPolyline" />
                  <circle
                    v-for="point in chartPoints"
                    :key="point.item.history_id"
                    class="chart-dot"
                    :cx="point.x"
                    :cy="point.y"
                    r="1.8"
                  />
                </svg>

                <div class="chart-axis">
                  <span>{{ chartStartLabel }}</span>
                  <span>{{ chartEndLabel }}</span>
                </div>

                <div v-if="latestRatingUpdate" class="chart-footnote">
                  <span>
                    {{ t('profile.game.ratingHistory.lastChange') }}
                    <strong>{{ formatSignedNumber(latestRatingUpdate.delta) }}</strong>
                  </span>
                  <span>{{ formatRatingDateTime(latestRatingUpdate.created_at) }}</span>
                </div>
              </div>
            </section>
          </template>

          <div v-else class="state-block">
            <p class="state-copy error">
              {{ activeGameError || t('profile.game.msg.failedLoad') }}
            </p>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>

<style scoped>
.profile-area {
  position: fixed;
  inset: var(--layout-inset, 92px 20px 20px 304px);
  display: grid;
  align-items: start;
  overflow: auto;
  transition: all var(--transition-slow) ease;
}

.profile-area.collapsed {
  --layout-inset: 92px 20px 20px 120px;
}

.container {
  max-width: 1100px;
  margin: 25px auto;
  width: 100%;
}

.profile-shell {
  display: grid;
  gap: 1rem;
  justify-items: center;
}

.profile-tabs {
  width: 100%;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
}

.profile-tab {
  appearance: none;
  min-width: 0;
  display: grid;
  gap: 0.25rem;
  padding: 0.95rem 1rem;
  border-radius: 1.15rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  background:
    linear-gradient(180deg, color-mix(in oklab, var(--color-surface), white 14%), var(--color-surface));
  color: var(--color-text);
  text-align: left;
  cursor: pointer;
  box-shadow: var(--shadow-sm);
  transition:
    transform var(--transition-fast) ease,
    border-color var(--transition-fast) ease,
    background var(--transition-fast) ease;
}

.profile-tab:hover {
  transform: translateY(-1px);
  border-color: color-mix(in oklab, var(--color-primary-secondary), transparent 18%);
}

.profile-tab.active {
  border-color: color-mix(in oklab, var(--color-primary-secondary), transparent 8%);
  background:
    radial-gradient(circle at top right, rgba(14, 165, 233, 0.2), transparent 46%),
    linear-gradient(135deg, color-mix(in oklab, var(--color-surface), white 20%), var(--color-surface));
}

.profile-tab strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.profile-tab span {
  color: var(--color-muted);
  font-size: 0.86rem;
  line-height: 1.35;
}

.panel-card {
  width: 100%;
  justify-self: center;
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

.identity-card,
.game-card,
.rating-card {
  align-self: start;
}

.identity-card {
  max-width: 760px;
}

.game-card {
  max-width: 820px;
}

.card-head,
.profile-header,
.actions,
.inline-actions,
.rating-head {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: flex-start;
  flex-wrap: wrap;
}

.card-head h2,
.editor h3,
.identity .name,
.rating-history h3 {
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
.label,
.helper,
.state-copy {
  color: var(--color-muted);
}

.section-copy,
.helper,
.state-copy,
.chart-footnote {
  margin: 0;
}

.profile-header {
  align-items: center;
}

.avatar {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  font-weight: 700;
  background: color-mix(in oklab, var(--color-surface), white 12%);
}

.avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar span {
  font-size: 1.5rem;
}

.identity {
  display: grid;
  gap: 0.2rem;
}

.identity p {
  margin: 0;
}

.meta,
.rating-history,
.chart-shell {
  display: grid;
  gap: 0.7rem;
}

.row {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
  align-items: baseline;
  padding-bottom: 0.55rem;
  border-bottom: 1px dashed color-mix(in oklab, var(--color-border), transparent 10%);
}

.value {
  text-align: right;
  font-weight: 600;
}

.value.ok {
  color: var(--color-success);
}

.value.warn,
.state-copy.error,
.feedback .err,
.chart-empty.error {
  color: var(--color-danger);
}

.value.balance,
.feedback .ok,
.summary-card strong.positive {
  color: var(--color-success);
}

.roles {
  display: inline-flex;
  gap: 0.4rem;
  flex-wrap: wrap;
}

.chip,
.period-chip {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  padding: 0.25rem 0.6rem;
  background: color-mix(in oklab, var(--color-surface), white 8%);
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 12%);
}

.editor,
.game-editor {
  display: grid;
  gap: 0.85rem;
}

.grid,
.rating-summary {
  display: grid;
  gap: 0.85rem;
}

.grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.rating-summary {
  grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
}

.summary-card {
  display: grid;
  gap: 0.35rem;
  padding: 0.95rem 1rem;
  border-radius: 1rem;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.18), rgba(255, 255, 255, 0.04));
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 14%);
}

.summary-card strong {
  font-size: 1.15rem;
}

.rank-with-tooltip {
  position: relative;
  width: fit-content;
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  color: var(--rank-text-color, var(--color-text));
  outline: none;
}

.rank-with-tooltip strong {
  color: currentColor;
}

.rank-text--bronze {
  --rank-text-color: #b45309;
}

.rank-text--silver {
  --rank-text-color: #64748b;
}

.rank-text--gold {
  --rank-text-color: #d97706;
}

.rank-text--platinum {
  --rank-text-color: #0891b2;
}

.rank-text--diamond {
  --rank-text-color: #7c3aed;
}

.rank-help {
  width: 1.25rem;
  height: 1.25rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  border: 1px solid color-mix(in oklab, currentColor, transparent 42%);
  background: color-mix(in oklab, currentColor, transparent 88%);
  font-size: 0.75rem;
  font-weight: 800;
}

.rank-tooltip {
  position: absolute;
  z-index: 5;
  left: 0;
  bottom: calc(100% + 0.55rem);
  width: max-content;
  max-width: min(320px, 70vw);
  padding: 0.7rem 0.8rem;
  border-radius: 0.85rem;
  border: 1px solid color-mix(in oklab, currentColor, transparent 68%);
  background: color-mix(in oklab, var(--color-surface), black 4%);
  color: var(--color-text);
  box-shadow: var(--shadow-md);
  font-size: 0.84rem;
  line-height: 1.35;
  opacity: 0;
  pointer-events: none;
  transform: translateY(0.35rem);
  transition:
    opacity var(--transition-fast) ease,
    transform var(--transition-fast) ease;
}

.rank-with-tooltip:hover .rank-tooltip,
.rank-with-tooltip:focus-visible .rank-tooltip {
  opacity: 1;
  transform: translateY(0);
}

.summary-label {
  color: var(--color-muted);
  font-size: 0.85rem;
}

.wide {
  grid-column: 1 / -1;
}

label {
  display: grid;
  gap: 0.35rem;
}

label span {
  color: var(--color-muted);
  font-size: 0.9rem;
}

input[type='text'],
input[type='email'],
input[type='password'],
input[type='number'] {
  width: 100%;
  padding: 0.8rem 0.95rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
  border-radius: 0.9rem;
  background: color-mix(in oklab, var(--color-surface), white 14%);
  color: var(--color-text);
}

.feedback {
  min-height: 1.2rem;
}

.btn,
.period-chip {
  appearance: none;
  cursor: pointer;
  transition:
    transform var(--transition-fast) ease,
    border-color var(--transition-fast) ease,
    background var(--transition-fast) ease;
}

.btn {
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  background: color-mix(in oklab, var(--color-surface), white 8%);
  color: var(--color-text);
  border-radius: 999px;
  padding: 0.8rem 1rem;
  font-weight: 600;
}

.btn:hover,
.period-chip:hover {
  transform: translateY(-1px);
}

.btn:disabled,
.period-chip:disabled {
  cursor: not-allowed;
  opacity: 0.6;
  transform: none;
}

.btn--primary {
  border-color: transparent;
  background: linear-gradient(135deg, #0f766e, #0284c7);
  color: #f0fdfa;
}

.period-switch {
  display: flex;
  gap: 0.45rem;
  flex-wrap: wrap;
}

.period-chip.active {
  background: linear-gradient(135deg, rgba(15, 118, 110, 0.14), rgba(2, 132, 199, 0.18));
  border-color: rgba(2, 132, 199, 0.35);
}

.chart-shell {
  justify-items: center;
  padding: 0.95rem;
  border-radius: 1.15rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 12%);
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.16), rgba(255, 255, 255, 0.04));
}

.rating-chart {
  width: 100%;
  max-width: 720px;
  height: clamp(150px, 20vw, 220px);
  overflow: visible;
}

.chart-axis,
.chart-footnote {
  width: 100%;
}

.chart-line {
  fill: none;
  stroke: #0891b2;
  stroke-width: 2.25;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.chart-dot {
  fill: #0284c7;
  stroke: white;
  stroke-width: 0.8;
}

.chart-axis,
.chart-footnote {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  color: var(--color-muted);
  font-size: 0.84rem;
}

.chart-empty {
  padding: 1rem;
  border-radius: 1rem;
  border: 1px dashed color-mix(in oklab, var(--color-border), transparent 10%);
  color: var(--color-muted);
}

.state-block {
  display: grid;
  gap: 0.75rem;
}

@media (max-width: 960px) {
  .profile-area,
  .profile-area.collapsed {
    position: static;
    inset: auto;
    margin: calc(76px + 0.75rem) 1rem 5.75rem;
  }
}

@media (max-width: 760px) {
  .profile-tabs {
    grid-template-columns: 1fr;
  }

  .panel-card {
    padding: 1rem;
  }

  .grid,
  .rating-summary {
    grid-template-columns: 1fr;
  }

  .row {
    flex-direction: column;
    align-items: flex-start;
  }

  .value {
    text-align: left;
  }

  .actions,
  .inline-actions {
    justify-content: stretch;
  }

  .actions .btn,
  .inline-actions .btn {
    width: 100%;
  }

  .chart-axis,
  .chart-footnote {
    flex-direction: column;
  }
}
</style>
