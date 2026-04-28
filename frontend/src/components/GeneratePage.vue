<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { api } from '../api.js'

const presets = ref([])
const presetTypes = ref([])
const selectedTypeId = ref(null)
const selectedPresetId = ref(null)
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

const llmAvailable = ref(false)
const llmModel = ref('')
const sdAvailable = ref(false)
const sdModel = ref('')
const kidsModeActive = ref(false)
let statusInterval = null

const recommendDesc = ref('')
const recommending = ref(false)
const recommendResult = ref(null)

const savedDescs = ref([])
const showSavedDescs = ref(false)

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

async function checkServices() {
  try {
    const status = await api.checkServices()
    llmAvailable.value = status.llm?.available || false
    llmModel.value = status.llm?.model || ''
    sdAvailable.value = status.sd?.available || false
    sdModel.value = status.sd?.model || ''
    kidsModeActive.value = await api.isKidsModeActive()
  } catch {}
}

async function loadPresets() {
  try {
    const [p, t] = await Promise.all([api.listPresets(), api.listPresetTypes()])
    presets.value = p || []
    presetTypes.value = t || []
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
  if (!selectedPresetId.value) {
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
    const result = await api.generateImage(selectedPresetId.value, extraPrompt.value, extraNegativePrompt.value)
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
  if (!selectedPresetId.value) {
    error.value = 'Select a preset first'
    return
  }
  saveGenState()

  if (description.value.trim() && promptDirty) {
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
    const result = await api.generateImage(selectedPresetId.value, extraPrompt.value, extraNegativePrompt.value)
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
    await api.createDescription(description.value.trim())
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

function useDescription(text) {
  description.value = text
  showSavedDescs.value = false
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

onMounted(async () => {
  loadPresets()
  checkServices()
  loadSavedDescs()
  statusInterval = setInterval(checkServices, 30000)
  try {
    const s = await api.getSettings()
    previewMode.value = s.preview_mode === 'true'
    if (s.gen_type_id) selectedTypeId.value = Number(s.gen_type_id) || null
    if (s.gen_preset_id) selectedPresetId.value = Number(s.gen_preset_id)
    if (s.gen_description) description.value = s.gen_description
    if (s.gen_negative) negative.value = s.gen_negative
    if (s.gen_extra_prompt) extraPrompt.value = s.gen_extra_prompt
    if (s.gen_extra_negative) extraNegativePrompt.value = s.gen_extra_negative
  } catch {}
  try {
    const last = await api.getLastImage()
    if (last && last.image) {
      generatedImage.value = last.image
      genInfo.value = last.info
      sourceImage.value = last.image
      sourceGenInfo.value = last.info
      isPreview.value = last.is_preview || false
    }
  } catch {}
})

onUnmounted(() => {
  if (statusInterval) clearInterval(statusInterval)
})
</script>

<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Generate</h1>
      <button class="btn btn-primary" @click="loadPresets">&#8635; Refresh</button>
    </div>

    <div class="service-status">
      <div class="status-badge" :class="llmAvailable ? 'status-ok' : 'status-down'">
        &#9679; LLM{{ llmModel ? ': ' + llmModel : '' }}
      </div>
      <div class="status-badge" :class="sdAvailable ? 'status-ok' : 'status-down'">
        &#9679; SD{{ sdModel ? ': ' + sdModel : '' }}
      </div>
      <div v-if="kidsModeActive" class="status-badge status-ok">
        &#9679; Kids Mode
      </div>
    </div>

    <div v-if="error" class="status status-error">{{ error }}</div>

    <div class="generate-layout">
      <div class="generate-section">
        <div class="card">
          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px;">
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
            <div v-if="showSavedDescs && savedDescs.length" style="margin-top: 8px; max-height: 200px; overflow-y: auto;">
              <div v-for="d in savedDescs" :key="d.id" style="display: flex; align-items: center; gap: 8px; padding: 6px 8px; background: var(--surface-2); border-radius: 6px; margin-bottom: 4px; cursor: pointer;" @click="useDescription(d.text)">
                <span style="flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 13px;">{{ d.text }}</span>
                <button class="btn btn-sm btn-secondary" style="padding: 2px 8px;" @click.stop="deleteDescription(d.id)">&times;</button>
              </div>
            </div>
            <div v-if="showSavedDescs && !savedDescs.length" style="margin-top: 6px; color: var(--text-dim); font-size: 13px;">No saved descriptions yet</div>
          </div>

          <div class="form-group">
            <label class="form-label">Negative</label>
            <textarea class="form-textarea" v-model="negative" rows="2" placeholder="What should NOT be in the image..." :disabled="generatingImage"></textarea>
          </div>

          <button class="btn btn-primary" style="width: 100%; justify-content: center; padding: 12px;" @click="generateImage" :disabled="generatingImage || !selectedPresetId">
            <span v-if="generatingImage" style="display: inline-flex; align-items: center; gap: 6px;">
              <span class="spinner" style="width: 14px; height: 14px; border-width: 2px;"></span>
              {{ generationStage === 'prompt' ? 'Generating prompt...' : 'Generating image...' }}
            </span>
            <span v-else>Generate Image</span>
          </button>

          <details style="margin-top: 8px;" class="card">
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
              <button class="btn btn-secondary" style="width: 100%; justify-content: center;" @click="sendToSD" :disabled="generatingImage || !selectedPresetId || !extraPrompt">
                Send to SD
              </button>
            </div>
          </details>
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
            <img :src="'data:image/png;base64,' + generatedImage" alt="Generated" style="border-radius: var(--radius-sm);" />
            <div style="display: flex; gap: 8px; margin-top: 12px; justify-content: center;">
              <button v-if="isPreview && previewMode" class="btn btn-primary btn-sm" @click="upscalePreview">Upscale to Full Size</button>
              <button v-if="!isPreview && savedPreview" class="btn btn-secondary btn-sm" @click="backToPreview">&larr; Back to Preview</button>
              <button v-if="generatedImage && !isPreview" class="btn btn-secondary btn-sm" @click="upscaleImageX2" :disabled="upscalingX2">Upscale x2</button>
              <button class="btn btn-secondary btn-sm" @click="downloadImage">Download</button>
              <button class="btn btn-secondary btn-sm" @click="generateImage">Regenerate</button>
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
  </div>
</template>
