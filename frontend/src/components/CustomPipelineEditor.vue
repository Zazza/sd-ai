<script setup>
import { ref, computed } from 'vue'
import { api } from '../api.js'
import { t } from '../i18n/index.js'
import { parseLoras, sanitizeWeight } from '../loraOverride.js'
import { MAX_CUSTOM_STEPS, newCustomStep, customStepLoras, customPresetLoras } from '../customSteps.js'
import SearchableSelect from './SearchableSelect.vue'

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
  presets: { type: Array, default: () => [] },
  disableLoras: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])

const steps = computed(() => props.modelValue || [])

const presetItems = computed(() => (props.presets || []).map(p => ({ value: p.id, label: p.name })))

const serverLoras = ref([])
let lorasLoaded = false

function emitSteps(next) {
  emit('update:modelValue', next)
}

function addStep() {
  if (steps.value.length >= MAX_CUSTOM_STEPS) return
  emitSteps([...steps.value, newCustomStep()])
}

function removeStep(idx) {
  if (steps.value.length <= 1) return
  emitSteps(steps.value.filter((_, i) => i !== idx))
}

function moveStep(idx, dir) {
  const target = idx + dir
  if (target < 0 || target >= steps.value.length) return
  const next = [...steps.value]
  const tmp = next[idx]
  next[idx] = next[target]
  next[target] = tmp
  emitSteps(next)
}

function onPresetSelected(step, val) {
  const wasTouched = customStepLoras(step, props.presets) !== null
  step.preset_id = val
  if (!wasTouched) {
    step.loraRows = customPresetLoras(props.presets, val).map(l => ({ name: l.name, weight: l.weight }))
  }
}

function clampDenoise(step) {
  const raw = step.denoising_strength
  if (raw === '' || raw === null || raw === undefined) {
    step.denoising_strength = 0.5
    return
  }
  const n = Number(raw)
  step.denoising_strength = Number.isFinite(n) ? Math.min(1, Math.max(0.05, n)) : 0.5
}

function loraBadge(step) {
  if (step.loraIgnoreAll) return t('custom.lora_off')
  const loras = customStepLoras(step, props.presets)
  if (loras === null) return t('custom.lora_inherit')
  return t('custom.loras_count', { count: parseLoras(loras).length })
}

function loraBadgeTouched(step) {
  return step.loraIgnoreAll || customStepLoras(step, props.presets) !== null
}

function addLora(step) {
  if (step.loraIgnoreAll) step.loraIgnoreAll = false
  step.loraRows.push({ name: '', weight: 0.6 })
}

function removeLora(step, idx) {
  step.loraRows.splice(idx, 1)
}

function onLoraRowEdit(step) {
  if (step.loraIgnoreAll) step.loraIgnoreAll = false
}

function onLorasToggle(e) {
  if (!e.target.open || lorasLoaded) return
  lorasLoaded = true
  api.getLoRAs()
    .then(lor => { serverLoras.value = lor || [] })
    .catch(() => { lorasLoaded = false })
}
</script>

<template>
  <div class="custom-pipeline">
    <div class="custom-pipeline-head">
      <label class="form-label" style="margin: 0;">{{ t('custom.steps_count', { count: steps.length }) }}</label>
      <button type="button" class="btn btn-sm btn-secondary" :disabled="steps.length >= MAX_CUSTOM_STEPS" @click="addStep">
        {{ t('custom.btn_add_step') }}
      </button>
    </div>
    <div v-if="steps.length >= MAX_CUSTOM_STEPS" class="custom-max-hint">{{ t('custom.max_steps_hint', { count: MAX_CUSTOM_STEPS }) }}</div>

    <div v-for="(step, idx) in steps" :key="idx" class="custom-step">
      <div class="custom-step-head">
        <span class="custom-step-title">
          {{ t('custom.step_label', { num: idx + 1, type: idx === 0 ? t('custom.step_txt2img') : t('custom.step_img2img') }) }}
        </span>
        <span v-if="!disableLoras" class="custom-lora-badge" :class="{ touched: loraBadgeTouched(step) }">{{ loraBadge(step) }}</span>
        <div class="custom-step-actions">
          <button type="button" class="btn btn-sm btn-secondary" style="padding: 2px 8px;" :disabled="idx === 0" :aria-label="t('custom.step_move_up')" @click="moveStep(idx, -1)">&#9650;</button>
          <button type="button" class="btn btn-sm btn-secondary" style="padding: 2px 8px;" :disabled="idx === steps.length - 1" :aria-label="t('custom.step_move_down')" @click="moveStep(idx, 1)">&#9660;</button>
          <button type="button" class="btn btn-sm btn-secondary" style="padding: 2px 8px; color: var(--danger);" :disabled="steps.length <= 1" :aria-label="t('custom.step_remove')" @click="removeStep(idx)">&times;</button>
        </div>
      </div>

      <div class="form-group" style="margin: 0 0 8px;">
        <label class="form-label">{{ t('compound.label_style') }}</label>
        <SearchableSelect
          :model-value="step.preset_id"
          :items="presetItems"
          :placeholder="t('compound.select_style')"
          :empty-text="t('searchable.no_results')"
          @update:model-value="val => onPresetSelected(step, val)"
        />
      </div>

      <div v-if="idx > 0" class="form-group custom-denoise" style="margin: 0;">
        <label class="form-label">{{ t('compound.label_denoising') }}</label>
        <input class="form-input" type="number" v-model.number="step.denoising_strength" min="0.05" max="1" step="0.05" @change="clampDenoise(step)" />
      </div>

      <details v-if="!disableLoras" class="custom-loras" @toggle="onLorasToggle">
        <summary>
          <span>{{ t('custom.loras_section') }}</span>
          <span class="custom-lora-badge" :class="{ touched: loraBadgeTouched(step) }">{{ loraBadge(step) }}</span>
        </summary>
        <div class="custom-loras-body">
          <label class="custom-loras-check">
            <input type="checkbox" v-model="step.loraIgnoreAll" />
            <span>{{ t('overrides.ignore_loras') }}</span>
          </label>
          <div class="custom-loras-section" :class="{ 'custom-loras-disabled': step.loraIgnoreAll }">
            <div class="custom-loras-head">
              <label class="form-label" style="margin: 0;">{{ t('overrides.loras') }}</label>
              <button type="button" class="btn btn-sm btn-secondary" :disabled="step.loraIgnoreAll" @click="addLora(step)">{{ t('overrides.loras_add') }}</button>
            </div>
            <div v-for="(row, ri) in step.loraRows" :key="ri" class="custom-lora-row">
              <input class="form-input" type="text" v-model="row.name" list="custom-lora-names" :placeholder="t('overrides.loras_name')" :disabled="step.loraIgnoreAll" @input="onLoraRowEdit(step)" />
              <input class="form-input" type="number" v-model.number="row.weight" min="0" max="2" step="0.05" :placeholder="0.6" :disabled="step.loraIgnoreAll" @change="row.weight = sanitizeWeight(row.weight)" />
              <button type="button" class="btn btn-sm btn-secondary custom-lora-remove" :disabled="step.loraIgnoreAll" @click="removeLora(step, ri)" :aria-label="t('overrides.loras_remove')">&times;</button>
            </div>
            <div class="custom-lora-empty">{{ t('overrides.loras_hint') }}</div>
          </div>
        </div>
      </details>
    </div>

    <datalist id="custom-lora-names">
      <option v-for="l in serverLoras" :key="l.name" :value="l.name" />
    </datalist>
  </div>
</template>

<style scoped>
.custom-pipeline-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.custom-max-hint {
  font-size: 11px;
  color: var(--text-dim);
  margin: -4px 0 8px;
}
.custom-step {
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 12px;
  margin-bottom: 8px;
  background: var(--surface-1);
}
.custom-step-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.custom-step-title {
  font-weight: 600;
  font-size: 13px;
}
.custom-step-actions {
  display: flex;
  gap: 4px;
  margin-left: auto;
}
.custom-lora-badge {
  font-size: 11px;
  color: var(--text-dim);
  background: var(--surface-2);
  border-radius: 4px;
  padding: 2px 8px;
  white-space: nowrap;
}
.custom-lora-badge.touched {
  color: var(--accent);
}
.custom-loras {
  margin-top: 8px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.custom-loras summary {
  cursor: pointer;
  color: var(--text-dim);
  font-size: 12px;
  padding: 6px 10px;
  user-select: none;
  display: flex;
  align-items: center;
  gap: 8px;
}
.custom-loras summary:hover {
  color: var(--text);
}
.custom-loras-body {
  padding: 4px 10px 10px;
}
.custom-loras-check {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
}
.custom-loras-check input {
  accent-color: var(--accent);
}
.custom-loras-check span {
  font-size: 12px;
}
.custom-loras-section {
  margin-top: 8px;
}
.custom-loras-disabled {
  opacity: 0.5;
}
.custom-loras-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}
.custom-lora-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 6px;
}
.custom-lora-row .form-input[type='text'] {
  flex: 1;
}
.custom-lora-row .form-input[type='number'] {
  flex: 0 0 90px;
}
.custom-lora-remove {
  padding: 4px 10px;
}
.custom-lora-empty {
  font-size: 11px;
  color: var(--text-dim);
}
@media (max-width: 640px) {
  .custom-step-head {
    flex-wrap: wrap;
  }
}
</style>
