<script setup>
import { ref, watch } from 'vue'
import { api } from '../api.js'
import { t } from '../i18n/index.js'
import { buildLorasOverride, sanitizeWeight, sameLoras } from '../loraOverride.js'

const props = defineProps({
  modelValue: { type: Object, default: null },
  presetLoras: { type: Array, default: () => [] },
})
const emit = defineEmits(['update:modelValue'])

const ovModel = ref('')
const ovSampler = ref('')
const ovScheduler = ref('')
const ovSteps = ref('')
const ovCfg = ref('')
const ovClipSkip = ref('')
const ovIgnoreLoras = ref(false)

const loraRows = ref([])
let loraSource = []
const serverLoras = ref([])

const models = ref([])
const samplers = ref([])
const schedulers = ref([])
const loaded = ref(false)
const loadError = ref(false)

function lorasTouched() {
  return !sameLoras(loraRows.value, loraSource)
}

function hasAnyValue() {
  return !!(ovModel.value || ovSampler.value || ovScheduler.value || ovSteps.value !== '' || ovCfg.value !== '' || ovClipSkip.value !== '' || ovIgnoreLoras.value || lorasTouched())
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
  const loras = buildLorasOverride(loraSource, loraRows.value, ovIgnoreLoras.value)
  if (loras !== null) ov.loras = loras
  return Object.keys(ov).length > 0 ? ov : null
}

function syncLoraRows() {
  const src = Array.isArray(props.presetLoras) ? props.presetLoras : []
  loraRows.value = src.map((l) => ({ name: l?.name || '', weight: sanitizeWeight(l?.weight) }))
  loraSource = src
}

function addLora() {
  loraRows.value.push({ name: '', weight: 0.6 })
}

function removeLora(idx) {
  loraRows.value.splice(idx, 1)
}

watch([ovModel, ovSampler, ovScheduler, ovSteps, ovCfg, ovClipSkip, ovIgnoreLoras], () => {
  emit('update:modelValue', buildOverrides())
})

watch(loraRows, () => {
  if (ovIgnoreLoras.value) ovIgnoreLoras.value = false
  emit('update:modelValue', buildOverrides())
}, { deep: true })

watch(() => props.presetLoras, () => {
  if (!lorasTouched()) syncLoraRows()
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
  syncLoraRows()
}

async function loadLists() {
  if (loaded.value) return
  loaded.value = true
  loadError.value = false
  try {
    const [m, s, sch, lor] = await Promise.all([
      api.getModels(),
      api.getSamplers(),
      api.getSchedulers(),
      api.getLoRAs(),
    ])
    models.value = m || []
    samplers.value = s || []
    schedulers.value = sch || []
    serverLoras.value = lor || []
  } catch (e) {
    loadError.value = true
    loaded.value = false
  }
}

function onToggle(e) {
  if (e.target.open) loadLists()
}

syncLoraRows()

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
      <div class="form-group lora-section" :class="{ 'lora-disabled': ovIgnoreLoras }">
        <div class="lora-head">
          <label class="form-label">{{ t('overrides.loras') }}</label>
          <button type="button" class="btn btn-sm btn-secondary" :disabled="ovIgnoreLoras" @click="addLora">{{ t('overrides.loras_add') }}</button>
        </div>
        <div v-for="(row, idx) in loraRows" :key="idx" class="lora-row">
          <input class="form-input" type="text" v-model="row.name" list="overrides-lora-names" :placeholder="t('overrides.loras_name')" :disabled="ovIgnoreLoras" />
          <input class="form-input" type="number" v-model.number="row.weight" min="0" max="2" step="0.05" :placeholder="0.6" :disabled="ovIgnoreLoras" />
          <button type="button" class="btn btn-sm btn-secondary lora-remove" :disabled="ovIgnoreLoras" @click="removeLora(idx)" :aria-label="t('overrides.loras_remove')">&times;</button>
        </div>
        <div class="lora-empty">{{ t('overrides.loras_hint') }}</div>
      </div>
      <datalist id="overrides-lora-names">
        <option v-for="l in serverLoras" :key="l.name" :value="l.name" />
      </datalist>
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
.lora-section {
  margin-top: 10px;
}
.lora-disabled {
  opacity: 0.5;
}
.lora-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}
.lora-head .form-label {
  margin-bottom: 0;
}
.lora-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 6px;
}
.lora-row .form-input[type='text'] {
  flex: 1;
}
.lora-row .form-input[type='number'] {
  flex: 0 0 90px;
}
.lora-remove {
  padding: 4px 10px;
}
.lora-empty {
  font-size: 11px;
  color: var(--text-dim);
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
