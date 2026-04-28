<script setup>
import { ref, computed } from 'vue'
import { api } from '../api.js'

const props = defineProps({
  presets: { type: Array, required: true },
})
const emit = defineEmits(['done', 'close'])

const selected = ref(props.presets.map(() => true))
const importing = ref(false)
const error = ref('')

const grouped = computed(() => {
  const map = new Map()
  for (let i = 0; i < props.presets.length; i++) {
    const file = props.presets[i].source_file || 'Unknown file'
    if (!map.has(file)) map.set(file, [])
    map.get(file).push(i)
  }
  return map
})

const fileCount = computed(() => grouped.value.size)

const allSelected = computed(() => selected.value.every(Boolean))
const selectedCount = computed(() => selected.value.filter(Boolean).length)

function toggleAll() {
  const val = !allSelected.value
  selected.value = selected.value.map(() => val)
}

function toggleFile(indices) {
  const allOn = indices.every(i => selected.value[i])
  for (const i of indices) {
    selected.value[i] = !allOn
  }
}

async function doImport() {
  const items = props.presets.filter((_, i) => selected.value[i])
  if (items.length === 0) return
  importing.value = true
  error.value = ''
  try {
    await api.importPresets(items)
    emit('done')
  } catch (e) {
    error.value = e.message || String(e)
  } finally {
    importing.value = false
  }
}
</script>

<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h2 class="modal-title">Import Presets ({{ presets.length }} from {{ fileCount }} file{{ fileCount > 1 ? 's' : '' }})</h2>
        <button class="modal-close" @click="$emit('close')">&times;</button>
      </div>

      <div v-if="error" class="import-error">{{ error }}</div>

      <div style="margin-bottom: 12px;">
        <label class="import-select-all">
          <input type="checkbox" :checked="allSelected" @change="toggleAll" />
          Select all
        </label>
        <span class="import-counter">{{ selectedCount }} selected</span>
      </div>

      <div class="import-list">
        <template v-for="[file, indices] of grouped" :key="file">
          <div v-if="fileCount > 1" class="import-file-header" @click="toggleFile(indices)">
            <input type="checkbox" :checked="indices.every(i => selected[i])" @click.stop="toggleFile(indices)" />
            <span>{{ file }}</span>
            <span class="import-file-count">{{ indices.length }}</span>
          </div>
          <label v-for="i in indices" :key="i" class="import-item" :class="{ 'import-item-indented': fileCount > 1 }">
            <input type="checkbox" v-model="selected[i]" />
            <div class="import-item-info">
              <div class="import-item-name">{{ presets[i].name || 'Untitled' }}</div>
              <div class="import-item-meta">
                <span>{{ presets[i].preset_type || 'general' }}</span>
                <span>{{ presets[i].sampler }}</span>
                <span>{{ presets[i].width }}x{{ presets[i].height }}</span>
              </div>
            </div>
          </label>
        </template>
      </div>

      <div style="display: flex; gap: 10px; justify-content: flex-end; margin-top: 20px;">
        <button class="btn btn-secondary" @click="$emit('close')">Cancel</button>
        <button class="btn btn-primary" :disabled="importing || selectedCount === 0" @click="doImport">
          {{ importing ? 'Importing...' : `Import (${selectedCount})` }}
        </button>
      </div>
    </div>
  </div>
</template>
