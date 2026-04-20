import { computed } from 'vue'
import { useSettingStore } from '@/stores/settingStore'
import { storeToRefs } from 'pinia'

interface LayoutInsetOptions {
  expanded?: string
  collapsed?: string
}

export function useLayoutInset(options?: LayoutInsetOptions) {
  const settingStore = useSettingStore()
  const { LeftTabHidden } = storeToRefs(settingStore)
  const expanded = options?.expanded ?? '92px 20px 20px 304px'
  const collapsed = options?.collapsed ?? '92px 20px 20px 120px'

  const layoutInset = computed(() => (LeftTabHidden.value ? collapsed : expanded))

  return {
    LeftTabHidden,
    layoutInset,
  }
}
