<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import LeftTab from '@/components/LeftTab.vue'
import UpTab from '@/components/UpTab.vue'
import { useAuthStore } from '@/stores/authStore'
import { useUserCabinetStore } from '@/stores/userCabinetStore'
import type { UserUpdateRequest } from '@/api/types'
import { useI18n } from '@/i18n'
import { useLayoutInset } from '@/composables/useLayoutInset'

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

onMounted(async () => {
  if (auth.isAuthenticated && !auth.User) {
    try {
      await auth.authenticate()
    } catch {}
  }

  if (auth.isAuthenticated) {
    void cabinet.ensureLoaded().catch(() => {})
  }
})

const fullName = computed(() => {
  const user = auth.User
  if (!user) return ''
  return [user.first_name, user.last_name].filter(Boolean).join(' ')
})

const avatarUrl = computed(() => auth.User?.photo || '')

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

function displayRole(role: string) {
  const normalized = role.toLowerCase()
  if (normalized === 'admin') return t('roles.admin')
  if (normalized === 'user') return t('roles.user')
  return role
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
    await cabinet.refresh()
  } catch (error: any) {
    gameErrorMsg.value = error?.message || t('profile.game.msg.failedLoad')
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
  } catch (error: any) {
    gameErrorMsg.value = error?.message || t('profile.game.msg.failedBalance')
  } finally {
    balanceSaving.value = false
  }
}

async function logout() {
  await auth.logout()
  cabinet.reset()
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
      <div class="profile-grid">
        <section class="panel-card">
          <div class="card-head">
            <div>
              <p class="eyebrow">{{ t('profile.identityEyebrow') }}</p>
              <h2>{{ t('profile.identityTitle') }}</h2>
              <p class="section-copy">{{ t('profile.identityDescription') }}</p>
            </div>
          </div>

          <div class="profile-header">
            <div class="avatar">
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

        <section class="panel-card">
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
              <!-- <div class="row">
                <span class="label">{{ t('profile.game.userId') }}</span>
                <span class="value">#{{ cabinet.profile.user_id }}</span>
              </div> -->
              <div class="row">
                <span class="label">{{ t('profile.game.nickname') }}</span>
                <span class="value">{{ cabinet.profile.nickname }}</span>
              </div>
              <div class="row">
                <span class="label">{{ t('profile.game.balance') }}</span>
                <span class="value balance">{{ formattedBalance }}</span>
              </div>
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
                  {{
                    nicknameSaving
                      ? t('common.saving')
                      : t('profile.game.saveNickname')
                  }}
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
                  {{
                    balanceSaving
                      ? t('common.saving')
                      : t('profile.game.applyDelta')
                  }}
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
      </div>
    </div>
  </div>
</template>

<style scoped>
.profile-area {
  position: fixed;
  inset: var(--layout-inset, 60px 20px 20px 310px);
  display: grid;
  align-items: start;
  overflow: auto;
  transition: all var(--transition-slow) ease;
}

.profile-area.collapsed {
  --layout-inset: 60px 20px 20px 80px;
}

.container {
  max-width: 1100px;
  margin: auto;
  width: 100%;
}

.profile-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.15fr) minmax(21rem, 0.85fr);
  gap: 1rem;
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
.profile-header,
.actions,
.inline-actions {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: flex-start;
  flex-wrap: wrap;
}

.card-head h2,
.editor h3,
.identity .name {
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
.state-copy {
  margin: 0;
}

.profile-header {
  align-items: center;
}

.avatar {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  border: 2px solid color-mix(in oklab, var(--color-primary-secondary), transparent 15%);
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

.meta {
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
.feedback .err {
  color: var(--color-danger);
}

.value.balance,
.feedback .ok {
  color: var(--color-success);
}

.roles {
  display: inline-flex;
  gap: 0.4rem;
  flex-wrap: wrap;
}

.chip {
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

.grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.85rem;
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

.state-block {
  display: grid;
  gap: 0.75rem;
}

@media (max-width: 1100px) {
  .profile-grid {
    grid-template-columns: 1fr;
  }
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
  .panel-card {
    padding: 1rem;
  }

  .grid {
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
}
</style>
