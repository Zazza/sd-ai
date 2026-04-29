<script setup>
import { ref, reactive, onMounted, onUnmounted, nextTick } from 'vue'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'
import { api } from '../api.js'

const expanded = ref(false)
const logs = reactive([])
const logContainer = ref(null)
const filterLevel = ref('all')

const services = reactive({
  llm: { available: false, label: 'LLM' },
  sd: { available: false, label: 'SD' },
  rembg: { available: false, label: 'Rembg' },
})

function addLog(raw) {
  try {
    const entry = JSON.parse(raw)
    logs.push(entry)
    if (logs.length > 500) {
      logs.splice(0, logs.length - 500)
    }
    if (expanded.value) {
      nextTick(() => {
        if (logContainer.value) {
          logContainer.value.scrollTop = logContainer.value.scrollHeight
        }
      })
    }
  } catch {}
}

function toggleLog() {
  expanded.value = !expanded.value
}

function levelClass(level) {
  switch (level) {
    case 'error': return 'log-error'
    case 'warn': return 'log-warn'
    case 'info': return 'log-info'
    case 'debug': return 'log-debug'
    default: return ''
  }
}

const filteredLogs = () => {
  if (filterLevel.value === 'all') return logs
  return logs.filter(l => l.level === filterLevel.value)
}

async function checkServices() {
  try {
    const status = await api.checkServices()
    services.llm.available = status.llm?.available || false
    services.sd.available = status.sd?.available || false
  } catch {
    services.llm.available = false
    services.sd.available = false
  }

  try {
    await api.checkRembg()
    services.rembg.available = true
  } catch {
    services.rembg.available = false
  }
}

onMounted(() => {
  EventsOn('log:entry', addLog)
  checkServices()
  const interval = setInterval(checkServices, 30000)
  onUnmounted(() => {
    EventsOff('log:entry')
    clearInterval(interval)
  })
})
</script>

<template>
  <div class="app-footer" :class="{ expanded }">
    <div class="footer-bar" @click="toggleLog">
      <div class="footer-status">
        <span v-for="(svc, key) in services" :key="key"
              class="status-dot"
              :class="{ online: svc.available, offline: !svc.available }">
          {{ svc.label }}
        </span>
      </div>
      <span class="footer-toggle">{{ expanded ? 'Hide Log' : 'Show Log' }}</span>
    </div>
    <div v-if="expanded" class="footer-log">
      <div class="log-toolbar">
        <div class="log-filters">
          <button class="filter-btn" :class="{ active: filterLevel === 'all' }" @click.stop="filterLevel = 'all'">All</button>
          <button class="filter-btn" :class="{ active: filterLevel === 'error' }" @click.stop="filterLevel = 'error'">Errors</button>
          <button class="filter-btn" :class="{ active: filterLevel === 'warn' }" @click.stop="filterLevel = 'warn'">Warnings</button>
          <button class="filter-btn" :class="{ active: filterLevel === 'info' }" @click.stop="filterLevel = 'info'">Info</button>
          <button class="filter-btn" :class="{ active: filterLevel === 'debug' }" @click.stop="filterLevel = 'debug'">Debug</button>
        </div>
        <button class="filter-btn" @click.stop="logs.splice(0, logs.length)">Clear</button>
      </div>
      <div class="log-entries" ref="logContainer">
        <div v-for="(entry, i) in filteredLogs()" :key="i" class="log-entry" :class="levelClass(entry.level)">
          <span class="log-time">{{ entry.timestamp }}</span>
          <span class="log-level">{{ entry.level.toUpperCase() }}</span>
          <span class="log-msg">{{ entry.message }}</span>
        </div>
        <div v-if="logs.length === 0" class="log-empty">No log entries yet</div>
      </div>
    </div>
  </div>
</template>

<script>
export default { name: 'AppFooter' }
</script>

<style scoped>
.app-footer {
  border-top: 1px solid var(--border);
  background: var(--surface);
  flex-shrink: 0;
}

.app-footer.expanded {
  height: 40vh;
  display: flex;
  flex-direction: column;
}

.footer-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 16px;
  cursor: pointer;
  user-select: none;
}

.footer-bar:hover {
  background: var(--surface-2);
}

.footer-status {
  display: flex;
  gap: 12px;
}

.status-dot {
  font-size: 12px;
  font-weight: 500;
  padding: 1px 6px;
  border-radius: 3px;
}

.status-dot.online {
  color: var(--success);
}

.status-dot.offline {
  color: var(--text-dim);
}

.footer-toggle {
  font-size: 11px;
  color: var(--text-dim);
}

.footer-log {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-top: 1px solid var(--border);
}

.log-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 12px;
  background: var(--surface-2);
  border-bottom: 1px solid var(--border);
}

.log-filters {
  display: flex;
  gap: 4px;
}

.filter-btn {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-dim);
  padding: 2px 8px;
  border-radius: 3px;
  font-size: 11px;
  cursor: pointer;
}

.filter-btn.active {
  background: var(--accent-bg);
  border-color: var(--accent);
  color: var(--text-bright);
}

.filter-btn:hover {
  border-color: var(--text-dim);
}

.log-entries {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
  font-family: 'Menlo', 'Consolas', 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.6;
}

.log-entry {
  display: flex;
  gap: 8px;
  padding: 0 12px;
  white-space: nowrap;
}

.log-entry:hover {
  background: var(--surface-2);
}

.log-time {
  color: var(--text-dim);
  flex-shrink: 0;
  width: 60px;
}

.log-level {
  flex-shrink: 0;
  width: 44px;
  font-weight: 600;
  font-size: 11px;
}

.log-error .log-level { color: var(--danger); }
.log-error { background: rgba(224, 85, 85, 0.06); }
.log-warn .log-level { color: var(--warning); }
.log-info .log-level { color: var(--accent); }
.log-debug .log-level { color: var(--text-dim); }
.log-debug { color: var(--text-dim); }

.log-msg {
  overflow: hidden;
  text-overflow: ellipsis;
}

.log-empty {
  text-align: center;
  color: var(--text-dim);
  padding: 20px;
  font-family: inherit;
  font-size: 13px;
}
</style>
