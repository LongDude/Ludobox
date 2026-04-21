import { useAuthStore } from '@/stores/authStore'
import type { AxiosError, AxiosInstance, InternalAxiosRequestConfig } from 'axios'

let refreshPromise: Promise<string> | null = null

export function attachAccessToken(config: InternalAxiosRequestConfig) {
  const auth = useAuthStore()
  const token = auth.AccessToken

  if (token) {
    config.headers = config.headers ?? {}
    ;(config.headers as any).Authorization = `Bearer ${token}`
  }

  return config
}

async function getRefreshedAccessToken() {
  const auth = useAuthStore()

  if (!refreshPromise) {
    refreshPromise = auth
      .refreshToken()
      .then((token) => {
        if (!token) {
          throw new Error('Refresh returned no access token')
        }

        return token
      })
      .catch(async (error) => {
        try {
          await auth.logout()
        } catch {}

        throw error
      })
      .finally(() => {
        refreshPromise = null
      })
  }

  return refreshPromise
}

export function shouldRefreshRequest(error: AxiosError) {
  const original = error.config
  const status = error.response?.status
  const reqUrl = (original?.url || '') as string
  const isRefreshCall = reqUrl.includes('/auth/refresh')
  const isLogoutCall = reqUrl.includes('/auth/logout')

  return Boolean(
    original && status === 401 && !(original as any)._retry && !isRefreshCall && !isLogoutCall,
  )
}

export async function retryRequestAfterRefresh(
  api: AxiosInstance,
  error: AxiosError,
  normalizeApiError: (error: AxiosError) => Error,
) {
  const original = error.config

  if (!original || !shouldRefreshRequest(error)) {
    return Promise.reject(normalizeApiError(error))
  }

  ;(original as any)._retry = true

  try {
    const accessToken = await getRefreshedAccessToken()
    original.headers = original.headers ?? {}
    ;(original.headers as any).Authorization = `Bearer ${accessToken}`
    return api(original)
  } catch {
    return Promise.reject(normalizeApiError(error))
  }
}
