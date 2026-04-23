<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const presets = ref([])
const selectedPresetId = ref(null)
const description = ref('')
const extraPrompt = ref('')
const extraNegativePrompt = ref('')
const generatedImage = ref('')
const genInfo = ref(null)

const generatingPrompt = ref(false)
const generatingImage = ref(false)
const error = ref('')

const savedDescriptions = ref([])
const showSavedDescriptions = ref(false)
const savedPrompts = ref([])
const showSavedPrompts = ref(false)

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
  } catch (e) {
    console.error(e)
  }
}

async function loadPrompts() {
  try {
    savedPrompts.value = await api.listPrompts()
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
    error.value = e.message
  }
}

async function deleteDescription(id) {
  try {
    await api.deleteDescription(id)
    await loadDescriptions()
  } catch (e) {
    error.value = e.message
  }
}

function useDescription(text) {
  description.value = text
  showSavedDescriptions.value = false
}

async function savePrompt() {
  if (!extraPrompt.value.trim()) return
  try {
    await api.createPrompt(extraPrompt.value)
    await loadPrompts()
  } catch (e) {
    error.value = e.message
  }
}

async function deletePrompt(id) {
  try {
    await api.deletePrompt(id)
    await loadPrompts()
  } catch (e) {
    error.value = e.message
  }
}

function usePrompt(text) {
  extraPrompt.value = text
  showSavedPrompts.value = false
}

async function generatePrompt() {
  if (!description.value.trim()) return
  generatingPrompt.value = true
  error.value = ''
  try {
    const preset = presets.value.find(p => p.id === Number(selectedPresetId.value))
    const result = await api.generateSdPrompt(description.value, preset?.preset_type || '')
    if (result && result.trim()) {
      extraPrompt.value = result
    } else {
      error.value = 'LLM returned empty response'
    }
  } catch (e) {
    error.value = e.message
  } finally {
    generatingPrompt.value = false
  }
}

async function generateImage() {
  if (!selectedPresetId.value) {
    error.value = 'Select a preset first'
    return
  }
  generatingImage.value = true
  error.value = ''
  generatedImage.value = ''
  genInfo.value = null
  try {
    const result = await api.generateImage(selectedPresetId.value, extraPrompt.value, extraNegativePrompt.value)
    generatedImage.value = result.image
    genInfo.value = result.info
  } catch (e) {
    error.value = e.message
  } finally {
    generatingImage.value = false
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
    error.value = 'Save failed: ' + e.message
  }
}

onMounted(() => {
  loadPresets()
  loadDescriptions()
  loadPrompts()
})
</script>

<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Generate</h1>
      <button class="btn btn-primary" @click="loadPresets">&#8635; Refresh</button>
    </div>

    <div v-if="error" class="status status-error">{{ error }}</div>

    <div class="generate-layout">
      <div class="generate-section">
        <div class="card">
          <div class="form-group">
            <label class="form-label">Preset</label>
            <select class="form-select" v-model="selectedPresetId">
              <option :value="null" disabled>Select preset...</option>
              <option v-for="p in presets" :key="p.id" :value="p.id">
                {{ p.name }} ({{ p.preset_type || 'general' }})
              </option>
            </select>
          </div>

          <div class="form-group">
            <div style="display: flex; justify-content: space-between; align-items: center;">
              <label class="form-label" style="margin-bottom: 0;">Describe in Russian</label>
              <div style="display: flex; gap: 6px;">
                <button class="btn btn-secondary btn-sm" @click="saveDescription" :disabled="!description.trim()" title="Save description">
                  &#9776; Save
                </button>
                <button class="btn btn-secondary btn-sm" @click="showSavedDescriptions = !showSavedDescriptions" title="Load saved">
                  &#128194; Saved
                </button>
              </div>
            </div>

            <div v-if="showSavedDescriptions" class="saved-list">
              <div v-if="savedDescriptions.length === 0" style="color: var(--text-dim); padding: 8px; font-size: 12px;">
                No saved descriptions yet
              </div>
              <div v-for="s in savedDescriptions" :key="s.id" class="saved-item">
                <span class="saved-item-text" @click="useDescription(s.text)">{{ s.text }}</span>
                <button class="btn btn-danger btn-sm" @click="deleteDescription(s.id)" title="Delete">&times;</button>
              </div>
            </div>

            <div style="display: flex; gap: 8px; margin-top: 8px;">
              <textarea class="form-textarea" v-model="description" rows="3" placeholder="Магический меч с синим свечением..."></textarea>
              <button class="btn btn-secondary" @click="generatePrompt" :disabled="generatingPrompt || !description.trim()" style="white-space: nowrap;">
                {{ generatingPrompt ? '...' : 'AI Prompt' }}
              </button>
            </div>
          </div>

          <div class="form-group">
            <div style="display: flex; justify-content: space-between; align-items: center;">
              <label class="form-label" style="margin-bottom: 0;">Extra Prompt</label>
              <div style="display: flex; gap: 6px;">
                <button class="btn btn-secondary btn-sm" @click="savePrompt" :disabled="!extraPrompt.trim()" title="Save prompt">
                  &#9776; Save
                </button>
                <button class="btn btn-secondary btn-sm" @click="showSavedPrompts = !showSavedPrompts" title="Load saved">
                  &#128194; Saved
                </button>
              </div>
            </div>

            <div v-if="showSavedPrompts" class="saved-list">
              <div v-if="savedPrompts.length === 0" style="color: var(--text-dim); padding: 8px; font-size: 12px;">
                No saved prompts yet
              </div>
              <div v-for="s in savedPrompts" :key="s.id" class="saved-item">
                <span class="saved-item-text" @click="usePrompt(s.text)">{{ s.text }}</span>
                <button class="btn btn-danger btn-sm" @click="deletePrompt(s.id)" title="Delete">&times;</button>
              </div>
            </div>

            <textarea class="form-textarea" v-model="extraPrompt" rows="2" placeholder="Additional tags..." style="margin-top: 8px;"></textarea>
          </div>

          <div class="form-group">
            <label class="form-label">Extra Negative</label>
            <input class="form-input" v-model="extraNegativePrompt" placeholder="Additional negative tags..." />
          </div>

          <button class="btn btn-primary" style="width: 100%; justify-content: center; padding: 12px;" @click="generateImage" :disabled="generatingImage || !selectedPresetId">
            {{ generatingImage ? 'Generating...' : 'Generate Image' }}
          </button>
        </div>
      </div>

      <div class="generate-section">
        <div class="generate-image-area">
          <div v-if="generatingImage" style="text-align: center;">
            <span class="spinner" style="width: 32px; height: 32px; border-width: 3px;"></span>
            <p style="margin-top: 12px; color: var(--text-dim);">Generating image...</p>
          </div>
          <div v-else-if="generatedImage" style="width: 100%; padding: 12px;">
            <img :src="'data:image/png;base64,' + generatedImage" alt="Generated" style="border-radius: var(--radius-sm);" />
            <div style="display: flex; gap: 8px; margin-top: 12px; justify-content: center;">
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
          <pre>{{ JSON.stringify(genInfo, null, 2) }}</pre>
        </details>
      </div>
    </div>
  </div>
</template>
