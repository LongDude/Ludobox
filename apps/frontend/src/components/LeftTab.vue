<script lang="ts" setup>
import { computed, watch, ref, onMounted, onBeforeUnmount} from 'vue'
import { useRouter } from 'vue-router'
import { useSettingStore } from '@/stores/settingStore'
import { storeToRefs } from 'pinia'
import { useAuthStore } from '@/stores/authStore'
import { useI18n } from '@/i18n'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
const authStore = useAuthStore()
const { t } = useI18n()
const router = useRouter()
const settingStore = useSettingStore()
const { LeftTabHidden } = storeToRefs(settingStore)
const leftTabHidden = LeftTabHidden
function toggleLeftTab() {
  settingStore.HideLeftTab()
}
function RedirecttoSettings() {
  router.push('/settings')
}
function RedirecttoHome() {
  router.push('/')
}
function RedirecttoAdmin() {
  router.push('/admin')
}

const props = defineProps<{
  hidden?: boolean
}>()

const hidden = computed(() => props.hidden)

watch(
  hidden,
  (val) => {
    if (typeof val === 'boolean' && val !== LeftTabHidden.value) {
      LeftTabHidden.value = val
    }
  },
  { immediate: true },
)

const openHistoryMenuId = ref<number | null>(null)
const isDeleteDialogOpen = ref(false)
const deleteCandidateId = ref<number | null>(null)


function closeHistoryMenu() {
  openHistoryMenuId.value = null
}

function handleOutsideClick(event: MouseEvent) {
  const target = event.target as HTMLElement | null
  if (!target) return
  if (target.closest('[data-history-menu]')) return
  closeHistoryMenu()
}

function handleDeleteCancel() {
  isDeleteDialogOpen.value = false
  deleteCandidateId.value = null
}

onMounted(() => {
  document.addEventListener('click', handleOutsideClick)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleOutsideClick)
})
</script>
<template>
  <div class="left-tab" :class="{ hidden: leftTabHidden }">
    <div class="header" :class="{ hidden: leftTabHidden }">
      <button class="btn btn-icon" type="button" @click="RedirecttoHome" aria-label="Home">
        <img alt="Home" src="./../assets/logo_micro.svg" class="logo" />
      </button>
      <button
        class="btn btn-icon sidebar-toggle"
        type="button"
        @click="toggleLeftTab"
        :aria-expanded="!leftTabHidden"
        aria-label="Toggle sidebar"
      >
        <svg class="chevron" viewBox="0 0 12 12" aria-hidden="true" focusable="false">
          <path
            d="M4 2l4 4-4 4"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>
    </div>
    <div class="menu">
      <!-- Author access -->
      <!-- <button class="btn-menu btn" v-if="canAuthorAccess" @click="RedirecttoWriterCabinet">
        <div class="icon-text">
          <img src="/src/assets/papers-icon.svg" alt="|=|" class="logo" />
          <p v-if="!leftTabHidden">{{ t('nav.addPaper') }}</p>
        </div>
      </button> -->
      <!-- Admin access -->
      <button
        class="btn-menu btn"
        v-if="authStore.User && authStore.isAdmin"
        @click="RedirecttoAdmin"
      >
        <div class="icon-text">
          <img src="/src/assets/manage-icon.svg" alt="A" class="logo" />
          <p v-if="!leftTabHidden">{{ t('nav.adminPanel') }}</p>
        </div>
      </button>
    </div>
    <div class="footer">
      <button class="btn-menu btn" @click="RedirecttoSettings">
        <div class="icon-text">
          <img src="/src/assets/manage-icon.svg" alt="S" class="logo" />
          <p v-if="!leftTabHidden">{{ t('nav.settings') }}</p>
        </div>
      </button>
    </div>
    <ConfirmDialog
      v-model="isDeleteDialogOpen"
      :title="t('history.deletePromptTitle')"
      :message="t('history.deleteConfirm')"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      @cancel="handleDeleteCancel"
    />
  </div>
</template>
<style lang="css" scoped>
.left-tab {
  display: flex;
  flex-direction: column;
  width: 250px;
  height: 100vh;
  background-color: var(--color-bg-secondary);
  border-right: 1px solid var(--color-border);
  border-bottom-right-radius: 25px;
  border-top-right-radius: 25px;
  padding: 15px 5px;
  padding-bottom: 5px;
  transition: all var(--transition-slow) ease;
}

.left-tab.hidden {
  width: 60px;
  border-bottom-right-radius: 0px;
  border-top-right-radius: 0px;
}

.header {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 0px 10px;
  transition: all var(--transition-slow) ease;
}

.header.hidden {
  justify-content: center;
  flex-direction: column;
  padding: 0px 5px;
  gap: 10px;
}

.icon-text {
  display: inline-flex;
  align-items: center;
  gap: 0.5em;
}

.icon-text p {
  margin: 0;
  line-height: 1.2;
}

.icon-text .logo {
  width: 1em;
  height: 1em;
}

.menu {
  margin-top: 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
}
.menu.history {
  align-items: stretch;
  gap: 4px;
}
.label {
  align-self: flex-start;
  margin-bottom: 10px;
  font-size: medium;
  padding-left: 16px;
  color: var(--color-text-secondary);
}

.btn-menu {
  text-align: left;
  font-size: medium;
  width: 100%;
  appearance: none;
  border: 1px solid var(--color-bg-secondary);
  background: var(--color-bg-secondary);
  color: var(--color-text);
  padding: 0.55rem 0.9rem;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition:
    background var(--transition-base),
    border-color var(--transition-base),
    transform var(--transition-fast);
}
.btn-menu:hover {
  border-color: var(--color-border);
  background: color-mix(in oklab, var(--color-surface), var(--color-text) 3%);
}
.btn-menu:active {
  transform: translateY(1px);
}

.history-item {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 4px;
  border-color: transparent;
}

.history-elem {
  position: relative;
  display: flex;
  justify-content: space-between;
  flex-direction: row;
  align-items: stretch;
  gap: 0;
  border-color: transparent;
  margin-right: 5px;
  z-index: 0;

  width: 100%;
  appearance: none;
  border: 1px solid var(--color-bg-secondary);
  background: var(--color-bg-secondary);
  color: var(--color-text);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition:
    /* background var(--transition-base), */
    border-color var(--transition-base),
    transform var(--transition-fast);
}

.history-elem:hover {
  border-color: var(--color-border);
  /* background: color-mix(in oklab, var(--color-surface), var(--color-text) 3%); */
}

.history-elem.active {
  transform: translateY(1px);
  border-color: var(--color-primary-secondary);
}

.history-elem:active {
  transform: translateY(2px);
}

.history-elem.menu-open {
  z-index: 50;
}

.history-actions {
  position: relative;
  display: inline-flex;
  align-items: stretch;
  border-radius: var(--radius-md);
  border: 0px;
}

.history-edit {
  appearance: none;
  margin: 0px;
  padding: 0px;
  border-top-right-radius: 0;
  border-bottom-right-radius: 0;
  border-top-left-radius: var(--radius-md);
  border-bottom-left-radius: var(--radius-md);
}

.history-edit-input {
  appearance: none;
  border-top-right-radius: 0;
  border-bottom-right-radius: 0;
  border-top-left-radius: var(--radius-md);
  border-bottom-left-radius: var(--radius-md);
  cursor: text;
  height: 100%;
  border: 0px;
  width: 100%;
  text-align: left;
  font-size: medium;
  background: var(--color-bg-secondary);
  color: var(--color-text);
  padding: 0.55rem 0.9rem;
  background: color-mix(in oklab, var(--color-surface), var(--color-text) 6%);
}

.history-edit-input:focus-visible {
  outline: none;
}

.history-edit-input::placeholder {
  color: var(--color-text-secondary);
}

.history-menu {
  position: absolute;
  top: calc(100% + 4px);
  right: 0;
  display: flex;
  flex-direction: column;
  min-width: 160px;
  padding: 4px 0;
  margin: 0;
  list-style: none;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-dropdown, 0 8px 16px rgba(15, 23, 42, 0.15));
  z-index: 100;
}

.history-menu-item {
  width: 100%;
  padding: 0.45rem 0.9rem;
  background: transparent;
  border: none;
  color: var(--color-text);
  text-align: left;
  font-size: 0.85rem;
  line-height: 1.3;
  cursor: pointer;
  transition:
    background var(--transition-base),
    color var(--transition-base);
}

.history-menu-item:hover,
.history-menu-item:focus-visible {
  background: color-mix(in oklab, var(--color-surface), var(--color-text) 6%);
  outline: none;
}

.history-menu-item.destructive {
  color: var(--color-danger, #d14343);
}

.history-menu-item.destructive:hover,
.history-menu-item.destructive:focus-visible {
  background: color-mix(in oklab, var(--color-surface), var(--color-danger, #d14343) 12%);
  color: var(--color-danger, #d14343);
}

.history-elem .btn-history:first-child {
  border-top-right-radius: 0px;
  border-bottom-right-radius: 0px;
}

.history-elem .btn-history:nth-child(2) {
  border-top-left-radius: 0px;
  border-bottom-left-radius: 0px;
}

.history-elem .btn-history:nth-child(2):hover {
  background: color-mix(in oklab, var(--color-surface), var(--color-text) 5%);
}

.history-elem .btn-icon,
.history-elem .btn-history,
.history-elem .btn {
  border: 0px !important;
}

.history-elem .btn-icon:active,
.history-elem .btn-history:active,
.history-elem .btn:active {
  transform: translateY(0px) !important;
}

.btn-history {
  text-align: left;
  font-size: medium;
  width: 100%;
  appearance: none;
  border: 1px solid var(--color-bg-secondary);
  background: var(--color-bg-secondary);
  color: var(--color-text);
  padding: 0.55rem 0.9rem;
  border-radius: var(--radius-md);
  cursor: pointer;
}

.history-item .history-title {
  font-size: 0.95rem;
  line-height: 1.2;
  color: inherit;
}
.history-item .history-meta {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}
.history-item.active {
  border-color: var(--color-primary-secondary);
}
.history-item.active .history-meta {
  color: var(--color-primary-secondary);
}
.placeholder {
  margin: 0;
  font-size: 0.85rem;
  color: var(--color-text-secondary);
  padding: 0 16px;
  text-align: center;
}

.btn.btn-icon {
  width: 40px;
  height: 40px;
  padding: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--color-bg-secondary);
  background: var(--color-bg-secondary);
  color: var(--color-text);
  border-radius: var(--radius-md);
  line-height: 1;
}
.btn.btn-icon:hover {
  border-color: var(--color-border);
  background: color-mix(in oklab, var(--color-surface), var(--color-text) 3%);
}
.btn.btn-icon:active {
  transform: translateY(1px);
}
.btn-icon .logo {
  width: 60px;
  height: 60px;
}
.sidebar-toggle .chevron {
  width: 14px;
  height: 14px;
  transition: transform var(--transition-fast) ease;
  transform: rotate(180deg);
}
.left-tab.hidden .sidebar-toggle .chevron {
  transform: rotate(0deg);
}
.footer {
  margin-top: auto;
  display: flex;
  justify-content: center;
  padding: 10px 0;
}
</style>
