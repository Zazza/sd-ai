<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api.js'
import PresetForm from './PresetForm.vue'
import ImportModal from './ImportModal.vue'

const presets = ref([])
const loading = ref(true)
const showForm = ref(false)
const editingPreset = ref(null)

const selectMode = ref(false)
const selectedIds = ref(new Set())

const showImport = ref(false)
const importPresets = ref([])

const pendingDeleteId = ref(null)
let deleteTimer = null

const selectedCount = computed(() => selectedIds.value.size)

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
  if (pendingDeleteId.value === id) {
    clearTimeout(deleteTimer)
    pendingDeleteId.value = null
    await api.deletePreset(id)
    await load()
  } else {
    clearTimeout(deleteTimer)
    pendingDeleteId.value = id
    deleteTimer = setTimeout(() => { pendingDeleteId.value = null }, 3000)
  }
}

function toggleSelectMode() {
  selectMode.value = !selectMode.value
  if (!selectMode.value) {
    selectedIds.value = new Set()
  }
}

function togglePreset(id) {
  const next = new Set(selectedIds.value)
  if (next.has(id)) {
    next.delete(id)
  } else {
    next.add(id)
  }
  selectedIds.value = next
}

function selectAll() {
  if (selectedIds.value.size === presets.value.length) {
    selectedIds.value = new Set()
  } else {
    selectedIds.value = new Set(presets.value.map(p => p.id))
  }
}

async function handleExport() {
  if (selectedIds.value.size === 0) return
  try {
    await api.exportPresets([...selectedIds.value])
    selectMode.value = false
    selectedIds.value = new Set()
  } catch (e) {
    if (e.message) alert('Export failed: ' + e.message)
  }
}

async function handleOpenImport() {
  try {
    const result = await api.openImportFile()
    if (result && result.presets && result.presets.length > 0) {
      importPresets.value = result.presets
      showImport.value = true
    }
  } catch (e) {
    if (e.message) alert('Import failed: ' + e.message)
  }
}

function handleImportDone() {
  showImport.value = false
  importPresets.value = []
  load()
}

onMounted(load)
</script>

<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Presets</h1>
      <div class="header-actions">
        <template v-if="selectMode">
          <button class="btn btn-secondary btn-sm" @click="selectAll">
            {{ selectedCount === presets.length ? 'Deselect All' : 'Select All' }}
          </button>
          <button class="btn btn-primary btn-sm" @click="handleExport" :disabled="selectedCount === 0">
            Export ({{ selectedCount }})
          </button>
          <button class="btn btn-secondary btn-sm" @click="toggleSelectMode">Cancel</button>
        </template>
        <template v-else>
          <button class="btn btn-secondary" @click="handleOpenImport">Import</button>
          <button class="btn btn-secondary" @click="toggleSelectMode" :disabled="presets.length === 0">Export</button>
          <button class="btn btn-primary" @click="openCreate">+ New Preset</button>
        </template>
      </div>
    </div>

    <div v-if="loading" style="padding: 40px; text-align: center;">
      <span class="spinner"></span>
    </div>

    <div v-else-if="presets.length === 0" class="empty-state">
      <div class="empty-state-icon">&#9776;</div>
      <p>No presets yet. Create your first one!</p>
    </div>

    <div v-else class="card-grid">
      <div v-for="p in presets" :key="p.id" class="card preset-card" :class="{ 'preset-selected': selectedIds.has(p.id) }">
        <div class="preset-card-header">
          <div style="display: flex; align-items: center; gap: 8px;">
            <input v-if="selectMode" type="checkbox" :checked="selectedIds.has(p.id)" @change="togglePreset(p.id)" />
            <div class="preset-name">{{ p.name }}</div>
          </div>
          <span class="preset-type">{{ p.preset_type || 'general' }}</span>
        </div>
        <div class="preset-prompt">{{ p.prompt || '(no prompt)' }}</div>
        <div class="preset-meta">
          <span>{{ p.sampler }}</span>
          <span>{{ p.steps }} steps</span>
          <span>{{ p.width }}x{{ p.height }}</span>
          <span>CFG {{ p.cfg_scale }}</span>
        </div>
        <div v-if="!selectMode" class="preset-actions">
          <button class="btn btn-secondary btn-sm" @click="openEdit(p)">Edit</button>
          <button class="btn btn-secondary btn-sm" @click="handleDuplicate(p)">Duplicate</button>
          <button class="btn btn-danger btn-sm" @click="handleDelete(p.id)">{{ pendingDeleteId === p.id ? 'Sure?' : 'Delete' }}</button>
        </div>
      </div>
    </div>

    <PresetForm
      v-if="showForm"
      :preset="editingPreset"
      @save="handleSave"
      @close="closeForm"
    />

    <ImportModal
      v-if="showImport"
      :presets="importPresets"
      @done="handleImportDone"
      @close="showImport = false"
    />
  </div>
</template>
