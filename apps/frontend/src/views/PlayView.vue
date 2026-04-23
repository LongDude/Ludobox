<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch, type CSSProperties } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import LeftTab from '@/components/LeftTab.vue'
import UpTab from '@/components/UpTab.vue'
import FooterTab from '@/components/FooterTab.vue'
import { GameApi } from '@/api/useMatchApi'
import type {
  GameJoinRoomResponse,
  GameParticipantInfo,
  GameRoundFinalizedEventData,
  GameRoomStateResponse,
  GameRoundEvent,
  GameRoundStatusResponse,
  GameWinnerInfo,
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

const isSpinning = ref(false)
const arrowAngle = ref(0)
const isRoundActive = ref(false)
const roundFinalized = ref(false)
let spinInterval: ReturnType<typeof window.setInterval> | null = null

const winnerSeat = computed(() => {
  if (!winners.value.length) return null
  return winners.value[0]?.number_in_room || null
})

const winnerName = computed(() => {
  if (!winners.value.length) return ''
  const winner = winners.value[0]

  if (winner.is_bot || winner.participant_id < 0) return t('gameRoom.round.winnerBot')
  return winner.nickname || `Player ${winner.participant_id}`
})

const winnerAmount = computed(() => {
  if (!winners.value.length) return 0
  return winners.value[0].winnings || 0
})

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

const finalizedRoundData = computed<GameRoundFinalizedEventData | null>(() => {
  const targetRoundId = activeRoundId.value
  if (!targetRoundId) return null

  const recentEvents = [...liveEvents.value, ...(roomState.value?.recent_events ?? [])]
  for (const event of recentEvents) {
    if (event.type !== 'round_finalized') continue
    const normalized = normalizeRoundFinalizedEventData(event.data, activeRoundId.value)
    if (normalized && normalized.round_id === targetRoundId) {
      return normalized
    }
  }

  return null
})
const winners = computed<GameWinnerInfo[]>(() => {
  if (finalizedRoundData.value?.winners?.length) {
    return finalizedRoundData.value.winners
  }

  return (roundStatus.value?.winners ?? []).map((winner) => ({
    participant_id: winner.participant_id,
    user_id: winner.user_id ?? null,
    nickname: winner.nickname ?? null,
    number_in_room: winner.number_in_room,
    winnings: winner.winning_money,
    gross_winnings: winner.winning_money,
    is_bot: winner.is_bot,
  }))
})
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
          rating: cabinet.profile?.rating ?? null,
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

// ==================================================================
// Watchers
// ==================================================================

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

// ==================================================================
// Functions
// ==================================================================

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
  // If clicking on a seat you already own (occupied by you)
  if (occupiedSeats.value.has(seat) && ownedSeatNumbers.value.has(seat)) {
    const participant = getParticipantOnSeat(seat)
    if (participant && canLeaveSeats.value) {
      // Directly leave the participant without confirmation
      leaveParticipantDirect(participant)
    }
    return
  }
  
  // Normal seat selection for empty seats
  if (!canJoinMoreSeats.value) return

  const alreadySelected = selectedSeatNumbers.value.has(seat)
  if (!alreadySelected && !canSelectSeat(seat)) return

  if (alreadySelected) {
    selectedSeats.value = selectedSeats.value.filter((value) => value !== seat)
    return
  }

  selectedSeats.value = [...selectedSeats.value, seat].sort((left, right) => left - right)
}

async function leaveParticipantDirect(participant: GameParticipantInfo) {
  clearFeedback()

  if (!canLeaveSeats.value) {
    errorMsg.value = isJoined.value ? t('gameRoom.controls.actionsLocked') : t('gameRoom.errors.joinFirst')
    return
  }

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

  // Handle round_started event
  if (event.type === 'round_started') {
    isRoundActive.value = true
    roundFinalized.value = false
    startRoundSpinning()
  }

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
    
    // Stop spinning and point to winner
    stopSpinningAndPointToWinner()
    
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

function normalizeWinnerInfo(value: unknown): GameWinnerInfo | null {
  if (!value || typeof value !== 'object') return null

  const winner = value as Record<string, unknown>
  const participantId = Number(winner.participant_id ?? 0)
  const numberInRoom = Number(winner.number_in_room ?? 0)
  const userID = Number(winner.user_id ?? 0)
  const nickname =
    typeof winner.nickname === 'string' && winner.nickname.trim().length > 0
      ? winner.nickname.trim()
      : null

  if (!Number.isFinite(participantId) || !Number.isFinite(numberInRoom) || numberInRoom <= 0) {
    return null
  }

  return {
    participant_id: participantId,
    user_id: Number.isFinite(userID) && userID > 0 ? userID : null,
    nickname,
    number_in_room: numberInRoom,
    winnings: Number(winner.winnings ?? winner.winning_money ?? 0) || 0,
    gross_winnings: Number(winner.gross_winnings ?? winner.winnings ?? winner.winning_money ?? 0) || 0,
    is_bot: Boolean(winner.is_bot),
  }
}

function normalizeRoundFinalizedEventData(
  value: unknown,
  fallbackRoundId: number | null = null,
): GameRoundFinalizedEventData | null {
  if (!value || typeof value !== 'object') return null

  const payload = value as Record<string, unknown>
  const winners = Array.isArray(payload.winners)
    ? payload.winners
        .map((winner) => normalizeWinnerInfo(winner))
        .filter((winner): winner is GameWinnerInfo => Boolean(winner))
        .sort((left, right) => left.number_in_room - right.number_in_room)
    : []

  const roundId = Number(payload.round_id ?? fallbackRoundId ?? 0)
  if (!Number.isFinite(roundId) || roundId <= 0) return null

  return {
    round_id: roundId,
    winners,
    payouts:
      payload.payouts && typeof payload.payouts === 'object'
        ? (payload.payouts as Record<string, number>)
        : undefined,
    next_round_id: Number(payload.next_round_id ?? 0) || null,
    next_round_delay: Number(payload.next_round_delay ?? 0) || null,
  }
}

function winnerLabel(winner: GameWinnerInfo) {
  if (winner.nickname) return winner.nickname
  if (winner.user_id) return t('gameRoom.participant.user', { id: winner.user_id })
  return t('gameRoom.participant.id', { id: winner.participant_id })
}

function isOwnWinner(winner: GameWinnerInfo) {
  if (winner.is_bot) return false
  return (
    ownedParticipantIds.value.has(winner.participant_id) || ownedSeatNumbers.value.has(winner.number_in_room)
  )
}

function eventDescription(event: GameRoundEvent) {
  const data = eventData(event)

  if (event.type === 'player_joined') {
    return t('gameRoom.events.playerJoinedDetails', {
      participant: String(data.nickname ?? 0) || '-',
      seat: Number(data.number_in_room ?? 0) || '-',
      players: Number(data.current_players ?? 0) || '-',
    })
  }

  if (event.type === 'player_left') {
    return t('gameRoom.events.playerLeftDetails', {
      participant: String(data.nickname ?? 0) || '-',
      seat: Number(data.number_in_room ?? 0) || '-',
      players: Number(data.current_players ?? 0) || '-',
    })
  }

  if (event.type === 'boost_purchased') {
    return t('gameRoom.events.boostPurchasedDetails', {
      participant: String(data.nickname ?? 0) || '-',
      power: Number(data.boost_power ?? 0) || '-',
    })
  }

  if (event.type === 'boost_cancelled') {
    return t('gameRoom.events.boostCancelledDetails', {
      participant: String(data.nickname ?? 0) || '-',
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

      // Get all free seats
      const allFreeSeats = getFreeSeatsList()
      
      if (allFreeSeats.length < targetRandomCount) {
        throw new Error(t('gameRoom.entry.roomFull'))
      }
      
      // Shuffle and pick random seats
      const shuffledFreeSeats = [...allFreeSeats]
      for (let i = shuffledFreeSeats.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [shuffledFreeSeats[i], shuffledFreeSeats[j]] = [shuffledFreeSeats[j], shuffledFreeSeats[i]]
      }
      
      const randomSeatsToReserve = shuffledFreeSeats.slice(0, targetRandomCount)
      
      // Reserve each random seat
      for (const seat of randomSeatsToReserve) {
        const response = await GameApi.joinRoomWithSeat(roomId.value, { number_in_room: seat })
        joinResult.value = response
        rememberOwnedParticipantIds([response.participant_id])
        reservedSeats.push(response.number_in_room)
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

// filter free seats
function getFreeSeatsList(): number[] {
  const freeSeats: number[] = []
  for (let i = 1; i <= roomCapacity.value; i++) {
    if (!occupiedSeats.value.has(i)) {
      freeSeats.push(i)
    }
  }
  return freeSeats
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
    // await refreshRoundView()
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
    // await refreshRoundView()
  } catch (error: any) {
    errorMsg.value = normalizeError(error, t('gameRoom.errors.cancelBoost'))
  } finally {
    actionLoading.value = ''
  }
}

function canBuyBoostForSeat(seatNumber: number) {
  const participant = getParticipantOnSeat(seatNumber)
  if (!participant) return false
  if (!ownedSeatNumbers.value.has(seatNumber)) return false
  if (participant.boost > 0) return false
  if (!canManageCurrentRound.value) return false
  if ((roomState.value?.is_boost ?? room.value?.is_boost) !== true) return false
  
  return true
}

async function purchaseBoostForSeat(seatNumber: number) {
  clearFeedback()
  
  const participant = getParticipantOnSeat(seatNumber)
  if (!participant) {
    errorMsg.value = t('gameRoom.errors.joinFirst')
    return
  }
  
  if (participant.boost > 0) {
    errorMsg.value = t('gameRoom.errors.boostAlreadyPurchased')
    return
  }

  actionLoading.value = `boost-${participant.participant_id}`

  try {
    const response = await GameApi.purchaseBoost(roomId.value, participant.participant_id)
    successMsg.value = t('gameRoom.messages.boostPurchased', {
      power: response.boost_power,
      cost: response.boost_cost,
    })
    refreshCabinetBalance()
  } catch (error: any) {
    errorMsg.value = normalizeError(error, t('gameRoom.errors.boost'))
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

function formatMoney(amount: number) {
  return amount
}

function getSeatPosition(index: number, totalSeats: number): CSSProperties {
  const angle = (360 / totalSeats) * index - 90
  const radius = 200
  const x = Math.cos((angle * Math.PI) / 180) * radius
  const y = Math.sin((angle * Math.PI) / 180) * radius

  return {
    transform: `translate(calc(-50% + ${x}px), calc(-50% + ${y}px))`,
    position: 'absolute',
  }
}

function getParticipantOnSeat(seatNumber: number) {
  return participants.value.find((participant) => participant.number_in_room === seatNumber)
}

function startRoundSpinning() {
  isRoundActive.value = true
  roundFinalized.value = false
  isSpinning.value = true
  
  // Reset arrow angle to 0 before starting
  arrowAngle.value = 0
  
  // Continuous spinning animation
  let currentAngle = 0
  spinInterval = setInterval(() => {
    currentAngle = (currentAngle + 15) % 360
    arrowAngle.value = currentAngle
  }, 50)
}

function stopSpinningAndPointToWinner() {
  if (spinInterval) {
    clearInterval(spinInterval)
    spinInterval = null
  }
  
  if (!winnerSeat.value || !seatOptions.value.length) {
    isSpinning.value = false
    return
  }
  
  // Calculate target angle based on winner seat position
  const winnerIndex = seatOptions.value.indexOf(winnerSeat.value)
  const totalSeats = seatOptions.value.length
  // const targetAngle = (winnerIndex * 360) / totalSeats - 90
  const targetAngle = 360/totalSeats * winnerIndex
  
  // Smooth animation to winner
  const startAngle = arrowAngle.value
  const angleDiff = ((targetAngle - startAngle) + 360) % 360
  const duration = 1000 // 1 second
  const startTime = performance.now()
  
  function animate(currentTime: number) {
    const elapsed = currentTime - startTime
    const progress = Math.min(1, elapsed / duration)
    
    // Easing function for smooth stop
    const easeOut = 1 - Math.pow(1 - progress, 3)
    const currentAngle = startAngle + (angleDiff * easeOut)
    arrowAngle.value = currentAngle % 360
    
    if (progress < 1) {
      requestAnimationFrame(animate)
    } else {
      arrowAngle.value = targetAngle
      isSpinning.value = false
      roundFinalized.value = true
      isRoundActive.value = false
    }
  }
  
  requestAnimationFrame(animate)
}

// Reset round state when new round starts
watch(() => roomRoundId.value, () => {
  isRoundActive.value = false
  roundFinalized.value = false
  if (spinInterval) {
    clearInterval(spinInterval)
    spinInterval = null
  }
  isSpinning.value = false
  arrowAngle.value = 0
})

// Clean up on unmount
onBeforeUnmount(() => {
  if (spinInterval) clearInterval(spinInterval)
})
</script>

<template>
  <UpTab :show-menu="false" :show-upload="false" />
  <LeftTab />

  <main class="play-area" :class="{ collapsed: leftHidden }" :style="{ '--layout-inset': layoutInset }">
    <section class="hero-card">
      <div>
        <h1>{{ t('matchmaking.play.title', { roomId }) }}</h1>
        <p class="description">
          {{ t('matchmaking.play.meta.entry') }}: {{ entryPrice }},
          {{ t('matchmaking.play.meta.capacity') }}: {{ roomCapacity }},
          {{ t('matchmaking.play.meta.boost') }}: {{ formatBoost() }}
        </p>
      </div>
      <button class="btn back-btn" type="button" @click="openRooms">
        {{ t('matchmaking.play.backRooms') }}
      </button>
    </section>

    <section class="game-area">
        <div class="join-box" :class="{ joined: isJoined, 'round-active': isRoundActive }">


          <div class="join-flow">
            <p class="join-title">
              {{ isJoined ? t('gameRoom.entry.addSeatTitle') : t('gameRoom.entry.notJoinedTitle') }}
            </p>
            <p class="description">{{ joinBlockedHint }}</p>

              <!-- Always show circular seat selection -->
              <div class="circular-seat-container">
                <div class="seats-circle">
                  <!-- Arrow always visible, spins during round -->
                  <div class="spinning-arrow" 
                       :class="{ spinning: isRoundActive && !roundFinalized }" 
                       :style="{ transform: `rotate(${arrowAngle}deg)` }">
                    <svg viewBox="0 0 100 100" class="arrow-svg">
                      <polygon points="50,10 40,30 60,30" fill="#f59e0b" stroke="#d97706" stroke-width="2"/>
                      <rect x="48" y="30" width="4" height="40" fill="#f59e0b" />
                    </svg>
                  </div>
                  
                  <!-- Seat Buttons - disabled during round but visible -->
                  <button
                    v-for="(seat, index) in seatOptions"
                    :key="seat"
                    class="circular-seat-button"
                    :class="{
                      occupied: occupiedSeats.has(seat),
                      selected: selectedSeatNumbers.has(seat),
                      'own-seat': ownedSeatNumbers.has(seat),
                      'winner-seat': winnerSeat === seat && roundFinalized,
                      'clickable-own-seat': occupiedSeats.has(seat) && ownedSeatNumbers.has(seat) && !isRoundActive,
                      'has-boost': (getParticipantOnSeat(seat)?.boost ?? 0) > 0
                    }"
                    :style="getSeatPosition(index, seatOptions.length)"
                    type="button"
                    :disabled="isRoundActive || (occupiedSeats.has(seat) && !ownedSeatNumbers.has(seat))"
                    @click="toggleSeatSelection(seat)"
                  >
                    <span class="seat-number">{{ seat }}</span>
                    <span v-if="getParticipantOnSeat(seat)" class="seat-player">
                      {{ getParticipantOnSeat(seat)?.nickname || `P${getParticipantOnSeat(seat)?.participant_id}` }}
                    </span>

                    <!-- Boost button overlay on hover for own seats without boost -->
                      <div 
                        v-if="canBuyBoostForSeat(seat) && !isRoundActive"
                        class="boost-overlay"
                        @click.stop="purchaseBoostForSeat(seat)"
                      >
                        <button 
                          class="boost-seat-btn"
                          :disabled="actionLoading === `boost-${getParticipantOnSeat(seat)?.participant_id}`"
                          :title="t('gameRoom.controls.buyBoost')"
                        >
                          ⚡
                        </button>
                      </div>
                      
                      <!-- Boost indicator for seats that already have boost -->
                      <div 
                        v-if="(getParticipantOnSeat(seat)?.boost ?? 0) > 0"
                        class="boost-active-indicator"
                        :title="t('gameRoom.controls.boostActiveOnSeat', { seat })"
                      >
                        ⚡
                      </div>
                  </button>
                </div>
                
                <!-- Timer display during round -->
                <div v-if="isRoundActive && !roundFinalized" class="round-timer-overlay">
                  <div class="timer-circle">
                    <span class="timer-label">Time Remaining</span>
                    <span class="timer-value">{{ roundTimerValue }}s</span>
                  </div>
                </div>

                <!-- Winner Display -->
                <div v-if="winnerSeat && roundFinalized" class="winner-announcement">
                  <div class="winner-content">
                    <span class="winner-label">🏆 WINNER! 🏆</span>
                    <span class="winner-name">Seat {{ winnerSeat }} - {{ winnerName }}</span>
                    <span class="winner-prize">Won {{ formatMoney(winnerAmount) }}</span>
                    <span class="winner-timer">{{ t('gameRoom.round.nextRoundTimer') }}: {{ pendingNextRoundCountdown }}</span>
                  </div>
                </div>
              </div>

              <!-- Selection controls - hidden during round -->
              <div v-if="!isRoundActive && canJoinMoreSeats" class="selection-controls">
                <label class="random-count-control">
                  <span>{{ t('gameRoom.entry.randomSeatsCount') }}</span>
                  <input
                    v-model.number="randomReserveCount"
                    type="number"
                    min="1"
                    :max="randomReserveMax"
                    :disabled="Boolean(selectedSeats.length)"
                  />
                </label>
                
                <p v-if="selectedSeats.length" class="selected-seats-info">
                  ✓ {{ t('gameRoom.entry.selectedSeats', { seats: selectedSeats.join(', ') }) }}
                </p>
                <p v-else class="selected-seats-info">
                  🎲 {{ t('gameRoom.entry.randomSeatsHint', { count: safeRandomReserveCount }) }}
                </p>

                <div class="actions stretch">
                  <button class="btn btn--primary" type="button" :disabled="joining" @click="reserveSeats">
                    {{ reserveSeatsLabel }}
                  </button>
                </div>
              </div>

              <!-- Show message during round that selection is disabled -->
              <div v-if="isRoundActive && !roundFinalized" class="round-message">
                <p>⚡ Round in progress - Seat selection disabled ⚡</p>
              </div>

          </div>
        </div>


      <article class="panel-card right-info">
        <div class="card-head row">
          <div>
            <h2>
              {{ t('gameRoom.round') }} {{ activeRoundId ? `#${activeRoundId}` : '-' }}
            </h2>
          </div>
          <button class="btn" type="button" :disabled="!activeRoundId || statusLoading" @click="refreshRoundView()">
            {{ statusLoading ? t('common.loading') : t('common.refresh') }}
          </button>
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
              <span v-if="participant.nickname">
                {{ participant.nickname }}
              </span>
              <span v-else-if="participant.user_id">
                {{ t('gameRoom.participant.user', { id: participant.user_id }) }}
              </span>
            </div>
          </div>
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

    <!-- bottom menu -->
    <section class="panel-card controls-card">
      <div class="card-head">
        <div>
          <h2>{{ t('gameRoom.controls.title') }}</h2>
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
  margin-top: 0.5rem;
  overflow: auto;
  align-content: start;
  transition: all var(--transition-slow) ease;
}

.play-area.collapsed {
  --layout-inset: 92px 20px 20px 120px;
}

/* game related */
.game-area {
  display: flex;
/*  flex-direction: column;*/
  gap: 1rem;
}

.right-info {
  width: 30%;
  min-width: 350px;

  display: flex;
  flex-direction: column;

  background:
  radial-gradient(circle at top right, rgba(14, 165, 233, 0.16), transparent 26%),
  linear-gradient(
    310deg,
    color-mix(in oklab, var(--color-bg-secondary), white 16%),
    color-mix(in oklab, var(--color-surface), transparent 4%)
  );
}

.circular-seat-container {
  position: relative;
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 420px;
  margin: 1rem 0;
}

.seats-circle {
  position: relative;
  width: 380px;
  height: 380px;
  border-radius: 50%;
  border: 3px solid color-mix(in oklab, var(--color-primary-secondary), transparent 50%);
  background: radial-gradient(circle at center, color-mix(in oklab, var(--color-surface), white 5%), transparent 70%);
  display: flex;
  justify-content: center;
  align-items: center;
}

.circular-seat-button {
  position: absolute;
  left: 50%;
  top: 50%;
  width: 65px;
  height: 65px;
  margin-left: 0;
  margin-top: 0;
  border-radius: 50%;
  border: 1px solid color-mix(in oklab, var(--color-primary-secondary), transparent 50%);
  background: linear-gradient(135deg, var(--color-surface), color-mix(in oklab, var(--color-surface), white 15%));
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  font-weight: bold;
  z-index: 2;
}

.circular-seat-button:hover:not(:disabled) {
  transform: scale(1.1);
  border-color: #f59e0b;
  box-shadow: 0 0 15px rgba(245, 158, 11, 0.4);
}

.circular-seat-button.selected {
  border-color: #10b981;
  background: linear-gradient(135deg, #10b981, #059669);
  color: white;
  transform: scale(1.05);
  box-shadow: 0 0 20px rgba(16, 185, 129, 0.5);
}

.circular-seat-button.occupied {
  cursor: not-allowed;
  opacity: 0.95;
  filter: grayscale(0.3);
}

.circular-seat-button.own-seat {
  border-color: #3b82f6;
  background: linear-gradient(135deg, #3b82f6, #2563eb);
  color: white;
}

.circular-seat-button.winner-seat {
  border-color: #fbbf24;
  background: linear-gradient(135deg, #fbbf24, #f59e0b);
  animation: winnerPulse 1s ease-in-out infinite;
  box-shadow: 0 0 30px rgba(251, 191, 36, 0.8);
  transform: scale(1.1);
  z-index: 3;
}

.seat-number {
  font-size: 1.2rem;
  font-weight: bold;
}

.seat-player {
  font-size: 0.65rem;
  margin-top: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 55px;
}

.spinning-arrow {
  position: absolute;
  width: 60px;
  height: 60px;
  z-index: 10;
  pointer-events: none;
  transition: transform 0.05s linear;
}

.spinning-arrow.spinning {
  animation: arrowPulse 0.4s ease-in-out infinite;
}

.arrow-svg {
  width: 100%;
  height: 100%;
  filter: drop-shadow(0 0 5px rgba(245, 158, 11, 0.5));
  transform: scale(1.6);
}

.winner-announcement {
  position: absolute;
  top: 60%;
  left: 50%;
  min-width: 220px;
  transform: translateX(-50%);
  z-index: 20;
  animation: winnerFloatUp 0.5s ease-out;
}

.winner-content {
  background: linear-gradient(135deg, #fbbf24, #f59e0b);
  padding: 1rem 2rem;
  border-radius: 50px;
  text-align: center;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
  min-width: 200px;
}

.winner-label {
  display: block;
  font-size: 0.85rem;
  font-weight: bold;
  letter-spacing: 2px;
  color: #fff;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
}

.winner-name {
  display: block;
  font-size: 1.5rem;
  font-weight: bold;
  margin: 0.3rem 0;
  color: #fff;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
}

.winner-player {
  display: block;
  font-size: 0.9rem;
  margin: 0.2rem 0;
  color: #fff;
  opacity: 0.95;
}

.winner-prize {
  display: block;
  font-size: 1.1rem;
  font-weight: 600;
  margin-top: 0.3rem;
  color: #fffbeb;
}

.selection-controls {
  margin-top: 1.5rem;
  text-align: center;
}

.random-count-control {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem;
}

.random-count-control span {
  text-wrap: nowrap;
}

.selected-seats-info {
  text-align: center;
  margin: 0.5rem 0;
  font-size: 0.9rem;
}

@keyframes winnerPulse {
  0%, 100% { 
/*    transform: scale(1.1); */
    box-shadow: 0 0 20px rgba(251, 191, 36, 0.5);
  }
  50% { 
/*    transform: scale(1.2); */
    box-shadow: 0 0 40px rgba(251, 191, 36, 0.8);
  }
}

@keyframes arrowPulse {
  0% { 
    transform: scale(1);
    transform: rotate(0deg); 
  }
  25% {
    transform: scale(1.1);
    transform: rotate(90deg);
  }
  50% { 
    transform: scale(1); 
    transform: rotate(180deg); 
  }
  75% {
    transform: scale(1.1);
    transform: rotate(270deg);
  }
  100% {
    transform: scale(1);
    transform: rotate(360deg); 
  }
}

@keyframes winnerFloatUp {
  0% { 
    transform: translateX(-50%) translateY(20px); 
    opacity: 0; 
  }
  100% { 
    transform: translateX(-50%) translateY(0); 
    opacity: 1; 
  }
}

.circular-seat-button.winner-seat {
  border-color: #fbbf24;
  background: linear-gradient(135deg, #fbbf24, #f59e0b);
  animation: winnerPulse 1s ease-in-out infinite;
  box-shadow: 0 0 30px rgba(251, 191, 36, 0.8);
  transform: scale(1.1);
  z-index: 3;
}

@keyframes spinWheel {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

@keyframes winnerPopup {
  0% {
    transform: scale(0);
    opacity: 0;
  }
  50% {
    transform: scale(1.1);
  }
  100% {
    transform: scale(1);
    opacity: 1;
  }
}

.round-message {
  text-align: center;
  padding: 1rem;
  background: rgba(245, 158, 11, 0.1);
  border-radius: 0.5rem;
  margin-top: 1rem;
}

.round-message p {
  color: #f59e0b;
  font-weight: bold;
  margin: 0;
}

.round-timer-overlay {
  position: absolute;
  top: 80%;
  left: 50%;
  transform: translate(-50%, -50%);
  z-index: 15;
  pointer-events: none;
}

.timer-circle {
  background: linear-gradient(135deg, var(--color-bg), var(--color-surface));
  backdrop-filter: blur(10px);
  border-radius: 50%;
  width: 130px;
  height: 130px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border: 3px solid #f59e0b;
  box-shadow: 0 0 30px rgba(245, 158, 11, 0.5);
  animation: timerPulse 1s ease-in-out infinite;
}

.timer-label {
  font-size: 0.7rem;
  color: color-mix(in oklab, #fbbf24, var(--color-text) 5%);
  text-transform: uppercase;
  letter-spacing: 1px;
}

.timer-value {
  font-size: 2rem;
  font-weight: bold;
  font-family: monospace;
  color: #f59e0b;
  line-height: 1;
}

@keyframes timerPulse {
  0%, 100% {
    transform: scale(1);
    box-shadow: 0 0 30px rgba(245, 158, 11, 0.5);
  }
  50% {
    transform: scale(1.05);
    box-shadow: 0 0 50px rgba(245, 158, 11, 0.8);
  }
}

/* other */
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
    radial-gradient(circle at left, rgba(245, 158, 11, 0.18), transparent 18%),
    radial-gradient(circle at right, rgba(14, 165, 233, 0.16), transparent 26%),
    linear-gradient(
      135deg,
      color-mix(in oklab, var(--color-bg-secondary), white 16%),
      color-mix(in oklab, var(--color-surface), transparent 4%)
    );
}

.hero-pills,
.actions {
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

/*.panel-card,*/
.control-block,
.join-box,
.next-round-box {
  display: grid;
  gap: 1rem;
}

.join-box {
  min-width: 460px;
  flex-grow: 1;

  background:
    radial-gradient(circle at top left, rgba(245, 158, 11, 0.18), transparent 8%),
    linear-gradient(
      20deg,
      color-mix(in oklab, var(--color-bg-secondary), white 14%),
      color-mix(in oklab, var(--color-surface), transparent 4%)
    );
}

.panel-card:not(.right-info) {
  background: color-mix(in oklab, var(--color-surface), white 10%);
    /*radial-gradient(circle at top left, color-mix(in oklab, #0ea5e9, white 88%), transparent 28%),
    linear-gradient(180deg, color-mix(in oklab, var(--color-surface), white 14%), var(--color-surface));*/
}

.card-head {
/*  display: grid;*/
  gap: 0.5rem;
}

.card-head.row {
  display: flex;
  justify-content: space-between;
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
.next-round-box {
  padding: 0.45rem;
  border-radius: 1rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  background: color-mix(in oklab, var(--color-surface), white 10%);
}

.join-box {
  padding: 0.85rem;
  border-radius: 1rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
/*  background: color-mix(in oklab, var(--color-surface), white 10%);*/
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

.circular-seat-button {
  position: relative;
}

.boost-overlay {
  position: absolute;
  top: -12px;
  right: -12px;
  z-index: 5;
}

.boost-seat-btn {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: linear-gradient(135deg, #f59e0b, #d97706);
  border: 2px solid #fff;
  color: white;
  font-size: 14px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  box-shadow: 0 2px 8px rgba(0,0,0,0.2);
}

.boost-seat-btn:hover:not(:disabled) {
  transform: scale(1.1);
  box-shadow: 0 0 12px rgba(245, 158, 11, 0.6);
}

.boost-seat-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.boost-active-indicator {
  position: absolute;
  top: -8px;
  right: -8px;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: linear-gradient(135deg, #10b981, #059669);
  border: 2px solid #fff;
  color: white;
  font-size: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  animation: boostPulse 2s ease-in-out infinite;
  z-index: 5;
}

.circular-seat-button.has-boost {
  border-color: #10b981;
  box-shadow: 0 0 15px rgba(16, 185, 129, 0.3);
}

@keyframes boostPulse {
  0%, 100% {
    transform: scale(1);
    box-shadow: 0 0 5px rgba(16, 185, 129, 0.5);
  }
  50% {
    transform: scale(1.05);
    box-shadow: 0 0 12px rgba(16, 185, 129, 0.8);
  }
}

label,
.seat-input,
.own-seats,
.join-flow,
.winners,
.winner-list,
.live-events,
.event-list {
  display: grid;
  gap: 0.55rem;
}

.live-events {
  max-height: 400px;
  overflow-y: auto;
}

.participants {
  flex-grow: 1;
  max-height: 40%;
}

.participants h3 {
  flex-grow: 0;
}

.description {
  flex-grow: 1;
}

.card-head {
  flex-grow: 0;
}

.own-seat-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.participant-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.participant-card {
  display: flex;
  max-width: 48%;
  width: 48%;

  align-items: center;
  gap: 0.5rem;
}

.winner-list {
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
}

.winner-card {
  display: grid;
  gap: 0.45rem;
  padding: 0.85rem;
  border-radius: 1rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  background: color-mix(in oklab, var(--color-surface), white 10%);
}

.winner-card.own {
  border-color: color-mix(in oklab, var(--color-success), transparent 18%);
  background:
    radial-gradient(circle at top right, color-mix(in oklab, var(--color-success), transparent 76%), transparent 55%),
    color-mix(in oklab, var(--color-success), transparent 92%);
}

.winner-card.bot {
  opacity: 0.88;
}

.winner-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.65rem;
  flex-wrap: wrap;
}

.winner-bot {
  color: var(--color-muted);
  font-weight: 600;
}

.own-seat-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.85rem;
  padding: 0.55rem;
  border-radius: 1rem;
  border: 1px solid color-mix(in oklab, var(--color-primary-secondary), transparent 35%);
  background: color-mix(in oklab, var(--color-primary-secondary), transparent 91%);
  max-width: 40%;
}

.own-seat-card div {
  display: grid;
  gap: 0.25rem;
}

.circular-seat-button.clickable-own-seat {
  cursor: pointer;
  position: relative;
}

.circular-seat-button.clickable-own-seat:hover::after {
  content: "✕";
  position: absolute;
  top: -11px;
  right: 45px;
  background: #ef4444;
  color: white;
  border-radius: 50%;
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: bold;
  box-shadow: 0 2px 4px rgba(0,0,0,0.2);
  border: 2px solid #fff;
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
  background: color-mix(in oklab, var(--color-surface), white 6%);
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

.back-btn {
  border-radius: 1rem;
  background: color-mix(in oklab, var(--color-surface), white 4%);
  border: 1px solid color-mix(in oklab, var(--color-surface), white 18%);
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

  .game-area {
    flex-direction: column;
  }

  .right-info {
    width: 100%;
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
