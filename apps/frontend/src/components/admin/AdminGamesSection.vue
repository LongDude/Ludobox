<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { UserApi } from '@/api/useUserApi'
import type { GameResponse, GameUpsertRequest } from '@/api/types'
import { useI18n } from '@/i18n'

type EditorMode = 'create' | 'edit'

const loading = ref(false)
const saving = ref(false)
const errorMsg = ref('')
const successMsg = ref('')
const games = ref<GameResponse[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(12)
const mode = ref<EditorMode>('create')
const editingGameId = ref<number | null>(null)
const { t } = useI18n()

const form = reactive<GameUpsertRequest>({
  name_game: '',
})

const normalizedName = computed(() => form.name_game.trim())
const editorTitle = computed(() =>
  mode.value === 'edit' && editingGameId.value
    ? t('admin.gamesSection.editTitle', { id: editingGameId.value })
    : t('admin.gamesSection.createTitle'),
)
const pageSummary = computed(() =>
  t('common.pageSummary', {
    page: page.value,
    pages: Math.max(1, Math.ceil(total.value / pageSize.value)),
    total: total.value,
    entity: t('admin.games.entity'),
  }),
)

onMounted(async () => {
  await loadGames()
})

function resetForm() {
  form.name_game = ''
  mode.value = 'create'
  editingGameId.value = null
  errorMsg.value = ''
  successMsg.value = ''
}

function editGame(game: GameResponse) {
  form.name_game = game.name_game
  mode.value = 'edit'
  editingGameId.value = game.game_id
  errorMsg.value = ''
  successMsg.value = ''
}

async function loadGames() {
  loading.value = true
  errorMsg.value = ''

  try {
    const response = await UserApi.listGames({
      page: page.value,
      page_size: pageSize.value,
      sort_field: 'game_id',
      sort_direction: 'desc',
    })

    games.value = response.items ?? []
    total.value = response.total ?? 0
    page.value = response.page ?? page.value
    pageSize.value = response.page_size ?? pageSize.value
  } catch (error: any) {
    errorMsg.value = error?.message || t('admin.gamesSection.error.load')
  } finally {
    loading.value = false
  }
}

async function saveGame() {
  errorMsg.value = ''
  successMsg.value = ''

  if (!normalizedName.value) {
    errorMsg.value = t('admin.gamesSection.error.validation')
    return
  }

  saving.value = true
  try {
    const payload: GameUpsertRequest = {
      name_game: normalizedName.value,
    }

    if (mode.value === 'edit' && editingGameId.value) {
      const updated = await UserApi.updateGame(editingGameId.value, payload)
      form.name_game = updated.name_game
      successMsg.value = t('admin.gamesSection.success.updated')
    } else {
      const created = await UserApi.createGame(payload)
      mode.value = 'edit'
      editingGameId.value = created.game_id
      form.name_game = created.name_game
      successMsg.value = t('admin.gamesSection.success.created')
    }

    await loadGames()
  } catch (error: any) {
    errorMsg.value = error?.message || t('admin.gamesSection.error.save')
  } finally {
    saving.value = false
  }
}

async function archiveGame(game: GameResponse) {
  const confirmed = window.confirm(t('admin.gamesSection.confirmArchive', { id: game.game_id }))
  if (!confirmed) return

  errorMsg.value = ''
  successMsg.value = ''

  try {
    await UserApi.deleteGame(game.game_id)
    if (editingGameId.value === game.game_id) {
      resetForm()
    }
    successMsg.value = t('admin.gamesSection.success.archived', { id: game.game_id })
    await loadGames()
  } catch (error: any) {
    errorMsg.value = error?.message || t('admin.gamesSection.error.archive')
  }
}

function prevPage() {
  if (page.value > 1) {
    page.value -= 1
    void loadGames()
  }
}

function nextPage() {
  const pages = Math.max(1, Math.ceil(total.value / pageSize.value))
  if (page.value < pages) {
    page.value += 1
    void loadGames()
  }
}
</script>

<template>
  <section class="section-card">
    <div class="section-header">
      <div>
        <p class="eyebrow">{{ t('admin.gamesSection.eyebrow') }}</p>
        <h2>{{ t('admin.gamesSection.title') }}</h2>
        <p class="section-copy">{{ t('admin.gamesSection.description') }}</p>
      </div>
      <div class="header-actions">
        <button class="button ghost" @click="resetForm">{{ t('admin.gamesSection.newGame') }}</button>
        <button class="button ghost" @click="loadGames" :disabled="loading">{{ t('common.refresh') }}</button>
      </div>
    </div>

    <div class="workspace">
      <div class="editor-card">
        <div class="editor-head">
          <div>
            <h3>{{ editorTitle }}</h3>
            <p class="muted">{{ t('admin.gamesSection.editorHint') }}</p>
          </div>
          <span class="mode-pill">
            {{ mode === 'edit' ? t('admin.gamesSection.editMode') : t('admin.gamesSection.createMode') }}
          </span>
        </div>

        <label class="field">
          <span>{{ t('admin.gamesSection.fields.name') }}</span>
          <input
            v-model="form.name_game"
            class="input"
            type="text"
            :placeholder="t('admin.gamesSection.fields.namePlaceholder')"
          />
        </label>

        <p v-if="errorMsg" class="state-copy error">{{ errorMsg }}</p>
        <p v-if="successMsg" class="state-copy success">{{ successMsg }}</p>

        <div class="editor-actions">
          <button class="button ghost" @click="resetForm">{{ t('admin.gamesSection.clearForm') }}</button>
          <button class="button primary" :disabled="saving" @click="saveGame">
            {{
              saving
                ? t('common.saving')
                : mode === 'edit'
                  ? t('admin.gamesSection.saveGame')
                  : t('admin.gamesSection.createGame')
            }}
          </button>
        </div>
      </div>

      <div class="list-card">
        <div class="list-head">
          <div>
            <h3>{{ t('admin.gamesSection.listTitle') }}</h3>
            <p class="muted">{{ t('admin.gamesSection.listHint') }}</p>
          </div>
        </div>

        <p v-if="loading" class="state-copy">{{ t('admin.gamesSection.loading') }}</p>

        <div v-else class="game-list">
          <article
            v-for="game in games"
            :key="game.game_id"
            class="game-card"
            :class="{ selected: editingGameId === game.game_id }"
          >
            <div class="game-topline">
              <strong>#{{ game.game_id }}</strong>
              <span class="badge">{{ t('admin.gamesSection.badge') }}</span>
            </div>
            <h4>{{ game.name_game }}</h4>
            <div class="game-actions">
              <button class="button ghost" @click="editGame(game)">{{ t('common.edit') }}</button>
              <button class="button danger" @click="archiveGame(game)">{{ t('common.archive') }}</button>
            </div>
          </article>

          <p v-if="games.length === 0" class="state-copy">{{ t('admin.gamesSection.noMatches') }}</p>
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
    radial-gradient(circle at top left, color-mix(in oklab, #f97316, white 82%), transparent 24%),
    linear-gradient(180deg, color-mix(in oklab, var(--color-surface), white 22%), var(--color-surface));
  box-shadow: var(--shadow-md);
}

.section-header,
.header-actions,
.editor-head,
.editor-actions,
.list-head,
.game-topline,
.game-actions,
.pager {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: flex-start;
  flex-wrap: wrap;
}

.section-header h2,
.editor-head h3,
.list-head h3,
.game-card h4 {
  margin: 0;
}

.eyebrow {
  margin: 0 0 0.35rem;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  font-size: 0.72rem;
  color: #c2410c;
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
  grid-template-columns: minmax(20rem, 26rem) minmax(0, 1fr);
  gap: 1rem;
}

.editor-card,
.list-card,
.game-card {
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

.field {
  display: grid;
  gap: 0.35rem;
}

.mode-pill,
.badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  font-size: 0.8rem;
}

.mode-pill {
  padding: 0.35rem 0.85rem;
  background: color-mix(in oklab, #c2410c, white 82%);
  color: #9a3412;
}

.badge {
  padding: 0.3rem 0.75rem;
  background: color-mix(in oklab, #ea580c, white 82%);
  color: #9a3412;
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
  background: linear-gradient(135deg, #f97316, #c2410c);
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

.header-actions .button,
.editor-actions .button,
.game-actions .button,
.pager .button {
  min-width: 8.5rem;
}

.game-list {
  display: grid;
  gap: 0.9rem;
}

.game-card {
  display: grid;
  gap: 0.85rem;
  padding: 1rem;
}

.game-card.selected {
  border-color: color-mix(in oklab, #f97316, white 42%);
  box-shadow: 0 0 0 1px color-mix(in oklab, #f97316, white 62%);
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

@media (max-width: 1080px) {
  .workspace {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .section-card {
    padding: 1rem;
  }

  .header-actions,
  .editor-actions,
  .game-actions {
    justify-content: stretch;
  }

  .header-actions .button,
  .editor-actions .button,
  .game-actions .button,
  .pager .button {
    width: 100%;
  }
}
</style>
