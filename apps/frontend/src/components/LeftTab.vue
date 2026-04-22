<script lang="ts" setup>
import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useSettingStore } from '@/stores/settingStore'
import { useAuthStore } from '@/stores/authStore'
import { useUserCabinetStore } from '@/stores/userCabinetStore'
import { useI18n } from '@/i18n'
import { rankFrameClass } from '@/utils/rankFrame'

interface NavItem {
  key: string
  label: string
  path: string
  icon: 'home' | 'admin' | 'history' | 'settings'
}

const authStore = useAuthStore()
const cabinetStore = useUserCabinetStore()
const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const settingStore = useSettingStore()
const { LeftTabHidden } = storeToRefs(settingStore)

const props = defineProps<{
  hidden?: boolean
}>()

watch(
  () => props.hidden,
  (value) => {
    if (typeof value === 'boolean' && value !== LeftTabHidden.value) {
      LeftTabHidden.value = value
    }
  },
  { immediate: true },
)

const navItems = computed<NavItem[]>(() => {
  const items: NavItem[] = [
    {
      key: 'home',
      label: t('nav.home'),
      path: '/',
      icon: 'home',
    },
  ]

  if (authStore.User && authStore.isAdmin) {
    items.push({
      key: 'admin',
      label: t('nav.adminPanel'),
      path: '/admin',
      icon: 'admin',
    })
  }

  if (authStore.isAuthenticated) {
    items.push({
      key: 'history',
      label: t('nav.gameHistory'),
      path: '/history',
      icon: 'history',
    })

    items.push({
      key: 'settings',
      label: t('nav.settings'),
      path: '/settings',
      icon: 'settings',
    })
  }

  return items
})

const profileName = computed(() => {
  const user = authStore.User
  return [user?.first_name, user?.last_name].filter(Boolean).join(' ') || user?.email || t('profile.title')
})

const profileRole = computed(() => {
  const primaryRole = authStore.roles[0]?.toLowerCase()
  if (primaryRole === 'admin' || authStore.isAdmin) return t('roles.admin')
  if (primaryRole === 'user') return t('roles.user')
  return authStore.roles[0] || t('roles.user')
})

const avatarUrl = computed(() => authStore.User?.photo || '')
const avatarRankClass = computed(() => rankFrameClass(cabinetStore.profile?.rank))

const avatarLetter = computed(() => {
  const user = authStore.User
  const name = [user?.first_name, user?.last_name].filter(Boolean).join(' ') || user?.email || ''
  const trimmed = name.trim()
  return trimmed ? trimmed[0].toUpperCase() : 'U'
})

function toggleLeftTab() {
  settingStore.HideLeftTab()
}

function redirectTo(path: string) {
  router.push(path)
}

function isActive(path: string) {
  return path === '/' ? route.path === '/' : route.path.startsWith(path)
}
</script>

<template>
  <aside class="left-tab" :class="{ hidden: LeftTabHidden }">
    <div class="shell">
      <div class="header">
        <button class="brand-button" type="button" @click="redirectTo('/')">
          <span class="brand-mark">
            <img alt="LudoBox" src="./../assets/logo_micro.svg" class="brand-logo" />
          </span>
          <span v-if="!LeftTabHidden" class="brand-copy">
            <strong>LudoBox</strong>
            <small>{{ t('layout.sidebarTagline') }}</small>
          </span>
        </button>

        <button
          class="toggle-button"
          type="button"
          @click="toggleLeftTab"
          :aria-expanded="!LeftTabHidden"
          :aria-label="t('layout.toggleSidebar')"
        >
          <svg class="chevron" viewBox="0 0 16 16" aria-hidden="true">
            <path
              d="M6 3l5 5-5 5"
              fill="none"
              stroke="currentColor"
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="1.8"
            />
          </svg>
        </button>
      </div>

      <nav class="nav-list" :aria-label="t('nav.primary')">
        <button
          v-for="item in navItems"
          :key="item.key"
          class="nav-button"
          :class="{ active: isActive(item.path), compact: LeftTabHidden }"
          type="button"
          @click="redirectTo(item.path)"
        >
          <span class="nav-icon" :class="item.icon">
            <svg v-if="item.icon === 'home'" viewBox="0 0 24 24" aria-hidden="true">
              <path
                d="M4 11.5L12 5l8 6.5"
                fill="none"
                stroke="currentColor"
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="1.8"
              />
              <path
                d="M6.5 10.5V19h11v-8.5"
                fill="none"
                stroke="currentColor"
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="1.8"
              />
            </svg>
            <svg v-else-if="item.icon === 'admin'" viewBox="0 0 24 24" aria-hidden="true">
              <rect
                x="4"
                y="4"
                width="6"
                height="6"
                rx="1.4"
                fill="none"
                stroke="currentColor"
                stroke-width="1.8"
              />
              <rect
                x="14"
                y="4"
                width="6"
                height="6"
                rx="1.4"
                fill="none"
                stroke="currentColor"
                stroke-width="1.8"
              />
              <rect
                x="4"
                y="14"
                width="6"
                height="6"
                rx="1.4"
                fill="none"
                stroke="currentColor"
                stroke-width="1.8"
              />
              <rect
                x="14"
                y="14"
                width="6"
                height="6"
                rx="1.4"
                fill="none"
                stroke="currentColor"
                stroke-width="1.8"
              />
            </svg>
            <svg v-else-if="item.icon === 'history'" viewBox="0 0 24 24" aria-hidden="true">
              <path
                d="M12 7v5l3 2"
                fill="none"
                stroke="currentColor"
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="1.8"
              />
              <circle
                cx="12"
                cy="12"
                r="7.5"
                fill="none"
                stroke="currentColor"
                stroke-width="1.8"
              />
              <path
                d="M5.2 8.5H3.5V4.8"
                fill="none"
                stroke="currentColor"
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="1.8"
              />
            </svg>
            <svg v-else viewBox="0 0 24 24" aria-hidden="true">
              <path
                d="M12 3.5l1.8 2.2 2.8-.3.9 2.7 2.6 1.2-1 2.6 1 2.6-2.6 1.2-.9 2.7-2.8-.3L12 20.5l-1.8-2.2-2.8.3-.9-2.7-2.6-1.2 1-2.6-1-2.6 2.6-1.2.9-2.7 2.8.3z"
                fill="none"
                stroke="currentColor"
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="1.6"
              />
              <circle
                cx="12"
                cy="12"
                r="3"
                fill="none"
                stroke="currentColor"
                stroke-width="1.6"
              />
            </svg>
          </span>
          <span v-if="!LeftTabHidden" class="nav-copy">{{ item.label }}</span>
        </button>
      </nav>

      <div v-if="authStore.isAuthenticated" class="profile-card">
        <button class="profile-button" type="button" @click="redirectTo('/profile')">
          <span class="profile-avatar rank-frame" :class="avatarRankClass">
            <img v-if="avatarUrl" :src="avatarUrl" alt="" class="profile-avatar-image" />
            <span v-else>{{ avatarLetter }}</span>
          </span>
          <span v-if="!LeftTabHidden" class="profile-copy">
            <strong>{{ profileName }}</strong>
            <small>{{ profileRole }}</small>
          </span>
        </button>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.left-tab {
  position: fixed;
  top: 1rem;
  left: 1rem;
  bottom: 1rem;
  z-index: 40;
  width: 272px;
  padding: 1rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 6%);
  border-radius: 1.6rem;
  background:
    radial-gradient(circle at top left, rgba(245, 158, 11, 0.14), transparent 28%),
    linear-gradient(
      180deg,
      color-mix(in oklab, var(--color-bg-secondary), white 12%),
      color-mix(in oklab, var(--color-surface), transparent 4%)
    );
  box-shadow: var(--shadow-md);
  transition:
    width var(--transition-slow) ease,
    background var(--transition-base) ease,
    border-radius var(--transition-slow) ease;
}

.left-tab.hidden {
  width: 88px;
}

.shell {
  height: 100%;
  display: grid;
  grid-template-rows: auto 1fr auto;
  gap: 1rem;
}

.header,
.brand-button,
.nav-button,
.profile-button,
.toggle-button {
  display: inline-flex;
  align-items: center;
}

.header {
  justify-content: space-between;
  gap: 0.75rem;
}

.left-tab.hidden .header {
  flex-direction: column;
  justify-content: flex-start;
  align-items: center;
}

.brand-button,
.nav-button,
.profile-button,
.toggle-button {
  appearance: none;
  box-sizing: border-box;
  border: none;
  background: transparent;
  color: var(--color-text);
}

.brand-button {
  flex: 1;
  justify-content: flex-start;
  gap: 0.85rem;
  min-width: 0;
  padding: 0.3rem;
  text-align: left;
  cursor: pointer;
}

.left-tab.hidden .brand-button {
  flex: none;
  width: auto;
  padding: 0;
  justify-content: center;
}

.brand-mark,
.toggle-button,
.nav-icon,
.profile-avatar {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.brand-mark {
  width: 2.8rem;
  height: 2.8rem;
  border-radius: 1rem;
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.2), rgba(236, 72, 153, 0.14));
  border: 1px solid color-mix(in oklab, var(--color-warning), white 62%);
}

.brand-logo {
  width: 2.8rem;
  height: 2.8rem;
  display: block;
}

.brand-copy,
.profile-copy {
  min-width: 0;
  display: grid;
}

.brand-copy strong,
.profile-copy strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.brand-copy small,
.profile-copy small {
  color: var(--color-muted);
}

.toggle-button {
  width: 2.6rem;
  height: 2.6rem;
  border-radius: 0.95rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  background: color-mix(in oklab, var(--color-surface), white 10%);
  box-shadow: var(--shadow-sm);
  cursor: pointer;
  transition:
    transform var(--transition-fast) ease,
    border-color var(--transition-fast) ease;
}

.toggle-button:hover {
  transform: translateY(-1px);
  border-color: color-mix(in oklab, var(--color-primary-secondary), transparent 18%);
}

.chevron {
  width: 1rem;
  height: 1rem;
  transition: transform var(--transition-fast) ease;
  transform: rotate(180deg);
}

.left-tab.hidden .chevron {
  transform: rotate(0deg);
}

.nav-list {
  min-width: 0;
  display: grid;
  align-content: start;
  gap: 0.55rem;
}

.nav-button {
  width: 100%;
  gap: 0.85rem;
  padding: 0.85rem 0.9rem;
  border-radius: 1rem;
  border: 1px solid transparent;
  text-align: left;
  cursor: pointer;
  transition:
    transform var(--transition-fast) ease,
    border-color var(--transition-fast) ease,
    background var(--transition-fast) ease;
}

.nav-button:hover {
  transform: translateY(-1px);
  border-color: color-mix(in oklab, var(--color-border), transparent 8%);
  background: color-mix(in oklab, var(--color-surface), white 8%);
}

.nav-button.active {
  border-color: color-mix(in oklab, var(--color-primary-secondary), transparent 12%);
  background: color-mix(in oklab, var(--color-primary-secondary), transparent 86%);
  box-shadow: var(--shadow-sm);
}

.nav-button.compact {
  justify-content: center;
  padding: 0.55rem;
}

.nav-icon {
  width: 2.2rem;
  height: 2.2rem;
  border-radius: 0.85rem;
  background: color-mix(in oklab, var(--color-surface), white 10%);
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
}

.nav-icon svg {
  width: 1.1rem;
  height: 1.1rem;
}

.nav-copy {
  font-weight: 600;
}

.profile-card {
  min-width: 0;
  margin-top: auto;
}

.profile-button {
  width: 100%;
  gap: 0.85rem;
  padding: 0.75rem;
  border-radius: 1.1rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  background: color-mix(in oklab, var(--color-surface), white 9%);
  text-align: left;
  cursor: pointer;
}

.left-tab.hidden .profile-button {
  justify-content: center;
  padding: 0.55rem;
}

.profile-avatar {
  width: 2.4rem;
  height: 2.4rem;
  border-radius: 50%;
  overflow: hidden;
  font-weight: 700;
  background: color-mix(in oklab, var(--color-primary-secondary), transparent 78%);
  color: var(--color-text);
}

.profile-avatar-image {
  width: 100%;
  height: 100%;
  display: block;
  object-fit: cover;
}

@media (max-width: 960px) {
  .left-tab,
  .left-tab.hidden {
    top: auto;
    bottom: 0.75rem;
    left: 0.75rem;
    right: 0.75rem;
    width: auto;
    min-height: 0;
    padding: 0.75rem;
    border-radius: 1.3rem;
  }

  .shell {
    grid-template-rows: auto;
  }

  .header,
  .profile-card {
    display: none;
  }

  .nav-list {
    grid-template-columns: repeat(auto-fit, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .left-tab {
    padding: 0.65rem;
  }

  .brand-copy {
    display: none;
  }

  .nav-button,
  .nav-button.compact {
    justify-content: center;
    padding-inline: 0.6rem;
  }

  .nav-copy {
    display: none;
  }
}
</style>
