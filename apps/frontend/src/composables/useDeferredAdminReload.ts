import { onBeforeUnmount, watch, type Ref } from 'vue'

export function useDeferredAdminReload(
  reload: () => Promise<void> | void,
  busy?: Readonly<Ref<boolean>>,
  delayMs = 350,
) {
  let timer: ReturnType<typeof setTimeout> | undefined
  let pending = false

  function clearReloadTimer() {
    if (timer) {
      clearTimeout(timer)
      timer = undefined
    }
  }

  function scheduleReload() {
    if (busy?.value) {
      pending = true
      return
    }

    clearReloadTimer()
    timer = setTimeout(() => {
      timer = undefined
      void reload()
    }, delayMs)
  }

  if (busy) {
    watch(busy, (isBusy) => {
      if (!isBusy && pending) {
        pending = false
        scheduleReload()
      }
    })
  }

  onBeforeUnmount(clearReloadTimer)

  return scheduleReload
}
