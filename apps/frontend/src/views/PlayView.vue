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

type LiveRoundEvent = GameRoundEvent & { id: number }

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
const selectedSeat = ref('')
const autoRefresh = ref(true)
const autoAdvanceNextRound = ref(true)
const sseConnected = ref(false)
const sseError = ref('')
const liveEvents = ref<LiveRoundEvent[]>([])
const displayedRoundId = ref<number | null>(null)
const pendingNextRoundId = ref<number | null>(null)
const pendingNextRoundCountdown = ref(0)
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
const currentRoomIsActive = computed(() => currentRoundPhase.value === 'playing')
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
  return new Set(
    (roundStatus.value?.participants ?? [])
      .filter((participant) => !participant.exited_at)
      .map((participant) => participant.number_in_room),
  )
})

const participants = computed(() =>
  [...(roundStatus.value?.participants ?? [])].sort(
    (left, right) => left.number_in_room - right.number_in_room,
  ),
)

const winners = computed(() => roundStatus.value?.winners ?? [])
const roomUserParticipants = computed(() =>
  (roomState.value?.current_user_participants ?? []).filter((participant) => !participant.exited_at),
)
const currentRoomParticipant = computed(() => {
  const participant = roomUserParticipants.value[0] ?? null
  if (!participant) return null

  if (activeRoundId.value === roomRoundId.value) {
    return (
      participants.value.find((item) => item.participant_id === participant.participant_id) ??
      participant
    )
  }

  return participant
})
const displayedOwnParticipants = computed(() => {
  const userId = currentUserId.value
  if (userId) {
    const matched = participants.value.filter(
      (participant) => participant.user_id === userId && !participant.exited_at,
    )
    if (matched.length > 0) return matched
  }

  if (participants.value.length > 0 && roomUserParticipants.value.length > 0) {
    const ownedParticipantIds = new Set(
      roomUserParticipants.value.map((participant) => participant.participant_id),
    )
    const matched = participants.value.filter(
      (participant) =>
        !participant.exited_at && ownedParticipantIds.has(participant.participant_id),
    )
    if (matched.length > 0) return matched
  }

  const participantId = initialParticipantId.value
  if (!participantId || initialParticipantRoundId.value !== activeRoundId.value) return []

  const matched = participants.value.find((participant) => participant.participant_id === participantId)
  return matched ? [matched] : []
})
const activeParticipant = computed(() => {
  if (activeRoundId.value === roomRoundId.value && currentRoomParticipant.value) {
    return currentRoomParticipant.value
  }

  if (displayedOwnParticipants.value.length > 0) {
    return displayedOwnParticipants.value[0] ?? null
  }

  return null
})
const activeParticipantId = computed(() => activeParticipant.value?.participant_id ?? null)
const currentParticipantId = computed(() => currentRoomParticipant.value?.participant_id ?? null)
const mySeat = computed(
  () => currentRoomParticipant.value?.number_in_room ?? activeParticipant.value?.number_in_room ?? null,
)
const isJoined = computed(() => {
  if (roomUserParticipants.value.length > 0) return true
  return Boolean(initialParticipantId.value && initialParticipantRoundId.value === roomRoundId.value)
})
const hasOwnedBoost = computed(() =>
  roomUserParticipants.value.some((participant) => participant.boost > 0),
)
const hasActiveParticipantBoost = computed(
  () => Boolean(currentRoomParticipant.value?.boost && currentRoomParticipant.value.boost > 0),
)
const canManageCurrentRound = computed(
  () => Boolean(currentParticipantId.value) && currentRoundPhase.value === 'waiting',
)
const canBoost = computed(
  () =>
    canManageCurrentRound.value &&
    isJoined.value &&
    (roomState.value?.is_boost ?? room.value?.is_boost) === true &&
    !hasOwnedBoost.value,
)
const canLeave = computed(() => isJoined.value && !currentRoomIsActive.value)
const boostHint = computed(() => {
  if (hasOwnedBoost.value) return t('gameRoom.controls.boostAlreadyActive')
  if (currentRoomIsActive.value) return t('gameRoom.controls.actionsLocked')
  return canBoost.value ? t('gameRoom.controls.boostHint') : t('gameRoom.controls.boostDisabled')
})
const nextRoundAvailable = computed(
  () => Boolean(pendingNextRoundId.value && pendingNextRoundId.value !== activeRoundId.value),
)
const roundTimerValue = computed(() => roundStatus.value?.time_left_seconds ?? '-')

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
    clearPendingNextRound()
    roundStatus.value = null
    liveEvents.value = []
    void loadRoomState(true)
  },
  { immediate: true },
)

watch([activeRoundId, autoRefresh, sseConnected], () => {
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
    if (enabled && nextRoundAvailable.value && pendingNextRoundCountdown.value === 0) {
      void goToNextRound()
    }
  },
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
  router.push('/rooms')
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
  return error?.message || error?.details?.message || error?.details?.error || fallback
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

function refreshCabinetBalance() {
  void cabinet.refresh().catch(() => {})
}

function participantState(participant: GameParticipantInfo) {
  if (participant.exited_at) return t('gameRoom.participant.exited')
  if (participant.is_bot) return t('gameRoom.participant.bot')
  return t('gameRoom.participant.active')
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
}

function stopStatusPolling() {
  if (statusTimer !== null) {
    window.clearInterval(statusTimer)
    statusTimer = null
  }
}

function restartStatusPolling() {
  stopStatusPolling()

  if (!activeRoundId.value || !autoRefresh.value || sseConnected.value) return

  statusTimer = window.setInterval(() => {
    void refreshRoundView(true)
  }, 5000)
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
    if (roundStatus.value && activeRoundId.value === roomRoundId.value) {
      roundStatus.value = {
        ...roundStatus.value,
        status: String(data.status || roundStatus.value.status || 'waiting'),
        time_left_seconds:
          Number(data.seconds_left ?? roundStatus.value.time_left_seconds ?? 0) || 0,
      }
    }
    return
  }

  liveEvents.value = [{ ...event, id: ++liveEventId }, ...liveEvents.value].slice(0, 5)

  if (event.type === 'round_finalized') {
    const data = eventData(event)
    const finalizedRoundId = Number(data.round_id ?? activeRoundId.value ?? 0) || 0
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

async function joinRoom() {
  clearFeedback()
  joining.value = true

  try {
    joinResult.value = await GameApi.joinRoom(roomId.value)
    displayedRoundId.value = null
    clearPendingNextRound()
    successMsg.value = t('gameRoom.messages.joined', { seat: joinResult.value.number_in_room })
    refreshCabinetBalance()
    await refreshRoundView()
  } catch (error: any) {
    errorMsg.value = normalizeError(error, t('gameRoom.errors.join'))
  } finally {
    joining.value = false
  }
}

async function joinRoomWithSeat() {
  clearFeedback()

  const seat = Number(selectedSeat.value)
  if (!Number.isInteger(seat) || seat <= 0) {
    errorMsg.value = t('gameRoom.errors.seatRequired')
    return
  }

  if (occupiedSeats.value.has(seat) && seat !== mySeat.value) {
    errorMsg.value = t('gameRoom.errors.seatTaken')
    return
  }

  joining.value = true

  try {
    joinResult.value = await GameApi.joinRoomWithSeat(roomId.value, { number_in_room: seat })
    displayedRoundId.value = null
    clearPendingNextRound()
    successMsg.value = t('gameRoom.messages.joined', { seat: joinResult.value.number_in_room })
    refreshCabinetBalance()
    await refreshRoundView()
  } catch (error: any) {
    errorMsg.value = normalizeError(error, t('gameRoom.errors.joinSeat'))
  } finally {
    joining.value = false
  }
}

async function purchaseBoost() {
  clearFeedback()

  const participantId = currentParticipantId.value
  if (!participantId) {
    errorMsg.value = t('gameRoom.errors.joinFirst')
    return
  }
  if (hasOwnedBoost.value) {
    errorMsg.value = t('gameRoom.errors.boostAlreadyPurchased')
    return
  }

  actionLoading.value = 'boost'

  try {
    const response = await GameApi.purchaseBoost(roomId.value, participantId)
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

  const participantId = currentParticipantId.value
  if (!participantId) {
    errorMsg.value = t('gameRoom.errors.joinFirst')
    return
  }

  actionLoading.value = 'cancel-boost'

  try {
    const response = await GameApi.cancelBoost(roomId.value, participantId)
    successMsg.value = t('gameRoom.messages.boostCancelled', { refund: response.refund ?? 0 })
    refreshCabinetBalance()
    await refreshRoundView()
  } catch (error: any) {
    errorMsg.value = normalizeError(error, t('gameRoom.errors.cancelBoost'))
  } finally {
    actionLoading.value = ''
  }
}

async function leaveRoom() {
  clearFeedback()

  if (!canLeave.value) {
    errorMsg.value = t('gameRoom.errors.joinFirst')
    return
  }

  const confirmed = window.confirm(t('gameRoom.confirmLeave'))
  if (!confirmed) return

  actionLoading.value = 'leave'

  try {
    const response = await GameApi.leaveRoom(roomId.value)
    successMsg.value = t('gameRoom.messages.left', { refund: response.refund ?? 0 })
    joinResult.value = null
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

async function goToNextRound() {
  if (!pendingNextRoundId.value) return

  displayedRoundId.value = null
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

function scheduleNextRoundTransition(nextRoundId: number | null, nextRoundDelay: number, timestamp: string) {
  clearPendingNextRound()

  if (!nextRoundId || nextRoundId <= 0) return

  pendingNextRoundId.value = nextRoundId

  const finalizedAt = new Date(timestamp).getTime()
  const baseTime = Number.isFinite(finalizedAt) ? finalizedAt : Date.now()
  const targetAt = baseTime + Math.max(0, nextRoundDelay) * 1000

  const tick = () => {
    const secondsLeft = Math.max(0, Math.ceil((targetAt - Date.now()) / 1000))
    pendingNextRoundCountdown.value = secondsLeft
    if (secondsLeft > 0) return

    stopNextRoundCountdown()
    if (autoAdvanceNextRound.value) {
      void goToNextRound()
    }
  }

  tick()
  if (targetAt > Date.now()) {
    nextRoundTimer = window.setInterval(tick, 1000)
  }
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
          <template v-if="isJoined">
            <p class="join-title">
              {{ t('gameRoom.entry.joinedAs', { participant: currentParticipantId ?? '-', seat: mySeat ?? '-' }) }}
            </p>
            <p class="description">
              {{ t('gameRoom.entry.joinedHint') }}
            </p>
          </template>

          <template v-else>
            <p class="join-title">{{ t('gameRoom.entry.notJoinedTitle') }}</p>
            <p class="description">{{ t('gameRoom.entry.notJoinedHint') }}</p>

            <div v-if="seatOptions.length" class="seat-grid" :aria-label="t('gameRoom.entry.seats')">
              <button
                v-for="seat in seatOptions"
                :key="seat"
                class="seat-button"
                :class="{ occupied: occupiedSeats.has(seat), selected: Number(selectedSeat) === seat }"
                type="button"
                :disabled="occupiedSeats.has(seat)"
                @click="selectedSeat = String(seat)"
              >
                {{ seat }}
              </button>
            </div>

            <label class="seat-input">
              <span>{{ t('gameRoom.entry.seatNumber') }}</span>
              <input
                v-model="selectedSeat"
                type="number"
                min="1"
                step="1"
                :placeholder="t('gameRoom.entry.seatPlaceholder')"
              />
            </label>

            <div class="actions stretch">
              <button class="btn btn--primary" type="button" :disabled="joining" @click="joinRoom">
                {{ joining ? t('gameRoom.entry.joining') : t('gameRoom.entry.joinAuto') }}
              </button>
              <button class="btn" type="button" :disabled="joining" @click="joinRoomWithSeat">
                {{ t('gameRoom.entry.joinSeat') }}
              </button>
            </div>
          </template>
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
          <div v-if="nextRoundAvailable" class="meta-item">
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
              :class="{ own: participant.participant_id === activeParticipantId }"
            >
              <span v-if="participant.participant_id === activeParticipantId" class="own-badge">
                {{ t('gameRoom.participant.you') }}
              </span>
              <strong>{{ t('gameRoom.participant.seat', { seat: participant.number_in_room }) }}</strong>
              <span>{{ t('gameRoom.participant.id', { id: participant.participant_id }) }}</span>
              <span v-if="participant.user_id">
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
            <p class="description">
              {{ t('gameRoom.round.nextRoundHint', { seconds: pendingNextRoundCountdown }) }}
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
              :disabled="!hasActiveParticipantBoost || !canManageCurrentRound || actionLoading === 'cancel-boost'"
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
              :disabled="!canLeave || actionLoading === 'leave'"
              @click="leaveRoom"
            >
              {{ actionLoading === 'leave' ? t('common.loading') : t('gameRoom.controls.leave') }}
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

label,
.seat-input,
.participants,
.winners,
.participant-list,
.live-events,
.event-list {
  display: grid;
  gap: 0.55rem;
}

.boost-value {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin: 0;
}

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
