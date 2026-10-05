<script setup>
import { ref, watch } from 'vue'
import { api } from '../api.js'
import { t } from '../i18n/index.js'

const props = defineProps({
  modelValue: { type: Object, default: null },
})
const emit = defineEmits(['update:modelValue'])

const ovModel = ref('')
const ovSampler = ref('')
const ovScheduler = ref('')
const ovSteps = ref('')
const ovCfg = ref('')
const ovClipSkip = ref('')
const ovIgnoreLoras = ref(false)

const models = ref([])
const samplers = ref([])
const schedulers = ref([])
const loaded = ref(false)
const loadError = ref(false)

function hasAnyValue() {
  return !!(ovModel.value || ovSampler.value || ovScheduler.value || ovSteps.value !== '' || ovCfg.value !== '' || ovClipSkip.value !== '' || ovIgnoreLoras.value)
}

function buildOverrides() {
  const ov = {}
  if (ovModel.value) ov.model_name = ovModel.value
  if (ovSampler.value) ov.sampler = ovSampler.value
  if (ovScheduler.value) ov.schedule_type = ovScheduler.value
  const steps = Number(ovSteps.value)
  if (ovSteps.value !== '' && Number.isFinite(steps) && steps > 0) ov.steps = Math.min(150, Math.round(steps))
  const cfg = Number(ovCfg.value)
  if (ovCfg.value !== '' && Number.isFinite(cfg) && cfg > 0) ov.cfg_scale = Math.min(30, cfg)
  const clip = Number(ovClipSkip.value)
  if (ovClipSkip.value !== '' && Number.isFinite(clip) && clip > 0) ov.clip_skip = Math.min(12, Math.round(clip))
  if (ovIgnoreLoras.value) ov.loras = '[]'
  return Object.keys(ov).length > 0 ? ov : null
}

watch([ovModel, ovSampler, ovScheduler, ovSteps, ovCfg, ovClipSkip, ovIgnoreLoras], () => {
  emit('update:modelValue', buildOverrides())
})

watch(() => props.modelValue, (v) => {
  if (v === null && hasAnyValue()) reset()
})

function reset() {
  ovModel.value = ''
  ovSampler.value = ''
  ovScheduler.value = ''
  ovSteps.value = ''
  ovCfg.value = ''
  ovClipSkip.value = ''
  ovIgnoreLoras.value = false
}

async function loadLists() {
  if (loaded.value) return
  loaded.value = true
  loadError.value = false
  try {
    const [m, s, sch] = await Promise.all([
      api.getModels(),
      api.getSamplers(),
      api.getSchedulers(),
    ])
    models.value = m || []
    samplers.value = s || []
    schedulers.value = sch || []
  } catch (e) {
    loadError.value = true
    loaded.value = false
  }
}

function onToggle(e) {
  if (e.target.open) loadLists()
}

defineExpose({ reset })
</script>

<template>
  <details class="preset-overrides" @toggle="onToggle">
    <summary>{{ t('overrides.title') }}</summary>
    <div class="preset-overrides-body">
      <div v-if="loadError" class="overrides-hint" style="color: var(--status-warn, #e6a23c);">{{ t('overrides.load_error') }}</div>
      <div class="overrides-grid">
        <div class="form-group">
          <label class="form-label">{{ t('overrides.model') }}</label>
          <select class="form-select" v-model="ovModel">
            <option value="">{{ t('overrides.as_preset') }}</option>
            <option v-for="m in models" :key="m.title" :value="m.title">{{ m.title }}</option>
          </select>
        </div>
        <div class="form-group">
          <label class="form-label">{{ t('overrides.sampler') }}</label>
          <select class="form-select" v-model="ovSampler">
            <option value="">{{ t('overrides.as_preset') }}</option>
            <option v-for="s in samplers" :key="s.name" :value="s.name">{{ s.name }}</option>
          </select>
        </div>
        <div class="form-group">
          <label class="form-label">{{ t('overrides.scheduler') }}</label>
          <select class="form-select" v-model="ovScheduler">
            <option value="">{{ t('overrides.as_preset') }}</option>
            <option v-for="s in schedulers" :key="s.name" :value="s.name">{{ s.label || s.name }}</option>
          </select>
        </div>
      </div>
      <div class="overrides-grid">
        <div class="form-group">
          <label class="form-label">{{ t('overrides.steps') }}</label>
          <input class="form-input" type="number" v-model.number="ovSteps" min="1" max="150" step="1" :placeholder="t('overrides.as_preset')" />
        </div>
        <div class="form-group">
          <label class="form-label">{{ t('overrides.cfg') }}</label>
          <input class="form-input" type="number" v-model.number="ovCfg" min="0.1" max="30" step="0.1" :placeholder="t('overrides.as_preset')" />
        </div>
        <div class="form-group">
          <label class="form-label">{{ t('overrides.clip_skip') }}</label>
          <input class="form-input" type="number" v-model.number="ovClipSkip" min="1" max="12" step="1" :placeholder="t('overrides.as_preset')" />
        </div>
      </div>
      <label class="overrides-check">
        <input type="checkbox" v-model="ovIgnoreLoras" />
        <span>{{ t('overrides.ignore_loras') }}</span>
      </label>
      <div class="overrides-hint">{{ t('overrides.hint') }}</div>
    </div>
  </details>
</template>

<style scoped>
.preset-overrides {
  margin-top: 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.preset-overrides summary {
  cursor: pointer;
  color: var(--text-dim);
  font-size: 13px;
  padding: 8px 12px;
  user-select: none;
}
.preset-overrides summary:hover {
  color: var(--text);
}
.preset-overrides-body {
  padding: 4px 12px 12px;
}
.overrides-grid {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 12px;
}
.overrides-check {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  margin-top: 4px;
}
.overrides-check input {
  accent-color: var(--accent);
}
.overrides-check span {
  font-size: 12px;
}
.overrides-hint {
  margin-top: 8px;
  font-size: 11px;
  color: var(--text-dim);
}
@media (max-width: 900px) {
  .overrides-grid {
    grid-template-columns: 1fr;
  }
}
</style>
