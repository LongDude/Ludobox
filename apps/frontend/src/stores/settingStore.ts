import { ref, watch } from 'vue'
import { defineStore } from 'pinia'

export const useSettingStore = defineStore('setting', () => {
  const STORAGE_KEY = 'layout.left-tab-hidden'

  function readLeftTabHidden() {
    try {
      const saved = typeof window !== 'undefined' ? localStorage.getItem(STORAGE_KEY) : null
      if (saved === 'true') return { value: true, restored: true }
      if (saved === 'false') return { value: false, restored: true }
    } catch {}

    return { value: false, restored: false }
  }

  const initialState = readLeftTabHidden()
  const LeftTabHidden = ref(initialState.value)
  const hasSavedLeftTabPreference = ref(initialState.restored)

  function setLeftTabHidden(value: boolean) {
    LeftTabHidden.value = value
    hasSavedLeftTabPreference.value = true
  }

  function applyInitialLeftTabHidden(value: boolean) {
    if (!hasSavedLeftTabPreference.value) {
      LeftTabHidden.value = value
    }
  }

  const HideLeftTab = () => {
    setLeftTabHidden(!LeftTabHidden.value)
  }

  watch(
    LeftTabHidden,
    (value) => {
      if (!hasSavedLeftTabPreference.value) return

      try {
        localStorage.setItem(STORAGE_KEY, String(value))
      } catch {}
    },
    { immediate: true },
  )

  return { LeftTabHidden, HideLeftTab, setLeftTabHidden, applyInitialLeftTabHidden }
})
