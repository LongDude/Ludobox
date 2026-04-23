<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { UserApi } from '@/api/useUserApi'
import type { AdminEvent, AdminEventResource, GameServerResponse, RoomResponse } from '@/api/types'
import { useI18n } from '@/i18n'

type AdminEventVersions = Partial<Record<AdminEventResource, number>>

interface ServerBucket {
  server: GameServerResponse
  rooms: RoomResponse[]
  openCount: number
  inGameCount: number
  completedCount: number
}

const OVERVIEW_METADATA_RELOAD_DELAY_MS = 700
const SERVER_EVENT_STALE_MS = 60_000
const SERVER_STATUS_REFRESH_INTERVAL_MS = 5_000

const props = defineProps<{
  adminEventVersions?: AdminEventVersions
  adminEvents?: AdminEvent[]
}>()

const loading = ref(false)
const backgroundRefreshing = ref(false)
const errorMsg = ref('')
const servers = ref<GameServerResponse[]>([])
const rooms = ref<RoomResponse[]>([])
const statusClock = ref(Date.now())
const { locale, t } = useI18n()
let overviewReloadTimer: ReturnType<typeof setTimeout> | undefined
let serverStatusTimer: ReturnType<typeof setInterval> | undefined
let processedAdminEvents = 0

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
const hasOverviewData = computed(() => servers.value.length > 0 || rooms.value.length > 0)
const refreshDisabled = computed(() => loading.value || backgroundRefreshing.value)

onMounted(async () => {
  startServerStatusTimer()
  await loadOverview()
})

onBeforeUnmount(() => {
  clearOverviewReloadTimer()
  stopServerStatusTimer()
})

watch(
  () => props.adminEvents?.length ?? 0,
  (length) => {
    const events = props.adminEvents ?? []
    const nextEvents = events.slice(processedAdminEvents, length)
    processedAdminEvents = length

    for (const event of nextEvents) {
      applyAdminEvent(event)
    }
  },
)

watch(
  () => props.adminEvents,
  (events) => {
    if (!events?.length) {
      processedAdminEvents = 0
      return
    }

    if (processedAdminEvents > events.length) {
      processedAdminEvents = 0
      for (const event of events) {
        applyAdminEvent(event)
      }
      processedAdminEvents = events.length
    }
  },
)

function clearOverviewReloadTimer() {
  if (!overviewReloadTimer) return
  clearTimeout(overviewReloadTimer)
  overviewReloadTimer = undefined
}

function startServerStatusTimer() {
  if (serverStatusTimer) return

  serverStatusTimer = setInterval(() => {
    statusClock.value = Date.now()
  }, SERVER_STATUS_REFRESH_INTERVAL_MS)
}

function stopServerStatusTimer() {
  if (!serverStatusTimer) return
  clearInterval(serverStatusTimer)
  serverStatusTimer = undefined
}

function scheduleOverviewReload() {
  if (overviewReloadTimer) return

  overviewReloadTimer = setTimeout(() => {
    overviewReloadTimer = undefined
    void loadOverview({ silent: true })
  }, OVERVIEW_METADATA_RELOAD_DELAY_MS)
}

async function loadOverview(options: { silent?: boolean } = {}) {
  if (loading.value || backgroundRefreshing.value) {
    if (options.silent) scheduleOverviewReload()
    return
  }

  const silent = Boolean(options.silent)
  if (silent) {
    backgroundRefreshing.value = true
  } else {
    clearOverviewReloadTimer()
    loading.value = true
  }

  if (!silent || !hasOverviewData.value) {
    errorMsg.value = ''
  }

  try {
    const [allServers, allRooms] = await Promise.all([loadAllServers(), loadAllRooms()])

    servers.value = allServers
    rooms.value = allRooms
  } catch (error: any) {
    if (!silent || !hasOverviewData.value) {
      errorMsg.value = error?.message || t('admin.overviewSection.error.load')
    }
  } finally {
    if (silent) {
      backgroundRefreshing.value = false
    } else {
      loading.value = false
    }
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
  return normalizedServerStatus(server) === 'up'
}

function applyAdminEvent(event: AdminEvent) {
  if (event.resource === 'rooms') {
    applyRoomEvent(event)
    return
  }
  if (event.resource === 'servers') {
    applyServerEvent(event)
    return
  }
  if (event.resource === 'configs' || event.resource === 'games') {
    scheduleOverviewReload()
  }
}

function applyServerEvent(event: AdminEvent) {
  const server = normalizeServerEventData(event)
  if (!server) return

  if (event.action === 'delete' || server.archived_at) {
    servers.value = servers.value.filter((item) => item.server_id !== server.server_id)
    return
  }

  upsertById(servers.value, server, 'server_id')
}

function applyRoomEvent(event: AdminEvent) {
  const room = normalizeRoomEventData(event)
  if (!room) return

  if (event.action === 'delete' || room.archived_at) {
    rooms.value = rooms.value.filter((item) => item.room_id !== room.room_id)
    return
  }

  const existing = rooms.value.find((item) => item.room_id === room.room_id)
  upsertById(
    rooms.value,
    {
      ...existing,
      ...room,
      config: existing?.config ?? room.config ?? null,
      server_name: room.server_name ?? existing?.server_name ?? null,
    },
    'room_id',
  )
}

function normalizeServerEventData(event: AdminEvent): GameServerResponse | null {
  const data = event.data
  const serverId = Number(data?.server_id ?? event.id ?? 0)
  if (!serverId) return null

  return {
    server_id: serverId,
    instance_key: String(data?.instance_key ?? ''),
    redis_host: String(data?.redis_host ?? ''),
    status: String(data?.status ?? ''),
    started_at: stringOrNull(data?.started_at),
    last_heartbeat_at: stringOrNull(data?.last_heartbeat_at) ?? event.timestamp,
    archived_at: stringOrNull(data?.archived_at),
  }
}

function normalizeRoomEventData(event: AdminEvent): RoomResponse | null {
  const data = event.data
  const roomId = Number(data?.room_id ?? event.id ?? 0)
  if (!roomId) return null

  return {
    room_id: roomId,
    config_id: Number(data?.config_id ?? 0),
    server_id: Number(data?.server_id ?? 0),
    current_players: Number(data?.current_players ?? 0),
    status: normalizeRoomStatus(data?.status),
    archived_at: stringOrNull(data?.archived_at),
    config: null,
    server_name: stringOrNull(data?.server_name),
  }
}

function normalizeRoomStatus(value: unknown): RoomResponse['status'] {
  return value === 'in_game' || value === 'completed' ? value : 'open'
}

function stringOrNull(value: unknown) {
  return typeof value === 'string' && value ? value : null
}

function upsertById<T extends Record<K, number>, K extends keyof T>(items: T[], next: T, key: K) {
  const index = items.findIndex((item) => item[key] === next[key])
  if (index >= 0) {
    items.splice(index, 1, next)
  } else {
    items.push(next)
  }
}

function roomSummary(room: RoomResponse) {
  const config = room.config
  if (!config) return t('admin.roomsSection.configFallback', { id: room.config_id })
  return t('admin.overviewSection.roomSummary', {
    game: config.game?.name_game ?? t('admin.configsSection.gameLabel', { id: config.game_id }),
    capacity: config.capacity,
    players: room.current_players,
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
  if (shouldMarkServerDown(server, normalized)) return 'down'

  return normalized || 'unknown'
}

function shouldMarkServerDown(server: GameServerResponse, normalizedStatus?: string) {
  const status = normalizedStatus ?? server.status.trim().toLowerCase()
  if (status === 'maintenance' || status === 'archived' || status === 'down') return false

  const lastActivityAt = parseTimestamp(server.last_heartbeat_at ?? server.started_at)
  if (lastActivityAt === null) return status === 'up'

  return statusClock.value - lastActivityAt > SERVER_EVENT_STALE_MS
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

function parseTimestamp(value?: string | null) {
  if (!value) return null

  const parsed = new Date(value).getTime()
  return Number.isNaN(parsed) ? null : parsed
}
</script>

<template>
  <section class="section-card">
    <div class="section-header">
      <div>
        <h2>{{ t('admin.overviewSection.title') }}</h2>
        <p class="section-copy">{{ t('admin.overviewSection.description') }}</p>
      </div>
      <button class="button ghost" @click="loadOverview()" :disabled="refreshDisabled">
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

    <p v-if="loading && !hasOverviewData" class="state-copy">{{ t('admin.overviewSection.loading') }}</p>
    <p v-else-if="errorMsg && !hasOverviewData" class="state-copy error">{{ errorMsg }}</p>

    <div v-else class="server-grid" :class="{ refreshing: backgroundRefreshing }">
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
    radial-gradient(circle at top right, color-mix(in oklab, #0ea5e9, var(--color-surface) 92%), transparent 60%),
    linear-gradient(180deg, color-mix(in oklab, var(--color-surface), white 12%), var(--color-surface));
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

.server-grid.refreshing {
  opacity: 0.92;
  transition: opacity var(--transition-fast) ease;
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
