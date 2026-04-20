import { api } from '@/api/base/useLudaApi'
import type {
  AdminListRequest,
  ConfigListResponse,
  ConfigResponse,
  ConfigUpsertRequest,
  GameServerListResponse,
  RoomCreateRequest,
  RoomListResponse,
  RoomResponse,
  RoomUpdateRequest,
} from './types'

function buildAdminParams(request?: AdminListRequest) {
  if (!request) return undefined

  const filters = (request.filters ?? []).filter(
    (filter) => filter.field && filter.operator && filter.value !== undefined && filter.value !== '',
  )

  return {
    page: request.page,
    page_size: request.page_size,
    sort_field: request.sort_field,
    sort_direction: request.sort_direction,
    filter_fields: filters.length ? filters.map((filter) => filter.field).join(',') : undefined,
    filter_operators: filters.length
      ? filters.map((filter) => filter.operator).join(',')
      : undefined,
    filter_values: filters.length
      ? filters.map((filter) => String(filter.value).trim()).join(',')
      : undefined,
  }
}

export const UserApi = {
  listConfigs(request?: AdminListRequest) {
    return api
      .get<ConfigListResponse>('/users/admin/configs/used', { params: buildAdminParams(request) })
      .then((response) => response.data)
  },
  getConfig(configId: number) {
    return api.get<ConfigResponse>(`/users/admin/config/${configId}`).then((response) => response.data)
  },
  createConfig(payload: ConfigUpsertRequest) {
    return api.post<ConfigResponse>('/users/admin/config', payload).then((response) => response.data)
  },
  updateConfig(configId: number, payload: ConfigUpsertRequest) {
    return api
      .put<ConfigResponse>(`/users/admin/config/${configId}`, payload)
      .then((response) => response.data)
  },
  deleteConfig(configId: number) {
    return api.delete<void>(`/users/admin/config/${configId}`).then((response) => response.data)
  },
  listRooms(request?: AdminListRequest) {
    return api
      .get<RoomListResponse>('/users/admin/rooms', { params: buildAdminParams(request) })
      .then((response) => response.data)
  },
  getRoom(roomId: number) {
    return api.get<RoomResponse>(`/users/admin/room/${roomId}`).then((response) => response.data)
  },
  createRoom(payload: RoomCreateRequest) {
    return api.post<RoomResponse>('/users/admin/room', payload).then((response) => response.data)
  },
  updateRoom(roomId: number, payload: RoomUpdateRequest) {
    return api
      .put<RoomResponse>(`/users/admin/room/${roomId}`, payload)
      .then((response) => response.data)
  },
  deleteRoom(roomId: number) {
    return api.delete<void>(`/users/admin/room/${roomId}`).then((response) => response.data)
  },
  listServers(request?: AdminListRequest) {
    return api
      .get<GameServerListResponse>('/users/admin/servers', { params: buildAdminParams(request) })
      .then((response) => response.data)
  },
}
