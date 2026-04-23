<script lang="ts" setup>
import { useSettingStore } from '@/stores/settingStore'
import { useI18n } from '@/i18n'
const useSetting = useSettingStore()
const { t } = useI18n()
</script>
<template>
  <div class="footer" :class="{ collapsed: useSetting.LeftTabHidden }">
  </div>
</template>
<style lang="css" scoped>
.footer {
  position: fixed;
  bottom: 0;
  left: 304px;
  right: 20px;
  height: 20px;
  font-size: x-small;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-slow) ease;
}
.footer.collapsed {
  left: 120px;
}
p {
  margin: 0;
}

@media (max-width: 960px) {
  .footer,
  .footer.collapsed {
    left: 1rem;
    right: 1rem;
    bottom: 5rem;
  }
}
</style>
