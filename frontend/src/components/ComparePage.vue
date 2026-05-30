<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { api } from '../api.js'
import { t } from '../i18n/index.js'
import TestPage from './TestPage.vue'

const tab = ref('prompt')
const uploadedImage = ref('')
const analyzing = ref(false)
const analyzedPrompt = ref('')
const error = ref('')

async function uploadImage() {
  try {
    const base64 = await api.readImageFile()
    if (base64) {
      uploadedImage.value = base64
      error.value = ''
    }
  } catch (e) {
    error.value = String(e)
  }
}

async function useLastImage() {
  try {
    const item = await api.getActiveSessionItem()
    if (item) {
      const b64 = await api.getSessionImage(item.id)
      if (b64) {
        uploadedImage.value = b64
        error.value = ''
      } else {
        error.value = t('fi.error_no_session_image')
      }
    } else {
      error.value = t('fi.error_no_active_item')
    }
  } catch (e) {
    error.value = String(e)
  }
}

async function pasteFromClipboard() {
  try {
    const base64 = await api.readClipboardImage()
    if (base64) {
      uploadedImage.value = base64
      error.value = ''
    }
  } catch (e) {
    error.value = String(e)
  }
}

function clearImage() {
  uploadedImage.value = ''
  analyzedPrompt.value = ''
  error.value = ''
}

function handlePaste(e) {
  const items = e.clipboardData?.items
  if (!items) return
  for (const item of items) {
    if (item.type.startsWith('image/')) {
      e.preventDefault()
      const file = item.getAsFile()
      if (!file) continue
      if (file.size > 16 * 1024 * 1024) {
        error.value = t('fi.error_image_too_large')
        return
      }
      const reader = new FileReader()
      reader.onload = () => {
        const base64 = reader.result.split(',')[1]
        uploadedImage.value = base64
        error.value = ''
      }
      reader.readAsDataURL(file)
      return
    }
  }
}

async function analyzeImage() {
  if (!uploadedImage.value) {
    error.value = t('compare.error_upload_first')
    return
  }
  analyzing.value = true
  error.value = ''
  try {
    const result = await api.analyzeImage(uploadedImage.value)
    analyzedPrompt.value = result || ''
  } catch (e) {
    error.value = t('compare.error_analysis', { error: String(e) })
  } finally {
    analyzing.value = false
  }
}

function onDrop(e) {
  e.preventDefault()
  const files = e.dataTransfer?.files
  if (files?.length) {
    for (const file of files) {
      if (file.type.startsWith('image/')) {
        const reader = new FileReader()
        reader.onload = () => {
          const base64 = reader.result.split(',')[1]
          uploadedImage.value = base64
          error.value = ''
        }
        reader.readAsDataURL(file)
        return
      }
    }
  }
}

onMounted(() => {
  window.addEventListener('paste', handlePaste)
})

onUnmounted(() => {
  window.removeEventListener('paste', handlePaste)
})
</script>

<template>
  <div @drop="onDrop" @dragover.prevent>
    <div class="page-header">
      <h1 class="page-title">{{ t('app.nav_compare') }}</h1>
    </div>

    <div class="tabs" style="margin-bottom: 16px;">
      <button class="tab" :class="{ active: tab === 'prompt' }" @click="tab = 'prompt'">{{ t('compare.tab_prompt') }}</button>
      <button class="tab" :class="{ active: tab === 'image' }" @click="tab = 'image'">{{ t('compare.tab_image') }}</button>
    </div>

    <div v-if="tab === 'image'" class="card" style="max-width: 800px; margin-bottom: 16px;">
      <div v-if="error" class="status status-error">{{ error }}</div>

      <div v-if="!uploadedImage" class="fi-drop-zone" @click="uploadImage" style="margin-bottom: 12px;">
        {{ t('fi.drop_image') }}
      </div>
      <div v-else style="margin-bottom: 12px;">
        <img :src="'data:image/png;base64,' + uploadedImage" style="max-width: 200px; max-height: 200px; border-radius: 6px; display: block; margin-bottom: 8px;" />
      </div>

      <div style="display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 12px;">
        <button class="btn btn-secondary btn-sm" @click="uploadImage">{{ t('fi.btn_change') }}</button>
        <button class="btn btn-secondary btn-sm" @click="useLastImage">{{ t('fi.btn_last_generated') }}</button>
        <button class="btn btn-secondary btn-sm" @click="pasteFromClipboard">{{ t('fi.btn_paste') }}</button>
        <button class="btn btn-secondary btn-sm" @click="clearImage">{{ t('fi.btn_clear') }}</button>
      </div>

      <button class="btn btn-primary" style="width: 100%; margin-bottom: 12px;" @click="analyzeImage" :disabled="analyzing || !uploadedImage">
        <span v-if="analyzing" style="display: inline-flex; align-items: center; gap: 6px;">
          <span class="spinner" style="width: 14px; height: 14px; border-width: 2px;"></span>
          {{ t('compare.analyzing') }}
        </span>
        <span v-else>{{ t('compare.btn_analyze') }}</span>
      </button>

      <div v-if="analyzedPrompt" class="form-group">
        <textarea class="form-textarea" v-model="analyzedPrompt" rows="4" :placeholder="t('compare.placeholder_analyzed')"></textarea>
      </div>
    </div>

    <TestPage
      :external-prompt="tab === 'image' ? analyzedPrompt : ''"
      :hide-prompt-input="tab === 'image'"
    />
  </div>
</template>
