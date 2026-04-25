<script setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { api } from '../api.js'

const presets = ref([])
const selectedPresetId = ref(null)
const description = ref('')
const extraPrompt = ref('')
const extraNegativePrompt = ref('')
const generatedImage = ref('')
const extraPromptEl = ref(null)
const genInfo = ref(null)
const sourceImage = ref('')
const sourceGenInfo = ref(null)
const isPreview = ref(false)
const savedPreview = ref(null)
const upscaling = ref(false)
const upscalingX2 = ref(false)
const previewMode = ref(false)

const generatingPrompt = ref(false)
const generatingImage = ref(false)
const error = ref('')
const promptElapsed = ref(0)
let promptTimerInterval = null
let cancelPromptFlag = false

const savedDescriptions = ref([])
const showDescModal = ref(false)
const descPage = ref(1)
const PAGE_SIZE = 10

const savedPrompts = ref([])
const showPromptModal = ref(false)
const promptPage = ref(1)

const descTotalPages = computed(() => Math.max(1, Math.ceil(savedDescriptions.value.length / PAGE_SIZE)))
const descPaginated = computed(() => {
  const start = (descPage.value - 1) * PAGE_SIZE
  return savedDescriptions.value.slice(start, start + PAGE_SIZE)
})

const promptTotalPages = computed(() => Math.max(1, Math.ceil(savedPrompts.value.length / PAGE_SIZE)))
const promptPaginated = computed(() => {
  const start = (promptPage.value - 1) * PAGE_SIZE
  return savedPrompts.value.slice(start, start + PAGE_SIZE)
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

const llmAvailable = ref(false)
const llmModel = ref('')
const sdAvailable = ref(false)
const sdModel = ref('')
const kidsModeActive = ref(false)
let statusInterval = null

function autoResize() {
  const el = extraPromptEl.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = el.scrollHeight + 'px'
}

watch(extraPrompt, () => nextTick(autoResize))

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
    presets.value = await api.listPresets()
  } catch (e) {
    console.error(e)
  }
}

async function loadDescriptions() {
  try {
    savedDescriptions.value = await api.listDescriptions()
    if (descPage.value > descTotalPages.value) descPage.value = descTotalPages.value
  } catch (e) {
    console.error(e)
  }
}

async function loadPrompts() {
  try {
    savedPrompts.value = await api.listPrompts()
    if (promptPage.value > promptTotalPages.value) promptPage.value = promptTotalPages.value
  } catch (e) {
    console.error(e)
  }
}

async function saveDescription() {
  if (!description.value.trim()) return
  try {
    await api.createDescription(description.value)
    await loadDescriptions()
  } catch (e) {
    error.value = String(e)
  }
}

async function deleteDescription(id) {
  try {
    await api.deleteDescription(id)
    await loadDescriptions()
  } catch (e) {
    error.value = String(e)
  }
}

function useDescription(text) {
  description.value = text
  showDescModal.value = false
}

function openDescModal() {
  showDescModal.value = true
  descPage.value = 1
}

async function savePrompt() {
  if (!extraPrompt.value.trim()) return
  try {
    await api.createPrompt(extraPrompt.value)
    await loadPrompts()
  } catch (e) {
    error.value = String(e)
  }
}

async function deletePrompt(id) {
  try {
    await api.deletePrompt(id)
    await loadPrompts()
  } catch (e) {
    error.value = String(e)
  }
}

function usePrompt(text) {
  extraPrompt.value = text
  showPromptModal.value = false
}

function openPromptModal() {
  showPromptModal.value = true
  promptPage.value = 1
}

function cancelPromptGeneration() {
  cancelPromptFlag = true
  generatingPrompt.value = false
  if (promptTimerInterval) {
    clearInterval(promptTimerInterval)
    promptTimerInterval = null
  }
  promptElapsed.value = 0
}

async function generatePrompt() {
  if (!description.value.trim()) return
  generatingPrompt.value = true
  cancelPromptFlag = false
  promptElapsed.value = 0
  error.value = ''
  promptTimerInterval = setInterval(() => { promptElapsed.value++ }, 1000)
  try {
    const preset = presets.value.find(p => p.id === Number(selectedPresetId.value))
    const result = await api.generateSdPrompt(description.value, preset?.preset_type || '')
    if (cancelPromptFlag) return
    if (result && result.trim()) {
      extraPrompt.value = result
    } else {
      error.value = 'LLM returned empty response'
    }
  } catch (e) {
    if (cancelPromptFlag) return
    error.value = String(e)
  } finally {
    if (!cancelPromptFlag) {
      generatingPrompt.value = false
    }
    if (promptTimerInterval) {
      clearInterval(promptTimerInterval)
      promptTimerInterval = null
    }
    promptElapsed.value = 0
  }
}

function saveGenState() {
  api.updateSettings({
    gen_preset_id: String(selectedPresetId.value || ''),
    gen_description: description.value || '',
    gen_extra_prompt: extraPrompt.value || '',
    gen_extra_negative: extraNegativePrompt.value || '',
  }).catch(() => {})
}

async function generateImage() {
  if (!selectedPresetId.value) {
    error.value = 'Select a preset first'
    return
  }
  saveGenState()
  generatingImage.value = true
  error.value = ''
  generatedImage.value = ''
  genInfo.value = null
  isPreview.value = false
  savedPreview.value = null
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
    }
  } catch (e) {
    error.value = String(e)
  } finally {
    generatingImage.value = false
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
  loadDescriptions()
  loadPrompts()
  checkServices()
  statusInterval = setInterval(checkServices, 30000)
  try {
    const s = await api.getSettings()
    previewMode.value = s.preview_mode === 'true'
    if (s.gen_preset_id) selectedPresetId.value = Number(s.gen_preset_id)
    if (s.gen_description) description.value = s.gen_description
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
          <div class="form-group">
            <label class="form-label">Preset</label>
            <select class="form-select" v-model="selectedPresetId" :disabled="generatingPrompt">
              <option :value="null" disabled>Select preset...</option>
              <option v-for="p in presets" :key="p.id" :value="p.id">
                {{ p.name }} ({{ p.preset_type || 'general' }})
              </option>
            </select>
          </div>

          <div class="form-group">
            <div style="display: flex; justify-content: space-between; align-items: center;">
              <label class="form-label" style="margin-bottom: 0;">Description</label>
              <div style="display: flex; gap: 6px;">
                <button class="btn btn-secondary btn-sm" @click="saveDescription" :disabled="generatingPrompt || !description.trim()" title="Save description">
                  &#9776; Save
                </button>
                <button class="btn btn-secondary btn-sm" @click="openDescModal" :disabled="generatingPrompt" title="Load saved">
                  &#128194; Saved
                </button>
              </div>
            </div>

            <div style="display: flex; gap: 8px; margin-top: 8px;">
              <textarea class="form-textarea" v-model="description" rows="3" placeholder="A glowing magic sword..." :disabled="generatingPrompt"></textarea>
              <div style="display: flex; flex-direction: column; gap: 4px;">
                <button class="btn btn-secondary" @click="generatePrompt" :disabled="generatingPrompt || !description.trim()" style="white-space: nowrap;">
                  <span v-if="generatingPrompt" style="display: inline-flex; align-items: center; gap: 6px;">
                    <span class="spinner" style="width: 14px; height: 14px; border-width: 2px;"></span>
                    {{ promptElapsed }}s
                  </span>
                  <span v-else>AI Prompt</span>
                </button>
                <button v-if="generatingPrompt" class="btn btn-danger btn-sm" @click="cancelPromptGeneration" style="white-space: nowrap;">Cancel</button>
              </div>
            </div>
          </div>

          <div class="form-group">
            <div style="display: flex; justify-content: space-between; align-items: center;">
              <label class="form-label" style="margin-bottom: 0;">Extra Prompt</label>
              <div style="display: flex; gap: 6px;">
                <button class="btn btn-secondary btn-sm" @click="savePrompt" :disabled="generatingPrompt || !extraPrompt.trim()" title="Save prompt">
                  &#9776; Save
                </button>
                <button class="btn btn-secondary btn-sm" @click="openPromptModal" :disabled="generatingPrompt" title="Load saved">
                  &#128194; Saved
                </button>
              </div>
            </div>

            <textarea ref="extraPromptEl" class="form-textarea" v-model="extraPrompt" rows="2" placeholder="Additional tags..." style="margin-top: 8px; resize: none; overflow: hidden;" :disabled="generatingPrompt" @input="autoResize"></textarea>
          </div>

          <div class="form-group">
            <label class="form-label">Extra Negative</label>
            <input class="form-input" v-model="extraNegativePrompt" placeholder="Additional negative tags..." :disabled="generatingPrompt" />
          </div>

          <button class="btn btn-primary" style="width: 100%; justify-content: center; padding: 12px;" @click="generateImage" :disabled="generatingImage || generatingPrompt || !selectedPresetId">
            {{ generatingImage ? 'Generating...' : 'Generate Image' }}
          </button>
        </div>
      </div>

      <div class="generate-section">
        <div class="generate-image-area">
          <div v-if="generatingImage || upscaling || upscalingX2" style="text-align: center;">
            <span class="spinner" style="width: 32px; height: 32px; border-width: 3px;"></span>
            <p style="margin-top: 12px; color: var(--text-dim);">{{ upscalingX2 ? 'Upscaling x2...' : upscaling ? 'Upscaling to full resolution...' : 'Generating image...' }}</p>
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
      </div>
    </div>

    <!-- Saved Descriptions Modal -->
    <div v-if="showDescModal" class="modal-overlay" @click.self="showDescModal = false">
      <div class="modal">
        <div class="modal-header">
          <h2 class="modal-title">Saved Descriptions</h2>
          <button class="modal-close" @click="showDescModal = false">&times;</button>
        </div>
        <div v-if="savedDescriptions.length === 0" style="color: var(--text-dim); text-align: center; padding: 24px;">
          No saved descriptions yet
        </div>
        <div v-else class="saved-modal-list">
          <div v-for="s in descPaginated" :key="s.id" class="saved-modal-item">
            <div class="saved-modal-text" @click="useDescription(s.text)">{{ s.text }}</div>
            <button class="btn btn-danger btn-sm" @click="deleteDescription(s.id)" title="Delete">&times;</button>
          </div>
        </div>
        <div v-if="descTotalPages > 1" class="pager">
          <button class="btn btn-secondary btn-sm" :disabled="descPage <= 1" @click="descPage--">&laquo;</button>
          <span class="pager-info">{{ descPage }} / {{ descTotalPages }}</span>
          <button class="btn btn-secondary btn-sm" :disabled="descPage >= descTotalPages" @click="descPage++">&raquo;</button>
        </div>
      </div>
    </div>

    <!-- Saved Prompts Modal -->
    <div v-if="showPromptModal" class="modal-overlay" @click.self="showPromptModal = false">
      <div class="modal">
        <div class="modal-header">
          <h2 class="modal-title">Saved Prompts</h2>
          <button class="modal-close" @click="showPromptModal = false">&times;</button>
        </div>
        <div v-if="savedPrompts.length === 0" style="color: var(--text-dim); text-align: center; padding: 24px;">
          No saved prompts yet
        </div>
        <div v-else class="saved-modal-list">
          <div v-for="s in promptPaginated" :key="s.id" class="saved-modal-item">
            <div class="saved-modal-text" @click="usePrompt(s.text)">{{ s.text }}</div>
            <button class="btn btn-danger btn-sm" @click="deletePrompt(s.id)" title="Delete">&times;</button>
          </div>
        </div>
        <div v-if="promptTotalPages > 1" class="pager">
          <button class="btn btn-secondary btn-sm" :disabled="promptPage <= 1" @click="promptPage--">&laquo;</button>
          <span class="pager-info">{{ promptPage }} / {{ promptTotalPages }}</span>
          <button class="btn btn-secondary btn-sm" :disabled="promptPage >= promptTotalPages" @click="promptPage++">&raquo;</button>
        </div>
      </div>
    </div>
  </div>
</template>
