<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { api } from '../api.js'

const uploadedImage = ref('')
const uploadedImageMime = ref('image/png')
const tags = ref('')
const presets = ref([])
const presetTypes = ref([])
const compoundPresets = ref([])
const selectedTypeId = ref(null)
const selectedPresetId = ref(null)
const selectedCompoundPresetId = ref(null)
const genMode = ref('preset')
const mode = ref('txt2img')
const denoisingStrength = ref(0.5)
const extraNegativePrompt = ref('')

const generatedImage = ref('')
const genInfo = ref(null)
const effectivePrompt = ref('')
const effectiveNegative = ref('')

const analyzing = ref(false)
const generatingImage = ref(false)
const generationStage = ref('')
const error = ref('')

const llmAvailable = ref(false)
const llmModel = ref('')
const sdAvailable = ref(false)
const sdModel = ref('')
const kidsModeActive = ref(false)
let statusInterval = null

const isDragOver = ref(false)

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
    const [p, t, c] = await Promise.all([api.listPresets(), api.listPresetTypes(), api.listCompoundPresets()])
    presets.value = p || []
    presetTypes.value = t || []
    compoundPresets.value = c || []
  } catch (e) {
    console.error(e)
  }
}

async function uploadImage() {
  try {
    const base64 = await api.readImageFile()
    if (base64) {
      uploadedImage.value = base64
      uploadedImageMime.value = 'image/png'
      tags.value = ''
      error.value = ''
    }
  } catch (e) {
    error.value = String(e)
  }
}

function onDrop(e) {
  isDragOver.value = false
  const file = e.dataTransfer?.files?.[0]
  if (!file) return
  if (!file.type.startsWith('image/')) {
    error.value = 'Please drop an image file'
    return
  }
  if (file.size > 16 * 1024 * 1024) {
    error.value = 'Image too large (max 16 MB)'
    return
  }
  const reader = new FileReader()
  reader.onload = () => {
    const dataUrl = reader.result
    const base64 = dataUrl.split(',')[1]
    const mime = dataUrl.split(':')[1]?.split(';')[0] || 'image/png'
    if (base64) {
      uploadedImage.value = base64
      uploadedImageMime.value = mime
      tags.value = ''
      error.value = ''
    }
  }
  reader.readAsDataURL(file)
}

function clearImage() {
  uploadedImage.value = ''
  uploadedImageMime.value = 'image/png'
  tags.value = ''
  generatedImage.value = ''
  genInfo.value = null
  effectivePrompt.value = ''
  effectiveNegative.value = ''
  error.value = ''
}

async function analyzeImage() {
  if (!uploadedImage.value) return
  analyzing.value = true
  error.value = ''
  try {
    const result = await api.analyzeImageForGen(uploadedImage.value)
    tags.value = result?.tags || ''
  } catch (e) {
    error.value = 'Analysis failed: ' + String(e)
  } finally {
    analyzing.value = false
  }
}

async function generate() {
  if (!uploadedImage.value) {
    error.value = 'Upload an image first'
    return
  }
  if (genMode.value === 'preset' && !selectedPresetId.value) {
    error.value = 'Select a preset first'
    return
  }
  if (genMode.value === 'compound' && !selectedCompoundPresetId.value) {
    error.value = 'Select a pipeline first'
    return
  }

  generatingImage.value = true
  generationStage.value = 'analyzing'
  generatedImage.value = ''
  genInfo.value = null
  effectivePrompt.value = ''
  effectiveNegative.value = ''
  error.value = ''

  try {
    generationStage.value = 'generating'
    const result = await api.generateFromImage({
      image_base64: uploadedImage.value,
      mode: mode.value,
      gen_mode: genMode.value,
      preset_id: selectedPresetId.value || 0,
      compound_preset_id: selectedCompoundPresetId.value || 0,
      denoising_strength: denoisingStrength.value,
      tags: tags.value,
      extra_negative_prompt: extraNegativePrompt.value,
    })
    if (!result || !result.image) {
      error.value = 'No image returned. Check preset settings.'
    } else {
      generatedImage.value = result.image
      genInfo.value = result.info
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

async function downloadImage() {
  if (!generatedImage.value) return
  try {
    const defaultName = 'sd-studio-from-image-' + Date.now() + '.png'
    await api.saveImage(generatedImage.value, defaultName)
  } catch (e) {
    error.value = 'Save failed: ' + String(e)
  }
}

onMounted(async () => {
  loadPresets()
  checkServices()
  statusInterval = setInterval(checkServices, 30000)
})

onUnmounted(() => {
  if (statusInterval) clearInterval(statusInterval)
})
</script>

<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Generate From Image</h1>
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
          <div
            class="drop-zone"
            :class="{ 'drop-zone-active': isDragOver, 'drop-zone-has-image': uploadedImage }"
            @dragover.prevent="isDragOver = true"
            @dragleave="isDragOver = false"
            @drop.prevent="onDrop"
            @click="!uploadedImage && uploadImage()"
          >
            <template v-if="!uploadedImage">
              <div style="font-size: 32px; color: var(--text-dim);">&#128444;</div>
              <p style="color: var(--text-dim); margin-top: 8px;">Drop image here or click to upload</p>
            </template>
            <template v-else>
              <img :src="'data:' + uploadedImageMime + ';base64,' + uploadedImage" alt="Source" style="max-height: 200px; border-radius: 6px;" />
            </template>
          </div>
          <div v-if="uploadedImage" style="display: flex; gap: 8px; margin-top: 8px;">
            <button class="btn btn-sm btn-secondary" @click="uploadImage">Change</button>
            <button class="btn btn-sm btn-secondary" @click="clearImage">Clear</button>
          </div>

          <div style="display: flex; gap: 8px; margin-top: 16px; margin-bottom: 12px;">
            <button class="btn btn-sm" :class="mode === 'txt2img' ? 'btn-primary' : 'btn-secondary'" @click="mode = 'txt2img'" :disabled="generatingImage">txt2img</button>
            <button class="btn btn-sm" :class="mode === 'img2img' ? 'btn-primary' : 'btn-secondary'" @click="mode = 'img2img'" :disabled="generatingImage">img2img</button>
          </div>

          <div v-if="mode === 'img2img'" class="form-group">
            <label class="form-label">Denoising Strength: {{ denoisingStrength.toFixed(2) }}</label>
            <input type="range" class="form-range" v-model.number="denoisingStrength" min="0.05" max="1.0" step="0.05" :disabled="generatingImage" />
            <div style="display: flex; justify-content: space-between; font-size: 11px; color: var(--text-dim);">
              <span>Keep original</span>
              <span>Full redraw</span>
            </div>
          </div>

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
                <option v-for="p in filteredPresets" :key="p.id" :value="p.id">{{ p.name }}</option>
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
            <label class="form-label">Extracted Tags</label>
            <div style="display: flex; gap: 8px; margin-bottom: 6px;">
              <button class="btn btn-sm btn-secondary" @click="analyzeImage" :disabled="analyzing || !uploadedImage">
                {{ analyzing ? 'Analyzing...' : 'Analyze Image' }}
              </button>
            </div>
            <textarea class="form-textarea" v-model="tags" rows="4" placeholder="Tags extracted from image will appear here. Click Analyze or generate directly." :disabled="generatingImage"></textarea>
          </div>

          <div class="form-group">
            <label class="form-label">Extra Negative</label>
            <textarea class="form-textarea" v-model="extraNegativePrompt" rows="2" placeholder="Additional negative prompt..." :disabled="generatingImage"></textarea>
          </div>

          <button class="btn btn-primary" style="width: 100%; justify-content: center; padding: 12px;" @click="generate" :disabled="generatingImage || !uploadedImage || (genMode === 'preset' ? !selectedPresetId : !selectedCompoundPresetId)">
            <span v-if="generatingImage" style="display: inline-flex; align-items: center; gap: 6px;">
              <span class="spinner" style="width: 14px; height: 14px; border-width: 2px;"></span>
              {{ generationStage === 'analyzing' ? 'Analyzing image...' : 'Generating image...' }}
            </span>
            <span v-else>Generate</span>
          </button>
        </div>
      </div>

      <div class="generate-section">
        <div class="generate-image-area">
          <div v-if="generatingImage" style="text-align: center;">
            <span class="spinner" style="width: 32px; height: 32px; border-width: 3px;"></span>
            <p style="margin-top: 12px; color: var(--text-dim);">{{ generationStage === 'analyzing' ? 'Analyzing image...' : 'Generating image...' }}</p>
          </div>
          <div v-else-if="generatedImage" style="width: 100%; padding: 12px;">
            <img :src="'data:image/png;base64,' + generatedImage" alt="Generated" style="border-radius: var(--radius-sm);" />
            <div style="display: flex; gap: 8px; margin-top: 12px; justify-content: center;">
              <button class="btn btn-secondary btn-sm" @click="downloadImage">Download</button>
              <button class="btn btn-secondary btn-sm" @click="generate">Regenerate</button>
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

<style scoped>
.drop-zone {
  border: 2px dashed var(--border);
  border-radius: var(--radius-sm);
  padding: 24px;
  text-align: center;
  cursor: pointer;
  transition: border-color 0.2s, background 0.2s;
  min-height: 120px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}
.drop-zone:hover {
  border-color: var(--accent);
  background: var(--surface-2);
}
.drop-zone-active {
  border-color: var(--accent);
  background: var(--surface-2);
}
.drop-zone-has-image {
  cursor: default;
  border-style: solid;
}
.form-range {
  width: 100%;
  accent-color: var(--accent);
}
</style>
