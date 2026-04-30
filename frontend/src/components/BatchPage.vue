<script setup>
import { ref, watch, onMounted, onUnmounted, inject } from 'vue'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'
import { BatchGenerate, BatchCompoundGenerate, SelectFolder } from '../wailsjs/go/main/App.js'
import { api } from '../api.js'

const shared = inject('sharedGenState', null)

const presets = ref([])
const compoundPresets = ref([])
const selectedPresetId = ref(null)
const selectedCompoundPresetId = ref(null)
const batchMode = ref('preset')
const prompt = ref('')
const negativePrompt = ref('')
const count = ref(4)
const outputFolder = ref('')
const generating = ref(false)
const error = ref('')
const progress = ref(null)
const generatedFiles = ref([])

const props = defineProps({
  prefillPrompt: { type: String, default: '' },
  prefillNegative: { type: String, default: '' },
  prefillPresetId: { type: Number, default: null },
  prefillCompoundPresetId: { type: Number, default: null },
})

async function loadPresets() {
  try {
    const [p, c] = await Promise.all([api.listPresets(), api.listCompoundPresets()])
    presets.value = p || []
    compoundPresets.value = c || []
  } catch (e) {
    console.error(e)
  }
}

async function selectFolder() {
  try {
    const folder = await SelectFolder()
    if (folder) {
      outputFolder.value = folder
    }
  } catch (e) {
    error.value = String(e)
  }
}

async function startGeneration() {
  if (!prompt.value.trim()) {
    error.value = 'Prompt is required'
    return
  }
  if (!outputFolder.value.trim()) {
    error.value = 'Output folder is required'
    return
  }
  if (count.value < 1 || count.value > 100) {
    error.value = 'Count must be between 1 and 100'
    return
  }

  generating.value = true
  error.value = ''
  progress.value = null
  generatedFiles.value = []

  try {
    if (batchMode.value === 'compound') {
      await BatchCompoundGenerate({
        compound_preset_id: selectedCompoundPresetId.value,
        extra_prompt: prompt.value,
        extra_negative_prompt: negativePrompt.value,
        count: count.value,
        output_folder: outputFolder.value,
      })
    } else {
      await BatchGenerate({
        preset_id: selectedPresetId.value || 0,
        prompt: prompt.value,
        negative_prompt: negativePrompt.value,
        count: count.value,
        output_folder: outputFolder.value,
      })
    }
  } catch (e) {
    error.value = String(e)
  } finally {
    generating.value = false
  }
}

function onBatchProgress(data) {
  progress.value = data
  if (data.file_path) {
    generatedFiles.value.push(data.file_path)
  }
}

onMounted(async () => {
  loadPresets()
  EventsOn('batch:progress', onBatchProgress)

  if (props.prefillPrompt) {
    prompt.value = props.prefillPrompt
  }
  if (props.prefillNegative) {
    negativePrompt.value = props.prefillNegative
  }
  if (props.prefillPresetId) {
    selectedPresetId.value = props.prefillPresetId
  }
  if (props.prefillCompoundPresetId) {
    selectedCompoundPresetId.value = props.prefillCompoundPresetId
    batchMode.value = 'compound'
  }

  try {
    const s = await api.getSettings()
    if (!props.prefillPrompt && s.batch_prompt) prompt.value = s.batch_prompt
    if (!props.prefillNegative && s.batch_negative) negativePrompt.value = s.batch_negative
    if (!props.prefillPresetId && s.batch_preset_id) selectedPresetId.value = Number(s.batch_preset_id)
    if (!props.prefillCompoundPresetId && s.batch_compound_preset_id) {
      selectedCompoundPresetId.value = Number(s.batch_compound_preset_id)
      batchMode.value = 'compound'
    }
    if (s.batch_mode) batchMode.value = s.batch_mode
    if (s.batch_output_folder) outputFolder.value = s.batch_output_folder
    if (s.batch_count) count.value = Number(s.batch_count) || 4
  } catch {}

  if (shared) {
    if (shared.selectedPresetId) selectedPresetId.value = shared.selectedPresetId
    if (shared.selectedCompoundPresetId) selectedCompoundPresetId.value = shared.selectedCompoundPresetId
    if (shared.genMode) batchMode.value = shared.genMode
    if (!props.prefillPrompt && shared.description) prompt.value = shared.description
    if (!props.prefillNegative && shared.negative) negativePrompt.value = shared.negative
  }
})

onUnmounted(() => {
  EventsOff('batch:progress')
  saveBatchState()
  if (shared) {
    shared.selectedPresetId = selectedPresetId.value
    shared.selectedCompoundPresetId = selectedCompoundPresetId.value
    shared.genMode = batchMode.value
    if (prompt.value) shared.description = prompt.value
    if (negativePrompt.value) shared.negative = negativePrompt.value
  }
})

function saveBatchState() {
  api.updateSettings({
    batch_preset_id: String(selectedPresetId.value || ''),
    batch_compound_preset_id: String(selectedCompoundPresetId.value || ''),
    batch_mode: batchMode.value,
    batch_prompt: prompt.value || '',
    batch_negative: negativePrompt.value || '',
    batch_count: String(count.value || ''),
    batch_output_folder: outputFolder.value || '',
  }).catch(() => {})
}
</script>

<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Batch Generation</h1>
    </div>

    <div v-if="error" class="status status-error">{{ error }}</div>

    <div class="card" style="max-width: 700px;">
      <div style="display: flex; gap: 8px; margin-bottom: 12px;">
        <button class="btn btn-sm" :class="batchMode === 'preset' ? 'btn-primary' : 'btn-secondary'" @click="batchMode = 'preset'">Preset</button>
        <button class="btn btn-sm" :class="batchMode === 'compound' ? 'btn-primary' : 'btn-secondary'" @click="batchMode = 'compound'">Pipeline</button>
      </div>

      <div v-if="batchMode === 'preset'" class="form-group">
        <label class="form-label">Preset (optional)</label>
        <select class="form-select" v-model="selectedPresetId" :disabled="generating">
          <option :value="null">No preset — use defaults</option>
          <option v-for="p in presets" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
      </div>

      <div v-if="batchMode === 'compound'" class="form-group">
        <label class="form-label">Pipeline</label>
        <select class="form-select" v-model="selectedCompoundPresetId" :disabled="generating">
          <option :value="null" disabled>Select pipeline...</option>
          <option v-for="c in compoundPresets" :key="c.id" :value="c.id">{{ c.name }} ({{ c.steps.length }} steps)</option>
        </select>
      </div>

      <div class="form-group">
        <label class="form-label">Prompt</label>
        <textarea class="form-textarea" v-model="prompt" rows="5" placeholder="SD positive prompt..." :disabled="generating"></textarea>
      </div>

      <div class="form-group">
        <label class="form-label">Negative Prompt</label>
        <textarea class="form-textarea" v-model="negativePrompt" rows="2" placeholder="SD negative prompt..." :disabled="generating"></textarea>
      </div>

      <div style="display: grid; grid-template-columns: 1fr auto; gap: 12px; align-items: end;">
        <div class="form-group">
          <label class="form-label">Output Folder</label>
          <div style="display: flex; gap: 8px;">
            <input class="form-input" v-model="outputFolder" placeholder="/path/to/output" :disabled="generating" style="flex: 1;" />
            <button class="btn btn-secondary" @click="selectFolder" :disabled="generating">Browse</button>
          </div>
        </div>
        <div class="form-group">
          <label class="form-label">Count</label>
          <input class="form-input" type="number" v-model.number="count" min="1" max="100" :disabled="generating" style="width: 80px;" />
        </div>
      </div>

      <button class="btn btn-primary" style="width: 100%; justify-content: center; padding: 12px; margin-top: 12px;" @click="startGeneration" :disabled="generating || !prompt.trim() || !outputFolder.trim() || (batchMode === 'compound' && !selectedCompoundPresetId)">
        <span v-if="generating" style="display: inline-flex; align-items: center; gap: 6px;">
          <span class="spinner" style="width: 14px; height: 14px; border-width: 2px;"></span>
          Generating {{ progress ? `${progress.current}/${progress.total}` : '...' }}
        </span>
        <span v-else>Generate {{ count }} Images</span>
      </button>
    </div>

    <div v-if="progress" class="card" style="max-width: 700px; margin-top: 12px;">
      <div style="margin-bottom: 8px;">
        <div style="display: flex; justify-content: space-between; margin-bottom: 6px;">
          <span style="color: var(--text-dim); font-size: 13px;">
            {{ progress.status === 'done' ? 'Complete' : `Image ${progress.current} of ${progress.total}` }}
          </span>
          <span style="color: var(--text-dim); font-size: 13px;">{{ progress.status }}</span>
        </div>
        <div style="background: var(--surface-2); border-radius: 4px; overflow: hidden; height: 6px;">
          <div :style="{ width: (progress.total ? (progress.current / progress.total * 100) : 0) + '%', background: 'var(--accent)', height: '100%', transition: 'width 0.3s' }"></div>
        </div>
      </div>
      <div v-if="generatedFiles.length" style="margin-top: 8px;">
        <div style="color: var(--text-dim); font-size: 12px; margin-bottom: 4px;">Saved files:</div>
        <div v-for="(f, i) in generatedFiles" :key="i" style="font-size: 12px; padding: 2px 0; color: var(--text);">{{ f }}</div>
      </div>
    </div>
  </div>
</template>
