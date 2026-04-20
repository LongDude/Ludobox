<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import LeftTab from '@/components/LeftTab.vue'
import UpTab from '@/components/UpTab.vue'
import FooterTab from '@/components/FooterTab.vue'
import { useMatchSessionStore } from '@/stores/matchSessionStore'
import { useI18n } from '@/i18n'
import { useLayoutInset } from '@/composables/useLayoutInset'

const route = useRoute()
const router = useRouter()
const session = useMatchSessionStore()
const { t } = useI18n()
const { LeftTabHidden: leftHidden, layoutInset } = useLayoutInset()

const roomId = computed(() => Number(route.params.roomId))

const room = computed(() => {
  const candidate = session.selectedRoom
  if (!candidate) return null
  return candidate.room_id === roomId.value ? candidate : null
})

const quickMatchMeta = computed(() => {
  if (!room.value || session.source !== 'quick-match') return null
  return session.quickMatchMeta
})

const sourceLabel = computed(() => {
  if (session.source === 'quick-match') return t('matchmaking.play.sourceQuick')
  if (session.source === 'recommendation') return t('matchmaking.play.sourceRecommendation')
  return t('matchmaking.play.sourceUnknown')
})

function backToHome() {
  router.push('/')
}

function openRooms() {
  router.push('/rooms')
}

function formatBoost() {
  if (!room.value) return t('common.off')
  if (!room.value.is_boost) return t('common.off')
  return t('matchmaking.results.boostValue', { value: room.value.boost_power })
}

function formatScore() {
  if (!room.value) return '-'
  return room.value.score.toFixed(2)
}
</script>

<template>
  <UpTab :show-menu="false" :show-upload="false" />
  <LeftTab />

  <main class="play-area" :class="{ collapsed: leftHidden }" :style="{ '--layout-inset': layoutInset }">
    <section class="hero-card">
      <div>
        <p class="eyebrow">{{ t('matchmaking.play.eyebrow') }}</p>
        <h1>{{ t('matchmaking.play.title', { roomId }) }}</h1>
        <p class="description">{{ t('matchmaking.play.description') }}</p>
      </div>
      <span class="source-pill">{{ sourceLabel }}</span>
    </section>

    <section class="play-grid">
      <article class="panel-card">
        <div class="card-head">
          <div>
            <p class="eyebrow accent">{{ t('matchmaking.play.entryEyebrow') }}</p>
            <h2>{{ t('matchmaking.play.entryTitle') }}</h2>
            <p class="description">{{ t('matchmaking.play.entryDescription') }}</p>
          </div>
        </div>

        <div v-if="room" class="meta-grid">
          <div class="meta-item">
            <span>{{ t('matchmaking.play.meta.entry') }}</span>
            <strong>{{ room.registration_price }}</strong>
          </div>
          <div class="meta-item">
            <span>{{ t('matchmaking.play.meta.capacity') }}</span>
            <strong>{{ room.capacity }}</strong>
          </div>
          <div class="meta-item">
            <span>{{ t('matchmaking.play.meta.players') }}</span>
            <strong>{{ room.current_players }}</strong>
          </div>
          <div class="meta-item">
            <span>{{ t('matchmaking.play.meta.minimumUsers') }}</span>
            <strong>{{ room.min_users }}</strong>
          </div>
          <div class="meta-item">
            <span>{{ t('matchmaking.play.meta.boost') }}</span>
            <strong>{{ formatBoost() }}</strong>
          </div>
          <div class="meta-item">
            <span>{{ t('matchmaking.play.meta.score') }}</span>
            <strong>{{ formatScore() }}</strong>
          </div>
        </div>
        <p v-else class="muted-copy">{{ t('matchmaking.play.missingSession') }}</p>
      </article>

      <article class="panel-card">
        <div class="card-head">
          <div>
            <p class="eyebrow">{{ t('matchmaking.play.connectionEyebrow') }}</p>
            <h2>{{ t('matchmaking.play.connectionTitle') }}</h2>
            <p class="description">{{ t('matchmaking.play.connectionDescription') }}</p>
          </div>
        </div>

        <div class="meta-grid compact">
          <div class="meta-item">
            <span>{{ t('matchmaking.play.meta.roomId') }}</span>
            <strong>#{{ roomId }}</strong>
          </div>
          <div class="meta-item">
            <span>{{ t('matchmaking.play.meta.source') }}</span>
            <strong>{{ sourceLabel }}</strong>
          </div>
          <template v-if="quickMatchMeta">
            <div class="meta-item">
              <span>{{ t('matchmaking.play.meta.roundId') }}</span>
              <strong>#{{ quickMatchMeta.round_id }}</strong>
            </div>
            <div class="meta-item">
              <span>{{ t('matchmaking.play.meta.seatNumber') }}</span>
              <strong>{{ quickMatchMeta.seat_number }}</strong>
            </div>
            <div class="meta-item wide">
              <span>{{ t('matchmaking.play.meta.reusedRoom') }}</span>
              <strong>
                {{
                  quickMatchMeta.reused_existing_room
                    ? t('common.yes')
                    : t('common.no')
                }}
              </strong>
            </div>
          </template>
        </div>

        <p class="helper-copy">{{ t('matchmaking.play.pending') }}</p>

        <div class="actions">
          <button class="btn" type="button" @click="backToHome">
            {{ t('matchmaking.play.backHome') }}
          </button>
          <button class="btn btn--primary" type="button" @click="openRooms">
            {{ t('matchmaking.play.backRooms') }}
          </button>
        </div>
      </article>
    </section>
  </main>

  <FooterTab />
</template>

<style scoped>
.play-area {
  position: fixed;
  inset: var(--layout-inset, 92px 20px 20px 304px);
  display: grid;
  gap: 1rem;
  overflow: auto;
  align-content: start;
  transition: all var(--transition-slow) ease;
}

.play-area.collapsed {
  --layout-inset: 92px 20px 20px 120px;
}

.hero-card,
.panel-card {
  padding: 1.35rem;
  border-radius: 1.6rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
  box-shadow: var(--shadow-md);
}

.hero-card {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 1rem;
  background:
    radial-gradient(circle at top right, rgba(245, 158, 11, 0.18), transparent 24%),
    linear-gradient(
      135deg,
      color-mix(in oklab, var(--color-bg-secondary), white 18%),
      color-mix(in oklab, var(--color-surface), transparent 6%)
    );
}

.play-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
}

.panel-card {
  display: grid;
  gap: 1rem;
  background:
    radial-gradient(circle at top left, color-mix(in oklab, #0ea5e9, white 88%), transparent 28%),
    linear-gradient(180deg, color-mix(in oklab, var(--color-surface), white 14%), var(--color-surface));
}

.card-head {
  display: grid;
  gap: 0.5rem;
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
.muted-copy,
.helper-copy,
.meta-item span {
  color: var(--color-muted);
}

.source-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  padding: 0.55rem 0.85rem;
  background: color-mix(in oklab, var(--color-surface), white 10%);
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
  font-weight: 600;
}

.meta-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.85rem;
}

.meta-grid.compact .wide {
  grid-column: 1 / -1;
}

.meta-item {
  display: grid;
  gap: 0.25rem;
  padding: 0.85rem;
  border-radius: 1rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  background: color-mix(in oklab, var(--color-surface), white 10%);
}

.actions {
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

.btn--primary {
  border-color: transparent;
  background: linear-gradient(135deg, #0f766e, #0284c7);
  color: #f0fdfa;
}

@media (max-width: 1080px) {
  .play-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 960px) {
  .play-area,
  .play-area.collapsed {
    position: static;
    inset: auto;
    margin: calc(76px + 0.75rem) 1rem 5.75rem;
  }
}

@media (max-width: 720px) {
  .hero-card,
  .meta-grid {
    grid-template-columns: 1fr;
  }

  .actions {
    justify-content: stretch;
  }

  .actions .btn {
    width: 100%;
  }
}
</style>
