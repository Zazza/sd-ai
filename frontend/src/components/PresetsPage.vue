<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api.js'
import { t } from '../i18n/index.js'
import PresetForm from './PresetForm.vue'
import ImportModal from './ImportModal.vue'

const presets = ref([])
const presetTypes = ref([])
const loading = ref(true)
const showForm = ref(false)
const editingPreset = ref(null)

const selectMode = ref(false)
const selectedIds = ref(new Set())

const showImport = ref(false)
const filterSearch = ref('')
const filterType = ref('')
const importPresets = ref([])

const installStatuses = ref([])
const installingId = ref(null)

const pendingDeleteId = ref(null)
let deleteTimer = null

const showDeleteConfirm = ref(false)

const selectedCount = computed(() => selectedIds.value.size)

async function loadInstallStatuses() {
  try {
    const statuses = await api.getPresetsInstallStatus()
    installStatuses.value = statuses || []
  } catch { installStatuses.value = [] }
}

function getStatus(presetId) {
  return installStatuses.value.find(s => s.id === presetId)
}

async function installDeps(presetId) {
  installingId.value = presetId
  try {
    await api.installPresetDeps(presetId)
    await loadInstallStatuses()
  } catch (e) {
    alert('Install failed: ' + e)
  } finally {
    installingId.value = null
  }
}

async function load() {
  loading.value = true
  try {
    const [p, t] = await Promise.all([api.listPresets(), api.listPresetTypes()])
    presets.value = p || []
    presetTypes.value = t || []
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
  loadInstallStatuses()
}

const filteredPresets = computed(() => {
  let list = presets.value
  if (filterType.value) {
    list = list.filter(p => {
      const pt = presetTypes.value.find(t => t.name === filterType.value)
      return pt && p.type_id === pt.id
    })
  }
  if (filterSearch.value) {
    const q = filterSearch.value.toLowerCase()
    list = list.filter(p => p.name.toLowerCase().includes(q) || p.prompt.toLowerCase().includes(q))
  }
  return list
})

const availableTypes = computed(() => {
  const types = new Set(presets.value.map(p => p.preset_type).filter(Boolean))
  return [...types].sort()
})

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
  const ids = [...selectedIds.value]
  try {
    const path = await api.preparePresetsExport(ids)
    if (path) {
      selectMode.value = false
      selectedIds.value = new Set()
      alert('Saved: ' + path)
    }
  } catch (e) {
    alert('Export failed: ' + e)
  }
}

function requestBatchDelete() {
  if (selectedIds.value.size === 0) return
  showDeleteConfirm.value = true
}

async function confirmBatchDelete() {
  showDeleteConfirm.value = false
  const ids = [...selectedIds.value]
  for (const id of ids) {
    try {
      await api.deletePreset(id)
    } catch (e) {
      console.error('Delete failed for', id, e)
    }
  }
  selectedIds.value = new Set()
  selectMode.value = false
  await load()
}

async function handleOpenImport() {
  try {
    const result = await api.openImportFile()
    if (result && result.presets && result.presets.length > 0) {
      importPresets.value = result.presets
      showImport.value = true
    }
  } catch (e) {
    if (String(e)) alert('Import failed: ' + e)
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
      <h1 class="page-title">{{ t('presets.title') }}</h1>
      <div class="header-actions">
        <template v-if="selectMode">
          <button class="btn btn-secondary btn-sm" @click="selectAll">
            {{ selectedCount === presets.length ? t('presets.btn_deselect_all') : t('presets.btn_select_all') }}
          </button>
          <button class="btn btn-primary btn-sm" @click="handleExport" :disabled="selectedCount === 0">
            {{ t('presets.btn_export', { count: selectedCount }) }}
          </button>
          <button class="btn btn-danger btn-sm" @click="requestBatchDelete" :disabled="selectedCount === 0">
            {{ t('presets.btn_delete_selected', { count: selectedCount }) }}
          </button>
          <button class="btn btn-secondary btn-sm" @click="toggleSelectMode">{{ t('presets.btn_cancel') }}</button>
        </template>
        <template v-else>
          <button class="btn btn-secondary" @click="handleOpenImport">{{ t('presets.btn_import') }}</button>
          <button class="btn btn-secondary" @click="toggleSelectMode" :disabled="presets.length === 0">{{ t('presets.btn_select') }}</button>
          <button class="btn btn-primary" @click="openCreate">{{ t('presets.btn_new') }}</button>
        </template>
      </div>
    </div>

    <div v-if="loading" style="padding: 40px; text-align: center;">
      <span class="spinner"></span>
    </div>

    <div v-else-if="presets.length === 0" class="empty-state">
      <div class="empty-state-icon">&#9776;</div>
      <p>{{ t('presets.no_presets') }}</p>
    </div>

    <template v-else>
      <div style="display: flex; gap: 8px; margin-bottom: 16px; align-items: center;">
        <input class="form-input" v-model="filterSearch" :placeholder="t('presets.search_presets')" style="flex: 1; max-width: 300px;" />
        <select class="form-select" v-model="filterType" style="width: auto; min-width: 140px;">
          <option value="">{{ t('presets.all_types') }}</option>
          <option v-for="at in availableTypes" :key="at" :value="at">{{ at }}</option>
        </select>
      </div>

      <div v-if="filteredPresets.length === 0" style="color: var(--text-dim); text-align: center; padding: 24px;">
        {{ t('presets.no_match') }}
      </div>

      <div v-else class="card-grid">
        <div v-for="p in filteredPresets" :key="p.id" class="card preset-card" :class="{ 'preset-selected': selectedIds.has(p.id) }">
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
          <span>CFG {{ p.cfg_scale }}</span>
          <template v-if="getStatus(p.id)">
            <span v-if="getStatus(p.id).installed" style="color: #4ade80;">installed</span>
            <span v-else style="color: #fbbf24;">
              missing: {{ [...(getStatus(p.id).missing_sd || []), ...(getStatus(p.id).missing_lora || [])].join(', ') }}
            </span>
          </template>
        </div>
        <div v-if="!selectMode && getStatus(p.id) && !getStatus(p.id).installed" class="preset-actions" style="margin-top: 4px;">
          <button class="btn btn-secondary btn-sm" @click="installDeps(p.id)" :disabled="installingId === p.id">
            {{ installingId === p.id ? 'Installing...' : 'Install' }}
          </button>
        </div>
        <div v-if="!selectMode" class="preset-actions">
          <button class="btn btn-secondary btn-sm" @click="openEdit(p)">{{ t('presets.btn_edit') }}</button>
          <button class="btn btn-secondary btn-sm" @click="handleDuplicate(p)">{{ t('presets.btn_duplicate') }}</button>
          <button class="btn btn-danger btn-sm" @click="handleDelete(p.id)">{{ pendingDeleteId === p.id ? t('presets.btn_sure') : t('presets.btn_delete') }}</button>
        </div>
      </div>
    </div>
    </template>

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

    <div v-if="showDeleteConfirm" class="modal-overlay" @click.self="showDeleteConfirm = false">
      <div class="modal">
        <div class="modal-header">
          <h2 class="modal-title">{{ t('presets.delete_title') }}</h2>
          <button class="modal-close" @click="showDeleteConfirm = false">&times;</button>
        </div>
        <p>{{ t('presets.delete_confirm') }} <strong>{{ selectedCount }}</strong> preset{{ selectedCount > 1 ? 's' : '' }}?</p>
        <p style="color: var(--text-dim); font-size: 0.9em;">{{ t('presets.delete_cannot_undo') }}</p>
        <div style="display: flex; gap: 8px; justify-content: flex-end; margin-top: 16px;">
          <button class="btn btn-secondary" @click="showDeleteConfirm = false">{{ t('presets.btn_cancel') }}</button>
          <button class="btn btn-danger" @click="confirmBatchDelete">{{ t('presets.btn_delete') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>
