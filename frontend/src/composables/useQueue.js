import { ref, computed, onMounted, onUnmounted } from 'vue'
import { EventsOn } from '../wailsjs/runtime/runtime'
import { api } from '../api.js'

const jobs = ref([])
const paused = ref(false)
const loading = ref(false)

const pending = computed(() => jobs.value.filter(j => j.status === 'pending'))
const running = computed(() => jobs.value.filter(j => j.status === 'running'))
const completed = computed(() => jobs.value.filter(j => j.status === 'completed'))
const failed = computed(() => jobs.value.filter(j => j.status === 'failed'))
const pausedJobs = computed(() => jobs.value.filter(j => j.status === 'paused'))
const currentJob = computed(() => running.value[0] || null)
const pendingCount = computed(() => pending.value.length)
const hasActiveJobs = computed(() => jobs.value.some(j => j.status === 'pending' || j.status === 'running'))
const hasPausedJobs = computed(() => pausedJobs.value.length > 0)

let subscribers = 0
let offChanged, offStarted, offProgress, offCompleted, offFailed, offPaused

async function refresh() {
  try {
    loading.value = true
    const result = await api.getQueue()
    jobs.value = result || []
    paused.value = await api.isQueuePaused().catch(() => false)
  } catch (e) {
    console.error('queue refresh error:', e)
  } finally {
    loading.value = false
  }
}

function subscribe() {
  offChanged = EventsOn('queue:changed', refresh)
  offStarted = EventsOn('queue:started', refresh)
  offProgress = EventsOn('queue:progress', refresh)
  offCompleted = EventsOn('queue:completed', refresh)
  offFailed = EventsOn('queue:failed', refresh)
  offPaused = EventsOn('queue:paused', refresh)
}

function unsubscribe() {
  offChanged?.()
  offStarted?.()
  offProgress?.()
  offCompleted?.()
  offFailed?.()
  offPaused?.()
}

export function useQueue() {
  onMounted(async () => {
    subscribers++
    if (subscribers === 1) {
      subscribe()
      await refresh()
    }
  })

  onUnmounted(() => {
    subscribers--
    if (subscribers <= 0) {
      subscribers = 0
      unsubscribe()
    }
  })

  return {
    jobs,
    paused,
    loading,
    pending,
    running,
    completed,
    failed,
    pausedJobs,
    currentJob,
    pendingCount,
    hasActiveJobs,
    hasPausedJobs,
    refresh,
  }
}
