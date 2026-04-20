<script setup lang="ts">
import UpTab from '@/components/UpTab.vue'
import LeftTab from '@/components/LeftTab.vue'
import { useI18n } from '@/i18n'
import { useLayoutInset } from '@/composables/useLayoutInset'
const { t } = useI18n()
const { LeftTabHidden: leftHidden, layoutInset } = useLayoutInset()
</script>

<template>
  <UpTab :show-menu="false" :show-upload="false" />
  <LeftTab />
  <div class="search-area" :class="{ collapsed: leftHidden }" :style="{ '--layout-inset': layoutInset }">
    <h2>{{ t('search.title') }}</h2>
    <p>{{ t('search.placeholder') }}</p>
  </div>
</template>

<style scoped>
.search-area {
  position: fixed;
  inset: var(--layout-inset, 92px 20px 20px 304px);
  overflow: auto;
  transition: all var(--transition-slow) ease;
}

.search-area.collapsed {
  --layout-inset: 92px 20px 20px 120px;
}

@media (max-width: 960px) {
  .search-area,
  .search-area.collapsed {
    position: static;
    inset: auto;
    margin: calc(76px + 0.75rem) 1rem 5.75rem;
  }
}
</style>
