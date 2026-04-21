import { ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { UserApi } from '@/api/useUserApi'
import type { CurrentUserProfileResponse } from '@/api/types'
import { useAuthStore } from '@/stores/authStore'

export const useUserCabinetStore = defineStore('userCabinet', () => {
  const auth = useAuthStore()

  const profile = ref<CurrentUserProfileResponse | null>(null)
  const loading = ref(false)
  const loaded = ref(false)
  const error = ref('')

  let pendingLoad: Promise<CurrentUserProfileResponse | null> | null = null
  let lastLoadId = 0
  let stopBalanceEvents: (() => void) | null = null

  function reset() {
    lastLoadId += 1
    pendingLoad = null
    stopBalanceSubscription()
    profile.value = null
    loading.value = false
    loaded.value = false
    error.value = ''
  }

  function loadProfile(force = false) {
    if (!auth.isAuthenticated) {
      reset()
      return Promise.resolve<CurrentUserProfileResponse | null>(null)
    }

    if (!force) {
      if (profile.value && loaded.value) {
        return Promise.resolve(profile.value)
      }
      if (pendingLoad) {
        return pendingLoad
      }
    } else if (pendingLoad) {
      return pendingLoad
    }

    error.value = ''
    loading.value = true
    const loadId = ++lastLoadId

    const loadPromise = UserApi.getCurrentUser()
      .then((response) => {
        if (loadId === lastLoadId) {
          profile.value = response
          loaded.value = true
          error.value = ''
        }
        return response
      })
      .catch((err: any) => {
        if (loadId === lastLoadId) {
          error.value = err?.message || 'Failed to load user profile.'
          loaded.value = Boolean(profile.value)
        }
        throw err
      })
      .finally(() => {
        if (pendingLoad === loadPromise) {
          pendingLoad = null
        }
        if (loadId === lastLoadId) {
          loading.value = false
        }
      })

    pendingLoad = loadPromise
    return loadPromise
  }

  function ensureLoaded() {
    return loadProfile(false)
  }

  function refresh() {
    return loadProfile(true)
  }

  function startBalanceSubscription() {
    if (stopBalanceEvents || !auth.isAuthenticated) return

    stopBalanceEvents = UserApi.subscribeBalanceEvents({
      onEvent(event) {
        if (event.type !== 'user_balance_changed' && event.type !== 'user_balance_snapshot') return

        if (profile.value) {
          profile.value = {
            ...profile.value,
            user_id: event.user_id,
            balance: event.balance,
          }
          loaded.value = true
          error.value = ''
          return
        }

        void refresh().catch(() => {})
      },
      onError(err) {
        error.value = err instanceof Error ? err.message : 'Failed to subscribe balance events.'
      },
    })
  }

  function stopBalanceSubscription() {
    stopBalanceEvents?.()
    stopBalanceEvents = null
  }

  async function updateNickname(nickname: string) {
    if (!auth.isAuthenticated) {
      throw new Error('Unauthorized')
    }

    const response = await UserApi.updateCurrentUser({ nickname: nickname.trim() })
    profile.value = response
    loaded.value = true
    error.value = ''
    return response
  }

  async function applyBalanceDelta(delta: number) {
    if (!auth.isAuthenticated) {
      throw new Error('Unauthorized')
    }

    const response = await UserApi.updateCurrentUserBalance({ delta })
    profile.value = response
    loaded.value = true
    error.value = ''
    return response
  }

  watch(
    () => auth.isAuthenticated,
    (isAuthenticated) => {
      if (!isAuthenticated) {
        reset()
        return
      }

      startBalanceSubscription()
      if (!loaded.value && !pendingLoad) {
        void ensureLoaded().catch(() => {})
      }
    },
    { immediate: true },
  )

  return {
    profile,
    loading,
    loaded,
    error,
    ensureLoaded,
    refresh,
    updateNickname,
    applyBalanceDelta,
    startBalanceSubscription,
    stopBalanceSubscription,
    reset,
  }
})
