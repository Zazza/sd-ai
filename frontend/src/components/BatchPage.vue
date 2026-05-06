<script setup>
import { ref, watch, onMounted, onUnmounted, inject } from 'vue'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'
import { BatchGenerate, BatchCompoundGenerate, SelectFolder } from '../wailsjs/go/main/App.js'
import { api } from '../api.js'
import { t } from '../i18n/index.js'
import { useGenerationProgress } from '../composables/useGenerationProgress.js'

const shared = inject('sharedGenState', null)

const presets = ref([])
const compoundPresets = ref([])
const selectedPresetId = ref(null)
const selectedCompoundPresetId = ref(null)
const batchMode = ref('preset')
const description = ref('')
const negative = ref('')
const count = ref(4)
const outputFolder = ref('')
const generating = ref(false)
const { sdProgress, interrupt: interruptGeneration, reset: resetProgress } = useGenerationProgress()
const error = ref('')
const progress = ref(null)
const generatedFiles = ref([])

const props = defineProps({
  prefillDescription: { type: String, default: '' },
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
  if (!description.value.trim()) {
    error.value = t('batch.error_prompt_required')
    return
  }
  if (!outputFolder.value.trim()) {
    error.value = t('batch.error_folder_required')
    return
  }
  if (count.value < 1 || count.value > 100) {
    error.value = t('batch.error_count_range')
    return
  }

  generating.value = true
  error.value = ''
  progress.value = null
  generatedFiles.value = []
  resetProgress()

  try {
    let batchPrompt = ''
    let batchNegative = ''

    let llmPresetId = selectedPresetId.value
    if (batchMode.value === 'compound') {
      const cp = compoundPresets.value.find(c => c.id === selectedCompoundPresetId.value)
      if (cp && cp.steps && cp.steps.length > 0) {
        llmPresetId = cp.steps[0].preset_id
      }
    }

    if (llmPresetId) {
      const promptResult = await api.generateSdPrompt({
        preset_id: llmPresetId,
        description: description.value,
        negative: negative.value,
      })
      if (promptResult && promptResult.prompt) {
        batchPrompt = promptResult.prompt
        batchNegative = promptResult.negative_prompt || ''
      }
    }

    if (!batchPrompt) {
      batchPrompt = description.value
      batchNegative = negative.value
    }

    if (batchMode.value === 'compound') {
      await BatchCompoundGenerate({
        compound_preset_id: selectedCompoundPresetId.value,
        extra_prompt: batchPrompt,
        extra_negative_prompt: batchNegative,
        count: count.value,
        output_folder: outputFolder.value,
      })
    } else {
      await BatchGenerate({
        preset_id: selectedPresetId.value || 0,
        prompt: batchPrompt,
        negative_prompt: batchNegative,
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

  if (props.prefillDescription) {
    description.value = props.prefillDescription
  }
  if (props.prefillNegative) {
    negative.value = props.prefillNegative
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
    if (!props.prefillDescription && s.batch_prompt) description.value = s.batch_prompt
    if (!props.prefillNegative && s.batch_negative) negative.value = s.batch_negative
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
    if (!props.prefillDescription && shared.description) description.value = shared.description
    if (!props.prefillNegative && shared.negative) negative.value = shared.negative
  }
})

onUnmounted(() => {
  EventsOff('batch:progress')
  saveBatchState()
  if (shared) {
    shared.selectedPresetId = selectedPresetId.value
    shared.selectedCompoundPresetId = selectedCompoundPresetId.value
    shared.genMode = batchMode.value
    if (description.value) shared.description = description.value
    if (negative.value) shared.negative = negative.value
  }
})

function saveBatchState() {
  api.updateSettings({
    batch_preset_id: String(selectedPresetId.value || ''),
    batch_compound_preset_id: String(selectedCompoundPresetId.value || ''),
    batch_mode: batchMode.value,
    batch_prompt: description.value || '',
    batch_negative: negative.value || '',
    batch_count: String(count.value || ''),
    batch_output_folder: outputFolder.value || '',
  }).catch(() => {})
}
</script>

<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">{{ t('batch.title') }}</h1>
    </div>

    <div v-if="error" class="status" :class="error === 'interrupted' ? 'status-warning' : 'status-error'">{{ error }}</div>

    <div v-if="generating && sdProgress && sdProgress.progress > 0" style="margin-bottom: 12px; padding: 10px 12px; background: var(--surface-2); border-radius: 6px;">
      <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 6px;">
        <span style="font-size: 13px;">SD: {{ Math.round(sdProgress.progress * 100) }}%</span>
        <span style="font-size: 12px; color: var(--text-dim);">{{ t('progress.sd_step', { current: Math.round(sdProgress.progress * sdProgress.steps), total: sdProgress.steps }) }}</span>
        <button class="btn btn-sm btn-secondary" @click="interruptGeneration" style="margin-left: auto; padding: 2px 8px; font-size: 11px;">{{ t('progress.btn_interrupt') }}</button>
      </div>
      <div style="background: var(--surface-1); border-radius: 4px; overflow: hidden; height: 4px;">
        <div :style="{ width: (sdProgress.progress * 100) + '%', background: 'var(--accent)', height: '100%', transition: 'width 0.3s' }"></div>
      </div>
    </div>

    <div class="card" style="max-width: 700px;">
      <div style="display: flex; gap: 8px; margin-bottom: 12px;">
        <button class="btn btn-sm" :class="batchMode === 'preset' ? 'btn-primary' : 'btn-secondary'" @click="batchMode = 'preset'">{{ t('batch.btn_preset') }}</button>
        <button class="btn btn-sm" :class="batchMode === 'compound' ? 'btn-primary' : 'btn-secondary'" @click="batchMode = 'compound'">{{ t('batch.btn_pipeline') }}</button>
      </div>

      <div v-if="batchMode === 'preset'" class="form-group">
        <label class="form-label">{{ t('batch.label_preset_optional') }}</label>
        <select class="form-select" v-model="selectedPresetId" :disabled="generating">
          <option :value="null">{{ t('batch.no_preset') }}</option>
          <option v-for="p in presets" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
      </div>

      <div v-if="batchMode === 'compound'" class="form-group">
        <label class="form-label">{{ t('batch.label_pipeline') }}</label>
        <select class="form-select" v-model="selectedCompoundPresetId" :disabled="generating">
          <option :value="null" disabled>{{ t('batch.select_pipeline') }}</option>
          <option v-for="c in compoundPresets" :key="c.id" :value="c.id">{{ c.name }} ({{ c.steps.length }} steps)</option>
        </select>
      </div>

      <div class="form-group">
        <label class="form-label">{{ t('batch.label_prompt') }}</label>
        <textarea class="form-textarea" v-model="description" rows="5" :placeholder="t('batch.placeholder_prompt')" :disabled="generating"></textarea>
      </div>

      <div class="form-group">
        <label class="form-label">{{ t('batch.label_negative') }}</label>
        <textarea class="form-textarea" v-model="negative" rows="2" :placeholder="t('batch.placeholder_negative')" :disabled="generating"></textarea>
      </div>

      <div style="display: grid; grid-template-columns: 1fr auto; gap: 12px; align-items: end;">
        <div class="form-group">
          <label class="form-label">{{ t('batch.label_output_folder') }}</label>
          <div style="display: flex; gap: 8px;">
            <input class="form-input" v-model="outputFolder" :placeholder="t('batch.placeholder_output')" :disabled="generating" style="flex: 1;" />
            <button class="btn btn-secondary" @click="selectFolder" :disabled="generating">{{ t('batch.btn_browse') }}</button>
          </div>
        </div>
        <div class="form-group">
          <label class="form-label">{{ t('batch.label_count') }}</label>
          <input class="form-input" type="number" v-model.number="count" min="1" max="100" :disabled="generating" style="width: 80px;" />
        </div>
      </div>

      <button class="btn btn-primary" style="width: 100%; justify-content: center; padding: 12px; margin-top: 12px;" @click="startGeneration" :disabled="generating || !description.trim() || !outputFolder.trim() || (batchMode === 'compound' && !selectedCompoundPresetId)">
        <span v-if="generating" style="display: inline-flex; align-items: center; gap: 6px;">
          <span class="spinner" style="width: 14px; height: 14px; border-width: 2px;"></span>
          Generating {{ progress ? `${progress.current}/${progress.total}` : t('settings.btn_loading') }}
        </span>
        <span v-else>Generate {{ count }} Images</span>
      </button>
    </div>

    <div v-if="progress" class="card" style="max-width: 700px; margin-top: 12px;">
      <div style="margin-bottom: 8px;">
        <div style="display: flex; justify-content: space-between; margin-bottom: 6px;">
          <span style="color: var(--text-dim); font-size: 13px;">
            {{ progress.status === 'done' ? t('batch.complete') : `Image ${progress.current} of ${progress.total}` }}
          </span>
          <span style="color: var(--text-dim); font-size: 13px;">{{ progress.status }}</span>
        </div>
        <div style="background: var(--surface-2); border-radius: 4px; overflow: hidden; height: 6px;">
          <div :style="{ width: (progress.total ? (progress.current / progress.total * 100) : 0) + '%', background: 'var(--accent)', height: '100%', transition: 'width 0.3s' }"></div>
        </div>
      </div>
      <div v-if="generatedFiles.length" style="margin-top: 8px;">
        <div style="color: var(--text-dim); font-size: 12px; margin-bottom: 4px;">{{ t('batch.saved_files') }}</div>
        <div v-for="(f, i) in generatedFiles" :key="i" style="font-size: 12px; padding: 2px 0; color: var(--text);">{{ f }}</div>
      </div>
    </div>
  </div>
</template>
