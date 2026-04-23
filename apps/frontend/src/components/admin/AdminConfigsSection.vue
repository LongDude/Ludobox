<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { UserApi } from '@/api/useUserApi'
import type {
  AdminEventResource,
  ConfigResponse,
  ConfigUpsertRequest,
  GameResponse,
} from '@/api/types'
import { useDeferredAdminReload } from '@/composables/useDeferredAdminReload'
import { useI18n } from '@/i18n'
import {
  distributionToInput,
  normalizeConfigDraft,
  parseDistributionInput,
  projectConfigEconomics,
  rebalanceDistribution,
  validateConfigDraft,
} from '@/utils/admin'

type EditorMode = 'create' | 'edit'
type AdminEventVersions = Partial<Record<AdminEventResource, number>>

const props = defineProps<{
  adminEventVersions?: AdminEventVersions
}>()

const loading = ref(false)
const saving = ref(false)
const errorMsg = ref('')
const successMsg = ref('')
const games = ref<GameResponse[]>([])
const configs = ref<ConfigResponse[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(12)
const { t } = useI18n()

const filters = reactive({
  gameId: '',
  capacity: '',
  boostMode: 'all' as 'all' | 'enabled' | 'disabled',
})

const mode = ref<EditorMode>('create')
const editingConfigId = ref<number | null>(null)
const gameById = computed(() => new Map(games.value.map((game) => [game.game_id, game])))

function makeDraft(): ConfigUpsertRequest {
  return {
    game_id: 1,
    capacity: 4,
    registration_price: 100,
    is_boost: true,
    boost_price: 30,
    boost_power: 20,
    number_winners: 1,
    winning_distribution: [100],
    commission: 20,
    time: 60,
    round_time: 15,
    next_round_delay: 15,
    min_users: 1,
  }
}

const form = reactive<ConfigUpsertRequest>(makeDraft())
const distributionInput = ref('100')

const formSnapshot = computed<ConfigUpsertRequest>(() => ({
  ...form,
  winning_distribution: parseDistributionInput(distributionInput.value),
}))

const issues = computed(() => validateConfigDraft(formSnapshot.value))
const blockingIssues = computed(() => issues.value.filter((issue) => issue.tone === 'error'))
const projection = computed(() => projectConfigEconomics(normalizeConfigDraft(formSnapshot.value)))
const editorTitle = computed(() =>
  mode.value === 'edit' && editingConfigId.value
    ? t('admin.configsSection.editTitle', { id: editingConfigId.value })
    : t('admin.configsSection.createTitle'),
)
const pageSummary = computed(() =>
  t('common.pageSummary', {
    page: page.value,
    pages: Math.max(1, Math.ceil(total.value / pageSize.value)),
    total: total.value,
    entity: t('admin.configs.entity'),
  }),
)

watch(
  () => form.number_winners,
  (count) => {
    const parsed = parseDistributionInput(distributionInput.value)
    if (parsed.length !== count) {
      distributionInput.value = distributionToInput(rebalanceDistribution(count, parsed))
    }
  },
)

watch(
  () => form.is_boost,
  (enabled) => {
    if (!enabled) {
      form.boost_price = 0
      form.boost_power = 0
    }
  },
)

onMounted(async () => {
  await Promise.all([loadConfigs(), loadAllGames()])
})

const scheduleConfigsReload = useDeferredAdminReload(loadConfigs, saving)
const scheduleGamesReload = useDeferredAdminReload(loadAllGames, saving)

watch(
  () => props.adminEventVersions?.configs,
  (version, previousVersion) => {
    if (version !== undefined && previousVersion !== undefined && version !== previousVersion) {
      scheduleConfigsReload()
    }
  },
)

watch(
  () => props.adminEventVersions?.games,
  (version, previousVersion) => {
    if (version !== undefined && previousVersion !== undefined && version !== previousVersion) {
      scheduleGamesReload()
    }
  },
)

function toRequest(config: ConfigResponse): ConfigUpsertRequest {
  return {
    game_id: config.game_id,
    capacity: config.capacity,
    registration_price: config.registration_price,
    is_boost: config.is_boost,
    boost_price: config.boost_price,
    boost_power: config.boost_power,
    number_winners: config.number_winners,
    winning_distribution: [...config.winning_distribution],
    commission: config.commission,
    time: config.time,
    round_time: config.round_time,
    next_round_delay: config.next_round_delay,
    min_users: config.min_users,
  }
}

function applyDraft(next: ConfigUpsertRequest, nextMode: EditorMode, nextConfigId: number | null) {
  const cloned = normalizeConfigDraft(next)
  Object.assign(form, cloned)
  distributionInput.value = distributionToInput(cloned.winning_distribution)
  mode.value = nextMode
  editingConfigId.value = nextConfigId
  successMsg.value = ''
  errorMsg.value = ''
}

function resetForm() {
  applyDraft(makeDraft(), 'create', null)
}

function editConfig(config: ConfigResponse) {
  applyDraft(toRequest(config), 'edit', config.config_id)
}

function duplicateConfig(config: ConfigResponse) {
  applyDraft(toRequest(config), 'create', null)
}

async function loadConfigs() {
  loading.value = true
  errorMsg.value = ''

  try {
    const response = await UserApi.listConfigs({
      page: page.value,
      page_size: pageSize.value,
      sort_field: 'config_id',
      sort_direction: 'desc',
      filters: [
        filters.gameId ? { field: 'game_id', operator: 'eq', value: filters.gameId } : undefined,
        filters.capacity
          ? { field: 'capacity', operator: 'eq', value: filters.capacity }
          : undefined,
        filters.boostMode === 'enabled'
          ? { field: 'is_boost', operator: 'eq', value: true }
          : undefined,
        filters.boostMode === 'disabled'
          ? { field: 'is_boost', operator: 'eq', value: false }
          : undefined,
      ].filter(Boolean) as NonNullable<Parameters<typeof UserApi.listConfigs>[0]>['filters'],
    })

    configs.value = response.items ?? []
    total.value = response.total ?? 0
    page.value = response.page ?? page.value
    pageSize.value = response.page_size ?? pageSize.value
  } catch (error: any) {
    errorMsg.value = error?.message || t('admin.configsSection.error.load')
  } finally {
    loading.value = false
  }
}

async function loadAllGames() {
  const collected: GameResponse[] = []
  let currentPage = 1
  let totalItems = 0

  do {
    const response = await UserApi.listGames({
      page: currentPage,
      page_size: 100,
      sort_field: 'game_id',
      sort_direction: 'asc',
    })

    totalItems = response.total ?? 0
    collected.push(...(response.items ?? []))
    currentPage += 1

    if (!response.items?.length) {
      break
    }
  } while (collected.length < totalItems && currentPage <= 20)

  games.value = collected
}

async function saveConfig() {
  successMsg.value = ''
  errorMsg.value = ''

  if (blockingIssues.value.length > 0) {
    errorMsg.value = t('admin.configsSection.error.validation')
    return
  }

  saving.value = true
  try {
    const payload = normalizeConfigDraft(formSnapshot.value)
    if (mode.value === 'edit' && editingConfigId.value) {
      const updated = await UserApi.updateConfig(editingConfigId.value, payload)
      applyDraft(toRequest(updated), 'edit', updated.config_id)
      successMsg.value = t('admin.configsSection.success.updated')
    } else {
      const created = await UserApi.createConfig(payload)
      applyDraft(toRequest(created), 'edit', created.config_id)
      successMsg.value = t('admin.configsSection.success.created')
    }

    await loadConfigs()
  } catch (error: any) {
    errorMsg.value = error?.message || t('admin.configsSection.error.save')
  } finally {
    saving.value = false
  }
}

async function archiveConfig(config: ConfigResponse) {
  const confirmed = window.confirm(
    t('admin.configsSection.confirmArchive', { id: config.config_id }),
  )
  if (!confirmed) return

  errorMsg.value = ''
  successMsg.value = ''

  try {
    await UserApi.deleteConfig(config.config_id)
    if (editingConfigId.value === config.config_id) {
      resetForm()
    }
    successMsg.value = t('admin.configsSection.success.archived', { id: config.config_id })
    await loadConfigs()
  } catch (error: any) {
    errorMsg.value = error?.message || t('admin.configsSection.error.archive')
  }
}

function applyFilters() {
  page.value = 1
  void loadConfigs()
}

function resetFilters() {
  filters.gameId = ''
  filters.capacity = ''
  filters.boostMode = 'all'
  page.value = 1
  void loadConfigs()
}

function prevPage() {
  if (page.value > 1) {
    page.value -= 1
    void loadConfigs()
  }
}

function nextPage() {
  const pages = Math.max(1, Math.ceil(total.value / pageSize.value))
  if (page.value < pages) {
    page.value += 1
    void loadConfigs()
  }
}

function gameLabel(config: ConfigResponse) {
  return (
    config.game?.name_game ??
    gameById.value.get(config.game_id)?.name_game ??
    t('admin.configsSection.gameLabel', { id: config.game_id })
  )
}

function boostLabel(config: ConfigResponse) {
  return config.is_boost
    ? t('admin.configsSection.boostSummary', {
        price: config.boost_price,
        power: config.boost_power,
      })
    : t('common.off')
}

function distributionLabel(value: number, index: number) {
  return t('admin.configsSection.distributionChip', {
    place: index + 1,
    value,
  })
}
</script>

<template>
  <section class="section-card">
    <div class="section-header">
      <div>
        <h2>{{ t('admin.configsSection.title') }}</h2>
        <p class="section-copy">{{ t('admin.configsSection.description') }}</p>
      </div>
      <div class="header-actions">
        <button class="button ghost" @click="resetForm">{{ t('admin.configsSection.newConfig') }}</button>
        <button class="button ghost" @click="loadConfigs" :disabled="loading">{{ t('common.refresh') }}</button>
      </div>
    </div>

    <div class="workspace">
      <div class="editor-card">
        <div class="editor-head">
          <div>
            <h3>{{ editorTitle }}</h3>
          </div>
          <span class="mode-pill">
            {{ mode === 'edit' ? t('admin.configsSection.revisionMode') : t('admin.configsSection.draftMode') }}
          </span>
        </div>

        <div class="form-grid">
          <label>
            <span>{{ t('admin.configsSection.fields.gameId') }}</span>
            <select v-if="games.length > 0" v-model.number="form.game_id" class="input">
              <option v-for="game in games" :key="game.game_id" :value="game.game_id">
                {{ game.name_game }}
              </option>
            </select>
            <input v-else v-model.number="form.game_id" class="input" type="number" min="1" />
          </label>
          <label>
            <span>{{ t('admin.configsSection.fields.capacity') }}</span>
            <input v-model.number="form.capacity" class="input" type="number" min="2" max="20" />
          </label>
          <label>
            <span>{{ t('admin.configsSection.fields.registrationPrice') }}</span>
            <input
              v-model.number="form.registration_price"
              class="input"
              type="number"
              min="0"
            />
          </label>
          <label>
            <span>{{ t('admin.configsSection.fields.roundTimer') }}</span>
            <input v-model.number="form.time" class="input" type="number" min="1" />
          </label>
          <label>
            <span>{{ t('admin.configsSection.fields.activeRoundTimer') }}</span>
            <input v-model.number="form.round_time" class="input" type="number" min="1" />
          </label>
          <label>
            <span>{{ t('admin.configsSection.fields.nextRoundDelay') }}</span>
            <input v-model.number="form.next_round_delay" class="input" type="number" min="0" />
          </label>
          <label>
            <span>{{ t('admin.configsSection.fields.minimumUsers') }}</span>
            <input v-model.number="form.min_users" class="input" type="number" min="1" />
          </label>
          <label>
            <span>{{ t('admin.configsSection.fields.commission') }}</span>
            <input
              v-model.number="form.commission"
              class="input"
              type="number"
              min="0"
              max="100"
            />
          </label>
          <label>
            <span>{{ t('admin.configsSection.fields.numberWinners') }}</span>
            <input
              v-model.number="form.number_winners"
              class="input"
              type="number"
              min="1"
              max="20"
              disabled
            />
          </label>
          <label class="boost-toggle">
            <span>{{ t('admin.configsSection.fields.boostEnabled') }}</span>
            <input v-model="form.is_boost" type="checkbox" />
          </label>
          <label>
            <span>{{ t('admin.configsSection.fields.boostPrice') }}</span>
            <input
              v-model.number="form.boost_price"
              class="input"
              type="number"
              min="0"
              :disabled="!form.is_boost"
            />
          </label>
          <label>
            <span>{{ t('admin.configsSection.fields.boostPower') }}</span>
            <input
              v-model.number="form.boost_power"
              class="input"
              type="number"
              min="0"
              max="100"
              :disabled="!form.is_boost"
            />
          </label>
          <label class="distribution-field">
            <span>{{ t('admin.configsSection.fields.winningDistribution') }}</span>
            <input
              v-model="distributionInput"
              class="input"
              type="text"
              :placeholder="t('admin.configsSection.distributionPlaceholder')"
              disabled
            />
            <small>{{ t('admin.configsSection.distributionHelp') }}</small>
          </label>
        </div>

        <div class="issues">
          <div v-if="issues.length === 0" class="issue good">{{ t('admin.configsSection.cleanChecks') }}</div>
          <div
            v-for="issue in issues"
            :key="`${issue.tone}-${issue.message}`"
            class="issue"
            :class="issue.tone"
          >
            {{ issue.message }}
          </div>
        </div>

        <div class="metric-grid">
          <article v-for="metric in projection.metrics" :key="metric.label" class="metric-card">
            <span class="metric-label">{{ metric.label }}</span>
            <strong>{{ metric.value }}</strong>
            <small>{{ metric.hint }}</small>
          </article>
        </div>

        <div class="winner-table">
          <div class="winner-head">
            <span>{{ t('admin.configsSection.metrics.winner') }}</span>
            <span>{{ t('admin.configsSection.metrics.share') }}</span>
            <span>{{ t('admin.configsSection.metrics.projectedPayout') }}</span>
          </div>
          <div v-for="winner in projection.winners" :key="winner.place" class="winner-row">
            <span>#{{ winner.place }}</span>
            <span>{{ winner.percent }}%</span>
            <span>{{ winner.amount }}</span>
          </div>
        </div>

        <p v-if="errorMsg" class="state-copy error">{{ errorMsg }}</p>
        <p v-if="successMsg" class="state-copy success">{{ successMsg }}</p>

        <div class="editor-actions">
          <button class="button ghost" @click="resetForm">{{ t('admin.configsSection.clearForm') }}</button>
          <button class="button primary" :disabled="saving" @click="saveConfig">
            {{
              saving
                ? t('common.saving')
                : mode === 'edit'
                  ? t('admin.configsSection.saveRevision')
                  : t('admin.configsSection.createConfig')
            }}
          </button>
        </div>
      </div>

      <div class="list-card">
        <div class="list-head">
          <div>
            <h3>{{ t('admin.configsSection.listTitle') }}</h3>
            <p class="muted">{{ t('admin.configsSection.listHint') }}</p>
          </div>
        </div>

        <div class="toolbar">
          <div class="toolbar-grid">
            <select v-if="games.length > 0" v-model="filters.gameId" class="input">
              <option value="">{{ t('admin.configsSection.filters.gameAny') }}</option>
              <option v-for="game in games" :key="game.game_id" :value="String(game.game_id)">
                {{ game.name_game }}
              </option>
            </select>
            <input
              v-else
              v-model="filters.gameId"
              class="input"
              type="number"
              min="1"
              :placeholder="t('admin.configsSection.filters.gameId')"
            />
            <input
              v-model="filters.capacity"
              class="input"
              type="number"
              min="2"
              :placeholder="t('admin.configsSection.filters.capacity')"
            />
            <select v-model="filters.boostMode" class="input">
              <option value="all">{{ t('admin.configsSection.filters.boostAny') }}</option>
              <option value="enabled">{{ t('admin.configsSection.filters.boostEnabled') }}</option>
              <option value="disabled">{{ t('admin.configsSection.filters.boostDisabled') }}</option>
            </select>
          </div>
          <div class="toolbar-actions">
            <button class="button primary" @click="applyFilters">{{ t('common.apply') }}</button>
            <button class="button ghost" @click="resetFilters">{{ t('common.reset') }}</button>
          </div>
        </div>

        <p v-if="loading" class="state-copy">{{ t('admin.configsSection.loading') }}</p>

        <div v-else class="config-list">
          <article
            v-for="config in configs"
            :key="config.config_id"
            class="config-card"
            :class="{ selected: editingConfigId === config.config_id }"
          >
            <div class="config-topline">
              <div>
                <strong>#{{ config.config_id }}</strong>
                <p class="muted">{{ gameLabel(config) }}</p>
              </div>
              <span class="badge">{{ t('admin.configsSection.seats', { count: config.capacity }) }}</span>
            </div>

            <dl class="config-meta">
              <div>
                <dt>{{ t('admin.configsSection.meta.entry') }}</dt>
                <dd>{{ config.registration_price }}</dd>
              </div>
              <div>
                <dt>{{ t('admin.configsSection.meta.winners') }}</dt>
                <dd>{{ config.number_winners }}</dd>
              </div>
              <div>
                <dt>{{ t('admin.configsSection.meta.commission') }}</dt>
                <dd>{{ config.commission }}%</dd>
              </div>
              <div>
                <dt>{{ t('admin.configsSection.meta.timer') }}</dt>
                <dd>{{ config.time }}s</dd>
              </div>
              <div>
                <dt>{{ t('admin.configsSection.meta.activeTimer') }}</dt>
                <dd>{{ config.round_time }}s</dd>
              </div>
              <div>
                <dt>{{ t('admin.configsSection.meta.nextRoundDelay') }}</dt>
                <dd>{{ config.next_round_delay }}s</dd>
              </div>
              <div>
                <dt>{{ t('admin.configsSection.meta.minUsers') }}</dt>
                <dd>{{ config.min_users }}</dd>
              </div>
              <div>
                <dt>{{ t('admin.configsSection.meta.boost') }}</dt>
                <dd>{{ boostLabel(config) }}</dd>
              </div>
            </dl>

            <div class="chip-row">
              <span class="chip" v-for="(value, index) in config.winning_distribution" :key="index">
                {{ distributionLabel(value, index) }}
              </span>
            </div>

            <div class="config-actions">
              <button class="button ghost" @click="editConfig(config)">{{ t('common.edit') }}</button>
              <button class="button ghost" @click="duplicateConfig(config)">{{ t('common.duplicate') }}</button>
              <button class="button danger" @click="archiveConfig(config)">{{ t('common.archive') }}</button>
            </div>
          </article>

          <p v-if="configs.length === 0" class="state-copy">
            {{ t('admin.configsSection.noMatches') }}
          </p>
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
      </div>
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
    radial-gradient(circle at top right, color-mix(in oklab, #0ea5e9, var(--color-surface) 92%), transparent 60%),
    linear-gradient(180deg, color-mix(in oklab, var(--color-surface), white 12%), var(--color-surface));
  box-shadow: var(--shadow-md);
}

.section-header,
.editor-head,
.list-head,
.config-topline,
.editor-actions,
.pager,
.header-actions {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: flex-start;
  flex-wrap: wrap;
}

.section-header h2,
.editor-head h3,
.list-head h3 {
  margin: 0;
}

.eyebrow {
  margin: 0 0 0.35rem;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  font-size: 0.72rem;
  color: #b45309;
}

.section-copy,
.muted,
.state-copy {
  color: var(--color-muted);
}

.section-copy {
  margin: 0.45rem 0 0;
  max-width: 46rem;
}

.workspace {
  display: grid;
  grid-template-columns: minmax(22rem, 32rem) minmax(0, 1fr);
  gap: 1rem;
}

.editor-card,
.list-card,
.metric-card,
.config-card {
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  border-radius: 1.35rem;
  background: color-mix(in oklab, var(--color-surface), white 12%);
}

.editor-card,
.list-card {
  display: grid;
  gap: 1rem;
  padding: 1.15rem;
  align-content: start;
}

.mode-pill,
.badge,
.chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  font-size: 0.8rem;
}

.mode-pill {
  padding: 0.35rem 0.85rem;
  background: color-mix(in oklab, #0f766e, white 82%);
  color: #115e59;
}

.form-grid,
.metric-grid {
  display: grid;
  gap: 0.8rem;
}

.form-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.form-grid label,
.distribution-field {
  display: grid;
  gap: 0.35rem;
}

.distribution-field {
  grid-column: 1 / -1;
}

.boost-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.85rem 1rem;
  border: 1px dashed color-mix(in oklab, var(--color-border), transparent 8%);
  border-radius: 1rem;
}

.input {
  width: 100%;
  padding: 0.75rem 0.9rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
  border-radius: 0.95rem;
  background: color-mix(in oklab, var(--color-surface), white 18%);
  color: var(--color-text);
}

.toolbar {
  display: grid;
  gap: 0.85rem;
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

.issues {
  display: grid;
  gap: 0.5rem;
}

.issue {
  padding: 0.8rem 0.95rem;
  border-radius: 1rem;
  font-size: 0.95rem;
}

.issue.good {
  background: color-mix(in oklab, var(--color-success), white 82%);
  color: #166534;
}

.issue.warning {
  background: color-mix(in oklab, var(--color-warning), white 82%);
  color: #9a3412;
}

.issue.error {
  background: color-mix(in oklab, var(--color-danger), white 84%);
  color: #991b1b;
}

.metric-grid {
  grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr));
}

.metric-card {
  display: grid;
  gap: 0.35rem;
  padding: 0.95rem;
}

.metric-label {
  color: var(--color-muted);
  font-size: 0.85rem;
}

.winner-table {
  display: grid;
  gap: 0.55rem;
}

.winner-head,
.winner-row {
  display: grid;
  grid-template-columns: 0.75fr 0.75fr 1fr;
  gap: 0.75rem;
  align-items: center;
}

.winner-head {
  padding: 0 0.15rem;
  color: var(--color-muted);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 0.78rem;
}

.winner-row {
  padding: 0.8rem 0.95rem;
  border-radius: 1rem;
  background: color-mix(in oklab, var(--color-surface), white 8%);
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
  background: linear-gradient(135deg, #d97706, #b45309);
  color: #fff7ed;
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

.toolbar-actions .button,
.editor-actions .button,
.header-actions .button,
.config-actions .button,
.pager .button {
  min-width: 8.5rem;
}

.config-list {
  display: grid;
  gap: 0.9rem;
}

.config-card {
  display: grid;
  gap: 0.85rem;
  padding: 1rem;
}

.config-card.selected {
  border-color: color-mix(in oklab, #d97706, white 42%);
  box-shadow: 0 0 0 1px color-mix(in oklab, #d97706, white 62%);
}

.badge {
  padding: 0.35rem 0.75rem;
  background: color-mix(in oklab, #0f766e, white 84%);
  color: #115e59;
}

.config-meta {
  display: grid;
  gap: 0.75rem;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin: 0;
}

.config-meta div {
  display: grid;
  gap: 0.15rem;
}

.config-meta dt {
  font-size: 0.8rem;
  color: var(--color-muted);
}

.config-meta dd {
  margin: 0;
  font-weight: 600;
}

.chip-row,
.config-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.chip {
  padding: 0.3rem 0.7rem;
  background: color-mix(in oklab, var(--color-surface), white 6%);
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
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

@media (max-width: 1180px) {
  .workspace {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .section-card {
    padding: 1rem;
  }

  .form-grid,
  .config-meta {
    grid-template-columns: 1fr;
  }

  .toolbar-actions {
    justify-content: stretch;
  }

  .toolbar-actions .button,
  .editor-actions .button,
  .header-actions .button,
  .config-actions .button,
  .pager .button {
    width: 100%;
  }

  .winner-head,
  .winner-row {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
</style>
