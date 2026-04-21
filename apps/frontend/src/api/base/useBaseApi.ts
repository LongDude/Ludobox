import { useAuthStore } from '@/stores/authStore'
import axios, { AxiosError } from 'axios'
import { SSO_CLIENT_ID_URL } from '@/config'
import { attachAccessToken, retryRequestAfterRefresh } from '@/api/base/authRetry'

export const api = axios.create({
  baseURL: SSO_CLIENT_ID_URL,
  timeout: 10000,
  withCredentials: true,
})

api.interceptors.request.use(attachAccessToken)

api.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => retryRequestAfterRefresh(api, error, normalizeApiError),
)

export class ApiError extends Error {
  status?: number
  details?: unknown
  constructor(message: string, status?: number, details?: unknown) {
    super(message)
    this.status = status
    this.details = details
  }
}

function normalizeApiError(err: AxiosError) {
  const status = err.response?.status
  const payload = (err.response?.data as any) ?? {}
  const message = payload.error || payload.message || err.message || 'Request failed'
  const details = err.response?.data
  return new ApiError(message, status, details)
}
