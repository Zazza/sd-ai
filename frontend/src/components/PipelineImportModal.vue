<script setup>
import { ref, computed } from 'vue'
import { api } from '../api.js'
import { t } from '../i18n/index.js'

const props = defineProps({
  pipelines: { type: Array, required: true },
})
const emit = defineEmits(['done', 'close'])

const selected = ref(props.pipelines.map(() => true))
const importing = ref(false)
const error = ref('')

const allSelected = computed(() => selected.value.length > 0 && selected.value.every(Boolean))
const selectedCount = computed(() => selected.value.filter(Boolean).length)

function toggleAll() {
  const val = !allSelected.value
  selected.value = selected.value.map(() => val)
}

async function doImport() {
  const items = props.pipelines.filter((_, i) => selected.value[i])
  if (items.length === 0) return
  importing.value = true
  error.value = ''
  try {
    await api.importCompoundPresets(items)
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
        <h2 class="modal-title">{{ t('pipeline_import.title', { count: pipelines.length }) }}</h2>
        <button class="modal-close" @click="$emit('close')">&times;</button>
      </div>

      <div v-if="error" class="import-error">{{ error }}</div>

      <div style="margin-bottom: 12px;">
        <label class="import-select-all">
          <input type="checkbox" :checked="allSelected" @change="toggleAll" />
          {{ t('pipeline_import.select_all') }}
        </label>
        <span class="import-counter">{{ t('pipeline_import.selected', { count: selectedCount }) }}</span>
      </div>

      <div class="import-list">
        <label v-for="(p, i) in pipelines" :key="i" class="import-item">
          <input type="checkbox" v-model="selected[i]" />
          <div class="import-item-info">
            <div class="import-item-name">{{ p.name || t('pipeline_import.untitled') }}</div>
            <div class="import-item-meta">
              <span>{{ p.steps.length }} steps</span>
              <span v-if="p.description">{{ p.description }}</span>
            </div>
          </div>
        </label>
      </div>

      <div style="display: flex; gap: 10px; justify-content: flex-end; margin-top: 20px;">
        <button class="btn btn-secondary" @click="$emit('close')">{{ t('pipeline_import.btn_cancel') }}</button>
        <button class="btn btn-primary" :disabled="importing || selectedCount === 0" @click="doImport">
          {{ importing ? t('pipeline_import.importing') : selectedCount === 0 ? t('pipeline_import.btn_import_short') : t('pipeline_import.btn_import', { count: selectedCount }) }}
        </button>
      </div>
    </div>
  </div>
</template>
