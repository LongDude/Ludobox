<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { UserApi } from '@/api/useUserApi'
import type { AdminEventResource, ConfigResponse, RoomResponse, RoomStatus } from '@/api/types'
import { useDeferredAdminReload } from '@/composables/useDeferredAdminReload'
import { useI18n } from '@/i18n'

type PlayerSortDirection = '' | 'asc' | 'desc'
type AdminEventVersions = Partial<Record<AdminEventResource, number>>

const props = defineProps<{
  adminEventVersions?: AdminEventVersions
}>()

const loading = ref(false)
const creating = ref(false)
const errorMsg = ref('')
const successMsg = ref('')
const rooms = ref<RoomResponse[]>([])
const configs = ref<ConfigResponse[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(12)
const { t } = useI18n()

const filters = reactive({
  status: '' as '' | RoomStatus,
  serverName: '',
  gameName: '',
  playersOrder: '' as PlayerSortDirection,
})

const createConfigId = ref('')
const editingRoomId = ref<number | null>(null)
const serverDrafts = reactive<Record<number, string>>({})
const savingRoomId = ref<number | null>(null)
const busy = computed(() => creating.value || savingRoomId.value !== null)

const configById = computed(() => {
  return new Map(configs.value.map((config) => [config.config_id, config]))
})
const pageSummary = computed(() =>
  t('common.pageSummary', {
    page: page.value,
    pages: Math.max(1, Math.ceil(total.value / pageSize.value)),
    total: total.value,
    entity: t('admin.rooms.entity'),
  }),
)

onMounted(async () => {
  await Promise.all([loadRooms(), loadConfigOptions()])
})

const scheduleRoomsReload = useDeferredAdminReload(loadRooms, busy)
const scheduleConfigOptionsReload = useDeferredAdminReload(loadConfigOptions, busy)

watch(
  () => props.adminEventVersions?.rooms,
  (version, previousVersion) => {
    if (version !== undefined && previousVersion !== undefined && version !== previousVersion) {
      scheduleRoomsReload()
    }
  },
)

watch(
  () => [props.adminEventVersions?.configs, props.adminEventVersions?.games] as const,
  ([configVersion, gameVersion], [previousConfigVersion, previousGameVersion]) => {
    const configsChanged =
      configVersion !== undefined &&
      previousConfigVersion !== undefined &&
      configVersion !== previousConfigVersion
    const gamesChanged =
      gameVersion !== undefined &&
      previousGameVersion !== undefined &&
      gameVersion !== previousGameVersion

    if (configsChanged || gamesChanged) {
      scheduleConfigOptionsReload()
      scheduleRoomsReload()
    }
  },
)

async function loadConfigOptions() {
  try {
    const response = await UserApi.listConfigs({
      page: 1,
      page_size: 100,
      sort_field: 'config_id',
      sort_direction: 'desc',
    })
    configs.value = response.items ?? []
  } catch {
    // Rooms stay manageable by id even when config metadata is unavailable.
  }
}

async function loadRooms() {
  loading.value = true
  errorMsg.value = ''

  try {
    const response = await UserApi.listRooms({
      page: page.value,
      page_size: pageSize.value,
      sort_field: filters.playersOrder ? 'current_players' : 'room_id',
      sort_direction: filters.playersOrder || 'desc',
      filters: [
        filters.status ? { field: 'status', operator: 'eq', value: filters.status } : undefined,
        filters.serverName
          ? { field: 'server_name', operator: 'like', value: filters.serverName }
          : undefined,
        filters.gameName
          ? { field: 'config_game_name', operator: 'like', value: filters.gameName }
          : undefined,
      ].filter(Boolean) as NonNullable<Parameters<typeof UserApi.listRooms>[0]>['filters'],
    })

    rooms.value = response.items ?? []
    total.value = response.total ?? 0
    page.value = response.page ?? page.value
    pageSize.value = response.page_size ?? pageSize.value
  } catch (error: any) {
    errorMsg.value = error?.message || t('admin.roomsSection.error.load')
  } finally {
    loading.value = false
  }
}

async function createRoom() {
  if (!createConfigId.value) {
    errorMsg.value = t('admin.roomsSection.error.selectConfig')
    return
  }

  errorMsg.value = ''
  successMsg.value = ''
  creating.value = true
  try {
    const room = await UserApi.createRoom({ config_id: Number(createConfigId.value) })
    successMsg.value = t('admin.roomsSection.success.created', { id: room.room_id })
    await loadRooms()
  } catch (error: any) {
    errorMsg.value = error?.message || t('admin.roomsSection.error.create')
  } finally {
    creating.value = false
  }
}

function startServerEdit(room: RoomResponse) {
  editingRoomId.value = room.room_id
  serverDrafts[room.room_id] = String(room.server_id)
}

function cancelServerEdit(roomId: number) {
  editingRoomId.value = null
  delete serverDrafts[roomId]
}

async function saveServer(room: RoomResponse) {
  const nextServerId = Number(serverDrafts[room.room_id])
  if (!Number.isInteger(nextServerId) || nextServerId <= 0) {
    errorMsg.value = t('admin.roomsSection.error.serverPositive')
    return
  }

  errorMsg.value = ''
  successMsg.value = ''
  savingRoomId.value = room.room_id
  try {
    await UserApi.updateRoom(room.room_id, { server_id: nextServerId })
    successMsg.value = t('admin.roomsSection.success.moved', {
      id: room.room_id,
      serverId: nextServerId,
    })
    editingRoomId.value = null
    await loadRooms()
  } catch (error: any) {
    errorMsg.value = error?.message || t('admin.roomsSection.error.update')
  } finally {
    savingRoomId.value = null
  }
}

async function archiveRoom(room: RoomResponse) {
  const confirmed = window.confirm(t('admin.roomsSection.confirmArchive', { id: room.room_id }))
  if (!confirmed) return

  errorMsg.value = ''
  successMsg.value = ''

  try {
    await UserApi.deleteRoom(room.room_id)
    successMsg.value = t('admin.roomsSection.success.archived', { id: room.room_id })
    if (editingRoomId.value === room.room_id) {
      cancelServerEdit(room.room_id)
    }
    await loadRooms()
  } catch (error: any) {
    errorMsg.value = error?.message || t('admin.roomsSection.error.archive')
  }
}

function applyFilters() {
  page.value = 1
  void loadRooms()
}

function resetFilters() {
  filters.status = ''
  filters.serverName = ''
  filters.gameName = ''
  filters.playersOrder = ''
  page.value = 1
  void loadRooms()
}

function prevPage() {
  if (page.value > 1) {
    page.value -= 1
    void loadRooms()
  }
}

function nextPage() {
  const pages = Math.max(1, Math.ceil(total.value / pageSize.value))
  if (page.value < pages) {
    page.value += 1
    void loadRooms()
  }
}

function statusVariant(status: RoomStatus) {
  if (status === 'open') return 'good'
  if (status === 'in_game') return 'accent'
  return 'neutral'
}

function statusLabel(status: RoomStatus) {
  if (status === 'open') return t('admin.status.open')
  if (status === 'in_game') return t('admin.status.inGame')
  return t('admin.status.completed')
}

function configLabel(configId: number) {
  const config = configById.value.get(configId)
  if (!config) return t('admin.roomsSection.configFallback', { id: configId })
  return t('admin.roomsSection.configSummary', {
    id: config.config_id,
    game: config.game?.name_game ?? t('admin.configsSection.gameLabel', { id: config.game_id }),
    capacity: config.capacity,
  })
}

function roomConfigLabel(room: RoomResponse) {
  const config = room.config ?? configById.value.get(room.config_id)
  if (!config) return t('admin.roomsSection.configFallback', { id: room.config_id })
  return t('admin.roomsSection.configSummary', {
    id: config.config_id,
    game: config.game?.name_game ?? t('admin.configsSection.gameLabel', { id: config.game_id }),
    capacity: config.capacity,
  })
}

function serverLabel(room: RoomResponse) {
  return room.server_name?.trim() || t('admin.roomsSection.serverFallback', { id: room.server_id })
}

function playersLabel(count: number) {
  return t('admin.roomsSection.playersCount', { count })
}
</script>

<template>
  <section class="section-card">
    <div class="section-header">
      <div>
        <p class="eyebrow">{{ t('admin.roomsSection.eyebrow') }}</p>
        <h2>{{ t('admin.roomsSection.title') }}</h2>
        <p class="section-copy">{{ t('admin.roomsSection.description') }}</p>
      </div>
      <button class="button ghost" @click="loadRooms" :disabled="loading">{{ t('common.refresh') }}</button>
    </div>

    <div class="create-panel">
      <div>
        <h3>{{ t('admin.roomsSection.createTitle') }}</h3>
        <p class="muted">{{ t('admin.roomsSection.createDescription') }}</p>
      </div>
      <div class="create-controls">
        <select v-model="createConfigId" class="input">
          <option value="">{{ t('admin.roomsSection.chooseConfig') }}</option>
          <option v-for="config in configs" :key="config.config_id" :value="config.config_id">
            {{ configLabel(config.config_id) }}
          </option>
        </select>
        <button class="button primary" :disabled="creating" @click="createRoom">
          {{ creating ? t('admin.roomsSection.creating') : t('admin.roomsSection.createRoom') }}
        </button>
      </div>
    </div>

    <div class="toolbar">
      <div class="toolbar-grid">
        <select v-model="filters.status" class="input">
          <option value="">{{ t('admin.roomsSection.filters.statusAny') }}</option>
          <option value="open">{{ t('admin.status.open') }}</option>
          <option value="in_game">{{ t('admin.status.inGame') }}</option>
          <option value="completed">{{ t('admin.status.completed') }}</option>
        </select>
        <input
          v-model="filters.serverName"
          class="input"
          type="text"
          :placeholder="t('admin.roomsSection.filters.serverName')"
        />
        <input
          v-model="filters.gameName"
          class="input"
          type="text"
          :placeholder="t('admin.roomsSection.filters.gameName')"
        />
        <select v-model="filters.playersOrder" class="input">
          <option value="">{{ t('admin.roomsSection.filters.playersOrderDefault') }}</option>
          <option value="desc">{{ t('admin.roomsSection.filters.playersOrderDesc') }}</option>
          <option value="asc">{{ t('admin.roomsSection.filters.playersOrderAsc') }}</option>
        </select>
      </div>
      <div class="toolbar-actions">
        <button class="button primary" @click="applyFilters">{{ t('common.apply') }}</button>
        <button class="button ghost" @click="resetFilters">{{ t('common.reset') }}</button>
      </div>
    </div>

    <p v-if="errorMsg" class="state-copy error">{{ errorMsg }}</p>
    <p v-if="successMsg" class="state-copy success">{{ successMsg }}</p>
    <p v-if="loading" class="state-copy">{{ t('admin.roomsSection.loading') }}</p>

    <div v-else class="room-list">
      <article v-for="room in rooms" :key="room.room_id" class="room-card">
        <div class="room-head">
          <div>
            <strong>{{ t('admin.roomsSection.roomTitle', { id: room.room_id }) }}</strong>
            <p class="muted">{{ roomConfigLabel(room) }}</p>
          </div>
          <span class="status-pill" :class="statusVariant(room.status)">
            {{ statusLabel(room.status) }}
          </span>
        </div>

        <dl class="room-meta">
          <div>
            <dt>{{ t('admin.roomsSection.meta.config') }}</dt>
            <dd>{{ roomConfigLabel(room) }}</dd>
          </div>
          <div>
            <dt>{{ t('admin.roomsSection.meta.server') }}</dt>
            <dd v-if="editingRoomId !== room.room_id">{{ serverLabel(room) }}</dd>
            <div v-else class="inline-edit">
              <input
                v-model="serverDrafts[room.room_id]"
                class="input"
                type="number"
                min="1"
                :placeholder="t('admin.roomsSection.serverInputPlaceholder')"
              />
            </div>
          </div>
          <div>
            <dt>{{ t('admin.roomsSection.meta.status') }}</dt>
            <dd>{{ statusLabel(room.status) }}</dd>
          </div>
          <div>
            <dt>{{ t('admin.roomsSection.meta.players') }}</dt>
            <dd>{{ playersLabel(room.current_players) }}</dd>
          </div>
        </dl>

        <div class="actions">
          <button
            v-if="editingRoomId !== room.room_id"
            class="button ghost"
            @click="startServerEdit(room)"
          >
            {{ t('admin.roomsSection.changeServer') }}
          </button>
          <template v-else>
            <button class="button ghost" @click="cancelServerEdit(room.room_id)">{{ t('common.cancel') }}</button>
            <button
              class="button primary"
              :disabled="savingRoomId === room.room_id"
              @click="saveServer(room)"
            >
              {{
                savingRoomId === room.room_id
                  ? t('common.saving')
                  : t('admin.roomsSection.saveServer')
              }}
            </button>
          </template>
          <button class="button danger" @click="archiveRoom(room)">{{ t('common.archive') }}</button>
        </div>
      </article>

      <p v-if="rooms.length === 0" class="state-copy">{{ t('admin.roomsSection.noMatches') }}</p>
    </div>

    <div v-if="total > 0" class="pager">
      <button class="button ghost" :disabled="page <= 1" @click="prevPage">{{ t('common.prev') }}</button>
      <span class="pager-copy">{{ pageSummary }}</span>
      <button
        class="button ghost"
        :disabled="page >= Math.max(1, Math.ceil(total / pageSize))"
        @click="nextPage"
      >
        {{ t('common.next') }}
      </button>
    </div>
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
    radial-gradient(circle at top left, color-mix(in oklab, #0f766e, white 84%), transparent 26%),
    linear-gradient(180deg, color-mix(in oklab, var(--color-surface), white 22%), var(--color-surface));
  box-shadow: var(--shadow-md);
}

.section-header,
.create-panel,
.create-controls,
.room-head,
.actions,
.pager {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: flex-start;
  flex-wrap: wrap;
}

.section-header h2,
.create-panel h3 {
  margin: 0;
}

.eyebrow {
  margin: 0 0 0.35rem;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  font-size: 0.72rem;
  color: #0f766e;
}

.section-copy,
.muted,
.state-copy {
  color: var(--color-muted);
}

.section-copy {
  margin: 0.45rem 0 0;
  max-width: 48rem;
}

.create-panel,
.room-card {
  padding: 1rem 1.1rem;
  border-radius: 1.3rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
  background: color-mix(in oklab, var(--color-surface), white 12%);
}

.create-panel {
  align-items: center;
}

.create-controls {
  width: min(32rem, 100%);
  align-items: stretch;
  flex-wrap: wrap;
}

.create-controls .input {
  flex: 1;
}

.toolbar {
  display: grid;
  gap: 0.8rem;
  padding: 0.9rem;
  border-radius: 1.15rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  background: color-mix(in oklab, var(--color-surface), white 6%);
}

.toolbar-grid {
  display: grid;
  gap: 0.8rem;
  grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr));
}

.toolbar-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.room-list {
  display: grid;
  gap: 0.9rem;
}

.room-card {
  display: grid;
  gap: 0.9rem;
}

.room-meta {
  display: grid;
  gap: 0.75rem;
  grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr));
  margin: 0;
}

.room-meta div {
  display: grid;
  gap: 0.15rem;
}

.room-meta dt {
  font-size: 0.8rem;
  color: var(--color-muted);
}

.room-meta dd {
  margin: 0;
  font-weight: 600;
}

.inline-edit {
  display: flex;
}

.input {
  width: 100%;
  padding: 0.75rem 0.9rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
  border-radius: 0.95rem;
  background: color-mix(in oklab, var(--color-surface), white 18%);
  color: var(--color-text);
}

.button {
  width: auto;
  border: 1px solid transparent;
  border-radius: 999px;
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

.button.primary {
  background: linear-gradient(135deg, #0f766e, #155e75);
  color: #ecfeff;
}

.button.ghost {
  border-color: color-mix(in oklab, var(--color-border), transparent 12%);
  background: color-mix(in oklab, var(--color-surface), white 10%);
  color: var(--color-text);
}

.button.danger {
  border-color: color-mix(in oklab, var(--color-danger), white 40%);
  background: color-mix(in oklab, var(--color-danger), white 86%);
  color: #991b1b;
}

.button:disabled {
  cursor: not-allowed;
  opacity: 0.6;
  transform: none;
}

.create-controls .button,
.toolbar-actions .button,
.actions .button,
.pager .button {
  min-width: 8.5rem;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.35rem 0.8rem;
  border-radius: 999px;
  text-transform: capitalize;
  font-size: 0.84rem;
}

.status-pill.good {
  background: color-mix(in oklab, var(--color-success), white 80%);
  color: #166534;
}

.status-pill.accent {
  background: color-mix(in oklab, #0ea5e9, white 80%);
  color: #0c4a6e;
}

.status-pill.neutral {
  background: color-mix(in oklab, var(--color-border), white 40%);
  color: var(--color-text);
}

.state-copy {
  margin: 0;
}

.state-copy.error {
  color: var(--color-danger);
}

.state-copy.success {
  color: var(--color-success);
}

.pager-copy {
  color: var(--color-muted);
}

@media (max-width: 760px) {
  .section-card {
    padding: 1rem;
  }

  .room-meta {
    grid-template-columns: 1fr;
  }

  .toolbar-actions {
    justify-content: stretch;
  }

  .create-controls .button,
  .toolbar-actions .button,
  .actions .button,
  .pager .button {
    width: 100%;
  }
}
</style>
