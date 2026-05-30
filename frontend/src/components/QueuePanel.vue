<script setup>
import { ref, computed } from 'vue'
import { useQueue } from '../composables/useQueue.js'
import { api } from '../api.js'
import { t } from '../i18n/index.js'
import { X, Pause, Play, Trash2, AlertCircle, CheckCircle, Loader, RotateCcw } from 'lucide-vue-next'

const {
  jobs, paused, pending, running, completed, failed, pausedJobs,
  currentJob, pendingCount, hasActiveJobs, hasPausedJobs, refresh,
} = useQueue()

const statusOrder = { running: 0, pending: 1, paused: 2, failed: 3, completed: 4 }

const sortedJobs = computed(() => {
  return [...jobs.value].sort((a, b) => {
    const diff = (statusOrder[a.status] ?? 9) - (statusOrder[b.status] ?? 9)
    if (diff !== 0) return diff
    return a.id - b.id
  })
})

const confirmClear = ref(false)
let clearTimer = null

const typeLabels = {
  txt2img: 'Generate',
  from_image: 'From Image',
  batch_item: 'Batch',
  compound: 'Compound',
  compare_item: 'Compare',
}

async function removeJob(id) {
  try {
    await api.removeQueueJob(id)
    await refresh()
  } catch {}
}

async function cancelJob(id) {
  try {
    await api.cancelQueueJob(id)
    await refresh()
  } catch {}
}

async function togglePause() {
  try {
    if (paused.value) {
      await api.resumeQueue()
    } else {
      await api.pauseQueue()
    }
    paused.value = !paused.value
  } catch {}
}

async function cancelAll() {
  try {
    await api.cancelQueue()
    await refresh()
  } catch {}
}

async function clearCompleted() {
  if (!confirmClear.value) {
    confirmClear.value = true
    clearTimeout(clearTimer)
    clearTimer = setTimeout(() => { confirmClear.value = false }, 3000)
    return
  }
  confirmClear.value = false
  clearTimeout(clearTimer)
  try {
    await api.clearCompletedQueueJobs()
    await refresh()
  } catch {}
}

async function resumePausedJobs() {
  try {
    await api.resumePausedQueueJobs()
    await refresh()
  } catch {}
}

function formatProgress(p) {
  return Math.round(p * 100) + '%'
}

function formatTime(ts) {
  if (!ts) return ''
  return ts.replace(/^.*?(\d{2}:\d{2}).*/, '$1')
}
</script>

<template>
  <div class="queue-panel">
    <div class="queue-toolbar">
      <div class="queue-info">
        <span v-if="currentJob" class="queue-active">
          <Loader :size="12" class="spin" />
          {{ typeLabels[currentJob.type] || currentJob.type }}
        </span>
        <span v-if="pendingCount > 0" class="queue-pending-count">
          {{ pendingCount }} pending
        </span>
      </div>
      <div class="queue-controls">
        <button class="q-btn" :class="{ active: paused }" @click="togglePause" :title="paused ? t('queue.resume') : t('queue.pause')">
          <Pause v-if="!paused" :size="12" />
          <Play v-else :size="12" />
          {{ paused ? t('queue.resume') : t('queue.pause') }}
        </button>
        <button v-if="hasPausedJobs" class="q-btn q-btn-resume" @click="resumePausedJobs" :title="t('queue.resume_paused')">
          <RotateCcw :size="12" /> {{ t('queue.resume_paused') }}
        </button>
        <button class="q-btn q-btn-danger" @click="cancelAll" :disabled="!hasActiveJobs" :title="t('queue.cancel_all')">
          <X :size="12" /> {{ t('queue.cancel_all') }}
        </button>
        <button class="q-btn" :class="{ confirm: confirmClear }" @click="clearCompleted" :disabled="completed.length === 0 && failed.length === 0">
          <Trash2 :size="12" />
          {{ confirmClear ? t('queue.confirm_clear') : t('queue.clear_done') }}
        </button>
      </div>
    </div>

    <div class="queue-list">
      <div v-if="jobs.length === 0" class="queue-empty">{{ t('queue.empty') }}</div>

      <div v-for="job in sortedJobs" :key="job.id" class="queue-item" :class="'status-' + job.status">
        <div class="qi-icon">
          <Loader v-if="job.status === 'running'" :size="14" class="spin" />
          <CheckCircle v-else-if="job.status === 'completed'" :size="14" class="text-success" />
          <AlertCircle v-else-if="job.status === 'failed'" :size="14" class="text-danger" />
          <Pause v-else-if="job.status === 'paused'" :size="14" class="text-warning" />
          <div v-else class="qi-dot" :class="{ active: job.status === 'pending' }"></div>
        </div>

        <div class="qi-content">
          <div class="qi-header">
            <span class="qi-type">{{ typeLabels[job.type] || job.type }}</span>
            <span class="qi-time">{{ formatTime(job.created_at) }}</span>
          </div>

          <div v-if="job.status === 'running' && job.progress > 0" class="qi-progress">
            <div class="qi-progress-bar">
              <div class="qi-progress-fill" :style="{ width: formatProgress(job.progress) }"></div>
            </div>
            <span class="qi-progress-text">{{ formatProgress(job.progress) }}</span>
          </div>

          <div v-if="job.status === 'failed' && job.error" class="qi-error" :title="job.error">
            {{ job.error }}
          </div>

          <div v-if="job.status === 'paused'" class="qi-paused-info">
            <span class="qi-paused-label">{{ t('queue.paused_label') }}</span>
            <span v-if="job.retry_count > 0" class="qi-retry-info">({{ job.retry_count }}/{{ job.max_retries }})</span>
          </div>

          <div v-if="job.status === 'pending' && job.retry_count > 0" class="qi-retry-badge">
            {{ t('queue.retrying') }} {{ job.retry_count }}/{{ job.max_retries }}
          </div>
        </div>

        <div class="qi-actions">
          <button v-if="job.status === 'paused'" class="qi-resume" @click="resumePausedJobs" :title="t('queue.resume_paused')">
            <RotateCcw :size="12" />
          </button>
          <button v-if="job.status === 'pending' || job.status === 'failed'" class="qi-remove" @click="removeJob(job.id)" :title="t('queue.remove')">
            <X :size="12" />
          </button>
          <button v-if="job.status === 'running'" class="qi-remove" @click="cancelJob(job.id)" :title="t('queue.cancel')">
            <X :size="12" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default { name: 'QueuePanel' }
</script>

<style scoped>
.queue-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.queue-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 12px;
  background: var(--surface-2);
  border-bottom: 1px solid var(--border);
  gap: 8px;
  flex-wrap: wrap;
}

.queue-info {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}

.queue-active {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--accent);
  font-weight: 500;
}

.queue-pending-count {
  color: var(--text-dim);
}

.queue-controls {
  display: flex;
  gap: 4px;
  align-items: center;
}

.q-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 3px;
  color: var(--text-dim);
  cursor: pointer;
  font-size: 11px;
  padding: 2px 8px;
  height: 24px;
}

.q-btn:hover:not(:disabled) {
  color: var(--text);
  border-color: var(--text-dim);
}

.q-btn:disabled {
  opacity: 0.3;
  cursor: default;
}

.q-btn.active {
  background: var(--accent-bg);
  border-color: var(--accent);
  color: var(--text-bright);
}

.q-btn-danger:hover:not(:disabled) {
  color: var(--danger, #e55);
  border-color: var(--danger, #e55);
}

.q-btn.confirm {
  color: #fff;
  background: var(--danger, #e55);
  border-color: var(--danger, #e55);
}

.queue-list {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}

.queue-empty {
  text-align: center;
  color: var(--text-dim);
  padding: 20px;
  font-size: 13px;
}

.queue-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  border-bottom: 1px solid transparent;
}

.queue-item:hover {
  background: var(--surface-2);
}

.queue-item.status-running {
  background: rgba(124, 92, 252, 0.05);
}

.queue-item.status-paused {
  background: rgba(240, 160, 48, 0.08);
}

.qi-icon {
  flex-shrink: 0;
  width: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.qi-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--text-dim);
  opacity: 0.4;
}

.qi-dot.active {
  opacity: 1;
  background: var(--warning, #f0a030);
}

.text-success { color: var(--success); }
.text-danger { color: var(--danger, #e55); }

.qi-content {
  flex: 1;
  min-width: 0;
}

.qi-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
}

.qi-type {
  font-weight: 500;
  color: var(--text);
}

.qi-time {
  color: var(--text-dim);
  font-size: 10px;
}

.qi-progress {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 4px;
}

.qi-progress-bar {
  flex: 1;
  height: 4px;
  background: var(--surface-2);
  border-radius: 2px;
  overflow: hidden;
}

.qi-progress-fill {
  height: 100%;
  background: var(--accent);
  border-radius: 2px;
  transition: width 0.3s ease;
}

.qi-progress-text {
  font-size: 10px;
  color: var(--text-dim);
  width: 32px;
  text-align: right;
}

.qi-error {
  font-size: 11px;
  color: var(--danger, #e55);
  margin-top: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.qi-paused-info {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 2px;
  font-size: 11px;
}

.qi-paused-label {
  color: var(--warning, #f0a030);
}

.qi-retry-info {
  color: var(--text-dim);
  font-size: 10px;
}

.qi-retry-badge {
  font-size: 10px;
  color: var(--warning, #f0a030);
  margin-top: 2px;
}

.text-warning { color: var(--warning, #f0a030); }

.qi-actions {
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.15s;
}

.queue-item:hover .qi-actions {
  opacity: 1;
}

.qi-remove {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 3px;
  color: var(--text-dim);
  cursor: pointer;
  padding: 0;
}

.qi-remove:hover {
  color: var(--danger, #e55);
  border-color: var(--danger, #e55);
}

.qi-resume {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 3px;
  color: var(--warning, #f0a030);
  cursor: pointer;
  padding: 0;
}

.qi-resume:hover {
  color: var(--success);
  border-color: var(--success);
}

.q-btn-resume {
  color: var(--warning, #f0a030);
  border-color: var(--warning, #f0a030);
}

.q-btn-resume:hover {
  color: var(--success);
  border-color: var(--success);
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
