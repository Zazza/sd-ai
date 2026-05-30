import { ref, computed, onMounted, onUnmounted } from 'vue'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'
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

export function useQueue() {
  onMounted(async () => {
    subscribers++
    if (subscribers === 1) {
      EventsOn('queue:changed', refresh)
      EventsOn('queue:started', refresh)
      EventsOn('queue:progress', refresh)
      EventsOn('queue:completed', refresh)
      EventsOn('queue:failed', refresh)
      EventsOn('queue:paused', refresh)
      await refresh()
    }
  })

  onUnmounted(() => {
    subscribers--
    if (subscribers <= 0) {
      subscribers = 0
      EventsOff('queue:changed')
      EventsOff('queue:started')
      EventsOff('queue:progress')
      EventsOff('queue:completed')
      EventsOff('queue:failed')
      EventsOff('queue:paused')
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
