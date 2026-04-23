<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { SSOApi } from '@/api/useSSOApi'
import type { UserListResponse, UserResponse, UserUpdateRequestWithRoles } from '@/api/types'
import { useI18n } from '@/i18n'

type UserEditing = UserResponse & {
  _editing?: boolean
  rolesString?: string
  _password?: string
  _saving?: boolean
  _error?: string
}

const loading = ref(false)
const errorMsg = ref('')
const users = reactive<UserEditing[]>([])
const total = ref(0)
const page = ref(1)
const limit = ref(20)

const filters = reactive({
  q: '',
  role: '',
  locale: '',
  email_confirmed: '' as '' | 'true' | 'false',
})
const { t } = useI18n()

const params = computed(() => ({
  q: filters.q || undefined,
  role: filters.role || undefined,
  locale: filters.locale || undefined,
  email_confirmed:
    filters.email_confirmed === '' ? undefined : filters.email_confirmed === 'true' ? true : false,
  page: page.value,
  limit: limit.value,
}))

onMounted(async () => {
  await loadUsers()
})

async function loadUsers() {
  errorMsg.value = ''
  loading.value = true
  try {
    const response = (await SSOApi.getUsers(params.value)) as UserListResponse
    users.splice(0, users.length, ...((response.items ?? []) as UserEditing[]))
    total.value = response.total ?? 0
    page.value = response.page ?? page.value
    limit.value = response.limit ?? limit.value
  } catch (error: any) {
    errorMsg.value = error?.message || t('admin.errFetch')
  } finally {
    loading.value = false
  }
}

function startEdit(user: UserEditing) {
  user._editing = true
  user.rolesString = user.roles?.join(', ') || ''
  user._password = ''
  user._error = ''
}

function cancelEdit(user: UserEditing) {
  user._editing = false
  user._error = ''
}

async function saveUser(user: UserEditing) {
  user._error = ''
  user._saving = true
  try {
    const roles =
      typeof user.rolesString === 'string'
        ? user.rolesString
            .split(',')
            .map((role) => role.trim().toUpperCase())
            .filter(Boolean)
        : user.roles ?? []

    const payload: UserUpdateRequestWithRoles = {
      roles,
      first_name: user.first_name,
      last_name: user.last_name,
      locale_type: user.locale_type,
    }

    if (user._password) {
      payload.password = user._password
    }

    if (typeof user.id !== 'number') {
      throw new Error(t('admin.usersSection.userIdMissing'))
    }

    const updated = await SSOApi.updateUserwithRoles(user.id, payload)
    user.first_name = updated.first_name
    user.last_name = updated.last_name
    user.locale_type = updated.locale_type
    user.photo = updated.photo
    user.email_confirmed = updated.email_confirmed
    user.roles = updated.roles
    user.rolesString = updated.roles.join(', ')
    user._editing = false
    user._password = ''
  } catch (error: any) {
    user._error = error?.message || t('profile.msg.failed')
  } finally {
    user._saving = false
  }
}

function applyFilters() {
  page.value = 1
  void loadUsers()
}

function resetFilters() {
  filters.q = ''
  filters.role = ''
  filters.locale = ''
  filters.email_confirmed = ''
  page.value = 1
  void loadUsers()
}

function prevPage() {
  if (page.value > 1) {
    page.value -= 1
    void loadUsers()
  }
}

function nextPage() {
  const pages = Math.max(1, Math.ceil(total.value / limit.value))
  if (page.value < pages) {
    page.value += 1
    void loadUsers()
  }
}

function changeLimit(nextLimit: number) {
  limit.value = nextLimit
  page.value = 1
  void loadUsers()
}
</script>

<template>
  <section class="section-card">
    <div class="section-header">
      <div>
        <h2>{{ t('admin.usersSection.title') }}</h2>
        <p class="section-copy">{{ t('admin.usersSection.description') }}</p>
      </div>
      <button class="button ghost" @click="loadUsers" :disabled="loading">{{ t('common.refresh') }}</button>
    </div>

    <div class="toolbar">
      <div class="toolbar-grid">
        <input
          v-model="filters.q"
          class="input"
          type="text"
          :placeholder="t('admin.usersSection.searchPlaceholder')"
        />
        <input
          v-model="filters.role"
          class="input"
          type="text"
          :placeholder="t('admin.usersSection.rolePlaceholder')"
        />
        <select v-model="filters.email_confirmed" class="input">
          <option value="">{{ t('admin.usersSection.emailAny') }}</option>
          <option value="true">{{ t('admin.usersSection.emailConfirmed') }}</option>
          <option value="false">{{ t('admin.usersSection.emailUnconfirmed') }}</option>
        </select>
        <select
          class="input"
          :value="limit"
          @change="changeLimit(Number(($event.target as HTMLSelectElement).value))"
        >
          <option :value="10">{{ t('common.rows', { count: 10 }) }}</option>
          <option :value="20">{{ t('common.rows', { count: 20 }) }}</option>
          <option :value="50">{{ t('common.rows', { count: 50 }) }}</option>
        </select>
      </div>
      <div class="toolbar-actions">
        <button class="button primary" @click="applyFilters">{{ t('common.apply') }}</button>
        <button class="button ghost" @click="resetFilters">{{ t('common.reset') }}</button>
      </div>
    </div>

    <p v-if="loading" class="state-copy">{{ t('admin.usersSection.loading') }}</p>
    <p v-else-if="errorMsg" class="state-copy error">{{ errorMsg }}</p>

    <div v-else class="table-shell">
      <div class="table-head user-grid">
        <div>{{ t('admin.columns.email') }}</div>
        <div>{{ t('admin.columns.first') }}</div>
        <div>{{ t('admin.columns.last') }}</div>
        <div>{{ t('admin.columns.locale') }}</div>
        <div>{{ t('admin.columns.confirmed') }}</div>
        <div>{{ t('admin.columns.roles') }}</div>
        <div>{{ t('admin.columns.password') }}</div>
        <div>{{ t('admin.columns.actions') }}</div>
      </div>

      <div v-for="user in users" :key="user.id ?? user.email" class="table-row user-grid">
        <div class="strong">{{ user.email }}</div>

        <div>
          <span v-if="!user._editing">{{ user.first_name }}</span>
          <input v-else v-model="user.first_name" class="input" type="text" />
        </div>

        <div>
          <span v-if="!user._editing">{{ user.last_name }}</span>
          <input v-else v-model="user.last_name" class="input" type="text" />
        </div>

        <div>
          <span v-if="!user._editing">{{ user.locale_type || '-' }}</span>
          <input v-else v-model="user.locale_type" class="input" type="text" />
        </div>

        <div>
          <span class="pill" :class="user.email_confirmed ? 'good' : 'warn'">
            {{ user.email_confirmed ? t('common.yes') : t('common.no') }}
          </span>
        </div>

        <div>
          <span v-if="!user._editing">{{ user.roles.join(', ') }}</span>
          <input
            v-else
            v-model="user.rolesString"
            class="input"
            type="text"
            :placeholder="t('admin.columns.roles')"
          />
        </div>

        <div>
          <span v-if="!user._editing">-</span>
          <input
            v-else
            v-model="user._password"
            class="input"
            type="password"
            :placeholder="t('admin.usersSection.passwordPlaceholder')"
          />
        </div>

        <div class="actions">
          <button v-if="!user._editing" class="button ghost" @click="startEdit(user)">{{ t('common.edit') }}</button>
          <template v-else>
            <button class="button ghost" @click="cancelEdit(user)">{{ t('common.cancel') }}</button>
            <button class="button primary" :disabled="user._saving" @click="saveUser(user)">
              {{ user._saving ? t('common.saving') : t('common.save') }}
            </button>
          </template>
        </div>

        <p v-if="user._error" class="inline-error user-grid-span">{{ user._error }}</p>
      </div>

      <p v-if="users.length === 0" class="state-copy">{{ t('admin.usersSection.noMatches') }}</p>
    </div>

    <div v-if="total > 0" class="pager">
      <button class="button ghost" :disabled="page <= 1" @click="prevPage">{{ t('common.prev') }}</button>
      <span class="pager-copy">
        {{
          t('common.pageSummary', {
            page,
            pages: Math.max(1, Math.ceil(total / limit)),
            total,
            entity: t('admin.users.entity'),
          })
        }}
      </span>
      <button
        class="button ghost"
        :disabled="page >= Math.max(1, Math.ceil(total / limit))"
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
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  border-radius: 1.5rem;
  background: 
    radial-gradient(circle at top right, color-mix(in oklab, #0ea5e9, var(--color-surface) 92%), transparent 60%),
    linear-gradient(180deg, color-mix(in oklab, var(--color-surface), white 12%), var(--color-surface));
  box-shadow: var(--shadow-md);
}

.section-header {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: flex-start;
  flex-wrap: wrap;
}

.eyebrow {
  margin: 0 0 0.35rem;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  font-size: 0.72rem;
  color: #b45309;
}

.section-header h2 {
  margin: 0;
}

.section-copy {
  margin: 0.45rem 0 0;
  max-width: 48rem;
  color: var(--color-muted);
}

.toolbar {
  display: grid;
  gap: 0.75rem;
  padding: 0.9rem;
  border-radius: 1.15rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  background: color-mix(in oklab, var(--color-surface), white 6%);
}

.toolbar-grid {
  display: grid;
  gap: 0.75rem;
  grid-template-columns: repeat(auto-fit, minmax(11rem, 1fr));
}

.toolbar-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  flex-wrap: wrap;
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
  color: #f8fafc;
}

.button.ghost {
  border-color: color-mix(in oklab, var(--color-border), transparent 12%);
  background: color-mix(in oklab, var(--color-surface), white 10%);
  color: var(--color-text);
}

.button:disabled {
  cursor: not-allowed;
  opacity: 0.6;
  transform: none;
}

.toolbar-actions .button,
.actions .button,
.pager .button {
  min-width: 8.5rem;
}

.table-shell {
  display: grid;
  gap: 0.65rem;
  overflow-x: auto;
}

.table-head,
.table-row {
  min-width: 72rem;
  display: grid;
  gap: 0.85rem;
  align-items: center;
}

.user-grid {
  grid-template-columns: 1.8fr repeat(2, 1fr) 0.9fr 0.8fr 1.4fr 1fr 1.1fr;
}

.table-head {
  padding: 0 0.2rem;
  color: var(--color-muted);
  font-size: 0.85rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.table-row {
  padding: 1rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  border-radius: 1.15rem;
  background: color-mix(in oklab, var(--color-surface), white 10%);
}

.strong {
  font-weight: 700;
}

.pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 4.25rem;
  padding: 0.35rem 0.7rem;
  border-radius: 999px;
  font-size: 0.82rem;
  border: 1px solid transparent;
}

.pill.good {
  background: color-mix(in oklab, var(--color-success), white 78%);
  color: #166534;
}

.pill.warn {
  background: color-mix(in oklab, var(--color-warning), white 80%);
  color: #9a3412;
}

.actions {
  display: flex;
  gap: 0.55rem;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.state-copy {
  margin: 0;
  color: var(--color-muted);
}

.state-copy.error,
.inline-error {
  color: var(--color-danger);
}

.inline-error {
  margin: 0;
  font-size: 0.9rem;
}

.user-grid-span {
  grid-column: 1 / -1;
}

.pager {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
}

.pager-copy {
  color: var(--color-muted);
}

@media (max-width: 900px) {
  .section-card {
    padding: 1rem;
  }

  .actions {
    justify-content: flex-start;
  }

  .toolbar-actions {
    justify-content: stretch;
  }

  .toolbar-actions .button,
  .actions .button,
  .pager .button {
    width: 100%;
  }
}
</style>
