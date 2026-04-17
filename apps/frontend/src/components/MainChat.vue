<script setup lang="ts">
import { ref } from 'vue'
import { useSettingStore } from '@/stores/settingStore'
import { useI18n } from '@/i18n'

const useSetting = useSettingStore()
const { t } = useI18n()

const loading = ref(false)
const errorMsg = ref('')



</script>
<template>
  <div class="chat" :class="{ collapsed: useSetting.LeftTabHidden }">
    <div class="main-chat" :class="{ collapsed: useSetting.LeftTabHidden }">
      <div class="status" v-if="loading || errorMsg">
        <span v-if="loading" class="status__loading">{{ t('chat.status.searching') }}</span>
        <span v-else class="status__error">{{ errorMsg }}</span>
      </div>
    </div>
  </div>
</template>
<style lang="css" scoped>
.chat {
  position: fixed;
  top: 80px;
  left: 310px;
  right: 60px;
  bottom: 20px;
  background-color: var(--color-bg-secondary);
  border-radius: 15px;
  padding: var(--space-4);
  display: grid;
  grid-template-rows: 1fr auto;
  gap: var(--space-3);
  transition: all var(--transition-slow) ease;
}
.chat.collapsed {
  left: 120px;
}

.main-chat {
  position: relative;
  display: grid;
  grid-template-rows: auto auto 1fr;
  gap: var(--space-3);
  overflow: hidden;
}

.main-header h2 {
  margin: 0;
  font-size: 1.4rem;
}
.main-header p {
  margin: 4px 0 0;
  color: var(--color-muted);
  font-size: 0.95rem;
}

.chat-log {
  position: relative;
  overflow-y: auto;
  padding-right: 4px;
  display: grid;
  gap: var(--space-4);
  padding-bottom: var(--space-2);
}

.chat-turn {
  background: rgba(0, 0, 0, 0.03);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  display: grid;
  gap: var(--space-3);
  border: 1px solid transparent;
  transition:
    border-color var(--transition-fast) ease,
    box-shadow var(--transition-fast) ease;
}
.chat-turn.active {
  border-color: var(--color-primary-secondary);
  box-shadow: 0 14px 28px rgba(0, 0, 0, 0.08);
}
.chat-turn__header {
  display: flex;
  justify-content: space-between;
  gap: var(--space-2);
  align-items: baseline;
}
.chat-turn__prompt {
  font-weight: 600;
  font-size: 1.05rem;
}
.chat-turn__time {
  font-size: 0.8rem;
  color: var(--color-muted);
}

.status {
  font-size: 0.95rem;
  min-height: 1.4rem;
}
.status__loading {
  color: var(--color-muted);
}
.status__error {
  color: var(--color-danger);
}

.results-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) minmax(280px, 0.9fr);
  gap: var(--space-4);
  align-items: start;
}

.cards {
  display: flex;
  gap: var(--space-3);
  flex-wrap: wrap;
  position: relative;
  z-index: 1;
}

.paper-card {
  position: relative;
  flex: 1 1 260px;
  min-width: 240px;
  background: var(--color-bg);
  border-radius: var(--radius-lg);
  padding: var(--space-3);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.08);
  border: 1px solid transparent;
  transition:
    transform var(--transition-fast) ease,
    box-shadow var(--transition-fast) ease,
    border-color var(--transition-fast) ease;
  cursor: pointer;
  outline: none;
}
.paper-card:hover,
.paper-card:focus-visible {
  transform: translateY(-8px) scale(1.02);
  box-shadow: 0 14px 28px rgba(0, 0, 0, 0.12);
  border-color: var(--color-primary-secondary);
  z-index: 2;
}

.paper-card--active {
  border-color: var(--color-primary-secondary);
  box-shadow: 0 16px 32px rgba(0, 0, 0, 0.14);
  transform: translateY(-6px) scale(1.015);
  z-index: 3;
}

.paper-card__header {
  display: grid;
  gap: 4px;
}
.paper-card__year {
  font-size: 0.75rem;
  color: var(--color-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.paper-card__title {
  margin: 0;
  font-size: 1.05rem;
}
.paper-card__abstract {
  margin: var(--space-2) 0 var(--space-3);
  color: var(--color-muted);
  line-height: 1.4;
  max-height: 6.5rem;
  overflow: hidden;
}
.paper-card__footer {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  font-size: 0.8rem;
  color: var(--color-muted);
}
.paper-card__badge {
  color: var(--color-success);
  font-weight: 600;
}

.no-results {
  margin: 0;
  color: var(--color-muted);
  font-size: 0.95rem;
}

.paper-preview {
  position: sticky;
  top: var(--space-4);
  background: var(--color-bg);
  border-radius: var(--radius-xl);
  padding: var(--space-4);
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.18);
  border: 1px solid var(--color-primary-secondary);
  max-height: 60vh;
  overflow-y: auto;
  align-self: start;
}
.paper-preview h3 {
  margin: 0 0 var(--space-2);
}
.paper-preview__abstract {
  margin: 0 0 var(--space-3);
  line-height: 1.5;
}
.paper-preview__meta {
  display: flex;
  gap: var(--space-3);
  flex-wrap: wrap;
  font-size: 0.85rem;
  color: var(--color-muted);
  margin-bottom: var(--space-3);
}
.paper-preview__links {
  display: flex;
  gap: var(--space-2);
}

.btn--tiny {
  font-size: 0.8rem;
  padding: 6px 10px;
  border-radius: 999px;
  background: var(--color-primary-secondary);
  color: var(--color-bg);
}

.empty-state {
  text-align: center;
  color: var(--color-muted);
  padding: var(--space-4) 0;
}

.input-area {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background-color: var(--color-bg);
  color: var(--color-text);
  display: flex;
  gap: var(--space-2);
  padding: 4px 6px;
  align-items: center;
}

.input-area input {
  border: none;
  outline: none;
  background: transparent;
  flex: 1;
  padding: 10px;
  font-size: 1rem;
}

.input-area button {
  min-width: 120px;
}
.blocked-note {
  color: var(--color-danger);
  margin-left: 8px;
}

@media (max-width: 1200px) {
  .results-grid {
    grid-template-columns: 1fr;
  }
  .paper-preview {
    position: relative;
    top: auto;
    max-height: none;
    margin-bottom: var(--space-3);
    margin-top: var(--space-3);
  }
}

@media (max-width: 960px) {
  .main-chat {
    grid-template-rows: auto 1fr;
  }
  .results-grid {
    gap: var(--space-3);
  }
  .paper-preview {
    box-shadow: none;
  }
}

@media (max-width: 720px) {
  .chat-turn {
    padding: var(--space-3);
  }
  .chat-turn__header {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--space-1, 4px);
  }
  .paper-card {
    min-width: auto;
  }
  .paper-preview__links {
    flex-direction: column;
    align-items: stretch;
  }
  .input-area {
    flex-direction: column;
    align-items: stretch;
    padding: var(--space-2);
  }
  .input-area button {
    width: 100%;
    min-width: 0;
  }
}
</style>
