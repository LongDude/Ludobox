export interface TokenResReq {
  access_token: string
}

export interface UserLoginRequest {
  login: string
  password: string
}

export interface UserRegisterRequest {
  email: string
  first_name: string
  last_name: string
  password: string
}

export interface PasswordResetRequest {
  email: string
}

export interface PasswordResetResponse {
  message?: string
}

export interface UserResponse {
  id?: number
  email: string
  email_confirmed: boolean
  first_name: string
  last_name: string
  locale_type?: string
  photo?: string
  roles: string[]
}

export interface User {
  id?: number
  email: string
  email_confirmed: boolean
  first_name: string
  last_name: string
  locale_type?: string
  photo?: string
  roles: string[]
}

export interface ErrorResponse {
  error: string
}

export interface UserUpdateRequest {
  email?: string
  first_name?: string
  last_name?: string
  locale_type?: string
  password?: string
}

export interface UserUpdateRequestWithRoles {
  email?: string
  first_name?: string
  last_name?: string
  locale_type?: string
  password?: string
  roles?: string[]
}

// Admin: list users response
export interface UserListResponse {
  items: UserResponse[]
  limit: number
  page: number
  total: number
}

// Admin: list users query
export interface UserListQuery {
  q?: string
  role?: string
  email_confirmed?: boolean
  locale?: string
  page?: number
  limit?: number
}

export type SortDirection = 'asc' | 'desc'

export type AdminFilterOperator =
  | 'eq'
  | 'neq'
  | 'gt'
  | 'lt'
  | 'gte'
  | 'lte'
  | 'in'
  | 'not_in'
  | 'contains'
  | 'contained'
  | 'overlap'
  | 'like'
  | 'not_like'

export interface AdminListFilter {
  field: string
  operator: AdminFilterOperator
  value: string | number | boolean
}

export interface AdminListRequest {
  page?: number
  page_size?: number
  sort_field?: string
  sort_direction?: SortDirection
  filters?: AdminListFilter[]
}

export interface ConfigUpsertRequest {
  game_id: number
  capacity: number
  registration_price: number
  is_boost: boolean
  boost_price: number
  boost_power: number
  number_winners: number
  winning_distribution: number[]
  commission: number
  time: number
  min_users: number
}

export interface ConfigResponse extends ConfigUpsertRequest {
  config_id: number
  archived_at?: string | null
}

export interface ConfigListResponse {
  items: ConfigResponse[]
  total: number
  page: number
  page_size: number
}

export type RoomStatus = 'open' | 'in_game' | 'completed'

export interface RoomResponse {
  room_id: number
  config_id: number
  server_id: number
  status: RoomStatus
  archived_at?: string | null
}

export interface RoomListResponse {
  items: RoomResponse[]
  total: number
  page: number
  page_size: number
}

export interface RoomCreateRequest {
  config_id: number
}

export interface RoomUpdateRequest {
  server_id?: number
  archived_at?: string | null
}

export type GameServerStatus = string

export interface GameServerResponse {
  server_id: number
  instance_key: string
  redis_host: string
  status: GameServerStatus
  started_at?: string | null
  last_heartbeat_at?: string | null
  archived_at?: string | null
}

export interface GameServerListResponse {
  items: GameServerResponse[]
  total: number
  page: number
  page_size: number
}
