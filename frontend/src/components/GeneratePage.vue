<script setup>
import { ref, computed, onMounted, onUnmounted, watch, inject } from 'vue'
import { EventsEmit } from '../wailsjs/runtime/runtime'
import { api } from '../api.js'
import SavedDescriptionsModal from './SavedDescriptionsModal.vue'
import ImageViewer from './ImageViewer.vue'

const presets = ref([])
const presetTypes = ref([])
const compoundPresets = ref([])
const selectedTypeId = ref(null)
const selectedPresetId = ref(null)
const selectedCompoundPresetId = ref(null)
const genMode = ref('preset')
const description = ref('')
const negative = ref('')
const extraPrompt = ref('')
const extraNegativePrompt = ref('')
const generatedImage = ref('')
const genInfo = ref(null)
const sourceImage = ref('')
const sourceGenInfo = ref(null)
const isPreview = ref(false)
const savedPreview = ref(null)
const upscaling = ref(false)
const upscalingX2 = ref(false)
const previewMode = ref(false)
const effectivePrompt = ref('')
const effectiveNegative = ref('')

const generatingImage = ref(false)
const generationStage = ref('')
const error = ref('')
let promptDirty = true

const kidsModeActive = ref(false)

const shared = inject('sharedGenState', null)

const recommendDesc = ref('')
const recommending = ref(false)
const recommendResult = ref(null)

const savedDescs = ref([])
const showSavedDescs = ref(false)
const showViewer = ref(false)

const filteredPresets = computed(() => {
  if (!selectedTypeId.value) return presets.value
  return presets.value.filter(p => p.type_id === selectedTypeId.value)
})

const formattedGenInfo = computed(() => {
  if (!genInfo.value) return ''
  try {
    const parsed = typeof genInfo.value === 'string' ? JSON.parse(genInfo.value) : genInfo.value
    return JSON.stringify(parsed, null, 2)
  } catch {
    return String(genInfo.value)
  }
})

watch([description, negative], () => {
  promptDirty = true
})

watch(selectedTypeId, () => {
  const filtered = filteredPresets.value
  if (selectedPresetId.value && !filtered.find(p => p.id === selectedPresetId.value)) {
    selectedPresetId.value = null
  }
  api.updateSettings({ gen_type_id: String(selectedTypeId.value || '') }).catch(() => {})
})

async function loadKidsMode() {
  try {
    kidsModeActive.value = await api.isKidsModeActive()
  } catch {}
}

async function loadPresets() {
  try {
    const [p, t, c] = await Promise.all([api.listPresets(), api.listPresetTypes(), api.listCompoundPresets()])
    presets.value = p || []
    presetTypes.value = t || []
    compoundPresets.value = c || []
  } catch (e) {
    console.error(e)
  }
}

function saveGenState() {
  api.updateSettings({
    gen_preset_id: String(selectedPresetId.value || ''),
    gen_type_id: String(selectedTypeId.value || ''),
    gen_description: description.value || '',
    gen_negative: negative.value || '',
    gen_extra_prompt: extraPrompt.value || '',
    gen_extra_negative: extraNegativePrompt.value || '',
    gen_mode: genMode.value,
    gen_compound_preset_id: String(selectedCompoundPresetId.value || ''),
  }).catch(() => {})
}

async function recommendPreset() {
  if (!recommendDesc.value.trim()) return
  recommending.value = true
  recommendResult.value = null
  error.value = ''
  try {
    const result = await api.recommendPreset(recommendDesc.value)
    if (result) {
      recommendResult.value = result
      if (result.preset_id) {
        selectedPresetId.value = result.preset_id
      }
      if (result.extra_prompt) {
        description.value = result.extra_prompt
      }
    }
  } catch (e) {
    error.value = 'Recommendation failed: ' + String(e)
  } finally {
    recommending.value = false
  }
}

async function sendToSD() {
  if (genMode.value === 'compound' && !selectedCompoundPresetId.value) {
    error.value = 'Select a pipeline first'
    return
  }
  if (genMode.value === 'preset' && !selectedPresetId.value) {
    error.value = 'Select a preset first'
    return
  }
  saveGenState()
  generationStage.value = 'image'
  generatingImage.value = true
  generatedImage.value = ''
  genInfo.value = null
  isPreview.value = false
  savedPreview.value = null
  effectivePrompt.value = ''
  effectiveNegative.value = ''
  try {
    let result
    if (genMode.value === 'compound') {
      result = await api.generateCompoundImage({
        compound_preset_id: selectedCompoundPresetId.value,
        extra_prompt: extraPrompt.value,
        extra_negative_prompt: extraNegativePrompt.value,
      })
    } else {
      result = await api.generateImage(selectedPresetId.value, extraPrompt.value, extraNegativePrompt.value)
    }
    if (!result || !result.image) {
      error.value = 'No image returned. Check preset settings (model, sampler, scheduler).'
    } else {
      generatedImage.value = result.image
      genInfo.value = result.info
      sourceImage.value = result.image
      sourceGenInfo.value = result.info
      isPreview.value = result.is_preview || false
      effectivePrompt.value = result.effective_prompt || ''
      effectiveNegative.value = result.effective_negative_prompt || ''
    }
  } catch (e) {
    error.value = String(e)
  } finally {
    generatingImage.value = false
    generationStage.value = ''
  }
}

async function generateImage() {
  if (genMode.value === 'compound' && !selectedCompoundPresetId.value) {
    error.value = 'Select a pipeline first'
    return
  }
  if (genMode.value === 'preset' && !selectedPresetId.value) {
    error.value = 'Select a preset first'
    return
  }
  saveGenState()

  if (genMode.value === 'preset' && description.value.trim() && promptDirty) {
    generatingImage.value = true
    generationStage.value = 'prompt'
    error.value = ''
    try {
      const promptResult = await api.generateSdPrompt({
        preset_id: selectedPresetId.value,
        description: description.value,
        negative: negative.value,
      })
      if (promptResult && promptResult.prompt) {
        extraPrompt.value = promptResult.prompt
        extraNegativePrompt.value = promptResult.negative_prompt || ''
        promptDirty = false
      } else {
        error.value = 'LLM returned empty response'
        generatingImage.value = false
        generationStage.value = ''
        return
      }
    } catch (e) {
      error.value = 'Prompt generation failed: ' + String(e)
      generatingImage.value = false
      generationStage.value = ''
      return
    }
  }

  generationStage.value = 'image'
  generatingImage.value = true
  generatedImage.value = ''
  genInfo.value = null
  isPreview.value = false
  savedPreview.value = null
  effectivePrompt.value = ''
  effectiveNegative.value = ''
  try {
    let result
    if (genMode.value === 'compound') {
      result = await api.generateCompoundImage({
        compound_preset_id: selectedCompoundPresetId.value,
        extra_prompt: extraPrompt.value,
        extra_negative_prompt: extraNegativePrompt.value,
      })
    } else {
      result = await api.generateImage(selectedPresetId.value, extraPrompt.value, extraNegativePrompt.value)
    }
    if (!result || !result.image) {
      error.value = 'No image returned. Check preset settings (model, sampler, scheduler).'
    } else {
      generatedImage.value = result.image
      genInfo.value = result.info
      sourceImage.value = result.image
      sourceGenInfo.value = result.info
      isPreview.value = result.is_preview || false
      effectivePrompt.value = result.effective_prompt || ''
      effectiveNegative.value = result.effective_negative_prompt || ''
    }
  } catch (e) {
    error.value = String(e)
  } finally {
    generatingImage.value = false
    generationStage.value = ''
  }
}

async function upscalePreview() {
  if (!generatedImage.value || !selectedPresetId.value || !genInfo.value) return
  savedPreview.value = {
    image: generatedImage.value,
    info: genInfo.value,
    sourceImage: sourceImage.value,
    sourceGenInfo: sourceGenInfo.value,
  }
  upscaling.value = true
  error.value = ''
  try {
    let seed = -1
    let info = genInfo.value
    if (typeof info === 'string') {
      try { info = JSON.parse(info) } catch { info = {} }
    }
    if (info && typeof info === 'object') {
      const s = info.seed ?? info.Seed
      if (s !== undefined) seed = Number(s)
    }
    const result = await api.upscalePreview(generatedImage.value, selectedPresetId.value, seed)
    if (!result || !result.image) {
      error.value = 'Upscale failed: no image returned'
    } else {
      generatedImage.value = result.image
      genInfo.value = result.info
      sourceImage.value = result.image
      sourceGenInfo.value = result.info
      isPreview.value = false
    }
  } catch (e) {
    error.value = String(e)
  } finally {
    upscaling.value = false
  }
}

function backToPreview() {
  if (!savedPreview.value) return
  generatedImage.value = savedPreview.value.image
  genInfo.value = savedPreview.value.info
  sourceImage.value = savedPreview.value.sourceImage
  sourceGenInfo.value = savedPreview.value.sourceGenInfo
  isPreview.value = true
}

async function upscaleImageX2() {
  if (!generatedImage.value) return
  upscalingX2.value = true
  error.value = ''
  try {
    const result = await api.upscaleImage(generatedImage.value, genInfo.value, selectedPresetId.value)
    if (!result || !result.image) {
      error.value = 'Upscale x2 failed: no image returned'
    } else {
      generatedImage.value = result.image
      genInfo.value = result.info
      sourceImage.value = result.image
      sourceGenInfo.value = result.info
      isPreview.value = false
    }
  } catch (e) {
    error.value = String(e)
  } finally {
    upscalingX2.value = false
  }
}

async function loadSavedDescs() {
  try {
    savedDescs.value = await api.listDescriptions() || []
  } catch (e) {
    console.error(e)
  }
}

async function saveDescription() {
  if (!description.value.trim()) return
  try {
    await api.createDescriptionFull({
      text: description.value.trim(),
      name: '',
      negative_prompt: negative.value || '',
      type: '',
    })
    await loadSavedDescs()
  } catch (e) {
    error.value = 'Save description failed: ' + String(e)
  }
}

async function deleteDescription(id) {
  try {
    await api.deleteDescription(id)
    await loadSavedDescs()
  } catch (e) {
    error.value = 'Delete description failed: ' + String(e)
  }
}

function useDescription(desc) {
  description.value = desc.text
  if (desc.negative_prompt) negative.value = desc.negative_prompt
  showSavedDescs.value = false
}

async function handleCreateDesc(data) {
  try {
    await api.createDescriptionFull(data)
    await loadSavedDescs()
  } catch (e) {
    error.value = 'Create failed: ' + String(e)
  }
}

async function handleUpdateDesc(data) {
  try {
    await api.updateDescription(data)
    await loadSavedDescs()
  } catch (e) {
    error.value = 'Update failed: ' + String(e)
  }
}

async function downloadImage() {
  if (!generatedImage.value) return
  try {
    const defaultName = 'sd-studio-' + Date.now() + '.png'
    const savedPath = await api.saveImage(generatedImage.value, defaultName)
    if (savedPath) {
      error.value = ''
    }
  } catch (e) {
    error.value = 'Save failed: ' + String(e)
  }
}

function copyPrompt() {
  const parts = []
  if (effectivePrompt.value) parts.push(effectivePrompt.value)
  if (effectiveNegative.value) parts.push('Negative: ' + effectiveNegative.value)
  if (parts.length) navigator.clipboard.writeText(parts.join('\n'))
}

async function openBatchGeneration() {
  if (!extraPrompt.value) return

  EventsEmit('navigate:batch', {
    prefillPrompt: extraPrompt.value,
    prefillNegative: extraNegativePrompt.value || '',
    prefillPresetId: genMode.value === 'preset' ? selectedPresetId.value || null : null,
    prefillCompoundPresetId: genMode.value === 'compound' ? selectedCompoundPresetId.value || null : null,
  })
}

onMounted(async () => {
  await loadPresets()
  loadKidsMode()
  loadSavedDescs()
  try {
    const s = await api.getSettings()
    previewMode.value = s.preview_mode === 'true'
    if (s.gen_type_id) selectedTypeId.value = Number(s.gen_type_id) || null
    if (s.gen_extra_prompt) extraPrompt.value = s.gen_extra_prompt
    if (s.gen_extra_negative) extraNegativePrompt.value = s.gen_extra_negative
    if (s.gen_preset_id) selectedPresetId.value = Number(s.gen_preset_id)
    if (s.gen_description) description.value = s.gen_description
    if (s.gen_negative) negative.value = s.gen_negative
    if (s.gen_mode) genMode.value = s.gen_mode
    if (s.gen_compound_preset_id) selectedCompoundPresetId.value = Number(s.gen_compound_preset_id)
  } catch {}
  if (shared) {
    if (shared.selectedPresetId) selectedPresetId.value = shared.selectedPresetId
    if (shared.selectedCompoundPresetId) selectedCompoundPresetId.value = shared.selectedCompoundPresetId
    if (shared.genMode) genMode.value = shared.genMode
    if (shared.description) description.value = shared.description
    if (shared.negative) negative.value = shared.negative
  }
  try {
    const item = await api.getActiveSessionItem()
    if (item) {
      const image = await api.getSessionImage(item.id)
      if (image) {
        generatedImage.value = image
        genInfo.value = item.info || null
        sourceImage.value = image
        sourceGenInfo.value = item.info || null
        isPreview.value = item.is_preview || false
      }
    }
  } catch {}

  document.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
  if (shared) {
    shared.selectedPresetId = selectedPresetId.value
    shared.selectedCompoundPresetId = selectedCompoundPresetId.value
    shared.genMode = genMode.value
    shared.description = description.value
    shared.negative = negative.value
  }
})

function onKeydown(e) {
  if (e.key === 'Escape' && showSavedDescs.value) {
    showSavedDescs.value = false
    return
  }
  if ((e.ctrlKey || e.metaKey) && e.key === 'Enter' && !generatingImage.value) {
    e.preventDefault()
    generateImage()
  }
}
</script>

<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Generate</h1>
      <button class="btn btn-primary" @click="loadPresets">&#8635; Refresh</button>
    </div>

    <div v-if="kidsModeActive" class="service-status">
      <div class="status-badge status-ok">
        &#9679; Kids Mode
      </div>
    </div>

    <div v-if="error" class="status status-error">{{ error }}</div>

    <div class="generate-layout">
      <div class="generate-section">
        <div class="card">
          <div style="display: flex; gap: 8px; margin-bottom: 12px;">
            <button class="btn btn-sm" :class="genMode === 'preset' ? 'btn-primary' : 'btn-secondary'" @click="genMode = 'preset'">Preset</button>
            <button class="btn btn-sm" :class="genMode === 'compound' ? 'btn-primary' : 'btn-secondary'" @click="genMode = 'compound'">Pipeline</button>
          </div>

          <div v-if="genMode === 'preset'" style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px;">
            <div class="form-group">
              <label class="form-label">Type</label>
              <select class="form-select" v-model="selectedTypeId" :disabled="generatingImage">
                <option :value="null">All types</option>
                <option v-for="t in presetTypes" :key="t.id" :value="t.id">{{ t.name }}</option>
              </select>
            </div>
            <div class="form-group">
              <label class="form-label">Preset</label>
              <select class="form-select" v-model="selectedPresetId" :disabled="generatingImage">
                <option :value="null" disabled>Select preset...</option>
                <option v-for="p in filteredPresets" :key="p.id" :value="p.id">
                  {{ p.name }}
                </option>
              </select>
            </div>
          </div>

          <div v-if="genMode === 'compound'" class="form-group">
            <label class="form-label">Pipeline</label>
            <select class="form-select" v-model="selectedCompoundPresetId" :disabled="generatingImage">
              <option :value="null" disabled>Select pipeline...</option>
              <option v-for="c in compoundPresets" :key="c.id" :value="c.id">{{ c.name }} ({{ c.steps.length }} steps)</option>
            </select>
          </div>

          <div class="form-group" style="margin-top: 12px;">
            <label class="form-label">Recommend Preset</label>
            <div style="display: flex; gap: 8px;">
              <input class="form-input" v-model="recommendDesc" placeholder="Describe what you want..." :disabled="recommending || generatingImage" style="flex: 1;" />
              <button class="btn btn-secondary" @click="recommendPreset" :disabled="recommending || !recommendDesc.trim()">
                {{ recommending ? '...' : 'Recommend' }}
              </button>
            </div>
            <div v-if="recommendResult" style="margin-top: 8px; padding: 8px; background: var(--surface-2); border-radius: 6px; font-size: 13px;">
              <div style="color: var(--text-bright);">{{ recommendResult.preset_name }}</div>
              <div v-if="recommendResult.reasoning" style="color: var(--text-dim); margin-top: 4px;">{{ recommendResult.reasoning }}</div>
            </div>
          </div>

          <div class="form-group">
            <label class="form-label">Description</label>
            <textarea class="form-textarea" v-model="description" rows="4" placeholder="Describe what to add or change in the image..." :disabled="generatingImage"></textarea>
            <div style="display: flex; gap: 8px; margin-top: 6px;">
              <button class="btn btn-sm btn-secondary" @click="saveDescription" :disabled="generatingImage || !description.trim()">Save</button>
              <button class="btn btn-sm btn-secondary" @click="showSavedDescs = !showSavedDescs">
                Saved {{ savedDescs.length ? '(' + savedDescs.length + ')' : '' }}
              </button>
            </div>
          </div>

          <div class="form-group">
            <label class="form-label">Negative</label>
            <textarea class="form-textarea" v-model="negative" rows="2" placeholder="What should NOT be in the image..." :disabled="generatingImage"></textarea>
          </div>

          <button class="btn btn-primary" :class="{ 'btn-generating': generatingImage }" style="width: 100%; justify-content: center; padding: 12px;" @click="generateImage" :disabled="generatingImage || (genMode === 'preset' ? !selectedPresetId : !selectedCompoundPresetId)">
            <span v-if="generatingImage" style="display: inline-flex; align-items: center; gap: 6px;">
              <span class="spinner" style="width: 14px; height: 14px; border-width: 2px;"></span>
              {{ generationStage === 'prompt' ? 'Generating prompt...' : 'Generating image...' }}
            </span>
            <span v-else>Generate Image</span>
          </button>
          <button class="btn btn-secondary" style="width: 100%; justify-content: center; padding: 8px; margin-top: 6px;" @click="openBatchGeneration" :disabled="generatingImage || !extraPrompt || (genMode === 'preset' ? !selectedPresetId : !selectedCompoundPresetId)">
            Batch Generation
          </button>

          <details v-if="!kidsModeActive" style="margin-top: 8px;" class="card">
            <summary style="cursor: pointer; color: var(--text-dim); font-size: 13px; padding: 8px;">Edit SD Prompt</summary>
            <div style="margin-top: 8px;">
              <div class="form-group">
                <label class="form-label">Positive Prompt</label>
                <textarea class="form-textarea" v-model="extraPrompt" rows="4" placeholder="SD positive prompt..." :disabled="generatingImage"></textarea>
              </div>
              <div class="form-group">
                <label class="form-label">Negative Prompt</label>
                <textarea class="form-textarea" v-model="extraNegativePrompt" rows="2" placeholder="SD negative prompt..." :disabled="generatingImage"></textarea>
              </div>
              <button class="btn btn-secondary" style="width: 100%; justify-content: center;" @click="sendToSD" :disabled="generatingImage || (genMode === 'preset' ? !selectedPresetId : !selectedCompoundPresetId) || !extraPrompt">
                Send to SD
              </button>
            </div>
          </details>
          <div v-else style="margin-top: 8px; padding: 8px; background: var(--bg-secondary); border-radius: 6px; text-align: center; font-size: 12px; color: var(--text-dim);">
            &#128274; Prompt editing disabled in Kids Mode
          </div>
        </div>
      </div>

      <div class="generate-section">
        <div class="generate-image-area">
          <div v-if="generatingImage || upscaling || upscalingX2" style="text-align: center;">
            <span class="spinner" style="width: 32px; height: 32px; border-width: 3px;"></span>
            <p style="margin-top: 12px; color: var(--text-dim);">{{ upscalingX2 ? 'Upscaling x2...' : upscaling ? 'Upscaling to full resolution...' : generationStage === 'prompt' ? 'Generating prompt...' : 'Generating image...' }}</p>
          </div>
          <div v-else-if="generatedImage" style="width: 100%; padding: 12px;">
            <div v-if="isPreview && previewMode" class="status status-info" style="margin-bottom: 8px; text-align: center;">
              Preview &mdash; click Upscale for full resolution. Tip: switch preset before upscale for style transfer.
            </div>
            <img :src="'data:image/png;base64,' + generatedImage" alt="Generated" class="img-fade-in" style="border-radius: var(--radius-sm); cursor: zoom-in;" @click="showViewer = true" />
            <div style="display: flex; gap: 8px; margin-top: 12px; justify-content: center;">
              <button v-if="isPreview && previewMode" class="btn btn-primary btn-sm" @click="upscalePreview">Upscale to Full Size</button>
              <button v-if="!isPreview && savedPreview" class="btn btn-secondary btn-sm" @click="backToPreview">&larr; Back to Preview</button>
              <button v-if="generatedImage && !isPreview" class="btn btn-secondary btn-sm" @click="upscaleImageX2" :disabled="upscalingX2">Upscale x2</button>
              <button class="btn btn-secondary btn-sm" @click="downloadImage" data-tooltip="Download">Download</button>
              <button class="btn btn-secondary btn-sm" @click="copyPrompt" data-tooltip="Copy prompt">Copy</button>
              <button class="btn btn-secondary btn-sm" @click="generateImage" data-tooltip="Regenerate">Regenerate</button>
            </div>
          </div>
          <div v-else class="generate-placeholder">
            <div class="generate-placeholder-icon">&#9744;</div>
            <p>Generated image will appear here</p>
          </div>
        </div>

        <details v-if="genInfo" class="gen-info card">
          <summary>Generation Info</summary>
          <pre style="white-space: pre-wrap; word-break: break-word; overflow-wrap: break-word;">{{ formattedGenInfo }}</pre>
        </details>

        <details v-if="effectivePrompt" class="gen-info card">
          <summary>Effective Prompt</summary>
          <div style="margin-bottom: 8px;">
            <div style="color: var(--text-dim); font-size: 11px; margin-bottom: 4px;">POSITIVE</div>
            <div style="font-size: 12px; line-height: 1.5; word-break: break-word;">{{ effectivePrompt }}</div>
          </div>
          <div v-if="effectiveNegative" style="margin-top: 8px; padding-top: 8px; border-top: 1px solid var(--border);">
            <div style="color: var(--text-dim); font-size: 11px; margin-bottom: 4px;">NEGATIVE</div>
            <div style="font-size: 12px; line-height: 1.5; word-break: break-word;">{{ effectiveNegative }}</div>
          </div>
        </details>
      </div>
    </div>

    <SavedDescriptionsModal
      :visible="showSavedDescs"
      :descriptions="savedDescs"
      @close="showSavedDescs = false"
      @use="useDescription"
      @create="handleCreateDesc"
      @update="handleUpdateDesc"
      @delete="deleteDescription"
    />

    <ImageViewer v-if="showViewer" :image-base64="generatedImage" @close="showViewer = false" />
  </div>
</template>
