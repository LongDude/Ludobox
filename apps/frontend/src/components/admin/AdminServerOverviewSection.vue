<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { UserApi } from '@/api/useUserApi'
import type { ConfigResponse, GameServerResponse, RoomResponse } from '@/api/types'
import { useI18n } from '@/i18n'

interface ServerBucket {
  server: GameServerResponse
  rooms: RoomResponse[]
  openCount: number
  inGameCount: number
  completedCount: number
}

const loading = ref(false)
const errorMsg = ref('')
const servers = ref<GameServerResponse[]>([])
const rooms = ref<RoomResponse[]>([])
const configs = ref<ConfigResponse[]>([])
const { locale, t } = useI18n()

const configById = computed(() => new Map(configs.value.map((config) => [config.config_id, config])))

const roomsByServer = computed(() => {
  const byServer = new Map<number, RoomResponse[]>()

  for (const room of rooms.value) {
    const current = byServer.get(room.server_id) ?? []
    current.push(room)
    byServer.set(room.server_id, current)
  }

  return byServer
})

const buckets = computed<ServerBucket[]>(() =>
  servers.value
    .map((server) => {
      const serverRooms = roomsByServer.value.get(server.server_id) ?? []

      return {
        server,
        rooms: serverRooms,
        openCount: serverRooms.filter((room) => room.status === 'open').length,
        inGameCount: serverRooms.filter((room) => room.status === 'in_game').length,
        completedCount: serverRooms.filter((room) => room.status === 'completed').length,
      }
    })
    .sort((left, right) => left.server.server_id - right.server.server_id),
)

const overviewStats = computed(() => ({
  servers: servers.value.length,
  availableServers: servers.value.filter((server) => isAvailableServer(server)).length,
  rooms: rooms.value.length,
  liveRounds: rooms.value.filter((room) => room.status === 'in_game').length,
}))

const idleServers = computed(() => buckets.value.filter((bucket) => bucket.rooms.length === 0).length)

onMounted(async () => {
  await loadOverview()
})

async function loadOverview() {
  loading.value = true
  errorMsg.value = ''

  try {
    const [allServers, allRooms, configResponse] = await Promise.all([
      loadAllServers(),
      loadAllRooms(),
      UserApi.listConfigs({
        page: 1,
        page_size: 100,
        sort_field: 'config_id',
        sort_direction: 'desc',
      }),
    ])

    servers.value = allServers
    rooms.value = allRooms
    configs.value = configResponse.items ?? []
  } catch (error: any) {
    errorMsg.value = error?.message || t('admin.overviewSection.error.load')
  } finally {
    loading.value = false
  }
}

async function loadAllServers() {
  const collected: GameServerResponse[] = []
  let currentPage = 1
  let totalItems = 0

  do {
    const response = await UserApi.listServers({
      page: currentPage,
      page_size: 100,
      sort_field: 'server_id',
      sort_direction: 'asc',
    })

    totalItems = response.total ?? 0
    collected.push(...(response.items ?? []))
    currentPage += 1

    if (!response.items?.length) {
      break
    }
  } while (collected.length < totalItems && currentPage <= 20)

  return collected
}

async function loadAllRooms() {
  const collected: RoomResponse[] = []
  let currentPage = 1
  let totalItems = 0

  do {
    const response = await UserApi.listRooms({
      page: currentPage,
      page_size: 100,
      sort_field: 'server_id',
      sort_direction: 'asc',
    })

    totalItems = response.total ?? 0
    collected.push(...(response.items ?? []))
    currentPage += 1

    if (!response.items?.length) {
      break
    }
  } while (collected.length < totalItems && currentPage <= 20)

  return collected
}

function isAvailableServer(server: GameServerResponse) {
  return !server.archived_at && server.status.trim().toLowerCase() === 'up'
}

function roomSummary(room: RoomResponse) {
  const config = configById.value.get(room.config_id)
  if (!config) return t('admin.roomsSection.configFallback', { id: room.config_id })
  return t('admin.overviewSection.roomSummary', {
    game: config.game?.name_game ?? t('admin.configsSection.gameLabel', { id: config.game_id }),
    capacity: config.capacity,
    price: config.registration_price,
  })
}

function roomStatusLabel(status: RoomResponse['status']) {
  if (status === 'open') return t('admin.status.open')
  if (status === 'in_game') return t('admin.status.inGame')
  return t('admin.status.completed')
}

function normalizedServerStatus(server: GameServerResponse) {
  if (server.archived_at) return 'archived'

  const normalized = server.status.trim().toLowerCase()
  return normalized || 'unknown'
}

function serverStatusLabel(server: GameServerResponse) {
  const status = normalizedServerStatus(server)

  if (status === 'up') return t('admin.overviewSection.serverStatus.up')
  if (status === 'down') return t('admin.overviewSection.serverStatus.down')
  if (status === 'maintenance') return t('admin.overviewSection.serverStatus.maintenance')
  if (status === 'archived') return t('admin.overviewSection.serverStatus.archived')

  return t('admin.overviewSection.serverStatus.unknown')
}

function serverStatusClass(server: GameServerResponse) {
  const status = normalizedServerStatus(server)

  if (status === 'up' || status === 'down' || status === 'maintenance' || status === 'archived') {
    return status
  }

  return 'unknown'
}

function formatTimestamp(value?: string | null) {
  if (!value) return '-'

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value

  return new Intl.DateTimeFormat(locale.value === 'ru' ? 'ru-RU' : 'en-US', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)
}
</script>

<template>
  <section class="section-card">
    <div class="section-header">
      <div>
        <p class="eyebrow">{{ t('admin.overviewSection.eyebrow') }}</p>
        <h2>{{ t('admin.overviewSection.title') }}</h2>
        <p class="section-copy">{{ t('admin.overviewSection.description') }}</p>
      </div>
      <button class="button ghost" @click="loadOverview" :disabled="loading">
        {{ t('common.refresh') }}
      </button>
    </div>

    <div class="stat-grid">
      <article class="stat-card">
        <span>{{ t('admin.overviewSection.stats.servers') }}</span>
        <strong>{{ overviewStats.servers }}</strong>
      </article>
      <article class="stat-card">
        <span>{{ t('admin.overviewSection.stats.availableServers') }}</span>
        <strong>{{ overviewStats.availableServers }}</strong>
      </article>
      <article class="stat-card">
        <span>{{ t('admin.overviewSection.stats.rooms') }}</span>
        <strong>{{ overviewStats.rooms }}</strong>
      </article>
      <article class="stat-card">
        <span>{{ t('admin.overviewSection.stats.liveRounds') }}</span>
        <strong>{{ overviewStats.liveRounds }}</strong>
      </article>
    </div>

    <p v-if="loading" class="state-copy">{{ t('admin.overviewSection.loading') }}</p>
    <p v-else-if="errorMsg" class="state-copy error">{{ errorMsg }}</p>

    <div v-else class="server-grid">
      <article v-for="bucket in buckets" :key="bucket.server.server_id" class="server-card">
        <div class="server-head">
          <div>
            <h3>{{ t('admin.overviewSection.serverTitle', { id: bucket.server.server_id }) }}</h3>
            <p class="muted">
              {{ t('admin.overviewSection.serverRooms', { count: bucket.rooms.length }) }}
            </p>
          </div>
          <span class="server-pill" :class="serverStatusClass(bucket.server)">
            {{ serverStatusLabel(bucket.server) }}
          </span>
        </div>

        <dl class="server-meta">
          <div>
            <dt>{{ t('admin.overviewSection.serverMeta.status') }}</dt>
            <dd>{{ bucket.server.status || t('admin.overviewSection.serverStatus.unknown') }}</dd>
          </div>
          <div>
            <dt>{{ t('admin.overviewSection.serverMeta.instanceKey') }}</dt>
            <dd>{{ bucket.server.instance_key || '-' }}</dd>
          </div>
          <div>
            <dt>{{ t('admin.overviewSection.serverMeta.redisHost') }}</dt>
            <dd>{{ bucket.server.redis_host || '-' }}</dd>
          </div>
          <div>
            <dt>{{ t('admin.overviewSection.serverMeta.heartbeat') }}</dt>
            <dd>{{ formatTimestamp(bucket.server.last_heartbeat_at) }}</dd>
          </div>
        </dl>

        <div class="mini-stats">
          <span class="mini-pill good">{{
            t('admin.overviewSection.open', { count: bucket.openCount })
          }}</span>
          <span class="mini-pill accent">{{
            t('admin.overviewSection.inGame', { count: bucket.inGameCount })
          }}</span>
          <span class="mini-pill neutral">{{
            t('admin.overviewSection.completed', { count: bucket.completedCount })
          }}</span>
        </div>

        <div class="room-stack">
          <article v-for="room in bucket.rooms" :key="room.room_id" class="room-chip">
            <div>
              <strong>{{ t('admin.overviewSection.roomTitle', { id: room.room_id }) }}</strong>
              <p>{{ roomSummary(room) }}</p>
            </div>
            <span class="status" :class="room.status">{{ roomStatusLabel(room.status) }}</span>
          </article>

          <p v-if="bucket.rooms.length === 0" class="empty-room-copy">
            {{ t('admin.overviewSection.noAssignedRooms') }}
          </p>
        </div>
      </article>

      <p v-if="buckets.length === 0" class="state-copy">
        {{ t('admin.overviewSection.noServers') }}
      </p>
    </div>

    <p v-if="!loading && !errorMsg && idleServers > 0" class="state-copy">
      {{ t('admin.overviewSection.idleServers', { count: idleServers }) }}
    </p>
  </section>
</template>

<style scoped>
.section-card {
  display: grid;
  gap: 1.25rem;
  padding: 1.5rem;
  border-radius: 1.5rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
  background:
    radial-gradient(circle at top right, color-mix(in oklab, #0ea5e9, white 84%), transparent 24%),
    linear-gradient(180deg, color-mix(in oklab, var(--color-surface), white 22%), var(--color-surface));
  box-shadow: var(--shadow-md);
}

.section-header,
.server-head {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: flex-start;
  flex-wrap: wrap;
}

.section-header h2,
.server-head h3 {
  margin: 0;
}

.eyebrow {
  margin: 0 0 0.35rem;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  font-size: 0.72rem;
  color: #0369a1;
}

.section-copy,
.muted,
.state-copy {
  color: var(--color-muted);
}

.section-copy {
  margin: 0.45rem 0 0;
  max-width: 50rem;
}

.stat-grid,
.server-grid {
  display: grid;
  gap: 0.9rem;
}

.stat-grid {
  grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr));
}

.stat-card,
.server-card {
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  border-radius: 1.3rem;
  background: color-mix(in oklab, var(--color-surface), white 12%);
}

.stat-card {
  display: grid;
  gap: 0.35rem;
  padding: 1rem;
}

.stat-card span {
  color: var(--color-muted);
}

.stat-card strong {
  font-size: 1.7rem;
}

.server-grid {
  grid-template-columns: repeat(auto-fit, minmax(18rem, 1fr));
}

.server-card {
  display: grid;
  gap: 1rem;
  padding: 1rem;
}

.server-pill,
.mini-pill,
.status,
.button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
}

.server-pill {
  padding: 0.35rem 0.75rem;
  font-size: 0.82rem;
}

.server-pill.up {
  background: color-mix(in oklab, var(--color-success), white 80%);
  color: #166534;
}

.server-pill.down {
  background: color-mix(in oklab, var(--color-danger), white 82%);
  color: #991b1b;
}

.server-pill.maintenance {
  background: color-mix(in oklab, var(--color-warning), white 80%);
  color: #9a3412;
}

.server-pill.archived,
.server-pill.unknown {
  background: color-mix(in oklab, var(--color-border), white 38%);
  color: var(--color-text);
}

.server-meta {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(11rem, 1fr));
  gap: 0.65rem;
  margin: 0;
}

.server-meta div {
  display: grid;
  gap: 0.15rem;
  padding: 0.7rem 0.8rem;
  border-radius: 0.95rem;
  background: color-mix(in oklab, var(--color-surface), white 8%);
}

.server-meta dt {
  color: var(--color-muted);
  font-size: 0.78rem;
}

.server-meta dd {
  margin: 0;
  word-break: break-word;
}

.mini-stats {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.mini-pill {
  padding: 0.3rem 0.7rem;
  font-size: 0.82rem;
}

.mini-pill.good {
  background: color-mix(in oklab, var(--color-success), white 80%);
  color: #166534;
}

.mini-pill.accent {
  background: color-mix(in oklab, #0ea5e9, white 82%);
  color: #075985;
}

.mini-pill.neutral {
  background: color-mix(in oklab, var(--color-border), white 38%);
  color: var(--color-text);
}

.room-stack {
  display: grid;
  gap: 0.65rem;
}

.room-chip,
.empty-room-copy {
  padding: 0.85rem 0.95rem;
  border-radius: 1rem;
  background: color-mix(in oklab, var(--color-surface), white 8%);
}

.room-chip {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
}

.room-chip p {
  margin: 0.2rem 0 0;
  color: var(--color-muted);
  font-size: 0.9rem;
}

.empty-room-copy {
  margin: 0;
  color: var(--color-muted);
}

.status {
  white-space: nowrap;
  height: fit-content;
  padding: 0.3rem 0.65rem;
  font-size: 0.8rem;
  text-transform: capitalize;
}

.status.open {
  background: color-mix(in oklab, var(--color-success), white 80%);
  color: #166534;
}

.status.in_game {
  background: color-mix(in oklab, #0ea5e9, white 80%);
  color: #075985;
}

.status.completed {
  background: color-mix(in oklab, var(--color-border), white 38%);
  color: var(--color-text);
}

.button {
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 12%);
  background: color-mix(in oklab, var(--color-surface), white 10%);
  color: var(--color-text);
  padding: 0.8rem 1rem;
  font-weight: 600;
  cursor: pointer;
  transition:
    transform var(--transition-fast) ease,
    border-color var(--transition-fast) ease,
    background var(--transition-fast) ease;
}

.button:hover {
  transform: translateY(-1px);
}

.button:disabled {
  cursor: not-allowed;
  opacity: 0.6;
  transform: none;
}

.state-copy {
  margin: 0;
}

.state-copy.error {
  color: var(--color-danger);
}

@media (max-width: 760px) {
  .section-card {
    padding: 1rem;
  }

  .room-chip {
    flex-direction: column;
  }
}
</style>
