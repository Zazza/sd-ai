<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { api } from '../api.js'
import { t } from '../i18n/index.js'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'

const exportPresets = ref([])
const selectedPresetId = ref(null)
const sourceImage = ref('')
const sourceName = ref('')
const imageInfo = ref({ width: 0, height: 0 })

const format = ref('png')
const width = ref(0)
const height = ref(0)
const lockRatio = ref(true)
const quality = ref(90)
const interpolation = ref('lanczos')
const filename = ref('')

const exporting = ref(false)
const error = ref('')
const successMsg = ref('')

const editingPreset = ref(null)
const presetName = ref('')
const showPresetForm = ref(false)

const formats = [
  { value: 'png', label: 'PNG' },
  { value: 'jpeg', label: 'JPEG' },
]

const interpolations = [
  { value: 'nearest', label: 'Nearest' },
  { value: 'linear', label: 'Linear' },
  { value: 'lanczos', label: 'Lanczos' },
]

const showQuality = computed(() => format.value !== 'png')

const aspectRatio = computed(() => {
  if (!imageInfo.value.width || !imageInfo.value.height) return 1
  return imageInfo.value.width / imageInfo.value.height
})

watch(width, (w) => {
  if (lockRatio.value && w > 0 && aspectRatio.value) {
    height.value = Math.round(w / aspectRatio.value)
  }
})

watch(height, (h) => {
  if (lockRatio.value && h > 0 && aspectRatio.value) {
    width.value = Math.round(h * aspectRatio.value)
  }
})

watch(format, () => {
  updateFilename()
})

function updateFilename() {
  const base = sourceName.value.replace(/\.[^.]+$/, '') || 'export'
  filename.value = `${base}_${new Date().toISOString().slice(0, 10)}.${format.value}`
}

async function loadLastImage() {
  try {
    const item = await api.getActiveSessionItem()
    if (item) {
      const image = await api.getSessionImage(item.id)
      if (image) {
        sourceImage.value = image
        loadImageDimensions(image)
        sourceName.value = item.file_name || 'generated_image'
        updateFilename()
      }
    }
  } catch {}
}

function loadImageDimensions(base64) {
  const img = new Image()
  img.onload = () => {
    imageInfo.value = { width: img.naturalWidth, height: img.naturalHeight }
    width.value = img.naturalWidth
    height.value = img.naturalHeight
  }
  const prefix = base64.startsWith('data:') ? '' : 'data:image/png;base64,'
  img.src = prefix + base64
}

async function uploadImage() {
  try {
    const result = await api.readImageFile()
    if (result) {
      sourceImage.value = result
      loadImageDimensions(result)
      sourceName.value = 'uploaded_image'
      updateFilename()
    }
  } catch {}
}

function applyPreset(preset) {
  selectedPresetId.value = preset.id
  format.value = preset.format
  interpolation.value = preset.interpolation
  quality.value = preset.quality || 90
  lockRatio.value = preset.lock_ratio

  if (preset.width > 0 || preset.height > 0) {
    if (preset.lock_ratio && preset.width > 0 && preset.height === 0) {
      width.value = preset.width
    } else if (preset.lock_ratio && preset.height > 0 && preset.width === 0) {
      height.value = preset.height
    } else {
      width.value = preset.width || imageInfo.value.width
      height.value = preset.height || imageInfo.value.height
    }
  } else {
    width.value = imageInfo.value.width
    height.value = imageInfo.value.height
  }
  updateFilename()
}

function onPresetSelect(id) {
  const p = exportPresets.value.find(ep => ep.id === Number(id))
  if (p) applyPreset(p)
}

async function savePreset() {
  try {
    const data = {
      name: presetName.value,
      format: format.value,
      width: width.value === imageInfo.value.width ? 0 : width.value,
      height: height.value === imageInfo.value.height ? 0 : height.value,
      lock_ratio: lockRatio.value,
      quality: quality.value,
      interpolation: interpolation.value,
    }
    if (editingPreset.value) {
      data.id = editingPreset.value.id
    }
    await api.saveExportPreset(data)
    await loadPresets()
    editingPreset.value = null
    presetName.value = ''
    showPresetForm.value = false
  } catch (e) {
    error.value = String(e)
  }
}

function startEditPreset(preset) {
  editingPreset.value = preset
  presetName.value = preset.name
  showPresetForm.value = true
}

async function deletePreset(id) {
  try {
    await api.deleteExportPreset(id)
    if (selectedPresetId.value === id) selectedPresetId.value = null
    await loadPresets()
  } catch (e) {
    error.value = String(e)
  }
}

async function loadPresets() {
  try {
    exportPresets.value = await api.listExportPresets()
  } catch {}
}

async function doExport() {
  if (!sourceImage.value) {
    error.value = t('export.error_no_image')
    return
  }
  exporting.value = true
  error.value = ''
  successMsg.value = ''
  try {
    const path = await api.exportImage({
      image_base64: sourceImage.value,
      format: format.value,
      width: width.value,
      height: height.value,
      lock_ratio: lockRatio.value,
      quality: quality.value,
      interpolation: interpolation.value,
      filename: filename.value,
    })
    if (path) {
      successMsg.value = `Saved: ${path}`
    }
  } catch (e) {
    error.value = String(e)
  } finally {
    exporting.value = false
  }
}

function resetSize() {
  width.value = imageInfo.value.width
  height.value = imageInfo.value.height
}

function openSavePreset() {
  editingPreset.value = null
  presetName.value = ''
  showPresetForm.value = true
}

onMounted(async () => {
  await loadPresets()
  await loadLastImage()
  EventsOn('session:active', () => {
    loadLastImage()
  })
})

onUnmounted(() => {
  EventsOff('session:active')
})
</script>

<template>
  <div class="page-enter export-page">
    <div class="export-layout">
      <div class="export-main">
        <div class="export-preview card">
          <div class="card-header">
            <span class="card-title">{{ t('export.source_image') }}</span>
            <div class="card-actions">
              <button class="btn btn-sm" @click="loadLastImage">{{ t('export.btn_load_last') }}</button>
              <button class="btn btn-sm" @click="uploadImage">{{ t('export.btn_upload') }}</button>
            </div>
          </div>
          <div class="preview-area">
            <div v-if="sourceImage" class="preview-image-wrap">
              <img :src="sourceImage.startsWith('data:') ? sourceImage : 'data:image/png;base64,' + sourceImage" alt="Source" />
              <div class="image-meta" v-if="imageInfo.width">
                {{ imageInfo.width }} x {{ imageInfo.height }}
              </div>
            </div>
            <div v-else class="preview-empty">
              {{ t('export.no_image') }}
            </div>
          </div>
        </div>

        <div class="export-settings card">
          <div class="card-header">
            <span class="card-title">{{ t('export.export_settings') }}</span>
          </div>
          <div class="card-body">
            <div class="form-group">
              <label class="form-label">{{ t('export.label_preset') }}</label>
              <div class="preset-row">
                <select class="form-select" :value="selectedPresetId" @change="onPresetSelect($event.target.value)">
                  <option :value="null">{{ t('export.custom') }}</option>
                  <option v-for="p in exportPresets" :key="p.id" :value="p.id">{{ p.name }}</option>
                </select>
                <button class="btn btn-sm" @click="openSavePreset" title="Save current as preset">{{ t('export.btn_save_preset') }}</button>
              </div>
            </div>

            <div class="preset-chips" v-if="exportPresets.length">
              <button
                v-for="p in exportPresets"
                :key="p.id"
                class="preset-chip"
                :class="{ active: selectedPresetId === p.id }"
                @click="applyPreset(p)"
              >
                {{ p.name }}
                <span class="chip-meta">{{ p.format.toUpperCase() }}</span>
              </button>
            </div>

            <div class="form-divider"></div>

            <div class="form-group">
              <label class="form-label">{{ t('export.label_format') }}</label>
              <select class="form-select" v-model="format">
                <option v-for="f in formats" :key="f.value" :value="f.value">{{ f.label }}</option>
              </select>
            </div>

            <div class="form-row">
              <div class="form-group form-group-half">
                <label class="form-label">{{ t('export.label_width') }}</label>
                <input class="form-control" type="number" v-model.number="width" min="1" />
              </div>
              <div class="form-group form-group-half">
                <label class="form-label">{{ t('export.label_height') }}</label>
                <input class="form-control" type="number" v-model.number="height" min="1" />
              </div>
            </div>

            <div class="form-group form-inline">
              <label class="form-label">
                <input type="checkbox" v-model="lockRatio" /> {{ t('export.lock_aspect') }}
              </label>
              <button class="btn btn-sm" @click="resetSize" v-if="imageInfo.width">{{ t('export.btn_reset') }}</button>
            </div>

            <div class="form-group" v-if="showQuality">
              <label class="form-label">{{ t('export.label_quality', { value: quality }) }}</label>
              <input type="range" min="1" max="100" v-model.number="quality" class="quality-slider" />
            </div>

            <div class="form-group">
              <label class="form-label">{{ t('export.label_interpolation') }}</label>
              <select class="form-select" v-model="interpolation">
                <option v-for="i in interpolations" :key="i.value" :value="i.value">{{ i.label }}</option>
              </select>
            </div>

            <div class="form-group">
              <label class="form-label">{{ t('export.label_filename') }}</label>
              <input class="form-control" type="text" v-model="filename" />
            </div>

            <div v-if="error" class="error-msg">{{ error }}</div>
            <div v-if="successMsg" class="success-msg">{{ successMsg }}</div>

            <button class="btn btn-primary btn-block" :disabled="exporting || !sourceImage" @click="doExport">
              {{ exporting ? t('export.exporting') : t('export.btn_export') }}
            </button>
          </div>
        </div>
      </div>

      <div class="export-sidebar">
        <div class="card">
          <div class="card-header">
            <span class="card-title">{{ t('export.saved_presets') }}</span>
          </div>
          <div class="card-body">
            <div class="preset-list">
              <div v-if="!exportPresets.length" class="preset-empty">{{ t('export.no_presets') }}</div>
              <div
                v-for="p in exportPresets"
                :key="p.id"
                class="preset-item"
                :class="{ active: selectedPresetId === p.id }"
                @click="applyPreset(p)"
              >
                <div class="preset-info">
                  <span class="preset-name">{{ p.name }}</span>
                  <span class="preset-meta">{{ p.format.toUpperCase() }} {{ p.width || 'orig' }}x{{ p.height || 'orig' }} Q{{ p.quality }}</span>
                </div>
                <div class="preset-actions">
                  <button class="btn-icon" @click.stop="startEditPreset(p)" title="Edit">&#9998;</button>
                  <button class="btn-icon btn-icon-danger" @click.stop="deletePreset(p.id)" title="Delete">&times;</button>
                </div>
              </div>
            </div>

            <div class="preset-form" v-if="showPresetForm">
              <div class="form-divider"></div>
              <div class="form-group">
                <input class="form-control" type="text" v-model="presetName" :placeholder="t('export.placeholder_preset_name')" />
              </div>
              <div class="preset-form-actions">
                <button class="btn btn-sm btn-primary" @click="savePreset" :disabled="!presetName">
                  {{ editingPreset ? t('export.btn_update') : t('export.btn_save') }}
                </button>
                <button class="btn btn-sm" @click="showPresetForm = false; editingPreset = null">{{ t('export.btn_cancel') }}</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.export-page {
  padding: 16px;
  height: 100%;
  overflow-y: auto;
}

.export-layout {
  display: grid;
  grid-template-columns: 1fr 260px;
  gap: 16px;
  height: 100%;
}

.export-main {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
}

.card-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-bright);
}

.card-actions {
  display: flex;
  gap: 6px;
}

.card-body {
  padding: 16px;
}

.preview-area {
  padding: 16px;
  min-height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.preview-image-wrap {
  max-width: 100%;
  max-height: 40vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.preview-image-wrap img {
  max-width: 100%;
  max-height: 35vh;
  object-fit: contain;
  border-radius: var(--radius-sm);
}

.image-meta {
  font-size: 12px;
  color: var(--text-dim);
}

.preview-empty {
  color: var(--text-dim);
  font-size: 13px;
  text-align: center;
  padding: 40px;
}

.preset-row {
  display: flex;
  gap: 8px;
}

.preset-row .form-select {
  flex: 1;
}

.preset-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 12px;
}

.preset-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 14px;
  color: var(--text);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
}

.preset-chip:hover {
  border-color: var(--accent);
  color: var(--text-bright);
}

.preset-chip.active {
  background: var(--accent-bg);
  border-color: var(--accent);
  color: var(--accent);
}

.chip-meta {
  font-size: 10px;
  color: var(--text-dim);
  text-transform: uppercase;
}

.preset-chip.active .chip-meta {
  color: var(--accent);
}

.form-row {
  display: flex;
  gap: 12px;
}

.form-group-half {
  flex: 1;
}

.form-inline {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.form-inline .form-label {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 0;
}

.quality-slider {
  width: 100%;
  accent-color: var(--accent);
}

.form-divider {
  border-top: 1px solid var(--border);
  margin: 12px 0;
}

.preset-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.preset-empty {
  color: var(--text-dim);
  font-size: 12px;
  text-align: center;
  padding: 16px;
}

.preset-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background 0.15s;
}

.preset-item:hover {
  background: var(--surface-2);
}

.preset-item.active {
  background: var(--accent-bg);
}

.preset-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.preset-name {
  font-size: 13px;
  color: var(--text-bright);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.preset-meta {
  font-size: 11px;
  color: var(--text-dim);
}

.preset-actions {
  display: flex;
  gap: 2px;
  opacity: 0;
  transition: opacity 0.15s;
  flex-shrink: 0;
}

.preset-item:hover .preset-actions {
  opacity: 1;
}

.btn-icon {
  background: none;
  border: none;
  color: var(--text-dim);
  cursor: pointer;
  padding: 2px 4px;
  font-size: 14px;
  border-radius: 3px;
  line-height: 1;
}

.btn-icon:hover {
  color: var(--text-bright);
  background: var(--surface-3);
}

.btn-icon-danger:hover {
  color: var(--danger);
}

.preset-form-actions {
  display: flex;
  gap: 8px;
}

.error-msg {
  color: var(--danger);
  font-size: 12px;
  margin-bottom: 8px;
}

.success-msg {
  color: var(--success);
  font-size: 12px;
  margin-bottom: 8px;
  word-break: break-all;
}

.btn-block {
  width: 100%;
}
</style>
