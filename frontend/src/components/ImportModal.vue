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

const allSelected = computed(() => selected.value.every(Boolean))
const selectedCount = computed(() => selected.value.filter(Boolean).length)

function toggleAll() {
  const val = !allSelected.value
  selected.value = selected.value.map(() => val)
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
        <h2 class="modal-title">Import Presets ({{ presets.length }})</h2>
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
        <label v-for="(p, i) in presets" :key="i" class="import-item">
          <input type="checkbox" v-model="selected[i]" />
          <div class="import-item-info">
            <div class="import-item-name">{{ p.name || 'Untitled' }}</div>
            <div class="import-item-meta">
              <span>{{ p.preset_type || 'general' }}</span>
              <span>{{ p.sampler }}</span>
              <span>{{ p.width }}x{{ p.height }}</span>
            </div>
          </div>
        </label>
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
