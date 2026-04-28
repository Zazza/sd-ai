<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const compounds = ref([])
const presets = ref([])
const editing = ref(null)
const showForm = ref(false)
const error = ref('')

const formName = ref('')
const formDescription = ref('')
const formSteps = ref([])

async function loadData() {
  try {
    const [c, p] = await Promise.all([api.listCompoundPresets(), api.listPresets()])
    compounds.value = c || []
    presets.value = p || []
  } catch (e) {
    error.value = String(e)
  }
}

function openCreate() {
  editing.value = null
  formName.value = ''
  formDescription.value = ''
  formSteps.value = [{ preset_id: null, width: 512, height: 512, denoising_strength: 0.5 }]
  showForm.value = true
  error.value = ''
}

function openEdit(cp) {
  editing.value = cp.id
  formName.value = cp.name
  formDescription.value = cp.description
  formSteps.value = cp.steps.map(s => ({
    preset_id: s.preset_id,
    width: s.width || 512,
    height: s.height || 512,
    denoising_strength: s.denoising_strength || 0.5,
  }))
  showForm.value = true
  error.value = ''
}

function cancelForm() {
  showForm.value = false
  editing.value = null
  error.value = ''
}

function addStep() {
  formSteps.value.push({ preset_id: null, width: 512, height: 512, denoising_strength: 0.5 })
}

function removeStep(idx) {
  formSteps.value.splice(idx, 1)
}

function moveStepUp(idx) {
  if (idx <= 0) return
  const tmp = formSteps.value[idx]
  formSteps.value[idx] = formSteps.value[idx - 1]
  formSteps.value[idx - 1] = tmp
  formSteps.value = [...formSteps.value]
}

function moveStepDown(idx) {
  if (idx >= formSteps.value.length - 1) return
  const tmp = formSteps.value[idx]
  formSteps.value[idx] = formSteps.value[idx + 1]
  formSteps.value[idx + 1] = tmp
  formSteps.value = [...formSteps.value]
}

async function saveCompound() {
  if (!formName.value.trim()) {
    error.value = 'Name is required'
    return
  }
  if (formSteps.value.some(s => !s.preset_id)) {
    error.value = 'Each step must have a preset'
    return
  }

  const data = {
    name: formName.value.trim(),
    description: formDescription.value.trim(),
    steps: formSteps.value.map((s, i) => ({
      step_order: i + 1,
      preset_id: s.preset_id,
      width: s.width,
      height: s.height,
      denoising_strength: s.denoising_strength,
    })),
  }

  try {
    if (editing.value) {
      data.id = editing.value
      await api.updateCompoundPreset(data)
    } else {
      await api.createCompoundPreset(data)
    }
    showForm.value = false
    editing.value = null
    await loadData()
  } catch (e) {
    error.value = String(e)
  }
}

async function deleteCompound(id) {
  try {
    await api.deleteCompoundPreset(id)
    await loadData()
  } catch (e) {
    error.value = String(e)
  }
}

function getPresetName(presetId) {
  const p = presets.value.find(p => p.id === presetId)
  return p ? p.name : `#${presetId}`
}

onMounted(loadData)
</script>

<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Pipelines</h1>
      <button class="btn btn-primary" @click="openCreate">+ New Pipeline</button>
    </div>

    <div v-if="error" class="status status-error">{{ error }}</div>

    <div v-if="showForm" class="card" style="max-width: 700px;">
      <h3 style="margin-bottom: 12px;">{{ editing ? 'Edit Pipeline' : 'New Pipeline' }}</h3>

      <div class="form-group">
        <label class="form-label">Name</label>
        <input class="form-input" v-model="formName" placeholder="Pipeline name..." />
      </div>

      <div class="form-group">
        <label class="form-label">Description</label>
        <input class="form-input" v-model="formDescription" placeholder="Optional description..." />
      </div>

      <div style="margin-bottom: 12px;">
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">
          <label class="form-label" style="margin: 0;">Steps ({{ formSteps.length }})</label>
          <button class="btn btn-sm btn-secondary" @click="addStep">+ Add Step</button>
        </div>

        <div v-for="(step, idx) in formSteps" :key="idx" style="border: 1px solid var(--border); border-radius: 6px; padding: 12px; margin-bottom: 8px; background: var(--surface-1);">
          <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">
            <span style="font-weight: 600; font-size: 13px;">Step {{ idx + 1 }}{{ idx === 0 ? ' (txt2img)' : ' (img2img)' }}</span>
            <div style="display: flex; gap: 4px;">
              <button class="btn btn-sm btn-secondary" style="padding: 2px 8px;" @click="moveStepUp(idx)" :disabled="idx === 0">&#9650;</button>
              <button class="btn btn-sm btn-secondary" style="padding: 2px 8px;" @click="moveStepDown(idx)" :disabled="idx === formSteps.length - 1">&#9660;</button>
              <button class="btn btn-sm btn-secondary" style="padding: 2px 8px; color: var(--error, #e55);" @click="removeStep(idx)" :disabled="formSteps.length <= 1">&times;</button>
            </div>
          </div>

          <div style="display: grid; grid-template-columns: 1fr; gap: 8px;">
            <div class="form-group" style="margin: 0;">
              <label class="form-label">Preset</label>
              <select class="form-select" v-model="step.preset_id">
                <option :value="null" disabled>Select preset...</option>
                <option v-for="p in presets" :key="p.id" :value="p.id">{{ p.name }}</option>
              </select>
            </div>

            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 8px;">
              <div class="form-group" style="margin: 0;">
                <label class="form-label">Width</label>
                <input class="form-input" type="number" v-model.number="step.width" min="64" max="2048" step="64" />
              </div>
              <div class="form-group" style="margin: 0;">
                <label class="form-label">Height</label>
                <input class="form-input" type="number" v-model.number="step.height" min="64" max="2048" step="64" />
              </div>
            </div>

            <div v-if="idx > 0" class="form-group" style="margin: 0;">
              <label class="form-label">Denoising Strength</label>
              <input class="form-input" type="number" v-model.number="step.denoising_strength" min="0" max="1" step="0.05" />
            </div>
          </div>
        </div>
      </div>

      <div style="display: flex; gap: 8px;">
        <button class="btn btn-primary" @click="saveCompound">{{ editing ? 'Update' : 'Create' }}</button>
        <button class="btn btn-secondary" @click="cancelForm">Cancel</button>
      </div>
    </div>

    <div v-if="!showForm && compounds.length === 0" class="card" style="max-width: 700px; text-align: center;">
      <p style="color: var(--text-dim);">No pipelines yet. Create one to chain presets into multi-step generation.</p>
    </div>

    <div v-if="!showForm" style="display: grid; gap: 12px; max-width: 700px;">
      <div v-for="cp in compounds" :key="cp.id" class="card">
        <div style="display: flex; justify-content: space-between; align-items: start;">
          <div>
            <h3 style="margin-bottom: 4px;">{{ cp.name }}</h3>
            <p v-if="cp.description" style="color: var(--text-dim); font-size: 13px; margin-bottom: 8px;">{{ cp.description }}</p>
          </div>
          <div style="display: flex; gap: 6px;">
            <button class="btn btn-sm btn-secondary" @click="openEdit(cp)">Edit</button>
            <button class="btn btn-sm btn-secondary" style="color: var(--error, #e55);" @click="deleteCompound(cp.id)">Delete</button>
          </div>
        </div>

        <div style="display: flex; flex-wrap: wrap; gap: 6px; align-items: center;">
          <template v-for="(step, idx) in cp.steps" :key="step.id">
            <div v-if="idx > 0" style="color: var(--text-dim);">&#8594;</div>
            <div style="background: var(--surface-2); border-radius: 4px; padding: 4px 10px; font-size: 12px;">
              <span style="font-weight: 600;">{{ getPresetName(step.preset_id) }}</span>
              <span style="color: var(--text-dim);">{{ step.width }}&times;{{ step.height }}</span>
            </div>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>
