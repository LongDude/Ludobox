<script lang="ts" setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/authStore'
import { useSettingStore } from '@/stores/settingStore'
import { useUserCabinetStore } from '@/stores/userCabinetStore'
import { useI18n } from '@/i18n'
import ToastCenter from '@/components/ToastCenter.vue'
import { rankFrameClass } from '@/utils/rankFrame'

defineProps<{
  showMenu?: boolean
  showUpload?: boolean
}>()

const authStore = useAuthStore()
const cabinetStore = useUserCabinetStore()
const settings = useSettingStore()
const route = useRoute()
const router = useRouter()
const { locale, t } = useI18n()

const avatarUrl = computed(() => authStore.User?.photo || '')
const avatarRankClass = computed(() => rankFrameClass(cabinetStore.profile?.rank))
const avatarLetter = computed(() => {
  const user = authStore.User
  const name = [user?.first_name, user?.last_name].filter(Boolean).join(' ') || user?.email || ''
  const trimmed = name.trim()
  return trimmed ? trimmed[0].toUpperCase() : 'U'
})

const userName = computed(() => {
  const user = authStore.User
  if (!user) return t('auth.login')
  return [user.first_name, user.last_name].filter(Boolean).join(' ') || user.email || t('profile.title')
})

const balanceValue = computed(() => {
  if (!authStore.isAuthenticated) return ''
  if (cabinetStore.loading && !cabinetStore.profile) return '...'
  if (cabinetStore.profile) {
    return new Intl.NumberFormat(locale.value === 'ru' ? 'ru-RU' : 'en-US').format(
      cabinetStore.profile.balance,
    )
  }
  return '-'
})

const routeMeta = computed(() => {
  if (route.path === '/') {
    return {
      eyebrow: t('layout.route.homeKicker'),
      title: t('layout.route.homeTitle'),
    }
  }

  if (route.path === '/rooms') {
    return {
      eyebrow: t('layout.route.roomsKicker'),
      title: t('layout.route.roomsTitle'),
    }
  }

  if (route.path.startsWith('/play/')) {
    return {
      eyebrow: t('layout.route.playKicker'),
      title: t('layout.route.playTitle'),
    }
  }

  if (route.path === '/admin') {
    return {
      eyebrow: t('layout.route.adminKicker'),
      title: t('layout.route.adminTitle'),
    }
  }

  if (route.path === '/settings') {
    return {
      eyebrow: t('layout.route.settingsKicker'),
      title: t('layout.route.settingsTitle'),
    }
  }

  if (route.path === '/profile') {
    return {
      eyebrow: t('layout.route.profileKicker'),
      title: t('layout.route.profileTitle'),
    }
  }

  if (route.path === '/history') {
    return {
      eyebrow: t('layout.route.historyKicker'),
      title: t('layout.route.historyTitle'),
    }
  }

  return {
    eyebrow: t('layout.route.defaultKicker'),
    title: t('layout.route.defaultTitle'),
  }
})

function redirectTo(path: string) {
  router.push(path)
}

async function logout() {
  await authStore.logout()
  router.replace('/auth')
}
</script>

<template>
  <header class="up-tab" :class="{ collapsed: settings.LeftTabHidden }">
    <button class="brand" type="button" @click="redirectTo('/')">
      <span class="brand-mark">
        <img class="brand-logo" src="./../assets/logo_micro.svg" alt="LudoBox" />
      </span>
    </button>

    <div class="page-copy">
      <span class="page-kicker">{{ routeMeta.eyebrow }}</span>
      <strong>{{ routeMeta.title }}</strong>
    </div>

    <div class="actions">
      <span class="locale-pill">{{ locale.toUpperCase() }}</span>

      <div v-if="authStore.isAuthenticated" class="balance-pill" :aria-label="t('layout.balanceLabel')">
        <img src="./../assets/balance.svg" alt="" class="balance-icon" />
        <strong>{{ balanceValue }}</strong>
      </div>

      <button
        v-if="authStore.isAuthenticated"
        class="action-button avatar mobile-profile-action"
        :class="['rank-frame', avatarRankClass]"
        type="button"
        :aria-label="userName"
        @click="redirectTo('/profile')"
      >
        <img v-if="avatarUrl" :src="avatarUrl" alt="" class="avatar-image" />
        <span v-else>{{ avatarLetter }}</span>
      </button>

      <button
        v-if="!authStore.isAuthenticated"
        class="action-button text-button"
        type="button"
        @click="redirectTo('/auth')"
      >
        {{ t('auth.login') }}
      </button>

      <button
        v-else
        class="action-button icon-button"
        type="button"
        :aria-label="userName"
        @click="logout"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path
            d="M15 4h2a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2h-2"
            fill="none"
            stroke="currentColor"
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="1.8"
          />
          <path
            d="M10 17l5-5-5-5"
            fill="none"
            stroke="currentColor"
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="1.8"
          />
          <path
            d="M15 12H5"
            fill="none"
            stroke="currentColor"
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="1.8"
          />
        </svg>
      </button>
    </div>
  </header>

  <ToastCenter />
</template>

<style scoped>
.up-tab {
  position: fixed;
  top: 1rem;
  left: 304px;
  right: 1.25rem;
  z-index: 35;
  min-height: 60px;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 1rem;
  padding: 0.7rem 1rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
  border-radius: 1.4rem;
  background:
    linear-gradient(
      135deg,
      color-mix(in oklab, var(--color-bg-secondary), white 22%),
      color-mix(in oklab, var(--color-surface), transparent 6%)
    );
  box-shadow: var(--shadow-md);
  backdrop-filter: blur(18px);
  transition:
    left var(--transition-slow) ease,
    right var(--transition-slow) ease,
    background var(--transition-base) ease;
}

.up-tab.collapsed {
  left: 120px;
}

.brand,
.action-button {
  appearance: none;
  border: none;
  background: transparent;
  color: var(--color-text);
}

.brand {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 0.85rem;
  padding: 0.15rem;
  cursor: pointer;
}

.brand-mark {
  width: 2.75rem;
  height: 2.75rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 1rem;
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.2), rgba(236, 72, 153, 0.14));
  border: 1px solid color-mix(in oklab, var(--color-warning), white 62%);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.2);
}

.brand-logo {
  width: 2.75rem;
  height: 2.75rem;
  display: block;
}

.brand-copy,
.page-copy {
  min-width: 0;
  display: grid;
}

.brand-copy strong,
.page-copy strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.brand-copy strong {
  font-size: 1rem;
}

.brand-copy small,
.page-kicker {
  text-transform: uppercase;
  letter-spacing: 0.12em;
  font-size: 0.68rem;
  color: var(--color-muted);
}

.page-copy strong {
  font-size: 0.95rem;
}

.actions {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.65rem;
}

.locale-pill,
.balance-pill,
.icon-button,
.text-button,
.avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
}

.locale-pill,
.balance-pill {
  height: 2.5rem;
  padding: 0 0.9rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 6%);
  background: color-mix(in oklab, var(--color-surface), white 8%);
}

.locale-pill {
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.12em;
  color: var(--color-muted);
}

.balance-pill {
  gap: 0.55rem;
  color: var(--color-text);
}

.balance-icon {
  width: 1.15rem;
  height: 1.15rem;
  display: block;
}

.action-button {
  height: 2.5rem;
  padding: 0 0.95rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 12%);
  background: color-mix(in oklab, var(--color-surface), white 10%);
  box-shadow: var(--shadow-sm);
  cursor: pointer;
  transition:
    transform var(--transition-fast) ease,
    border-color var(--transition-fast) ease,
    background var(--transition-fast) ease;
}

.action-button:hover {
  transform: translateY(-1px);
  border-color: color-mix(in oklab, var(--color-primary-secondary), transparent 18%);
}

.text-button {
  font-weight: 600;
}

.icon-button {
  width: 2.5rem;
  padding: 0;
}

.icon-button svg {
  width: 1.1rem;
  height: 1.1rem;
}

.avatar {
  display: none;
  width: 2.5rem;
  padding: 0;
  overflow: hidden;
  font-weight: 700;
  border-color: color-mix(in oklab, var(--color-primary-secondary), transparent 6%);
}

.avatar-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

@media (max-width: 1100px) {
  .page-copy {
    display: none;
  }
}

@media (max-width: 960px) {
  .up-tab,
  .up-tab.collapsed {
    left: 0.75rem;
    right: 0.75rem;
    top: 0.75rem;
  }

  .mobile-profile-action {
    display: inline-flex;
  }
}

@media (max-width: 720px) {
  .up-tab {
    grid-template-columns: auto auto;
    gap: 0.75rem;
  }

  .brand-copy,
  .locale-pill {
    display: none;
  }

  .actions {
    gap: 0.45rem;
  }

  .balance-pill {
    padding: 0 0.75rem;
  }
}
</style>
