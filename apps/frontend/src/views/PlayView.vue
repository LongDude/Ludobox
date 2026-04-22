<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import LeftTab from '@/components/LeftTab.vue'
import UpTab from '@/components/UpTab.vue'
import FooterTab from '@/components/FooterTab.vue'
import { GameApi } from '@/api/useMatchApi'
import type {
  GameJoinRoomResponse,
  GameParticipantInfo,
  GameRoomStateResponse,
  GameRoundEvent,
  GameRoundStatusResponse,
} from '@/api/types'
import { useAuthStore } from '@/stores/authStore'
import { useMatchSessionStore } from '@/stores/matchSessionStore'
import { useUserCabinetStore } from '@/stores/userCabinetStore'
import { useI18n } from '@/i18n'
import { useLayoutInset } from '@/composables/useLayoutInset'
import { filtersToQuery } from '@/utils/matchmaking'

type LiveRoundEvent = GameRoundEvent & { id: number }

const DEFAULT_STATUS_POLLING_INTERVAL_MS = 5000
const FINISHED_STATUS_POLLING_INTERVAL_MS = 1000

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const session = useMatchSessionStore()
const cabinet = useUserCabinetStore()
const { t } = useI18n()
const { LeftTabHidden: leftHidden, layoutInset } = useLayoutInset()

const roomId = computed(() => Number(route.params.roomId))

const joinResult = ref<GameJoinRoomResponse | null>(null)
const roomState = ref<GameRoomStateResponse | null>(null)
const roundStatus = ref<GameRoundStatusResponse | null>(null)
const joining = ref(false)
const statusLoading = ref(false)
const actionLoading = ref('')
const errorMsg = ref('')
const successMsg = ref('')
const selectedSeats = ref<number[]>([])
const randomReserveCount = ref(1)
const selectedBoostParticipantId = ref('')
const autoRefresh = ref(true)
const autoAdvanceNextRound = ref(true)
const sseConnected = ref(false)
const sseError = ref('')
const liveEvents = ref<LiveRoundEvent[]>([])
const displayedRoundId = ref<number | null>(null)
const pendingNextRoundId = ref<number | null>(null)
const pendingNextRoundCountdown = ref(0)
const pendingNextRoundAt = ref<number | null>(null)
const knownOwnedParticipantIds = ref<number[]>([])
let statusTimer: number | null = null
let roundEventsStop: (() => void) | null = null
let nextRoundTimer: number | null = null
let liveEventId = 0

const room = computed(() => {
  const candidate = session.selectedRoom
  if (!candidate) return null
  return candidate.room_id === roomId.value ? candidate : null
})

const quickMatchMeta = computed(() => {
  if (session.source !== 'quick-match') return null
  return session.quickMatchMeta
})

const sourceLabel = computed(() => {
  if (session.source === 'quick-match') return t('matchmaking.play.sourceQuick')
  if (session.source === 'recommendation') return t('matchmaking.play.sourceRecommendation')
  return t('matchmaking.play.sourceUnknown')
})

const roomRoundId = computed(() => {
  const roundId = roomState.value?.round_id || joinResult.value?.round_id || quickMatchMeta.value?.round_id || 0
  return roundId > 0 ? roundId : null
})

const activeRoundId = computed(() => displayedRoundId.value ?? roomRoundId.value ?? null)

const initialParticipantId = computed(() => {
  const participantId =
    joinResult.value?.participant_id || quickMatchMeta.value?.round_participant_id || 0
  return participantId > 0 ? participantId : null
})

const initialParticipantRoundId = computed(() => {
  const roundId = joinResult.value?.round_id || quickMatchMeta.value?.round_id || 0
  return roundId > 0 ? roundId : null
})

const currentUserId = computed(() => auth.User?.id ?? null)

const roomCapacity = computed(
  () => roomState.value?.room_capacity || joinResult.value?.room_capacity || room.value?.capacity || 0,
)
const entryPrice = computed(
  () => roomState.value?.entry_price ?? joinResult.value?.entry_price ?? room.value?.registration_price ?? '-',
)
const minPlayers = computed(
  () => roomState.value?.min_players ?? joinResult.value?.min_players ?? room.value?.min_users ?? '-',
)
const currentPlayers = computed(() => {
  if (roundStatus.value?.participants?.length) {
    return roundStatus.value.participants.filter((participant) => !participant.exited_at).length
  }

  return (
    roomState.value?.current_players ??
    joinResult.value?.current_players ??
    room.value?.current_players ??
    0
  )
})

const currentRoomStatusLabel = computed(
  () => roomState.value?.round_status || joinResult.value?.round_status || '',
)
const statusLabel = computed(() => {
  if (roundStatus.value?.status) return roundStatus.value.status
  if (activeRoundId.value === roomRoundId.value) return currentRoomStatusLabel.value || '-'
  return joinResult.value?.round_status || '-'
})
const roundPhase = computed(() => normalizeRoundPhase(statusLabel.value))
const currentRoundPhase = computed(() => normalizeRoundPhase(currentRoomStatusLabel.value))
const isReviewingPreviousRound = computed(
  () => Boolean(activeRoundId.value && roomRoundId.value && activeRoundId.value !== roomRoundId.value),
)
const roundPhases = computed(() =>
  [
    {
      key: 'waiting',
      title: t('gameRoom.round.phaseWaiting'),
      description: t('gameRoom.round.phaseWaitingHint'),
    },
    {
      key: 'playing',
      title: t('gameRoom.round.phasePlaying'),
      description: t('gameRoom.round.phasePlayingHint'),
    },
    {
      key: 'finalized',
      title: t('gameRoom.round.phaseFinalized'),
      description: t('gameRoom.round.phaseFinalizedHint'),
    },
  ].map((phase, index, phases) => {
    const activeIndex = phases.findIndex((item) => item.key === roundPhase.value)
    return {
      ...phase,
      active: phase.key === roundPhase.value,
      complete: activeIndex > index,
    }
  }),
)

const seatOptions = computed(() => {
  if (!roomCapacity.value) return []
  return Array.from({ length: roomCapacity.value }, (_, index) => index + 1)
})

const occupiedSeats = computed(() => {
  const seats = new Set<number>()
  for (const participant of roundStatus.value?.participants ?? []) {
    if (!participant.exited_at) seats.add(participant.number_in_room)
  }
  for (const participant of roomUserParticipants.value) {
    if (!participant.exited_at) seats.add(participant.number_in_room)
  }
  return seats
})

const participants = computed(() =>
  [...(roundStatus.value?.participants ?? [])].sort(
    (left, right) => left.number_in_room - right.number_in_room,
  ),
)

const winners = computed(() => roundStatus.value?.winners ?? [])
const roomUserParticipants = computed(() => {
  if (activeRoundId.value !== roomRoundId.value) return []
  return (roomState.value?.current_user_participants ?? []).filter((participant) => !participant.exited_at)
})
const ownedParticipants = computed(() => {
  const owned = new Map<number, GameParticipantInfo>()
  const knownOwnedIds = new Set(knownOwnedParticipantIds.value)
  const addParticipant = (participant: GameParticipantInfo | null | undefined) => {
    if (!participant || participant.exited_at || participant.participant_id <= 0) return
    owned.set(participant.participant_id, participant)
  }

  for (const participant of roomUserParticipants.value) {
    addParticipant(participant)
  }

  const userId = currentUserId.value
  for (const participant of participants.value) {
    if (userId && participant.user_id === userId) {
      addParticipant(participant)
      continue
    }

    if (knownOwnedIds.has(participant.participant_id)) {
      addParticipant(participant)
      continue
    }

    if (owned.has(participant.participant_id)) {
      addParticipant(participant)
    }
  }

  const participantId = initialParticipantId.value
  if (
    participantId &&
    initialParticipantRoundId.value === activeRoundId.value &&
    !owned.has(participantId)
  ) {
    const matched = participants.value.find((participant) => participant.participant_id === participantId)
    if (matched) {
      addParticipant(matched)
    } else {
      const seat = joinResult.value?.number_in_room || quickMatchMeta.value?.seat_number || 0
      if (seat > 0) {
        addParticipant({
          participant_id: participantId,
          user_id: currentUserId.value,
          nickname: cabinet.profile?.nickname ?? null,
          number_in_room: seat,
          boost: 0,
          winning_money: 0,
          is_bot: false,
          exited_at: null,
        })
      }
    }
  }

  return [...owned.values()].sort((left, right) => left.number_in_room - right.number_in_room)
})
const ownedParticipantIds = computed(
  () => new Set(ownedParticipants.value.map((participant) => participant.participant_id)),
)
const ownedSeatNumbers = computed(
  () => new Set(ownedParticipants.value.map((participant) => participant.number_in_room)),
)
const selectedSeatNumbers = computed(() => new Set(selectedSeats.value))
const isJoined = computed(() => ownedParticipants.value.length > 0)
const maxOwnSeats = computed(() => Math.max(1, Math.floor((roomCapacity.value || 1) / 2)))
const remainingSeatCapacity = computed(() =>
  Math.max(0, maxOwnSeats.value - ownedParticipants.value.length),
)
const freeSeatsCount = computed(
  () => seatOptions.value.filter((seat) => !occupiedSeats.value.has(seat)).length,
)
const ownSeatsLimitReached = computed(
  () => ownedParticipants.value.length >= maxOwnSeats.value,
)
const seatSelectionLimitReached = computed(
  () => remainingSeatCapacity.value > 0 && selectedSeats.value.length >= remainingSeatCapacity.value,
)
const canJoinMoreSeats = computed(
  () =>
    canManageCurrentRound.value &&
    roomCapacity.value > 0 &&
    freeSeatsCount.value > 0 &&
    !ownSeatsLimitReached.value,
)
const boostedParticipant = computed(
  () => ownedParticipants.value.find((participant) => participant.boost > 0) ?? null,
)
const selectedBoostParticipant = computed(() => {
  const selectedId = Number(selectedBoostParticipantId.value)
  if (selectedId > 0) {
    const selected = ownedParticipants.value.find(
      (participant) => participant.participant_id === selectedId,
    )
    if (selected) return selected
  }

  return boostedParticipant.value ?? ownedParticipants.value[0] ?? null
})
const hasOwnedBoost = computed(() =>
  Boolean(boostedParticipant.value),
)
const canManageCurrentRound = computed(
  () => !isReviewingPreviousRound.value && currentRoundPhase.value === 'waiting',
)
const canManageOwnedSeats = computed(() => isJoined.value && canManageCurrentRound.value)
const canBoost = computed(
  () =>
    canManageCurrentRound.value &&
    isJoined.value &&
    Boolean(selectedBoostParticipant.value) &&
    (roomState.value?.is_boost ?? room.value?.is_boost) === true &&
    !hasOwnedBoost.value,
)
const canCancelBoost = computed(() => canManageCurrentRound.value && hasOwnedBoost.value)
const canLeaveSeats = computed(() => canManageOwnedSeats.value)
const canLeaveRoom = computed(() => canManageOwnedSeats.value)
const randomReserveMax = computed(() => Math.max(1, remainingSeatCapacity.value))
const safeRandomReserveCount = computed(() => {
  const count = Number(randomReserveCount.value)
  if (!Number.isFinite(count) || count <= 0) return 1
  return Math.floor(count)
})
const joinBlockedHint = computed(() => {
  if (canJoinMoreSeats.value) {
    if (selectedSeats.value.length && seatSelectionLimitReached.value) {
      return t('gameRoom.entry.selectionLimitReached', {
        count: selectedSeats.value.length,
        limit: maxOwnSeats.value,
      })
    }
    return isJoined.value
      ? t('gameRoom.entry.addSeatHint', {
          count: ownedParticipants.value.length,
          limit: maxOwnSeats.value,
        })
      : t('gameRoom.entry.notJoinedHint')
  }
  if (isReviewingPreviousRound.value) return t('gameRoom.entry.waitForNextRound')
  if (currentRoundPhase.value !== 'waiting') return t('gameRoom.entry.actionsLocked')
  if (!roomCapacity.value) return t('gameRoom.entry.roomUnavailable')
  if (ownSeatsLimitReached.value) {
    return t('gameRoom.entry.seatLimitReached', { limit: maxOwnSeats.value })
  }
  if (freeSeatsCount.value <= 0) return t('gameRoom.entry.roomFull')
  return t('gameRoom.entry.notJoinedHint')
})
const reserveSeatsLabel = computed(() => {
  if (joining.value) return t('gameRoom.entry.joining')
  if (selectedSeats.value.length) {
    return t('gameRoom.entry.reserveSelected', { count: selectedSeats.value.length })
  }
  return t('gameRoom.entry.reserveRandomCount', { count: safeRandomReserveCount.value })
})
const boostHint = computed(() => {
  if (!isJoined.value) return t('gameRoom.errors.joinFirst')
  if (hasOwnedBoost.value && boostedParticipant.value) {
    return t('gameRoom.controls.boostActiveOnSeat', {
      seat: boostedParticipant.value.number_in_room,
    })
  }
  if (!canManageCurrentRound.value) return t('gameRoom.controls.actionsLocked')
  return canBoost.value ? t('gameRoom.controls.boostHint') : t('gameRoom.controls.boostDisabled')
})
const nextRoundAvailable = computed(
  () => Boolean(pendingNextRoundId.value && pendingNextRoundId.value !== activeRoundId.value),
)
const showNextRoundCountdown = computed(
  () => autoAdvanceNextRound.value && nextRoundAvailable.value && pendingNextRoundCountdown.value > 0,
)
const roundTimerValue = computed(() => roundStatus.value?.time_left_seconds ?? '-')
const waitingForFinishedStatus = computed(
  () =>
    normalizeRoundPhase(roundStatus.value?.status ?? '') === 'playing' &&
    Number(roundStatus.value?.time_left_seconds ?? 1) <= 0,
)

watch(
  () => [quickMatchMeta.value, room.value] as const,
  () => {
    if (joinResult.value || !quickMatchMeta.value) return

    joinResult.value = {
      participant_id: quickMatchMeta.value.round_participant_id,
      round_id: quickMatchMeta.value.round_id,
      number_in_room: quickMatchMeta.value.seat_number,
      room_capacity: room.value?.capacity ?? 0,
      current_players: room.value?.current_players ?? 0,
      min_players: room.value?.min_users ?? 0,
      entry_price: room.value?.registration_price ?? 0,
      round_status: '',
    }

    displayedRoundId.value = null
    clearPendingNextRound()
    successMsg.value = t('gameRoom.messages.quickSessionRestored')
    void loadRoomState(true)
    void loadRoundStatus(true)
  },
  { immediate: true },
)

watch(
  () => roomId.value,
  () => {
    displayedRoundId.value = null
    selectedSeats.value = []
    selectedBoostParticipantId.value = ''
    clearPendingNextRound()
    knownOwnedParticipantIds.value = []
    roundStatus.value = null
    liveEvents.value = []
    void loadRoomState(true)
  },
  { immediate: true },
)

watch(
  () => initialParticipantId.value,
  (participantId) => {
    rememberOwnedParticipantIds(participantId ? [participantId] : [])
  },
  { immediate: true },
)

watch(
  () => roomUserParticipants.value.map((participant) => participant.participant_id),
  (participantIds) => {
    rememberOwnedParticipantIds(participantIds)
  },
  { immediate: true },
)

watch(
  () =>
    participants.value
      .filter((participant) => participant.user_id && participant.user_id === currentUserId.value)
      .map((participant) => participant.participant_id),
  (participantIds) => {
    rememberOwnedParticipantIds(participantIds)
  },
  { immediate: true },
)

watch(
  ownedParticipants,
  (owned) => {
    const boosted = boostedParticipant.value
    if (boosted) {
      selectedBoostParticipantId.value = String(boosted.participant_id)
      return
    }

    const selectedId = Number(selectedBoostParticipantId.value)
    if (owned.some((participant) => participant.participant_id === selectedId)) return

    selectedBoostParticipantId.value = owned[0] ? String(owned[0].participant_id) : ''
  },
  { immediate: true },
)

watch([activeRoundId, autoRefresh, sseConnected, waitingForFinishedStatus], () => {
  restartStatusPolling()
})

watch(
  () => [roomId.value, roomRoundId.value] as const,
  () => {
    restartRoundEvents()
  },
  { immediate: true },
)

watch(
  () => autoAdvanceNextRound.value,
  (enabled) => {
    if (!pendingNextRoundId.value) return

    if (!enabled) {
      stopNextRoundCountdown()
      pendingNextRoundCountdown.value = 0
      return
    }

    if (pendingNextRoundAt.value === null) {
      if (nextRoundAvailable.value) {
        void goToNextRound()
      }
      return
    }

    startNextRoundCountdown()
  },
)

watch(
  () => [
    activeRoundId.value,
    roomRoundId.value,
    remainingSeatCapacity.value,
    [...occupiedSeats.value].join(','),
  ] as const,
  () => {
    if (!canJoinMoreSeats.value) {
      selectedSeats.value = []
      randomReserveCount.value = 1
      return
    }

    selectedSeats.value = selectedSeats.value
      .filter((seat) => !occupiedSeats.value.has(seat))
      .slice(0, remainingSeatCapacity.value)
    randomReserveCount.value = Math.min(
      Math.max(1, randomReserveCount.value),
      Math.max(1, remainingSeatCapacity.value),
    )
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  stopStatusPolling()
  stopRoundEvents()
  stopNextRoundCountdown()
})

function backToHome() {
  router.push('/')
}

function openRooms() {
  router.push({
    path: '/rooms',
    query: filtersToQuery(session.filters ?? {}),
  })
}

function formatBoost() {
  const isBoost = roomState.value?.is_boost ?? room.value?.is_boost
  const boostPower = roomState.value?.boost_power ?? room.value?.boost_power
  if (!isBoost) return t('common.off')
  return t('matchmaking.results.boostValue', { value: boostPower ?? 0 })
}

function formatScore() {
  if (!room.value) return '-'
  return room.value.score.toFixed(2)
}

function normalizeError(error: any, fallback: string) {
  const code = error?.details?.code
  if (code === 'BOOST_ALREADY_PURCHASED') return t('gameRoom.errors.boostAlreadyPurchased')
  if (code === 'BOOST_DISABLED') return t('gameRoom.errors.boostDisabled')
  if (code === 'GAME_STARTED') return t('gameRoom.errors.gameStarted')
  if (code === 'MAX_SEATS_EXCEEDED') return t('gameRoom.errors.maxSeatsExceeded')
  return error?.message || error?.details?.message || error?.details?.error || fallback
}

function errorCode(error: any) {
  return String(error?.details?.code || '')
}

function isRecoverableRandomJoinError(error: any) {
  const code = errorCode(error)
  return code === 'ROOM_FULL' || code === 'MAX_SEATS_EXCEEDED' || code === 'ROUND_NOT_JOINABLE'
}

function normalizeRoundPhase(status: string) {
  const normalized = String(status || '').toLowerCase()
  if (['completed', 'complete', 'finished', 'finalized', 'archived'].includes(normalized)) {
    return 'finalized'
  }
  if (['in_game', 'in-game', 'started', 'running', 'active', 'playing'].includes(normalized)) {
    return 'playing'
  }
  return 'waiting'
}

function clearFeedback() {
  errorMsg.value = ''
  successMsg.value = ''
}

function clearInitialJoinContext() {
  joinResult.value = null
  if (session.source === 'quick-match') {
    session.clearQuickMatchMeta()
  }
}

function rememberOwnedParticipantIds(participantIds: number[]) {
  if (!participantIds.length) return

  const nextIds = new Set(knownOwnedParticipantIds.value)
  for (const participantId of participantIds) {
    if (participantId > 0) nextIds.add(participantId)
  }
  knownOwnedParticipantIds.value = [...nextIds]
}

function refreshCabinetBalance() {
  void cabinet.refresh().catch(() => {})
}

function participantState(participant: GameParticipantInfo) {
  if (participant.exited_at) return t('gameRoom.participant.exited')
  if (participant.is_bot) return t('gameRoom.participant.bot')
  return t('gameRoom.participant.active')
}

function canSelectSeat(seat: number) {
  if (occupiedSeats.value.has(seat)) return false
  if (selectedSeatNumbers.value.has(seat)) return true
  return !seatSelectionLimitReached.value
}

function toggleSeatSelection(seat: number) {
  if (!canJoinMoreSeats.value) return

  const alreadySelected = selectedSeatNumbers.value.has(seat)
  if (!alreadySelected && !canSelectSeat(seat)) return

  if (alreadySelected) {
    selectedSeats.value = selectedSeats.value.filter((value) => value !== seat)
    return
  }

  selectedSeats.value = [...selectedSeats.value, seat].sort((left, right) => left - right)
}

function stopNextRoundCountdown() {
  if (nextRoundTimer !== null) {
    window.clearInterval(nextRoundTimer)
    nextRoundTimer = null
  }
}

function clearPendingNextRound() {
  stopNextRoundCountdown()
  pendingNextRoundId.value = null
  pendingNextRoundCountdown.value = 0
  pendingNextRoundAt.value = null
}

function stopStatusPolling() {
  if (statusTimer !== null) {
    window.clearInterval(statusTimer)
    statusTimer = null
  }
}

function restartStatusPolling() {
  stopStatusPolling()

  if (!activeRoundId.value || !autoRefresh.value) return
  if (sseConnected.value && !waitingForFinishedStatus.value) return

  const interval = waitingForFinishedStatus.value
    ? FINISHED_STATUS_POLLING_INTERVAL_MS
    : DEFAULT_STATUS_POLLING_INTERVAL_MS
  statusTimer = window.setInterval(() => {
    void refreshRoundView(true)
  }, interval)
}

function stopRoundEvents() {
  if (roundEventsStop) {
    roundEventsStop()
    roundEventsStop = null
  }
  sseConnected.value = false
}

function restartRoundEvents() {
  stopRoundEvents()
  sseError.value = ''

  if (!roomRoundId.value) return

  roundEventsStop = GameApi.subscribeRoundEvents(roomId.value, roomRoundId.value, {
    onOpen: () => {
      sseConnected.value = true
      sseError.value = ''
      void loadRoomState(true)
      if (activeRoundId.value === roomRoundId.value) {
        void loadRoundStatus(true)
      }
    },
    onEvent: handleRoundEvent,
    onError: (error) => {
      sseConnected.value = false
      sseError.value = normalizeError(error, t('gameRoom.errors.events'))
    },
    onClose: () => {
      sseConnected.value = false
    },
  })
}

function handleRoundEvent(event: GameRoundEvent) {
  if (event.type === 'round_timer') {
    const data = eventData(event)
    const status = String(data.status || roundStatus.value?.status || 'waiting')
    const secondsLeft = Number(data.seconds_left ?? roundStatus.value?.time_left_seconds ?? 0) || 0
    if (roundStatus.value && activeRoundId.value === roomRoundId.value) {
      roundStatus.value = {
        ...roundStatus.value,
        status,
        time_left_seconds: secondsLeft,
      }
    }
    if (normalizeRoundPhase(status) === 'playing' && secondsLeft <= 0) {
      void refreshRoundView(true)
    }
    return
  }

  liveEvents.value = [{ ...event, id: ++liveEventId }, ...liveEvents.value].slice(0, 5)

  if (event.type === 'round_finalized') {
    const data = eventData(event)
    const finalizedRoundId = Number(data.round_id ?? activeRoundId.value ?? 0) || 0
    if (roundStatus.value && roundStatus.value.round_id === finalizedRoundId) {
      roundStatus.value = {
        ...roundStatus.value,
        status: 'finished',
        time_left_seconds: 0,
      }
    }
    stopStatusPolling()
    if (finalizedRoundId > 0) {
      displayedRoundId.value = finalizedRoundId
    }
    scheduleNextRoundTransition(
      Number(data.next_round_id ?? roomRoundId.value ?? 0) || null,
      Number(data.next_round_delay ?? roomState.value?.next_round_delay ?? 0) || 0,
      event.timestamp,
    )
  }

  if (activeRoundId.value) {
    void loadRoundStatus(true)
  }
  if (['player_joined', 'player_left', 'round_finalized', 'round_started'].includes(event.type)) {
    void loadRoomState(true)
  }
  if (['boost_purchased', 'boost_cancelled', 'player_left', 'round_finalized'].includes(event.type)) {
    refreshCabinetBalance()
  }
}

function formatEventTime(timestamp: string) {
  const date = new Date(timestamp)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleTimeString()
}

function eventTitle(event: GameRoundEvent) {
  const key = `gameRoom.events.${event.type}`
  const label = t(key)
  if (label !== key) return label

  return event.type.replace(/_/g, ' ')
}

function eventData(event: GameRoundEvent) {
  if (!event.data || typeof event.data !== 'object') return {}
  return event.data as Record<string, unknown>
}

function eventDescription(event: GameRoundEvent) {
  const data = eventData(event)

  if (event.type === 'player_joined') {
    return t('gameRoom.events.playerJoinedDetails', {
      participant: Number(data.participant_id ?? 0) || '-',
      seat: Number(data.number_in_room ?? 0) || '-',
      players: Number(data.current_players ?? 0) || '-',
    })
  }

  if (event.type === 'player_left') {
    return t('gameRoom.events.playerLeftDetails', {
      participant: Number(data.participant_id ?? 0) || '-',
      seat: Number(data.number_in_room ?? 0) || '-',
      players: Number(data.current_players ?? 0) || '-',
    })
  }

  if (event.type === 'boost_purchased') {
    return t('gameRoom.events.boostPurchasedDetails', {
      participant: Number(data.participant_id ?? 0) || '-',
      power: Number(data.boost_power ?? 0) || '-',
    })
  }

  if (event.type === 'boost_cancelled') {
    return t('gameRoom.events.boostCancelledDetails', {
      participant: Number(data.participant_id ?? 0) || '-',
    })
  }

  if (event.type === 'round_started') {
    return t('gameRoom.events.roundStartedDetails', {
      players: Number(data.final_players ?? 0) || '-',
      seconds: Number(data.game_duration_sec ?? 0) || '-',
    })
  }

  if (event.type === 'round_finalized') {
    const winnersCount = Array.isArray(data.winners) ? data.winners.length : 0
    return t('gameRoom.events.roundFinalizedDetails', { winners: winnersCount })
  }

  if (event.type === 'round_timer') {
    return `${String(data.status || 'waiting')} - ${Number(data.seconds_left ?? 0) || 0}s`
  }

  return t('gameRoom.events.genericDetails')
}

async function loadRoomState(silent = false) {
  if (!roomId.value) return

  try {
    roomState.value = await GameApi.getRoomState(roomId.value)
    syncLiveEventsFromRoomState()
    syncPendingNextRoundFromState()
  } catch (error: any) {
    if (!silent) {
      errorMsg.value = normalizeError(error, t('gameRoom.errors.status'))
    }
  }
}

async function loadRoundStatus(silent = false) {
  if (!activeRoundId.value) return

  if (!silent) {
    statusLoading.value = true
    errorMsg.value = ''
  }

  try {
    roundStatus.value = await GameApi.getRoundStatus(roomId.value, activeRoundId.value)
    syncPendingNextRoundFromState()
  } catch (error: any) {
    if (!silent) {
      errorMsg.value = normalizeError(error, t('gameRoom.errors.status'))
    }
  } finally {
    if (!silent) {
      statusLoading.value = false
    }
  }
}

async function refreshRoundView(silent = false) {
  if (!roomId.value) return

  if (!silent) {
    statusLoading.value = true
    errorMsg.value = ''
  }

  try {
    await loadRoomState(true)
    if (activeRoundId.value) {
      roundStatus.value = await GameApi.getRoundStatus(roomId.value, activeRoundId.value)
      syncPendingNextRoundFromState()
    } else {
      roundStatus.value = null
    }
  } catch (error: any) {
    if (!silent) {
      errorMsg.value = normalizeError(error, t('gameRoom.errors.status'))
    }
  } finally {
    if (!silent) {
      statusLoading.value = false
    }
  }
}

async function reserveSeats() {
  clearFeedback()

  if (!canJoinMoreSeats.value) {
    errorMsg.value = joinBlockedHint.value
    return
  }

  joining.value = true
  const requestedSeats = [...selectedSeats.value]
  const reservedSeats: number[] = []
  const requestedRandomCount = requestedSeats.length
    ? 0
    : Math.min(safeRandomReserveCount.value, randomReserveMax.value)
  const targetRandomCount = Math.min(requestedRandomCount || 1, remainingSeatCapacity.value, freeSeatsCount.value)

  try {
    if (!requestedSeats.length) {
      if (targetRandomCount <= 0) {
        throw new Error(t('gameRoom.entry.roomFull'))
      }

      for (let attempt = 0; attempt < targetRandomCount; attempt += 1) {
        try {
          joinResult.value = await GameApi.joinRoom(roomId.value)
          rememberOwnedParticipantIds([joinResult.value.participant_id])
          reservedSeats.push(joinResult.value.number_in_room)
        } catch (error: any) {
          if (reservedSeats.length > 0 && isRecoverableRandomJoinError(error)) {
            break
          }
          throw error
        }
      }
    } else {
      for (const seat of requestedSeats) {
        if (occupiedSeats.value.has(seat)) {
          throw new Error(t('gameRoom.errors.seatTaken'))
        }

        const response = await GameApi.joinRoomWithSeat(roomId.value, { number_in_room: seat })
        joinResult.value = response
        rememberOwnedParticipantIds([response.participant_id])
        reservedSeats.push(response.number_in_room)
      }
    }

    selectedSeats.value = []
    randomReserveCount.value = 1
    displayedRoundId.value = null
    clearPendingNextRound()
    successMsg.value = !requestedSeats.length && reservedSeats.length < requestedRandomCount
      ? t('gameRoom.messages.joinedPartialAvailable', {
          count: reservedSeats.length,
          requested: requestedRandomCount,
          seats: reservedSeats.join(', '),
        })
      : reservedSeats.length > 1
        ? t('gameRoom.messages.joinedMultiple', {
            count: reservedSeats.length,
            seats: reservedSeats.join(', '),
          })
        : t('gameRoom.messages.joined', { seat: reservedSeats[0] ?? joinResult.value?.number_in_room ?? '-' })
    refreshCabinetBalance()
    await refreshRoundView()
  } catch (error: any) {
    const fallback = requestedSeats.length ? t('gameRoom.errors.joinSeat') : t('gameRoom.errors.join')
    const message = normalizeError(error, fallback)
    errorMsg.value =
      reservedSeats.length > 0
        ? t('gameRoom.errors.partialSeatReservation', {
            count: reservedSeats.length,
            seats: reservedSeats.join(', '),
            error: message,
          })
        : message
    await refreshRoundView(true)
  } finally {
    joining.value = false
  }
}

async function purchaseBoost() {
  clearFeedback()

  const participant = selectedBoostParticipant.value
  if (!participant) {
    errorMsg.value = t('gameRoom.errors.joinFirst')
    return
  }
  if (hasOwnedBoost.value) {
    errorMsg.value = t('gameRoom.errors.boostAlreadyPurchased')
    return
  }

  actionLoading.value = 'boost'

  try {
    const response = await GameApi.purchaseBoost(roomId.value, participant.participant_id)
    successMsg.value = t('gameRoom.messages.boostPurchased', {
      power: response.boost_power,
      cost: response.boost_cost,
    })
    refreshCabinetBalance()
    await refreshRoundView()
  } catch (error: any) {
    errorMsg.value = normalizeError(error, t('gameRoom.errors.boost'))
  } finally {
    actionLoading.value = ''
  }
}

async function cancelBoost() {
  clearFeedback()

  const participant = boostedParticipant.value
  if (!participant) {
    errorMsg.value = t('gameRoom.errors.joinFirst')
    return
  }

  actionLoading.value = 'cancel-boost'

  try {
    const response = await GameApi.cancelBoost(roomId.value, participant.participant_id)
    successMsg.value = t('gameRoom.messages.boostCancelled', { refund: response.refund ?? 0 })
    refreshCabinetBalance()
    await refreshRoundView()
  } catch (error: any) {
    errorMsg.value = normalizeError(error, t('gameRoom.errors.cancelBoost'))
  } finally {
    actionLoading.value = ''
  }
}

async function leaveParticipant(participant: GameParticipantInfo) {
  clearFeedback()

  if (!canLeaveSeats.value) {
    errorMsg.value = isJoined.value ? t('gameRoom.controls.actionsLocked') : t('gameRoom.errors.joinFirst')
    return
  }

  const confirmed = window.confirm(
    t('gameRoom.confirmLeaveSeat', { seat: participant.number_in_room }),
  )
  if (!confirmed) return

  actionLoading.value = `leave-${participant.participant_id}`

  try {
    await GameApi.leaveParticipant(roomId.value, participant.participant_id)
    successMsg.value = t('gameRoom.messages.leftSeat', { seat: participant.number_in_room })
    if (joinResult.value?.participant_id === participant.participant_id) {
      clearInitialJoinContext()
    }
    displayedRoundId.value = null
    clearPendingNextRound()
    await refreshRoundView(true)
    if (!activeRoundId.value) {
      roundStatus.value = null
      stopStatusPolling()
      stopRoundEvents()
    }
    refreshCabinetBalance()
  } catch (error: any) {
    errorMsg.value = normalizeError(error, t('gameRoom.errors.leave'))
  } finally {
    actionLoading.value = ''
  }
}

async function leaveRoomFully() {
  clearFeedback()

  if (!canLeaveRoom.value) {
    errorMsg.value = isJoined.value ? t('gameRoom.controls.actionsLocked') : t('gameRoom.errors.joinFirst')
    return
  }

  const confirmed = window.confirm(t('gameRoom.confirmLeaveRoom'))
  if (!confirmed) return

  actionLoading.value = 'leave-room'

  try {
    const response = await GameApi.leaveRoom(roomId.value)
    clearInitialJoinContext()
    selectedSeats.value = []
    randomReserveCount.value = 1
    displayedRoundId.value = null
    clearPendingNextRound()
    await refreshRoundView(true)
    successMsg.value = t('gameRoom.messages.leftRoom', { refund: response.refund ?? 0 })
    refreshCabinetBalance()
  } catch (error: any) {
    errorMsg.value = normalizeError(error, t('gameRoom.errors.leave'))
  } finally {
    actionLoading.value = ''
  }
}

async function goToNextRound() {
  if (!pendingNextRoundId.value) return

  displayedRoundId.value = null
  selectedSeats.value = []
  clearPendingNextRound()
  await refreshRoundView(true)
}

function syncLiveEventsFromRoomState() {
  const events = roomState.value?.recent_events ?? []
  if (!events.length) {
    liveEvents.value = []
    return
  }

  liveEvents.value = events.slice(0, 5).map((event) => ({
    ...event,
    id: ++liveEventId,
  }))
}

function syncPendingNextRoundFromState() {
  if (!roundStatus.value || roundStatus.value.status !== 'finished') {
    if (!nextRoundAvailable.value) {
      clearPendingNextRound()
    }
    return
  }
  if (!roomRoundId.value || activeRoundId.value === roomRoundId.value) {
    clearPendingNextRound()
    return
  }

  const history = roomState.value?.recent_events ?? []
  const finalizedEvent = history.find((event) => {
    if (event.type !== 'round_finalized') return false
    const data = eventData(event)
    return Number(data.round_id ?? 0) === activeRoundId.value
  })

  if (!finalizedEvent) {
    pendingNextRoundId.value = roomRoundId.value
    pendingNextRoundCountdown.value = 0
    return
  }

  const data = eventData(finalizedEvent)
  scheduleNextRoundTransition(
    Number(data.next_round_id ?? roomRoundId.value ?? 0) || null,
    Number(data.next_round_delay ?? roomState.value?.next_round_delay ?? 0) || 0,
    finalizedEvent.timestamp,
  )
}

function startNextRoundCountdown() {
  stopNextRoundCountdown()

  if (!autoAdvanceNextRound.value || !pendingNextRoundId.value || pendingNextRoundAt.value === null) {
    pendingNextRoundCountdown.value = 0
    return
  }

  const tick = () => {
    if (pendingNextRoundAt.value === null) {
      stopNextRoundCountdown()
      pendingNextRoundCountdown.value = 0
      return
    }

    const secondsLeft = Math.max(0, Math.ceil((pendingNextRoundAt.value - Date.now()) / 1000))
    pendingNextRoundCountdown.value = secondsLeft
    if (secondsLeft > 0) return

    stopNextRoundCountdown()
    void goToNextRound()
  }

  tick()
  if (pendingNextRoundAt.value > Date.now()) {
    nextRoundTimer = window.setInterval(tick, 1000)
  }
}

function scheduleNextRoundTransition(nextRoundId: number | null, nextRoundDelay: number, timestamp: string) {
  clearPendingNextRound()

  if (!nextRoundId || nextRoundId <= 0) return

  pendingNextRoundId.value = nextRoundId

  const finalizedAt = new Date(timestamp).getTime()
  const baseTime = Number.isFinite(finalizedAt) ? finalizedAt : Date.now()
  pendingNextRoundAt.value = baseTime + Math.max(0, nextRoundDelay) * 1000
  startNextRoundCountdown()
}
</script>

<template>
  <UpTab :show-menu="false" :show-upload="false" />
  <LeftTab />

  <main class="play-area" :class="{ collapsed: leftHidden }" :style="{ '--layout-inset': layoutInset }">
    <section class="hero-card">
      <div>
        <p class="eyebrow">{{ t('matchmaking.play.eyebrow') }}</p>
        <h1>{{ t('matchmaking.play.title', { roomId }) }}</h1>
        <p class="description">{{ t('gameRoom.prototypeDescription') }}</p>
      </div>
      <div class="hero-pills">
        <span class="source-pill">{{ sourceLabel }}</span>
        <span class="source-pill" :class="{ joined: isJoined }">
          {{ isJoined ? t('gameRoom.state.joined') : t('gameRoom.state.notJoined') }}
        </span>
      </div>
    </section>

    <div v-if="successMsg || errorMsg" class="feedback-bar" :class="{ error: errorMsg }">
      {{ errorMsg || successMsg }}
    </div>

    <section class="play-grid">
      <article class="panel-card">
        <div class="card-head">
          <div>
            <p class="eyebrow accent">{{ t('gameRoom.entry.eyebrow') }}</p>
            <h2>{{ t('gameRoom.entry.title') }}</h2>
            <p class="description">{{ t('gameRoom.entry.description') }}</p>
          </div>
        </div>

        <div class="meta-grid">
          <div class="meta-item">
            <span>{{ t('matchmaking.play.meta.entry') }}</span>
            <strong>{{ entryPrice }}</strong>
          </div>
          <div class="meta-item">
            <span>{{ t('matchmaking.play.meta.capacity') }}</span>
            <strong>{{ roomCapacity || '-' }}</strong>
          </div>
          <div class="meta-item">
            <span>{{ t('matchmaking.play.meta.players') }}</span>
            <strong>{{ currentPlayers }}</strong>
          </div>
          <div class="meta-item">
            <span>{{ t('matchmaking.play.meta.minimumUsers') }}</span>
            <strong>{{ minPlayers }}</strong>
          </div>
          <div class="meta-item">
            <span>{{ t('matchmaking.play.meta.boost') }}</span>
            <strong>{{ formatBoost() }}</strong>
          </div>
          <div class="meta-item">
            <span>{{ t('matchmaking.play.meta.score') }}</span>
            <strong>{{ formatScore() }}</strong>
          </div>
        </div>

        <div class="join-box" :class="{ joined: isJoined }">
          <div v-if="isJoined" class="own-seats">
            <p class="join-title">
              {{ t('gameRoom.entry.ownedSeatsTitle', { count: ownedParticipants.length }) }}
            </p>
            <p class="description">
              {{ t('gameRoom.entry.ownedSeatsHint', { limit: maxOwnSeats }) }}
            </p>

            <div class="own-seat-list">
              <article
                v-for="participant in ownedParticipants"
                :key="participant.participant_id"
                class="own-seat-card"
              >
                <div>
                  <strong>{{ t('gameRoom.participant.seat', { seat: participant.number_in_room }) }}</strong>
                  <span>{{ t('gameRoom.participant.id', { id: participant.participant_id }) }}</span>
                  <span v-if="participant.boost > 0">
                    {{ t('gameRoom.entry.boostActiveSeat', { seat: participant.number_in_room }) }}
                  </span>
                </div>
                <button
                  class="btn btn--danger"
                  type="button"
                  :disabled="!canLeaveSeats || actionLoading === `leave-${participant.participant_id}`"
                  @click="leaveParticipant(participant)"
                >
                  {{
                    actionLoading === `leave-${participant.participant_id}`
                      ? t('common.loading')
                      : t('gameRoom.entry.releaseSeat')
                  }}
                </button>
              </article>
            </div>
          </div>

          <div class="join-flow">
            <p class="join-title">
              {{ isJoined ? t('gameRoom.entry.addSeatTitle') : t('gameRoom.entry.notJoinedTitle') }}
            </p>
            <p class="description">{{ joinBlockedHint }}</p>

            <template v-if="canJoinMoreSeats">
              <label class="seat-input">
                <span>{{ t('gameRoom.entry.randomSeatsCount') }}</span>
                <input
                  v-model.number="randomReserveCount"
                  type="number"
                  min="1"
                  :max="randomReserveMax"
                  :disabled="Boolean(selectedSeats.length)"
                />
              </label>

              <div v-if="seatOptions.length" class="seat-grid" :aria-label="t('gameRoom.entry.seats')">
                <button
                  v-for="seat in seatOptions"
                  :key="seat"
                  class="seat-button"
                  :class="{
                    occupied: occupiedSeats.has(seat),
                    selected: selectedSeatNumbers.has(seat),
                    locked: !occupiedSeats.has(seat) && !selectedSeatNumbers.has(seat) && !canSelectSeat(seat),
                    'own-seat': ownedSeatNumbers.has(seat),
                  }"
                  type="button"
                  :disabled="!canSelectSeat(seat)"
                  @click="toggleSeatSelection(seat)"
                >
                  {{ seat }}
                </button>
              </div>

              <p v-if="selectedSeats.length" class="description">
                {{ t('gameRoom.entry.selectedSeats', { seats: selectedSeats.join(', ') }) }}
              </p>
              <p v-else class="description">
                {{ t('gameRoom.entry.randomSeatsHint', { count: safeRandomReserveCount }) }}
              </p>

              <div class="actions stretch">
                <button class="btn btn--primary" type="button" :disabled="joining" @click="reserveSeats">
                  {{ reserveSeatsLabel }}
                </button>
              </div>
            </template>
          </div>
        </div>
      </article>

      <article class="panel-card">
        <div class="card-head row">
          <div>
            <p class="eyebrow">{{ t('gameRoom.round.eyebrow') }}</p>
            <h2>{{ t('gameRoom.round.title') }}</h2>
            <p class="description">{{ t('gameRoom.round.description') }}</p>
          </div>
          <button class="btn" type="button" :disabled="!activeRoundId || statusLoading" @click="refreshRoundView()">
            {{ statusLoading ? t('common.loading') : t('common.refresh') }}
          </button>
        </div>

        <div class="meta-grid">
          <div class="meta-item">
            <span>{{ t('matchmaking.play.meta.roundId') }}</span>
            <strong>{{ activeRoundId ? `#${activeRoundId}` : '-' }}</strong>
          </div>
          <div class="meta-item">
            <span>{{ t('gameRoom.round.status') }}</span>
            <strong>{{ statusLabel }}</strong>
          </div>
          <div class="meta-item">
            <span>{{ t('gameRoom.round.timer') }}</span>
            <strong>{{ roundTimerValue }}</strong>
          </div>
          <div class="meta-item">
            <span>{{ t('gameRoom.round.fallbackPolling') }}</span>
            <label class="switch-line">
              <input v-model="autoRefresh" type="checkbox" />
              <span>{{ autoRefresh ? t('common.yes') : t('common.no') }}</span>
            </label>
          </div>
          <div class="meta-item">
            <span>{{ t('gameRoom.round.autoAdvanceNext') }}</span>
            <label class="switch-line">
              <input v-model="autoAdvanceNextRound" type="checkbox" />
              <span>{{ autoAdvanceNextRound ? t('common.yes') : t('common.no') }}</span>
            </label>
          </div>
          <div class="meta-item">
            <span>{{ t('gameRoom.round.liveStatus') }}</span>
            <strong :class="{ live: sseConnected }">
              {{ sseConnected ? t('gameRoom.round.liveConnected') : t('gameRoom.round.liveDisconnected') }}
            </strong>
          </div>
          <div v-if="showNextRoundCountdown" class="meta-item">
            <span>{{ t('gameRoom.round.nextRoundTimer') }}</span>
            <strong>{{ pendingNextRoundCountdown }}s</strong>
          </div>
        </div>

        <p class="description live-hint">
          {{ sseConnected ? t('gameRoom.round.ssePrimary') : t('gameRoom.round.pollingFallback') }}
        </p>

        <div class="round-timeline" :aria-label="t('gameRoom.round.timeline')">
          <div
            v-for="phase in roundPhases"
            :key="phase.key"
            class="phase-step"
            :class="{ active: phase.active, complete: phase.complete }"
          >
            <strong>{{ phase.title }}</strong>
            <span>{{ phase.description }}</span>
          </div>
        </div>

        <div class="participants">
          <h3>{{ t('gameRoom.round.participants') }}</h3>
          <p v-if="!participants.length" class="description">{{ t('gameRoom.round.empty') }}</p>
          <div v-else class="participant-list">
            <div
              v-for="participant in participants"
              :key="participant.participant_id"
              class="participant-card"
              :class="{ own: ownedParticipantIds.has(participant.participant_id) }"
            >
              <span v-if="ownedParticipantIds.has(participant.participant_id)" class="own-badge">
                {{ t('gameRoom.participant.you') }}
              </span>
              <strong>{{ t('gameRoom.participant.seat', { seat: participant.number_in_room }) }}</strong>
              <span>{{ t('gameRoom.participant.id', { id: participant.participant_id }) }}</span>
              <span v-if="participant.nickname">
                {{ participant.nickname }}
              </span>
              <span v-else-if="participant.user_id">
                {{ t('gameRoom.participant.user', { id: participant.user_id }) }}
              </span>
              <span>{{ participantState(participant) }}</span>
              <span>{{ t('gameRoom.participant.boost', { value: participant.boost }) }}</span>
              <span v-if="participant.winning_money">
                {{ t('gameRoom.participant.winning', { value: participant.winning_money }) }}
              </span>
            </div>
          </div>
        </div>

        <div v-if="winners.length" class="winners">
          <h3>{{ t('gameRoom.round.winners') }}</h3>
          <span v-for="winner in winners" :key="winner.participant_id" class="winner-chip">
            {{ t('gameRoom.participant.seat', { seat: winner.number_in_room }) }} -
            {{ winner.winning_money }}
          </span>
        </div>

        <div v-if="nextRoundAvailable" class="next-round-box">
          <div>
            <h3>{{ t('gameRoom.round.nextRoundTitle') }}</h3>
            <p class="description" v-if="showNextRoundCountdown">
              {{ t('gameRoom.round.nextRoundHint', { seconds: pendingNextRoundCountdown }) }}
            </p>
            <p class="description" v-else>
              {{ t('gameRoom.round.nextRoundManualHint') }}
            </p>
          </div>
          <button class="btn btn--primary" type="button" @click="goToNextRound">
            {{ t('gameRoom.round.goToNextRound') }}
          </button>
        </div>

        <div class="live-events">
          <h3>{{ t('gameRoom.round.liveEvents') }}</h3>
          <p v-if="sseError" class="description">{{ sseError }}</p>
          <p v-else-if="!liveEvents.length" class="description">{{ t('gameRoom.round.noLiveEvents') }}</p>
          <div v-else class="event-list">
            <article v-for="event in liveEvents" :key="event.id" class="event-card">
              <div>
                <strong>{{ eventTitle(event) }}</strong>
                <span>{{ eventDescription(event) }}</span>
              </div>
              <time>{{ formatEventTime(event.timestamp) }}</time>
            </article>
          </div>
        </div>
      </article>
    </section>

    <section class="panel-card controls-card">
      <div class="card-head">
        <div>
          <p class="eyebrow accent">{{ t('gameRoom.controls.eyebrow') }}</p>
          <h2>{{ t('gameRoom.controls.title') }}</h2>
          <p class="description">{{ t('gameRoom.controls.description') }}</p>
        </div>
      </div>

      <div class="control-grid">
        <div class="control-block">
          <h3>{{ t('gameRoom.controls.boostTitle') }}</h3>
          <p class="description">
            {{ boostHint }}
          </p>
          <p class="boost-value">
            <span>{{ t('gameRoom.controls.boostValue') }}</span>
            <strong>{{ formatBoost() }}</strong>
          </p>
          <label class="seat-input">
            <span>{{ t('gameRoom.controls.boostSeat') }}</span>
            <select v-model="selectedBoostParticipantId" :disabled="!ownedParticipants.length || hasOwnedBoost">
              <option value="">{{ t('gameRoom.controls.boostSeatPlaceholder') }}</option>
              <option
                v-for="participant in ownedParticipants"
                :key="participant.participant_id"
                :value="String(participant.participant_id)"
              >
                {{
                  t('gameRoom.controls.boostSeatOption', {
                    seat: participant.number_in_room,
                    participant: participant.participant_id,
                  })
                }}
              </option>
            </select>
          </label>
          <div class="actions stretch">
            <button
              class="btn btn--primary"
              type="button"
              :disabled="!canBoost || actionLoading === 'boost'"
              @click="purchaseBoost"
            >
              {{ actionLoading === 'boost' ? t('common.loading') : t('gameRoom.controls.buyBoost') }}
            </button>
            <button
              class="btn"
              type="button"
              :disabled="!canCancelBoost || actionLoading === 'cancel-boost'"
              @click="cancelBoost"
            >
              {{ actionLoading === 'cancel-boost' ? t('common.loading') : t('gameRoom.controls.cancelBoost') }}
            </button>
          </div>
        </div>

        <div class="control-block">
          <h3>{{ t('gameRoom.controls.roomTitle') }}</h3>
          <p class="description">{{ t('gameRoom.controls.roomHint') }}</p>
          <div class="actions stretch">
            <button
              class="btn btn--danger"
              type="button"
              :disabled="!canLeaveRoom || actionLoading === 'leave-room'"
              @click="leaveRoomFully"
            >
              {{ actionLoading === 'leave-room' ? t('common.loading') : t('gameRoom.controls.leaveRoom') }}
            </button>
            <button class="btn" type="button" @click="backToHome">
              {{ t('matchmaking.play.backHome') }}
            </button>
            <button class="btn" type="button" @click="openRooms">
              {{ t('matchmaking.play.backRooms') }}
            </button>
          </div>
        </div>
      </div>
    </section>
  </main>

  <FooterTab />
</template>

<style scoped>
.play-area {
  position: fixed;
  inset: var(--layout-inset, 92px 20px 20px 304px);
  display: grid;
  gap: 1rem;
  overflow: auto;
  align-content: start;
  transition: all var(--transition-slow) ease;
}

.play-area.collapsed {
  --layout-inset: 92px 20px 20px 120px;
}

.hero-card,
.panel-card,
.feedback-bar {
  padding: 1.35rem;
  border-radius: 1.6rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
  box-shadow: var(--shadow-md);
}

.hero-card {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 1rem;
  background:
    radial-gradient(circle at top right, rgba(245, 158, 11, 0.18), transparent 24%),
    linear-gradient(
      135deg,
      color-mix(in oklab, var(--color-bg-secondary), white 18%),
      color-mix(in oklab, var(--color-surface), transparent 6%)
    );
}

.hero-pills,
.actions,
.winner-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.65rem;
  flex-wrap: wrap;
}

.play-grid,
.control-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
}

.panel-card,
.control-block,
.join-box,
.next-round-box {
  display: grid;
  gap: 1rem;
}

.panel-card {
  background:
    radial-gradient(circle at top left, color-mix(in oklab, #0ea5e9, white 88%), transparent 28%),
    linear-gradient(180deg, color-mix(in oklab, var(--color-surface), white 14%), var(--color-surface));
}

.card-head {
  display: grid;
  gap: 0.5rem;
}

.card-head.row {
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: start;
}

.eyebrow {
  margin: 0;
  font-size: 0.72rem;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: #0369a1;
}

.eyebrow.accent {
  color: #b45309;
}

h1,
h2,
h3,
p,
dl {
  margin: 0;
}

.description,
.meta-item span,
.participant-card span,
label span,
.boost-value span {
  color: var(--color-muted);
}

.source-pill,
.winner-chip,
.own-badge {
  justify-content: center;
  border-radius: 999px;
  padding: 0.55rem 0.85rem;
  background: color-mix(in oklab, var(--color-surface), white 10%);
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
  font-weight: 600;
}

.source-pill.joined {
  background: color-mix(in oklab, var(--color-success), transparent 82%);
}

strong.live {
  color: var(--color-success);
}

.feedback-bar {
  background: color-mix(in oklab, var(--color-success), transparent 86%);
  color: var(--color-success);
}

.feedback-bar.error {
  background: color-mix(in oklab, var(--color-danger), transparent 88%);
  color: var(--color-danger);
}

.meta-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.85rem;
}

.live-hint {
  padding: 0.75rem 0.9rem;
  border-radius: 1rem;
  background: color-mix(in oklab, #0ea5e9, transparent 90%);
}

.round-timeline {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.65rem;
}

.phase-step {
  display: grid;
  gap: 0.35rem;
  min-height: 5.25rem;
  padding: 0.85rem;
  border-radius: 1rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
  background: color-mix(in oklab, var(--color-surface), white 8%);
}

.phase-step.active {
  border-color: color-mix(in oklab, #0284c7, transparent 20%);
  background:
    radial-gradient(circle at top right, color-mix(in oklab, #0ea5e9, transparent 78%), transparent 52%),
    color-mix(in oklab, var(--color-surface), white 12%);
}

.phase-step.complete {
  border-color: color-mix(in oklab, var(--color-success), transparent 25%);
}

.phase-step span {
  color: var(--color-muted);
}

.meta-item,
.participant-card,
.control-block,
.join-box,
.next-round-box {
  padding: 0.85rem;
  border-radius: 1rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  background: color-mix(in oklab, var(--color-surface), white 10%);
}

.join-box.joined,
.next-round-box {
  border-color: color-mix(in oklab, var(--color-success), transparent 28%);
}

.next-round-box {
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
}

.join-title {
  font-weight: 700;
}

.seat-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(2.65rem, 1fr));
  gap: 0.5rem;
}

.seat-button {
  appearance: none;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  background: color-mix(in oklab, var(--color-surface), white 8%);
  color: var(--color-text);
  border-radius: 0.8rem;
  min-height: 2.65rem;
  font-weight: 700;
  cursor: pointer;
}

.seat-button.selected {
  border-color: color-mix(in oklab, var(--color-primary-secondary), transparent 10%);
  background: color-mix(in oklab, var(--color-primary-secondary), transparent 82%);
}

.seat-button.occupied {
  cursor: not-allowed;
  opacity: 0.45;
}

.seat-button.locked {
  cursor: not-allowed;
  opacity: 0.55;
}

.seat-button.own-seat {
  border-color: color-mix(in oklab, var(--color-primary-secondary), transparent 20%);
}

label,
.seat-input,
.own-seats,
.own-seat-list,
.join-flow,
.participants,
.winners,
.participant-list,
.live-events,
.event-list {
  display: grid;
  gap: 0.55rem;
}

.own-seat-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.85rem;
  padding: 0.85rem;
  border-radius: 1rem;
  border: 1px solid color-mix(in oklab, var(--color-primary-secondary), transparent 35%);
  background: color-mix(in oklab, var(--color-primary-secondary), transparent 91%);
}

.own-seat-card div {
  display: grid;
  gap: 0.25rem;
}

.boost-value {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin: 0;
}

select,
input[type='number'] {
  width: 100%;
  padding: 0.8rem 0.95rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 8%);
  border-radius: 0.9rem;
  background: color-mix(in oklab, var(--color-surface), white 14%);
  color: var(--color-text);
}

.participant-list {
  grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
}

.participant-card.own {
  border-color: color-mix(in oklab, var(--color-primary-secondary), transparent 12%);
  background: color-mix(in oklab, var(--color-primary-secondary), transparent 88%);
}

.own-badge {
  display: inline-flex;
  width: fit-content;
  padding: 0.35rem 0.65rem;
  color: #075985;
  background: color-mix(in oklab, #38bdf8, transparent 82%);
}

.event-card {
  display: flex;
  justify-content: space-between;
  gap: 0.85rem;
  padding: 0.8rem 0.9rem;
  border-radius: 1rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  background: color-mix(in oklab, var(--color-surface), white 8%);
}

.event-card div {
  display: grid;
  gap: 0.25rem;
}

.event-card span,
.event-card time {
  color: var(--color-muted);
}

.event-card time {
  white-space: nowrap;
}

.switch-line {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
}

.actions {
  justify-content: flex-end;
}

.actions.stretch {
  justify-content: stretch;
}

.actions.stretch .btn {
  flex: 1 1 160px;
}

.btn {
  appearance: none;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  background: color-mix(in oklab, var(--color-surface), white 8%);
  color: var(--color-text);
  border-radius: 999px;
  padding: 0.8rem 1rem;
  font-weight: 600;
  cursor: pointer;
  transition:
    transform var(--transition-fast) ease,
    border-color var(--transition-fast) ease,
    background var(--transition-fast) ease;
}

.btn:hover {
  transform: translateY(-1px);
}

.btn:disabled {
  cursor: not-allowed;
  opacity: 0.6;
  transform: none;
}

.btn--primary {
  border-color: transparent;
  background: linear-gradient(135deg, #0f766e, #0284c7);
  color: #f0fdfa;
}

.btn--danger {
  border-color: transparent;
  background: linear-gradient(135deg, #b91c1c, #ea580c);
  color: #fff7ed;
}

@media (max-width: 1120px) {
  .play-grid,
  .control-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 960px) {
  .play-area,
  .play-area.collapsed {
    position: static;
    inset: auto;
    margin: calc(76px + 0.75rem) 1rem 5.75rem;
  }
}

@media (max-width: 720px) {
  .hero-card,
  .card-head.row,
  .meta-grid,
  .round-timeline,
  .next-round-box {
    grid-template-columns: 1fr;
  }

  .actions {
    justify-content: stretch;
  }

  .actions .btn {
    width: 100%;
  }
}
</style>
