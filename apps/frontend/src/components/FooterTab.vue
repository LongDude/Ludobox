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
@media (max-width: 960px) {
  .footer,
  .footer.collapsed {
    left: 1rem;
    right: 1rem;
    bottom: 5rem;
  }
}
</style>
