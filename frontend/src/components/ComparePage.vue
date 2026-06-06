<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { api } from '../api.js'
import { t } from '../i18n/index.js'
import { MAX_IMAGE_SIZE } from '../constants.js'
import TestPage from './TestPage.vue'

const props = defineProps({
  active: { type: Boolean, default: false }
})

const tab = ref('prompt')
const uploadedImage = ref('')
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
      if (file.size > MAX_IMAGE_SIZE) {
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
    </div>

    <TestPage
      :init-image="tab === 'image' ? uploadedImage : ''"
      :hide-prompt-input="tab === 'image'"
      :active="active"
    />
  </div>
</template>
