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

export interface CurrentUserProfileResponse {
  user_id: number
  nickname: string
  balance: number
  rating: number
  rank: UserRank
}

export interface CurrentUserProfileUpdateRequest {
  nickname: string
}

export interface CurrentUserBalanceUpdateRequest {
  delta: number
}

export interface UserBalanceEvent {
  type: string
  action: string
  user_id: number
  balance: number
  timestamp: string
}

export type UserRank = 'bronze' | 'silver' | 'gold' | 'platinum' | 'diamond' | string

export interface UserRatingHistoryRequest {
  date_from?: string
  date_to?: string
}

export interface UserRatingHistoryItem {
  history_id: number
  round_id?: number | null
  room_id?: number | null
  game_id?: number | null
  game_name?: string | null
  source: string
  delta: number
  rating_after: number
  rank: UserRank
  created_at: string
}

export interface UserRatingHistoryResponse {
  current_rating: number
  current_rank: UserRank
  period_change: number
  items: UserRatingHistoryItem[]
}

export type UserGameHistoryResult =
  | 'won'
  | 'lost'
  | 'left'
  | 'cancelled'
  | 'waiting'
  | 'active'
  | 'finished'
  | string

export interface UserGameHistoryRequest {
  page?: number
  page_size?: number
  game_id?: number
  room_id?: number
  status?: UserGameHistoryResult | ''
  date_from?: string
  date_to?: string
}

export interface UserGameHistoryItem {
  round_id: number
  room_id: number
  game_id: number
  game_name: string
  round_status: string
  result: UserGameHistoryResult
  reserved_seats: number[]
  winning_seats: number[]
  reserved_seats_count: number
  winning_seats_count: number
  entry_fee: number
  boost_fee: number
  total_spent: number
  winning_money: number
  net_result: number
  joined_at: string
  finished_at?: string | null
}

export interface UserGameHistoryListResponse {
  items: UserGameHistoryItem[]
  total: number
  page: number
  page_size: number
}

export interface MatchmakingFilters {
  min_registration_price?: number
  max_registration_price?: number
  min_capacity?: number
  max_capacity?: number
  is_boost?: boolean
  min_boost_power?: number
  page?: number
  page_size?: number
}

export interface Pagination {
  total: number
  page: number
  page_size: number
}

export interface RoomRecommendationResponse {
  room_id: number
  config_id: number
  server_id: number
  game_id: number
  registration_price: number
  capacity: number
  min_users: number
  is_boost: boolean
  boost_power: number
  current_players: number
  instance_key: string
  redis_host: string
  score: number
}

export interface RecommendRoomsResponse {
  items: RoomRecommendationResponse[]
  cached: boolean
  pagination: Pagination
}

export interface QuickMatchResponse {
  room: RoomRecommendationResponse
  round_id: number
  round_participant_id: number
  seat_number: number
  reused_existing_room: boolean
}

export interface GameJoinRoomWithSeatRequest {
  number_in_room: number
}

export interface GameJoinRoomResponse {
  participant_id: number
  round_id: number
  nickname?: string | null
  rating?: number | null
  number_in_room: number
  room_capacity: number
  current_players: number
  min_players: number
  entry_price: number
  round_status: string
  timer_starts_at?: string | null
}

export interface GameRoomStateResponse {
  room_id: number
  round_id: number
  room_capacity: number
  current_players: number
  min_players: number
  entry_price: number
  round_status: string
  is_boost: boolean
  boost_power: number
  boost_price: number
  waiting_time: number
  round_time: number
  next_round_delay: number
  timer_starts_at?: string | null
  current_user_participants?: GameParticipantInfo[]
  recent_events?: GameRoundEvent[]
}

export interface GameParticipantInfo {
  participant_id: number
  user_id?: number | null
  nickname?: string | null
  rating?: number | null
  number_in_room: number
  boost: number
  winning_money: number
  is_bot: boolean
  exited_at?: string | null
}

export interface GameRoundStatusResponse {
  round_id: number
  status: string
  created_at: string
  time_left_seconds: number
  participants: GameParticipantInfo[]
  winners: GameParticipantInfo[]
}

export interface GamePurchaseBoostResponse {
  success: boolean
  message?: string
  boost_power: number
  boost_cost: number
}

export interface GameCancelBoostResponse {
  success: boolean
  message?: string
  refund?: number
}

export interface GameLeaveRoomResponse {
  success: boolean
  message?: string
  refund?: number
}

export interface GameRoundEvent {
  type: string
  timestamp: string
  data: unknown
}

export interface GameWinnerInfo {
  participant_id: number
  user_id?: number | null
  nickname?: string | null
  rating?: number | null
  number_in_room: number
  winnings: number
  gross_winnings: number
  is_bot: boolean
}

export interface GameRoundFinalizedEventData {
  round_id: number
  winners: GameWinnerInfo[]
  payouts?: Record<string, number>
  next_round_id?: number | null
  next_round_delay?: number | null
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

export interface GameUpsertRequest {
  name_game: string
}

export interface GameResponse extends GameUpsertRequest {
  game_id: number
  archived_at?: string | null
}

export interface GameListResponse {
  items: GameResponse[]
  total: number
  page: number
  page_size: number
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
  round_time: number
  next_round_delay: number
  min_users: number
}

export interface ConfigResponse extends ConfigUpsertRequest {
  config_id: number
  game?: GameResponse | null
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
  config?: ConfigResponse | null
  server_id: number
  server_name?: string | null
  current_players: number
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

export type AdminEventResource = 'games' | 'configs' | 'rooms' | 'servers'

export interface AdminEvent {
  type: string
  resource?: AdminEventResource | ''
  action: string
  id?: number
  data?: Record<string, unknown> | null
  timestamp: string
}
