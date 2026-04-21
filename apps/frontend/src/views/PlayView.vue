<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import LeftTab from '@/components/LeftTab.vue'
import UpTab from '@/components/UpTab.vue'
import FooterTab from '@/components/FooterTab.vue'
import { GameApi } from '@/api/useMatchApi'
import type { GameJoinRoomResponse, GameParticipantInfo, GameRoundStatusResponse } from '@/api/types'
import { useMatchSessionStore } from '@/stores/matchSessionStore'
import { useUserCabinetStore } from '@/stores/userCabinetStore'
import { useI18n } from '@/i18n'
import { useLayoutInset } from '@/composables/useLayoutInset'

const route = useRoute()
const router = useRouter()
const session = useMatchSessionStore()
const cabinet = useUserCabinetStore()
const { t } = useI18n()
const { LeftTabHidden: leftHidden, layoutInset } = useLayoutInset()

const roomId = computed(() => Number(route.params.roomId))

const joinResult = ref<GameJoinRoomResponse | null>(null)
const roundStatus = ref<GameRoundStatusResponse | null>(null)
const joining = ref(false)
const statusLoading = ref(false)
const actionLoading = ref('')
const errorMsg = ref('')
const successMsg = ref('')
const selectedSeat = ref('')
const boostValue = ref('')
const autoRefresh = ref(true)
let statusTimer: number | null = null

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

const activeRoundId = computed(() => {
  const roundId = joinResult.value?.round_id || quickMatchMeta.value?.round_id || 0
  return roundId > 0 ? roundId : null
})

const activeParticipantId = computed(() => {
  const participantId =
    joinResult.value?.participant_id || quickMatchMeta.value?.round_participant_id || 0
  return participantId > 0 ? participantId : null
})

const mySeat = computed(() => {
  const seat = joinResult.value?.number_in_room || quickMatchMeta.value?.seat_number || 0
  return seat > 0 ? seat : null
})

const roomCapacity = computed(() => joinResult.value?.room_capacity || room.value?.capacity || 0)
const entryPrice = computed(() => joinResult.value?.entry_price ?? room.value?.registration_price ?? '-')
const minPlayers = computed(() => joinResult.value?.min_players ?? room.value?.min_users ?? '-')
const currentPlayers = computed(() => {
  if (roundStatus.value?.participants?.length) {
    return roundStatus.value.participants.filter((participant) => !participant.exited_at).length
  }

  return joinResult.value?.current_players ?? room.value?.current_players ?? 0
})

const statusLabel = computed(() => roundStatus.value?.status || joinResult.value?.round_status || '-')

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
const canBoost = computed(() => Boolean(activeParticipantId.value) && room.value?.is_boost !== false)
const isJoined = computed(() => Boolean(activeParticipantId.value && activeRoundId.value))

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

    successMsg.value = t('gameRoom.messages.quickSessionRestored')
    void loadRoundStatus(true)
  },
  { immediate: true },
)

watch(
  () => room.value,
  (nextRoom) => {
    if (!boostValue.value && nextRoom?.boost_power) {
      boostValue.value = String(nextRoom.boost_power)
    }
  },
  { immediate: true },
)

watch([activeRoundId, autoRefresh], () => {
  restartStatusPolling()
})

onBeforeUnmount(() => {
  stopStatusPolling()
})

function backToHome() {
  router.push('/')
}

function openRooms() {
  router.push('/rooms')
}

function formatBoost() {
  if (!room.value) return t('common.off')
  if (!room.value.is_boost) return t('common.off')
  return t('matchmaking.results.boostValue', { value: room.value.boost_power })
}

function formatScore() {
  if (!room.value) return '-'
  return room.value.score.toFixed(2)
}

function normalizeError(error: any, fallback: string) {
  return error?.message || error?.details?.message || error?.details?.error || fallback
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

function stopStatusPolling() {
  if (statusTimer !== null) {
    window.clearInterval(statusTimer)
    statusTimer = null
  }
}

function restartStatusPolling() {
  stopStatusPolling()

  if (!activeRoundId.value || !autoRefresh.value) return

  statusTimer = window.setInterval(() => {
    void loadRoundStatus(true)
  }, 5000)
}

async function loadRoundStatus(silent = false) {
  if (!activeRoundId.value) return

  if (!silent) {
    statusLoading.value = true
    errorMsg.value = ''
  }

  try {
    roundStatus.value = await GameApi.getRoundStatus(roomId.value, activeRoundId.value)
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
    successMsg.value = t('gameRoom.messages.joined', { seat: joinResult.value.number_in_room })
    refreshCabinetBalance()
    await loadRoundStatus()
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
    successMsg.value = t('gameRoom.messages.joined', { seat: joinResult.value.number_in_room })
    refreshCabinetBalance()
    await loadRoundStatus()
  } catch (error: any) {
    errorMsg.value = normalizeError(error, t('gameRoom.errors.joinSeat'))
  } finally {
    joining.value = false
  }
}

async function purchaseBoost() {
  clearFeedback()

  const participantId = activeParticipantId.value
  const value = Number(boostValue.value)
  if (!participantId) {
    errorMsg.value = t('gameRoom.errors.joinFirst')
    return
  }
  if (!Number.isInteger(value) || value <= 0) {
    errorMsg.value = t('gameRoom.errors.boostRequired')
    return
  }

  actionLoading.value = 'boost'

  try {
    const response = await GameApi.purchaseBoost(roomId.value, participantId, { boost_value: value })
    successMsg.value = t('gameRoom.messages.boostPurchased', {
      power: response.boost_power,
      cost: response.boost_cost,
    })
    refreshCabinetBalance()
    await loadRoundStatus()
  } catch (error: any) {
    errorMsg.value = normalizeError(error, t('gameRoom.errors.boost'))
  } finally {
    actionLoading.value = ''
  }
}

async function cancelBoost() {
  clearFeedback()

  const participantId = activeParticipantId.value
  if (!participantId) {
    errorMsg.value = t('gameRoom.errors.joinFirst')
    return
  }

  actionLoading.value = 'cancel-boost'

  try {
    const response = await GameApi.cancelBoost(roomId.value, participantId)
    successMsg.value = t('gameRoom.messages.boostCancelled', { refund: response.refund ?? 0 })
    refreshCabinetBalance()
    await loadRoundStatus()
  } catch (error: any) {
    errorMsg.value = normalizeError(error, t('gameRoom.errors.cancelBoost'))
  } finally {
    actionLoading.value = ''
  }
}

async function leaveRoom() {
  clearFeedback()

  const participantId = activeParticipantId.value
  if (!participantId) {
    errorMsg.value = t('gameRoom.errors.joinFirst')
    return
  }

  const confirmed = window.confirm(t('gameRoom.confirmLeave'))
  if (!confirmed) return

  actionLoading.value = 'leave'

  try {
    const response = await GameApi.leaveRoom(roomId.value, participantId)
    successMsg.value = t('gameRoom.messages.left', { refund: response.refund ?? 0 })
    joinResult.value = null
    roundStatus.value = null
    stopStatusPolling()
    refreshCabinetBalance()
  } catch (error: any) {
    errorMsg.value = normalizeError(error, t('gameRoom.errors.leave'))
  } finally {
    actionLoading.value = ''
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
              {{ t('gameRoom.entry.joinedAs', { participant: activeParticipantId ?? '-', seat: mySeat ?? '-' }) }}
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
          <button class="btn" type="button" :disabled="!activeRoundId || statusLoading" @click="loadRoundStatus()">
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
            <strong>{{ roundStatus?.time_left_seconds ?? '-' }}</strong>
          </div>
          <div class="meta-item">
            <span>{{ t('gameRoom.round.autoRefresh') }}</span>
            <label class="switch-line">
              <input v-model="autoRefresh" type="checkbox" />
              <span>{{ autoRefresh ? t('common.yes') : t('common.no') }}</span>
            </label>
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
              <strong>{{ t('gameRoom.participant.seat', { seat: participant.number_in_room }) }}</strong>
              <span>{{ t('gameRoom.participant.id', { id: participant.participant_id }) }}</span>
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
            {{ t('gameRoom.participant.seat', { seat: winner.number_in_room }) }} ·
            {{ winner.winning_money }}
          </span>
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
            {{ canBoost ? t('gameRoom.controls.boostHint') : t('gameRoom.controls.boostDisabled') }}
          </p>
          <label>
            <span>{{ t('gameRoom.controls.boostValue') }}</span>
            <input v-model="boostValue" type="number" min="1" step="1" placeholder="10" />
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
              :disabled="!activeParticipantId || actionLoading === 'cancel-boost'"
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
              :disabled="!activeParticipantId || actionLoading === 'leave'"
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
.join-box {
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
label span {
  color: var(--color-muted);
}

.source-pill,
.winner-chip {
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

.meta-item,
.participant-card,
.control-block,
.join-box {
  padding: 0.85rem;
  border-radius: 1rem;
  border: 1px solid color-mix(in oklab, var(--color-border), transparent 10%);
  background: color-mix(in oklab, var(--color-surface), white 10%);
}

.join-box.joined {
  border-color: color-mix(in oklab, var(--color-success), transparent 28%);
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
.participant-list {
  display: grid;
  gap: 0.55rem;
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
  .meta-grid {
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
