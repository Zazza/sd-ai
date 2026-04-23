<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'
import PresetForm from './PresetForm.vue'

const presets = ref([])
const loading = ref(true)
const showForm = ref(false)
const editingPreset = ref(null)

async function load() {
  loading.value = true
  try {
    presets.value = await api.listPresets()
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingPreset.value = null
  showForm.value = true
}

function openEdit(preset) {
  editingPreset.value = { ...preset }
  showForm.value = true
}

function closeForm() {
  showForm.value = false
  editingPreset.value = null
}

async function handleSave(data) {
  if (editingPreset.value) {
    await api.updatePreset(editingPreset.value.id, data)
  } else {
    await api.createPreset(data)
  }
  closeForm()
  await load()
}

async function handleDuplicate(preset) {
  const { id, created_at, updated_at, ...data } = preset
  await api.createPreset({ ...data, name: data.name + ' (copy)' })
  await load()
}

async function handleDelete(id) {
  if (!confirm('Delete this preset?')) return
  await api.deletePreset(id)
  await load()
}

onMounted(load)
</script>

<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Presets</h1>
      <button class="btn btn-primary" @click="openCreate">+ New Preset</button>
    </div>

    <div v-if="loading" style="padding: 40px; text-align: center;">
      <span class="spinner"></span>
    </div>

    <div v-else-if="presets.length === 0" class="empty-state">
      <div class="empty-state-icon">&#9776;</div>
      <p>No presets yet. Create your first one!</p>
    </div>

    <div v-else class="card-grid">
      <div v-for="p in presets" :key="p.id" class="card preset-card">
        <div class="preset-card-header">
          <div class="preset-name">{{ p.name }}</div>
          <span class="preset-type">{{ p.preset_type || 'general' }}</span>
        </div>
        <div class="preset-prompt">{{ p.prompt || '(no prompt)' }}</div>
        <div class="preset-meta">
          <span>{{ p.sampler }}</span>
          <span>{{ p.steps }} steps</span>
          <span>{{ p.width }}x{{ p.height }}</span>
          <span>CFG {{ p.cfg_scale }}</span>
        </div>
        <div class="preset-actions">
          <button class="btn btn-secondary btn-sm" @click="openEdit(p)">Edit</button>
          <button class="btn btn-secondary btn-sm" @click="handleDuplicate(p)">Duplicate</button>
          <button class="btn btn-danger btn-sm" @click="handleDelete(p.id)">Delete</button>
        </div>
      </div>
    </div>

    <PresetForm
      v-if="showForm"
      :preset="editingPreset"
      @save="handleSave"
      @close="closeForm"
    />
  </div>
</template>
