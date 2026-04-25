<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { api } from '../api.js'

const imageBase64 = ref('')
const imagePreview = ref('')
const result = ref('')
const analyzing = ref(false)
const error = ref('')
const elapsed = ref(0)
let timerInterval = null

const llmAvailable = ref(false)
const llmModel = ref('')
let statusInterval = null

async function checkServices() {
  try {
    const status = await api.checkServices()
    llmAvailable.value = status.llm?.available || false
    llmModel.value = status.llm?.model || ''
  } catch {}
}

function handlePaste(e) {
  const items = e.clipboardData?.items
  if (!items) return
  for (const item of items) {
    if (item.type.startsWith('image/')) {
      e.preventDefault()
      const file = item.getAsFile()
      if (!file) continue
      const reader = new FileReader()
      reader.onload = () => {
        const base64 = reader.result.split(',')[1]
        imageBase64.value = base64
        imagePreview.value = reader.result
      }
      reader.readAsDataURL(file)
      return
    }
  }
}

async function openFile() {
  try {
    const b64 = await api.readImageFile()
    if (b64) {
      imageBase64.value = b64
      imagePreview.value = 'data:image/png;base64,' + b64
    }
  } catch (e) {
    error.value = String(e)
  }
}

async function useLastImage() {
  try {
    const last = await api.getLastImage()
    if (last && last.image) {
      imageBase64.value = last.image
      imagePreview.value = 'data:image/png;base64,' + last.image
    } else {
      error.value = 'No last generated image found'
    }
  } catch (e) {
    error.value = String(e)
  }
}

async function analyze() {
  if (!imageBase64.value) return
  analyzing.value = true
  error.value = ''
  result.value = ''
  elapsed.value = 0
  timerInterval = setInterval(() => { elapsed.value++ }, 1000)
  try {
    const tags = await api.analyzeImage(imageBase64.value)
    if (tags && tags.trim()) {
      result.value = tags
    } else {
      error.value = 'Vision model returned empty response'
    }
  } catch (e) {
    error.value = String(e)
  } finally {
    analyzing.value = false
    if (timerInterval) {
      clearInterval(timerInterval)
      timerInterval = null
    }
  }
}

function copyResult() {
  if (!result.value) return
  navigator.clipboard.writeText(result.value).catch(() => {})
}

function clearImage() {
  imageBase64.value = ''
  imagePreview.value = ''
}

onMounted(() => {
  checkServices()
  statusInterval = setInterval(checkServices, 30000)
  document.addEventListener('paste', handlePaste)
})

onUnmounted(() => {
  if (statusInterval) clearInterval(statusInterval)
  document.removeEventListener('paste', handlePaste)
})
</script>

<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Analyze</h1>
    </div>

    <div class="service-status">
      <div class="status-badge" :class="llmAvailable ? 'status-ok' : 'status-down'">
        &#9679; Vision{{ llmModel ? ': ' + llmModel : '' }}
      </div>
    </div>

    <div v-if="error" class="status status-error">{{ error }}</div>

    <div class="generate-layout">
      <div class="generate-section">
        <div class="card">
          <div class="form-label" style="margin-bottom: 12px;">Image Source</div>

          <div class="analyze-btn-row">
            <button class="btn btn-secondary" @click="openFile">Open File</button>
            <button class="btn btn-secondary" @click="useLastImage">Last Generated</button>
            <button v-if="imagePreview" class="btn btn-secondary" @click="clearImage">Clear</button>
          </div>

          <div style="margin-top: 12px; color: var(--text-dim); font-size: 12px;">
            You can also paste an image from clipboard (Ctrl+V)
          </div>

          <div class="analyze-image-preview" v-if="imagePreview" style="margin-top: 16px;">
            <img :src="imagePreview" alt="Source" />
          </div>

          <button
            class="btn btn-primary"
            style="width: 100%; justify-content: center; padding: 12px; margin-top: 16px;"
            @click="analyze"
            :disabled="analyzing || !imageBase64"
          >
            <span v-if="analyzing" style="display: inline-flex; align-items: center; gap: 6px;">
              <span class="spinner" style="width: 14px; height: 14px; border-width: 2px;"></span>
              {{ elapsed }}s
            </span>
            <span v-else>Analyze Image</span>
          </button>
        </div>
      </div>

      <div class="generate-section">
        <div class="card">
          <div class="form-label" style="margin-bottom: 12px;">SD Tags</div>

          <textarea
            class="form-textarea"
            v-model="result"
            rows="16"
            placeholder="Analyzed tags will appear here..."
            readonly
            style="resize: vertical; min-height: 200px;"
          ></textarea>

          <div class="analyze-btn-row" style="margin-top: 12px;">
            <button class="btn btn-secondary" @click="copyResult" :disabled="!result">Copy</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.analyze-btn-row {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.analyze-image-preview {
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 8px;
  text-align: center;
}

.analyze-image-preview img {
  max-width: 100%;
  max-height: 400px;
  border-radius: var(--radius-sm);
  object-fit: contain;
}
</style>
