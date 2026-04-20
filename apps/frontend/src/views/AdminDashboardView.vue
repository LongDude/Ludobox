<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import LeftTab from '@/components/LeftTab.vue'
import UpTab from '@/components/UpTab.vue'
import { useLayoutInset } from '@/composables/useLayoutInset'
import { useI18n } from '@/i18n'
import AdminUsersSection from '@/components/admin/AdminUsersSection.vue'
import AdminGamesSection from '@/components/admin/AdminGamesSection.vue'
import AdminConfigsSection from '@/components/admin/AdminConfigsSection.vue'
import AdminRoomsSection from '@/components/admin/AdminRoomsSection.vue'
import AdminServerOverviewSection from '@/components/admin/AdminServerOverviewSection.vue'

type AdminTab = 'overview' | 'games' | 'users' | 'configs' | 'rooms'

interface TabMeta {
  key: AdminTab
  label: string
  title: string
  description: string
}

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const { LeftTabHidden: leftHidden, layoutInset } = useLayoutInset({
  expanded: '92px 20px 20px 304px',
  collapsed: '92px 20px 20px 120px',
})

const tabs = computed<TabMeta[]>(() => [
  {
    key: 'overview',
    label: t('admin.dashboard.tabs.overview.label'),
    title: t('admin.dashboard.tabs.overview.title'),
    description: t('admin.dashboard.tabs.overview.description'),
  },
  {
    key: 'games',
    label: t('admin.dashboard.tabs.games.label'),
    title: t('admin.dashboard.tabs.games.title'),
    description: t('admin.dashboard.tabs.games.description'),
  },
  {
    key: 'configs',
    label: t('admin.dashboard.tabs.configs.label'),
    title: t('admin.dashboard.tabs.configs.title'),
    description: t('admin.dashboard.tabs.configs.description'),
  },
  {
    key: 'rooms',
    label: t('admin.dashboard.tabs.rooms.label'),
    title: t('admin.dashboard.tabs.rooms.title'),
    description: t('admin.dashboard.tabs.rooms.description'),
  },
  {
    key: 'users',
    label: t('admin.dashboard.tabs.users.label'),
    title: t('admin.dashboard.tabs.users.title'),
    description: t('admin.dashboard.tabs.users.description'),
  },
])

const componentMap: Record<AdminTab, typeof AdminUsersSection> = {
  overview: AdminServerOverviewSection,
  games: AdminGamesSection,
  users: AdminUsersSection,
  configs: AdminConfigsSection,
  rooms: AdminRoomsSection,
}

function normalizeTab(value: unknown): AdminTab {
  const candidate = Array.isArray(value) ? value[0] : value
  if (
    candidate === 'overview' ||
    candidate === 'games' ||
    candidate === 'users' ||
    candidate === 'configs' ||
    candidate === 'rooms'
  ) {
    return candidate
  }
  return 'overview'
}

const activeTab = computed<AdminTab>(() => normalizeTab(route.query.tab))
const activeMeta = computed(
  () => tabs.value.find((tab) => tab.key === activeTab.value) ?? tabs.value[0],
)
const activeComponent = computed(() => componentMap[activeTab.value])

function setTab(tab: AdminTab) {
  router.replace({
    path: '/admin',
    query: tab === 'overview' ? {} : { tab },
  })
}
</script>

<template>
  <UpTab :show-menu="false" :show-upload="false" />
  <LeftTab />

  <div class="admin-shell" :class="{ collapsed: leftHidden }" :style="{ '--layout-inset': layoutInset }">
    <section class="intro-card">
      <div class="intro-copy">
        <p class="eyebrow">{{ t('admin.dashboard.kicker') }}</p>
        <h1>{{ t('admin.dashboard.title') }}</h1>
        <p class="intro-text">{{ t('admin.dashboard.intro') }}</p>
      </div>

      <div class="intro-metrics">
        <article class="metric-tile">
          <span>{{ t('admin.dashboard.scopeLabel') }}</span>
          <strong>{{ t('admin.dashboard.scopeValue') }}</strong>
          <small>{{ t('admin.dashboard.scopeHint') }}</small>
        </article>
        <article class="metric-tile">
          <span>{{ t('admin.dashboard.visibilityLabel') }}</span>
          <strong>{{ t('admin.dashboard.visibilityValue') }}</strong>
          <small>{{ t('admin.dashboard.visibilityHint') }}</small>
        </article>
      </div>
    </section>

    <nav class="tab-strip" :aria-label="t('admin.dashboard.tabsAria')">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        class="tab-pill"
        :class="{ active: activeTab === tab.key }"
        type="button"
        @click="setTab(tab.key)"
      >
        <small>{{ tab.label }}</small>
        <strong>{{ tab.title }}</strong>
      </button>
    </nav>

    <section class="section-frame">
      <div class="section-header">
        <div>
          <p class="eyebrow accent">{{ activeMeta.label }}</p>
          <h2>{{ activeMeta.title }}</h2>
          <p class="section-text">{{ activeMeta.description }}</p>
        </div>
      </div>

      <KeepAlive>
        <component :is="activeComponent" />
      </KeepAlive>
    </section>
  </div>
</template>

<style scoped>
.admin-shell {
  position: fixed;
  inset: var(--layout-inset, 92px 20px 20px 304px);
  display: grid;
  gap: 1rem;
  overflow: auto;
  transition: all var(--transition-slow) ease;
}

.admin-shell.collapsed {
  --layout-inset: 92px 20px 20px 120px;
}

.intro-card,
.section-frame {
  border-radius: 1.6rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
  box-shadow: var(--shadow-md);
}

.intro-card {
  display: grid;
  grid-template-columns: minmax(0, 1.5fr) minmax(18rem, 0.9fr);
  gap: 1rem;
  padding: 1.3rem;
  background:
    radial-gradient(circle at top left, rgba(245, 158, 11, 0.18), transparent 30%),
    radial-gradient(circle at bottom right, rgba(14, 165, 233, 0.16), transparent 26%),
    linear-gradient(
      135deg,
      color-mix(in oklab, var(--color-bg-secondary), white 16%),
      color-mix(in oklab, var(--color-surface), transparent 4%)
    );
}

.eyebrow {
  margin: 0 0 0.35rem;
  text-transform: uppercase;
  letter-spacing: 0.14em;
  font-size: 0.72rem;
  color: #b45309;
}

.eyebrow.accent {
  color: #0f766e;
}

.intro-card h1,
.section-frame h2 {
  margin: 0;
}

.intro-text,
.section-text {
  margin: 0.7rem 0 0;
  color: var(--color-muted);
}

.intro-metrics {
  display: grid;
  gap: 0.8rem;
}

.metric-tile {
  display: grid;
  gap: 0.35rem;
  padding: 1rem;
  border-radius: 1.2rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  background: color-mix(in oklab, var(--color-surface), white 10%);
}

.metric-tile span,
.metric-tile small {
  color: var(--color-muted);
}

.metric-tile strong {
  font-size: 1.05rem;
}

.tab-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.tab-pill {
  min-width: 12rem;
  display: grid;
  gap: 0.2rem;
  padding: 0.85rem 1rem;
  text-align: left;
  border-radius: 1.1rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  background: color-mix(in oklab, var(--color-surface), white 10%);
  color: var(--color-text);
  cursor: pointer;
  transition:
    transform var(--transition-fast) ease,
    border-color var(--transition-fast) ease,
    box-shadow var(--transition-fast) ease;
}

.tab-pill:hover {
  transform: translateY(-1px);
  box-shadow: var(--shadow-sm);
}

.tab-pill.active {
  border-color: color-mix(in oklab, #0f766e, white 46%);
  background: color-mix(in oklab, #0f766e, transparent 88%);
}

.tab-pill small {
  text-transform: uppercase;
  letter-spacing: 0.12em;
  font-size: 0.68rem;
  color: var(--color-muted);
}

.section-frame {
  display: grid;
  gap: 1rem;
  padding: 1.15rem;
  background:
    linear-gradient(180deg, color-mix(in oklab, var(--color-surface), white 18%), var(--color-bg));
}

@media (max-width: 1100px) {
  .intro-card {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 960px) {
  .admin-shell,
  .admin-shell.collapsed {
    position: static;
    inset: auto;
    margin: calc(76px + 0.75rem) 1rem 5.75rem;
  }
}

@media (max-width: 720px) {
  .intro-card,
  .section-frame {
    padding: 1rem;
    border-radius: 1.25rem;
  }

  .tab-strip {
    display: grid;
    grid-template-columns: 1fr;
  }

  .tab-pill {
    min-width: 0;
  }
}
</style>
