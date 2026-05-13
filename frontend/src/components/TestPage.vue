<script setup>
import { ref, computed, onMounted, onUnmounted, inject } from 'vue'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'
import { api } from '../api.js'
import { t } from '../i18n/index.js'
import { useGenerationProgress } from '../composables/useGenerationProgress.js'
import ResolutionSelector from './ResolutionSelector.vue'
import HiresProfileSelector from './HiresProfileSelector.vue'

const shared = inject('sharedGenState', null)

const mode = ref('presets')
const presets = ref([])
const models = ref([])
const compoundPresets = ref([])
const samplers = ref([])
const schedulers = ref([])
const selectedPresetIds = ref([])
const selectedModelNames = ref([])
const selectedCompoundIds = ref([])
const prompt = ref('')
const negativePrompt = ref('')
const showAdvanced = ref(false)
const sampler = ref('Euler a')
const scheduleType = ref('')
const steps = ref(20)
const cfgScale = ref(7.0)
const width = ref(512)
const height = ref(512)
const generating = ref(false)
const { sdProgress, interrupt: interruptGeneration, reset: resetProgress } = useGenerationProgress()
const error = ref('')
const progress = ref(null)
const results = ref([])
const selectedResolutionId = ref(null)
const selectedHiresProfileId = ref(null)

const selectedItems = computed(() => {
  if (mode.value === 'presets') return selectedPresetIds.value
  if (mode.value === 'compounds') return selectedCompoundIds.value
  return selectedModelNames.value
})

function togglePreset(id) {
  const idx = selectedPresetIds.value.indexOf(id)
  if (idx >= 0) selectedPresetIds.value.splice(idx, 1)
  else selectedPresetIds.value.push(id)
}

function toggleModel(name) {
  const idx = selectedModelNames.value.indexOf(name)
  if (idx >= 0) selectedModelNames.value.splice(idx, 1)
  else selectedModelNames.value.push(name)
}

function toggleCompound(id) {
  const idx = selectedCompoundIds.value.indexOf(id)
  if (idx >= 0) selectedCompoundIds.value.splice(idx, 1)
  else selectedCompoundIds.value.push(id)
}

function selectAll() {
  if (mode.value === 'presets') {
    selectedPresetIds.value = presets.value.map(p => p.id)
  } else if (mode.value === 'compounds') {
    selectedCompoundIds.value = compoundPresets.value.map(c => c.id)
  } else {
    selectedModelNames.value = models.value.map(m => m.title)
  }
}

function deselectAll() {
  if (mode.value === 'presets') {
    selectedPresetIds.value = []
  } else if (mode.value === 'compounds') {
    selectedCompoundIds.value = []
  } else {
    selectedModelNames.value = []
  }
}

async function loadData() {
  try {
    const [p, m, s, sch, c] = await Promise.all([
      api.listPresets(),
      api.getModels(),
      api.getSamplers(),
      api.getSchedulers(),
      api.listCompoundPresets(),
    ])
    presets.value = p || []
    models.value = m || []
    samplers.value = s || []
    schedulers.value = sch || []
    compoundPresets.value = c || []
  } catch (e) {
    console.error(e)
  }
}

async function generate() {
  if (selectedItems.value.length === 0) {
    error.value = t('test.error_select_item')
    return
  }
  if (!prompt.value.trim()) {
    error.value = t('test.error_prompt_required')
    return
  }

  generating.value = true
  error.value = ''
  progress.value = null
  results.value = []
  resetProgress()

  try {
    let res
    if (mode.value === 'compounds') {
      res = await api.testCompoundGenerate({
        selected_ids: selectedCompoundIds.value,
        prompt: prompt.value,
        negative_prompt: negativePrompt.value,
        resolution_id: selectedResolutionId.value,
        hires_profile_id: selectedHiresProfileId.value,
      })
    } else {
      res = await api.testGenerate({
        mode: mode.value,
        selected_ids: mode.value === 'presets' ? selectedPresetIds.value : [],
        selected_models: mode.value === 'models' ? selectedModelNames.value : [],
        prompt: prompt.value,
        negative_prompt: negativePrompt.value,
        sampler: showAdvanced.value ? sampler.value : '',
        schedule_type: showAdvanced.value ? scheduleType.value : '',
        steps: showAdvanced.value ? steps.value : 0,
        cfg_scale: showAdvanced.value ? cfgScale.value : 0,
        width: showAdvanced.value ? width.value : 0,
        height: showAdvanced.value ? height.value : 0,
        resolution_id: selectedResolutionId.value,
        hires_profile_id: selectedHiresProfileId.value,
      })
    }
    results.value = res || []
  } catch (e) {
    error.value = String(e)
  } finally {
    generating.value = false
  }
}

async function downloadImage(imageBase64, name) {
  try {
    const defaultName = (name || 'test') + '-' + Date.now() + '.png'
    await api.saveImage(imageBase64, defaultName)
  } catch (e) {
    error.value = 'Save failed: ' + String(e)
  }
}

function onProgress(data) {
  progress.value = data
}

onMounted(async () => {
  loadData()
  EventsOn('test:progress', onProgress)
  if (shared) {
    if (!prompt.value) prompt.value = shared.description || ''
    if (!negativePrompt.value) negativePrompt.value = shared.negative || ''
  }
  try {
    const s = await api.getSettings()
    if (s.test_mode) mode.value = s.test_mode
    if (s.test_prompt) prompt.value = s.test_prompt
    else if (shared?.description) prompt.value = shared.description
    if (s.test_negative) negativePrompt.value = s.test_negative
    else if (shared?.negative) negativePrompt.value = shared.negative
    if (s.test_sampler) sampler.value = s.test_sampler
    if (s.test_schedule_type) scheduleType.value = s.test_schedule_type
    if (s.test_steps) steps.value = Number(s.test_steps)
    if (s.test_cfg_scale) cfgScale.value = Number(s.test_cfg_scale)
    if (s.test_width) width.value = Number(s.test_width)
    if (s.test_height) height.value = Number(s.test_height)
  } catch {}
})

onUnmounted(() => {
  EventsOff('test:progress')
  saveTestState()
  if (shared) {
    if (prompt.value) shared.description = prompt.value
    if (negativePrompt.value) shared.negative = negativePrompt.value
  }
})

function saveTestState() {
  api.updateSettings({
    test_mode: mode.value,
    test_prompt: prompt.value || '',
    test_negative: negativePrompt.value || '',
    test_sampler: sampler.value || '',
    test_schedule_type: scheduleType.value || '',
    test_steps: String(steps.value || ''),
    test_cfg_scale: String(cfgScale.value || ''),
    test_width: String(width.value || ''),
    test_height: String(height.value || ''),
  }).catch(() => {})
}
</script>

<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">{{ t('test.title') }}</h1>
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

    <div class="card" style="max-width: 800px;">
      <div style="display: flex; gap: 8px; margin-bottom: 16px;">
        <button class="btn" :class="mode === 'presets' ? 'btn-primary' : 'btn-secondary'" @click="mode = 'presets'; results = []">{{ t('test.btn_presets') }}</button>
        <button class="btn" :class="mode === 'models' ? 'btn-primary' : 'btn-secondary'" @click="mode = 'models'; results = []">{{ t('test.btn_models') }}</button>
        <button class="btn" :class="mode === 'compounds' ? 'btn-primary' : 'btn-secondary'" @click="mode = 'compounds'; results = []">{{ t('test.btn_pipelines') }}</button>
      </div>

      <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">
        <label class="form-label" style="margin: 0;">
          {{ mode === 'presets' ? t('test.select_presets') : mode === 'compounds' ? t('test.select_pipelines') : t('test.select_models') }}
          {{ t('test.selected_count', { count: selectedItems.length }) }}
        </label>
        <div style="display: flex; gap: 6px;">
          <button class="btn btn-sm btn-secondary" @click="selectAll" :disabled="generating">{{ t('test.btn_all') }}</button>
          <button class="btn btn-sm btn-secondary" @click="deselectAll" :disabled="generating">{{ t('test.btn_none') }}</button>
        </div>
      </div>

      <div class="test-select-list">
        <template v-if="mode === 'presets'">
          <div v-for="p in presets" :key="p.id" class="test-select-item" :class="{ active: selectedPresetIds.includes(p.id) }" @click="!generating && togglePreset(p.id)">
            <input type="checkbox" :checked="selectedPresetIds.includes(p.id)" @click.prevent />
            <span>{{ p.name }}</span>
            <span v-if="p.model_name" style="color: var(--text-dim); font-size: 11px; margin-left: auto;">{{ p.model_name }}</span>
          </div>
          <div v-if="!presets.length" style="color: var(--text-dim); padding: 12px; text-align: center;">{{ t('test.no_presets') }}</div>
        </template>
        <template v-else-if="mode === 'compounds'">
          <div v-for="c in compoundPresets" :key="c.id" class="test-select-item" :class="{ active: selectedCompoundIds.includes(c.id) }" @click="!generating && toggleCompound(c.id)">
            <input type="checkbox" :checked="selectedCompoundIds.includes(c.id)" @click.prevent />
            <span>{{ c.name }}</span>
            <span style="color: var(--text-dim); font-size: 11px; margin-left: auto;">{{ c.steps.length }} steps</span>
          </div>
          <div v-if="!compoundPresets.length" style="color: var(--text-dim); padding: 12px; text-align: center;">{{ t('test.no_pipelines') }}</div>
        </template>
        <template v-else>
          <div v-for="m in models" :key="m.title" class="test-select-item" :class="{ active: selectedModelNames.includes(m.title) }" @click="!generating && toggleModel(m.title)">
            <input type="checkbox" :checked="selectedModelNames.includes(m.title)" @click.prevent />
            <span>{{ m.title }}</span>
          </div>
          <div v-if="!models.length" style="color: var(--text-dim); padding: 12px; text-align: center;">{{ t('test.no_models') }}</div>
        </template>
      </div>

      <div class="form-group" style="margin-top: 16px;">
        <label class="form-label">{{ t('test.label_prompt') }}</label>
        <textarea class="form-textarea" v-model="prompt" rows="3" :placeholder="t('test.placeholder_prompt')" :disabled="generating"></textarea>
      </div>

      <div class="form-group">
        <label class="form-label">{{ t('test.label_negative') }}</label>
        <textarea class="form-textarea" v-model="negativePrompt" rows="2" :placeholder="t('test.placeholder_negative')" :disabled="generating"></textarea>
      </div>

      <div v-if="mode !== 'compounds'" style="margin-bottom: 12px;">
        <button class="btn btn-secondary btn-sm" @click="showAdvanced = !showAdvanced">
          {{ showAdvanced ? t('test.hide_params') : t('test.show_params') }}
        </button>
      </div>

      <div v-if="showAdvanced && mode !== 'compounds'" style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-bottom: 16px;">
        <div class="form-group">
          <label class="form-label">{{ t('test.label_sampler') }}</label>
          <select class="form-select" v-model="sampler" :disabled="generating">
            <option value="">{{ t('test.default') }}</option>
            <option v-for="s in samplers" :key="s.name" :value="s.name">{{ s.name }}</option>
          </select>
        </div>
        <div class="form-group">
          <label class="form-label">{{ t('test.label_schedule') }}</label>
          <select class="form-select" v-model="scheduleType" :disabled="generating">
            <option value="">{{ t('test.default') }}</option>
            <option v-for="s in schedulers" :key="s.name" :value="s.name">{{ s.label || s.name }}</option>
          </select>
        </div>
        <div class="form-group">
          <label class="form-label">{{ t('test.label_width') }}</label>
          <input class="form-input" type="number" v-model.number="width" min="64" max="2048" step="64" :disabled="generating" />
        </div>
        <div class="form-group">
          <label class="form-label">{{ t('test.label_height') }}</label>
          <input class="form-input" type="number" v-model.number="height" min="64" max="2048" step="64" :disabled="generating" />
        </div>
        <div class="form-group">
          <label class="form-label">{{ t('test.label_steps') }}</label>
          <input class="form-input" type="number" v-model.number="steps" min="1" max="150" :disabled="generating" />
        </div>
        <div class="form-group">
          <label class="form-label">{{ t('test.label_cfg') }}</label>
          <input class="form-input" type="number" v-model.number="cfgScale" min="1" max="30" step="0.5" :disabled="generating" />
        </div>
      </div>

      <ResolutionSelector v-model="selectedResolutionId" />
      <HiresProfileSelector v-model="selectedHiresProfileId" />

      <div v-if="generating && progress" style="margin-bottom: 12px;">
        <div style="display: flex; justify-content: space-between; margin-bottom: 4px;">
          <span style="color: var(--text-dim); font-size: 12px;">{{ progress.current }} / {{ progress.total }}</span>
          <span style="color: var(--text-dim); font-size: 12px;">{{ progress.status }}</span>
        </div>
        <div style="background: var(--surface-2); border-radius: 4px; overflow: hidden; height: 4px;">
          <div :style="{ width: (progress.total ? (progress.current / progress.total * 100) : 0) + '%', background: 'var(--accent)', height: '100%', transition: 'width 0.3s' }"></div>
        </div>
      </div>

      <button class="btn btn-primary" style="width: 100%; justify-content: center; padding: 12px;" @click="generate" :disabled="generating || selectedItems.length === 0 || !prompt.trim()">
        <span v-if="generating" style="display: inline-flex; align-items: center; gap: 6px;">
          <span class="spinner" style="width: 14px; height: 14px; border-width: 2px;"></span>
          Generating {{ progress ? `${progress.current}/${progress.total}` : '...' }}
        </span>
        <span v-else>Generate ({{ selectedItems.length }} items)</span>
      </button>
    </div>

    <div v-if="results.length" style="margin-top: 16px; max-width: 800px;">
      <div v-if="mode !== 'compounds'" style="color: var(--text-dim); font-size: 12px; margin-bottom: 8px;">
        {{ width }}&times;{{ height }}, {{ steps }} steps, CFG {{ cfgScale }}
      </div>
      <div class="test-results-grid">
        <div v-for="(r, i) in results" :key="i" class="test-result-card">
          <div v-if="r.error" class="test-result-error">
            <strong>{{ r.name }}</strong>
            <div style="font-size: 12px; margin-top: 4px;">{{ r.error }}</div>
          </div>
          <template v-else>
            <img class="test-result-image" :src="'data:image/png;base64,' + r.image" :alt="r.name" />
            <div class="test-result-meta">
              <div class="test-result-name">{{ r.name }}</div>
              <div class="test-result-info">
                <span>Seed: {{ r.seed }}</span>
                <span v-if="r.model_name" style="margin-left: 8px;">{{ r.model_name }}</span>
              </div>
              <button class="btn btn-secondary btn-sm" @click="downloadImage(r.image, r.name)" style="margin-top: 6px; width: 100%;">{{ t('test.btn_download') }}</button>
            </div>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.test-select-list {
  max-height: 220px;
  overflow-y: auto;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--surface-1);
}
.test-select-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  border-bottom: 1px solid var(--border);
  transition: background 0.15s;
  font-size: 13px;
}
.test-select-item:last-child {
  border-bottom: none;
}
.test-select-item:hover {
  background: var(--surface-2);
}
.test-select-item.active {
  background: color-mix(in srgb, var(--accent) 12%, transparent);
}
.test-select-item input[type="checkbox"] {
  pointer-events: none;
}
.test-results-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 12px;
}
.test-result-card {
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
  background: var(--surface-1);
}
.test-result-image {
  width: 100%;
  display: block;
  aspect-ratio: 1;
  object-fit: cover;
  background: var(--surface-2);
}
.test-result-meta {
  padding: 10px;
}
.test-result-name {
  font-weight: 600;
  font-size: 13px;
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.test-result-info {
  font-size: 11px;
  color: var(--text-dim);
}
.test-result-error {
  padding: 16px;
  color: var(--error, #e55);
  font-size: 13px;
}
</style>
