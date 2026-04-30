<script setup>
import { ref, reactive, computed, onMounted, onUnmounted, provide } from 'vue'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'
import { api } from '../api.js'
import GeneratePage from './GeneratePage.vue'
import GenerateFromImagePage from './GenerateFromImagePage.vue'
import BatchPage from './BatchPage.vue'
import TestPage from './TestPage.vue'

const activeTab = ref('generate')
const batchProps = ref({})
const droppedImage = ref(null)

const batchKey = computed(() => JSON.stringify(batchProps.value))

const shared = reactive({
  description: '',
  negative: '',
  selectedPresetId: null,
  selectedCompoundPresetId: null,
  genMode: 'preset',
})

provide('sharedGenState', shared)

function onNavigateToBatch(data) {
  batchProps.value = {
    prefillPrompt: data?.prefillPrompt || '',
    prefillNegative: data?.prefillNegative || '',
    prefillPresetId: data?.prefillPresetId || null,
    prefillCompoundPresetId: data?.prefillCompoundPresetId || null,
  }
  if (data?.prefillPresetId) shared.selectedPresetId = data.prefillPresetId
  if (data?.prefillCompoundPresetId) {
    shared.selectedCompoundPresetId = data.prefillCompoundPresetId
    shared.genMode = 'compound'
  }
  activeTab.value = 'batch'
}

const isDragOver = ref(false)

function onDragOver(e) {
  if (e.dataTransfer?.types?.includes('Files')) {
    e.preventDefault()
    isDragOver.value = true
  }
}

function onDragLeave(e) {
  if (!e.currentTarget?.contains(e.relatedTarget)) {
    isDragOver.value = false
  }
}

function onDrop(e) {
  e.preventDefault()
  isDragOver.value = false
  const files = e.dataTransfer?.files
  if (files?.length) {
    for (const file of files) {
      if (file.type.startsWith('image/')) {
        const reader = new FileReader()
        reader.onload = () => {
          droppedImage.value = reader.result
          activeTab.value = 'from-image'
        }
        reader.readAsDataURL(file)
        return
      }
    }
  }
}

onMounted(async () => {
  EventsOn('navigate:batch', onNavigateToBatch)
  try {
    const s = await api.getSettings()
    if (s.gen_preset_id) shared.selectedPresetId = Number(s.gen_preset_id)
    if (s.gen_compound_preset_id) shared.selectedCompoundPresetId = Number(s.gen_compound_preset_id)
    if (s.gen_mode) shared.genMode = s.gen_mode
    if (s.gen_description) shared.description = s.gen_description
    if (s.gen_negative) shared.negative = s.gen_negative
  } catch {}
})

onUnmounted(() => {
  EventsOff('navigate:batch')
  api.updateSettings({
    gen_preset_id: String(shared.selectedPresetId || ''),
    gen_compound_preset_id: String(shared.selectedCompoundPresetId || ''),
    gen_mode: shared.genMode,
    gen_description: shared.description || '',
    gen_negative: shared.negative || '',
  }).catch(() => {})
})
</script>

<template>
  <div class="page-enter" @dragover="onDragOver" @dragleave="onDragLeave" @drop="onDrop">
    <div v-if="isDragOver" class="drop-zone-overlay">
      Drop image for img2img mode
    </div>
    <div class="tabs">
      <button class="tab" :class="{ active: activeTab === 'generate' }" @click="activeTab = 'generate'">Generate</button>
      <button class="tab" :class="{ active: activeTab === 'from-image' }" @click="activeTab = 'from-image'">From Image</button>
      <button class="tab" :class="{ active: activeTab === 'batch' }" @click="activeTab = 'batch'">Batch</button>
      <button class="tab" :class="{ active: activeTab === 'test' }" @click="activeTab = 'test'">Compare</button>
    </div>
    <div class="unified-tab-content">
      <GeneratePage v-if="activeTab === 'generate'" />
      <GenerateFromImagePage v-else-if="activeTab === 'from-image'" :dropped-image="droppedImage" @clear-dropped="droppedImage = null" @transfer-tags="activeTab = 'generate'" />
      <BatchPage v-else-if="activeTab === 'batch'" v-bind="batchProps" :key="batchKey" />
      <TestPage v-else-if="activeTab === 'test'" />
    </div>
  </div>
</template>
