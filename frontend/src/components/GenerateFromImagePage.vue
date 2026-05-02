<script setup>
import { ref, computed, nextTick, watch, onMounted, onUnmounted, inject } from 'vue'
import { api } from '../api.js'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'
import ImageViewer from './ImageViewer.vue'

const props = defineProps({
  droppedImage: { type: String, default: null }
})
const emit = defineEmits(['clear-dropped', 'transfer-tags'])

watch(() => props.droppedImage, (val) => {
  if (val) {
    uploadedImage.value = val
    uploadedImageMime.value = 'image/png'
    tags.value = ''
    recommendation.value = null
    error.value = ''
    clearMask()
    mode.value = 'img2img'
    emit('clear-dropped')
  }
})

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
const mode = ref('img2img')
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

const analyzeMode = ref('quick')
const chainStep = ref(0)
const chainTotal = ref(0)
const analyzeElapsed = ref(0)
let analyzeTimer = null

const recommending = ref(false)
const recommendation = ref(null)

const kidsModeActive = ref(false)
const showViewer = ref(false)

const shared = inject('sharedGenState', null)

const isDragOver = ref(false)

const removeStage = ref('')
const brushSize = ref(30)
const maskBlur = ref(4)
const maskPadding = ref(8)
const maskFeather = ref(8)
const inpaintFill = ref(1)
const inpaintFullRes = ref(true)
const invertMask = ref(false)
const isDrawing = ref(false)
const maskCanvasRef = ref(null)
const maskHistory = ref([])
const imgEl = ref(null)

const fullscreenMask = ref(false)
const fsCanvasRef = ref(null)
const fsDrawing = ref(false)
const fsHistory = ref([])

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

const hasMask = computed(() => maskHistory.value.length > 0)

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

async function uploadImage() {
  try {
    const base64 = await api.readImageFile()
    if (base64) {
      uploadedImage.value = base64
      uploadedImageMime.value = 'image/png'
      tags.value = ''
      recommendation.value = null
      error.value = ''
      clearMask()
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
        uploadedImageMime.value = 'image/png'
        tags.value = ''
        recommendation.value = null
        error.value = ''
        clearMask()
      } else {
        error.value = 'No image found for active session item'
      }
    } else {
      error.value = 'No active session item found'
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
      uploadedImageMime.value = 'image/png'
      tags.value = ''
      recommendation.value = null
      error.value = ''
      clearMask()
    }
  } catch (e) {
    error.value = String(e)
  }
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
        error.value = 'Image too large (max 16 MB)'
        return
      }
      const reader = new FileReader()
      reader.onload = () => {
        const base64 = reader.result.split(',')[1]
        const mime = reader.result.split(':')[1]?.split(';')[0] || 'image/png'
        if (base64) {
          uploadedImage.value = base64
          uploadedImageMime.value = mime
          tags.value = ''
          recommendation.value = null
          error.value = ''
          clearMask()
        }
      }
      reader.readAsDataURL(file)
      return
    }
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
      recommendation.value = null
      error.value = ''
      clearMask()
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
  recommendation.value = null
  error.value = ''
  clearMask()
}

async function analyzeImage() {
  if (!uploadedImage.value) return
  analyzing.value = true
  error.value = ''
  chainStep.value = 0
  chainTotal.value = 0
  analyzeElapsed.value = 0

  if (analyzeMode.value === 'deep') {
    analyzeTimer = setInterval(() => { analyzeElapsed.value++ }, 1000)
  }

  try {
    const result = await api.analyzeImage(uploadedImage.value)
    tags.value = result || ''
    if (analyzeMode.value === 'deep' && tags.value.trim()) {
      recommendPreset(tags.value)
    }
  } catch (e) {
    error.value = 'Analysis failed: ' + String(e)
  } finally {
    analyzing.value = false
    if (analyzeTimer) {
      clearInterval(analyzeTimer)
      analyzeTimer = null
    }
  }
}

async function recommendPreset(tagsText) {
  recommending.value = true
  recommendation.value = null
  try {
    const rec = await api.recommendPreset(tagsText)
    if (rec) recommendation.value = rec
  } catch (e) {
    console.error('Recommend preset failed:', e)
  } finally {
    recommending.value = false
  }
}

function initMaskCanvas() {
  const canvas = maskCanvasRef.value
  if (!canvas) return
  const img = imgEl.value
  if (!img) return

  canvas.width = img.naturalWidth
  canvas.height = img.naturalHeight

  const container = canvas.parentElement
  canvas.style.width = img.clientWidth + 'px'
  canvas.style.height = img.clientHeight + 'px'

  const ctx = canvas.getContext('2d')
  ctx.clearRect(0, 0, canvas.width, canvas.height)
  maskHistory.value = []
}

function getCanvasCoords(e) {
  const canvas = maskCanvasRef.value
  if (!canvas) return { x: 0, y: 0 }
  const rect = canvas.getBoundingClientRect()
  const scaleX = canvas.width / rect.width
  const scaleY = canvas.height / rect.height
  const clientX = e.touches ? e.touches[0].clientX : e.clientX
  const clientY = e.touches ? e.touches[0].clientY : e.clientY
  return {
    x: (clientX - rect.left) * scaleX,
    y: (clientY - rect.top) * scaleY,
  }
}

function saveMaskState() {
  const canvas = maskCanvasRef.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  maskHistory.value.push(ctx.getImageData(0, 0, canvas.width, canvas.height))
  if (maskHistory.value.length > 30) {
    maskHistory.value.shift()
  }
}

function startDraw(e) {
  e.preventDefault()
  isDrawing.value = true
  saveMaskState()
  draw(e)
}

function draw(e) {
  if (!isDrawing.value) return
  e.preventDefault()
  const canvas = maskCanvasRef.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  const { x, y } = getCanvasCoords(e)
  const scaledBrush = brushSize.value * (canvas.width / canvas.getBoundingClientRect().width)

  ctx.globalCompositeOperation = 'source-over'
  ctx.fillStyle = 'rgba(255, 255, 255, 0.6)'
  ctx.beginPath()
  ctx.arc(x, y, scaledBrush / 2, 0, Math.PI * 2)
  ctx.fill()
}

function stopDraw() {
  isDrawing.value = false
}

function clearMask() {
  const canvas = maskCanvasRef.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  ctx.clearRect(0, 0, canvas.width, canvas.height)
  maskHistory.value = []
}

function undoMask() {
  if (maskHistory.value.length === 0) return
  const canvas = maskCanvasRef.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  const prev = maskHistory.value.pop()
  ctx.putImageData(prev, 0, 0)
}

function getMaskBase64() {
  const canvas = maskCanvasRef.value
  if (!canvas) return ''
  const w = canvas.width
  const h = canvas.height
  const ctx = canvas.getContext('2d')
  const src = ctx.getImageData(0, 0, w, h)

  const maskCanvas = document.createElement('canvas')
  maskCanvas.width = w
  maskCanvas.height = h
  const maskCtx = maskCanvas.getContext('2d')
  const maskData = maskCtx.createImageData(w, h)

  for (let i = 0; i < src.data.length; i += 4) {
    const alpha = src.data[i + 3]
    const isMasked = alpha > 10
    const fill = invertMask.value ? !isMasked : isMasked
    if (fill) {
      maskData.data[i] = 255
      maskData.data[i + 1] = 255
      maskData.data[i + 2] = 255
      maskData.data[i + 3] = 255
    }
  }
  maskCtx.putImageData(maskData, 0, 0)

  if (maskPadding.value > 0) {
    const dilated = document.createElement('canvas')
    dilated.width = w
    dilated.height = h
    const dCtx = dilated.getContext('2d')
    dCtx.filter = `blur(${maskPadding.value}px)`
    dCtx.drawImage(maskCanvas, 0, 0)
    dCtx.filter = 'none'

    const dData = dCtx.getImageData(0, 0, w, h)
    for (let i = 0; i < dData.data.length; i += 4) {
      if (dData.data[i + 3] > 0) {
        dData.data[i] = 255
        dData.data[i + 1] = 255
        dData.data[i + 2] = 255
        dData.data[i + 3] = 255
      }
    }
    dCtx.putImageData(dData, 0, 0)

    maskCtx.clearRect(0, 0, w, h)
    maskCtx.drawImage(dilated, 0, 0)
  }

  if (maskFeather.value > 0) {
    const feathered = document.createElement('canvas')
    feathered.width = w
    feathered.height = h
    const fCtx = feathered.getContext('2d')
    fCtx.filter = `blur(${maskFeather.value}px)`
    fCtx.drawImage(maskCanvas, 0, 0)

    maskCtx.clearRect(0, 0, w, h)
    maskCtx.drawImage(feathered, 0, 0)
  }

  const dataUrl = maskCanvas.toDataURL('image/png')
  return dataUrl.split(',')[1] || ''
}

watch(mode, (newMode) => {
  if ((newMode === 'inpaint' || newMode === 'remove') && uploadedImage.value) {
    nextTick(() => {
      const img = imgEl.value
      if (img && img.complete) {
        initMaskCanvas()
      } else if (img) {
        img.onload = () => initMaskCanvas()
      }
    })
  }
  if (newMode === 'remove') {
    denoisingStrength.value = 0.6
    maskBlur.value = 4
    inpaintFill.value = 0
    inpaintFullRes.value = true
  }
})

watch(uploadedImage, () => {
  if (mode.value === 'inpaint' || mode.value === 'remove') {
    nextTick(() => {
      const img = imgEl.value
      if (img && img.complete) {
        initMaskCanvas()
      } else if (img) {
        img.onload = () => initMaskCanvas()
      }
    })
  }
})

async function generate() {
  if (!uploadedImage.value) {
    error.value = 'Upload an image first'
    return
  }
  if (genMode.value === 'preset' && !selectedPresetId.value && mode.value !== 'remove') {
    error.value = 'Select a preset first'
    return
  }
  if (genMode.value === 'compound' && !selectedCompoundPresetId.value && mode.value !== 'remove') {
    error.value = 'Select a pipeline first'
    return
  }
  if ((mode.value === 'inpaint' || mode.value === 'remove') && !hasMask.value) {
    error.value = 'Draw a mask on the image first'
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
    const params = {
      image_base64: uploadedImage.value,
      mode: mode.value === 'remove' ? 'inpaint' : mode.value,
      gen_mode: genMode.value,
      preset_id: selectedPresetId.value || 0,
      compound_preset_id: selectedCompoundPresetId.value || 0,
      denoising_strength: denoisingStrength.value,
      tags: mode.value === 'remove' ? '' : tags.value,
      extra_negative_prompt: extraNegativePrompt.value,
      remove_object: mode.value === 'remove',
    }
    if (mode.value === 'inpaint' || mode.value === 'remove') {
      params.mask_base64 = getMaskBase64()
      params.mask_blur = maskBlur.value
      params.inpaint_fill = inpaintFill.value
      params.inpaint_full_res = inpaintFullRes.value
    }
    const result = await api.generateFromImage(params)
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
    removeStage.value = ''
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

function transferToGenerate() {
  if (shared) {
    shared.description = tags.value
  }
  emit('transfer-tags')
}

function applyRecommendation() {
  if (!recommendation.value) return
  if (recommendation.value.preset_id) {
    selectedPresetId.value = recommendation.value.preset_id
    genMode.value = 'preset'
  }
  if (recommendation.value.extra_prompt) {
    const current = tags.value.trim()
    tags.value = current ? current + ', ' + recommendation.value.extra_prompt : recommendation.value.extra_prompt
  }
}

onMounted(async () => {
  loadPresets()
  loadKidsMode()
  document.addEventListener('paste', handlePaste)
  document.addEventListener('keydown', onKeydown)
  EventsOn("analyze:step", (step, total) => {
    chainStep.value = step
    chainTotal.value = total
  })
  EventsOn("remove:stage", (stage) => {
    removeStage.value = stage
  })
  EventsOn("session:added", () => {
    // Don't auto-load — user picks when to load via Last Generated
  })
  try {
    const s = await api.getSettings()
    if (s.fi_mode) mode.value = s.fi_mode
    if (s.fi_preset_id) selectedPresetId.value = Number(s.fi_preset_id)
    if (s.fi_compound_preset_id) selectedCompoundPresetId.value = Number(s.fi_compound_preset_id)
    if (s.fi_gen_mode) genMode.value = s.fi_gen_mode
    if (s.fi_denoising) denoisingStrength.value = Number(s.fi_denoising)
    if (s.fi_extra_negative) extraNegativePrompt.value = s.fi_extra_negative
    if (s.fi_analyze_mode) analyzeMode.value = s.fi_analyze_mode
    if (s.fi_mask_padding) maskPadding.value = Number(s.fi_mask_padding)
    if (s.fi_mask_feather) maskFeather.value = Number(s.fi_mask_feather)
  } catch {}
  if (shared) {
    if (shared.selectedPresetId) selectedPresetId.value = shared.selectedPresetId
    if (shared.selectedCompoundPresetId) selectedCompoundPresetId.value = shared.selectedCompoundPresetId
    if (shared.genMode) genMode.value = shared.genMode
  }
  if (!props.droppedImage && !uploadedImage.value) {
    await useLastImage()
  }
})

onUnmounted(() => {
  document.removeEventListener('paste', handlePaste)
  document.removeEventListener('keydown', onKeydown)
  EventsOff("analyze:step")
  EventsOff("remove:stage")
  EventsOff("session:added")
  saveFIState()
  if (shared) {
    shared.selectedPresetId = selectedPresetId.value
    shared.selectedCompoundPresetId = selectedCompoundPresetId.value
    shared.genMode = genMode.value
  }
})

function saveFIState() {
  api.updateSettings({
    fi_mode: mode.value,
    fi_preset_id: String(selectedPresetId.value || ''),
    fi_compound_preset_id: String(selectedCompoundPresetId.value || ''),
    fi_gen_mode: genMode.value,
    fi_denoising: String(denoisingStrength.value || ''),
    fi_extra_negative: extraNegativePrompt.value || '',
    fi_analyze_mode: analyzeMode.value || '',
    fi_mask_padding: String(maskPadding.value || ''),
    fi_mask_feather: String(maskFeather.value || ''),
  }).catch(() => {})
}

function copyPrompt() {
  const parts = []
  if (effectivePrompt.value) parts.push(effectivePrompt.value)
  if (effectiveNegative.value) parts.push('Negative: ' + effectiveNegative.value)
  if (parts.length) navigator.clipboard.writeText(parts.join('\n'))
}

function openFullscreenMask() {
  fullscreenMask.value = true
  nextTick(() => {
    const canvas = fsCanvasRef.value
    if (!canvas) return
    canvas.width = maskCanvasRef.value?.width || imgEl.value?.naturalWidth || 512
    canvas.height = maskCanvasRef.value?.height || imgEl.value?.naturalHeight || 512
    const ctx = canvas.getContext('2d')
    ctx.clearRect(0, 0, canvas.width, canvas.height)
    if (maskCanvasRef.value) {
      ctx.drawImage(maskCanvasRef.value, 0, 0)
    }
  })
}

function closeFullscreenMask() {
  if (fsCanvasRef.value && maskCanvasRef.value) {
    const ctx = maskCanvasRef.value.getContext('2d')
    ctx.clearRect(0, 0, maskCanvasRef.value.width, maskCanvasRef.value.height)
    ctx.drawImage(fsCanvasRef.value, 0, 0)
    if (maskHistory.value.length === 0) {
      maskHistory.value.push(ctx.getImageData(0, 0, maskCanvasRef.value.width, maskCanvasRef.value.height))
    }
  }
  fullscreenMask.value = false
}

function getFsCoords(e) {
  const canvas = fsCanvasRef.value
  if (!canvas) return { x: 0, y: 0 }
  const rect = canvas.getBoundingClientRect()
  const scaleX = canvas.width / rect.width
  const scaleY = canvas.height / rect.height
  const clientX = e.touches ? e.touches[0].clientX : e.clientX
  const clientY = e.touches ? e.touches[0].clientY : e.clientY
  return { x: (clientX - rect.left) * scaleX, y: (clientY - rect.top) * scaleY }
}

function fsStartDraw(e) {
  e.preventDefault()
  const canvas = fsCanvasRef.value
  if (canvas) {
    const ctx = canvas.getContext('2d')
    fsHistory.value.push(ctx.getImageData(0, 0, canvas.width, canvas.height))
    if (fsHistory.value.length > 30) fsHistory.value.shift()
  }
  fsDrawing.value = true
  fsDraw(e)
}

function fsDraw(e) {
  if (!fsDrawing.value) return
  e.preventDefault()
  const canvas = fsCanvasRef.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  const { x, y } = getFsCoords(e)
  const scaledBrush = brushSize.value * (canvas.width / canvas.getBoundingClientRect().width)
  ctx.globalCompositeOperation = 'source-over'
  ctx.fillStyle = 'rgba(255, 255, 255, 0.6)'
  ctx.beginPath()
  ctx.arc(x, y, scaledBrush / 2, 0, Math.PI * 2)
  ctx.fill()
}

function fsStopDraw() { fsDrawing.value = false }

function fsClearMask() {
  const canvas = fsCanvasRef.value
  if (!canvas) return
  canvas.getContext('2d').clearRect(0, 0, canvas.width, canvas.height)
  fsHistory.value = []
}

function fsUndoMask() {
  if (fsHistory.value.length === 0) return
  const canvas = fsCanvasRef.value
  if (!canvas) return
  const prev = fsHistory.value.pop()
  canvas.getContext('2d').putImageData(prev, 0, 0)
}

function onKeydown(e) {
  if (e.key === 'Escape' && fullscreenMask.value) {
    e.preventDefault()
    closeFullscreenMask()
    return
  }
  if ((e.ctrlKey || e.metaKey) && e.key === 'Enter' && !generatingImage.value) {
    e.preventDefault()
    generate()
  }
}
</script>

<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Generate From Image</h1>
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
          <div
            v-if="!uploadedImage"
            class="drop-zone"
            :class="{ 'drop-zone-active': isDragOver }"
            @dragover.prevent="isDragOver = true"
            @dragleave="isDragOver = false"
            @drop.prevent="onDrop"
            @click="uploadImage()"
          >
            <div style="font-size: 32px; color: var(--text-dim);">&#128444;</div>
            <p style="color: var(--text-dim); margin-top: 8px;">Drop image here, click to upload, or Ctrl+V</p>
          </div>

          <div v-else class="inpaint-canvas-container">
            <img
              ref="imgEl"
              :src="'data:' + uploadedImageMime + ';base64,' + uploadedImage"
              alt="Source"
              class="inpaint-source-img"
              @load="(mode === 'inpaint' || mode === 'remove') && initMaskCanvas()"
            />
            <canvas
              v-if="mode === 'inpaint' || mode === 'remove'"
              ref="maskCanvasRef"
              class="inpaint-mask-canvas"
              :style="{ cursor: 'crosshair' }"
              @mousedown="startDraw"
              @mousemove="draw"
              @mouseup="stopDraw"
              @mouseleave="stopDraw"
              @touchstart="startDraw"
              @touchmove="draw"
              @touchend="stopDraw"
            />
            <div v-if="mode === 'inpaint' || mode === 'remove'" class="inpaint-brush-indicator"></div>
          </div>

          <div v-if="uploadedImage" class="fi-btn-row">
            <button class="btn btn-sm btn-secondary" @click="uploadImage">Change</button>
            <button class="btn btn-sm btn-secondary" @click="useLastImage">Last Generated</button>
            <button class="btn btn-sm btn-secondary" @click="pasteFromClipboard">Paste</button>
            <button class="btn btn-sm btn-secondary" @click="clearImage">Clear</button>
          </div>
          <div v-else class="fi-btn-row" style="margin-top: 8px;">
            <button class="btn btn-sm btn-secondary" @click="useLastImage">Last Generated</button>
            <button class="btn btn-sm btn-secondary" @click="pasteFromClipboard">Paste</button>
          </div>

          <div style="display: flex; gap: 8px; margin-top: 16px; margin-bottom: 12px; flex-wrap: wrap;">
            <button class="btn btn-sm" :class="mode === 'img2img' ? 'btn-primary' : 'btn-secondary'" @click="mode = 'img2img'" :disabled="generatingImage">img2img</button>
            <button class="btn btn-sm" :class="mode === 'inpaint' ? 'btn-primary' : 'btn-secondary'" @click="mode = 'inpaint'" :disabled="generatingImage">inpaint</button>
            <button class="btn btn-sm" :class="mode === 'remove' ? 'btn-primary' : 'btn-secondary'" @click="mode = 'remove'" :disabled="generatingImage">remove</button>
          </div>

          <div v-if="mode === 'img2img' || mode === 'inpaint'" class="form-group">
            <label class="form-label">Denoising Strength: {{ denoisingStrength.toFixed(2) }}</label>
            <input type="range" class="form-range" v-model.number="denoisingStrength" min="0.05" max="1.0" step="0.05" :disabled="generatingImage" />
            <div style="display: flex; justify-content: space-between; font-size: 11px; color: var(--text-dim);">
              <span>Keep original</span>
              <span>Full redraw</span>
            </div>
          </div>

          <div v-if="mode === 'remove'" class="form-group" style="margin-top: 4px;">
            <div style="font-size: 11px; color: var(--text-dim);">
              Denoising: {{ denoisingStrength.toFixed(2) }} | Mask Blur: {{ maskBlur }} | Fill: Fill | Full Res: on
            </div>
          </div>

          <div v-if="mode === 'inpaint' || mode === 'remove'" class="inpaint-controls">
            <div class="form-group">
              <label class="form-label">Brush Size: {{ brushSize }}px</label>
              <input type="range" class="form-range" v-model.number="brushSize" min="5" max="100" step="1" />
            </div>
            <div class="fi-btn-row">
              <button class="btn btn-sm btn-secondary" @click="clearMask" :disabled="!hasMask">Clear Mask</button>
              <button class="btn btn-sm btn-secondary" @click="undoMask" :disabled="!hasMask">Undo</button>
              <button class="btn btn-sm btn-secondary" @click="openFullscreenMask" title="Open fullscreen mask editor">&#x26F6; Fullscreen</button>
            </div>
            <div v-if="mode === 'inpaint'" style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-top: 8px;">
              <div class="form-group">
                <label class="form-label">Mask Blur: {{ maskBlur }}</label>
                <input type="range" class="form-range" v-model.number="maskBlur" min="0" max="64" step="1" />
              </div>
              <div class="form-group">
                <label class="form-label">Inpaint Fill</label>
                <select class="form-select" v-model="inpaintFill">
                  <option :value="0">Fill</option>
                  <option :value="1">Original</option>
                  <option :value="2">Latent Noise</option>
                </select>
              </div>
            </div>
            <div v-if="mode === 'inpaint'" style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-top: 4px;">
              <div class="form-group">
                <label class="form-label">Mask Padding: {{ maskPadding }}px</label>
                <input type="range" class="form-range" v-model.number="maskPadding" min="0" max="64" step="1" />
              </div>
              <div class="form-group">
                <label class="form-label">Mask Feather: {{ maskFeather }}px</label>
                <input type="range" class="form-range" v-model.number="maskFeather" min="0" max="64" step="1" />
              </div>
            </div>
            <div v-if="mode === 'inpaint'" class="form-group" style="margin-top: 4px;">
              <label style="display: flex; align-items: center; gap: 6px; cursor: pointer;">
                <input type="checkbox" v-model="inpaintFullRes" style="accent-color: var(--accent);" />
                <span style="font-size: 12px;">Inpaint Full Resolution</span>
              </label>
              <label style="display: flex; align-items: center; gap: 6px; cursor: pointer; margin-top: 4px;">
                <input type="checkbox" v-model="invertMask" style="accent-color: var(--accent);" />
                <span style="font-size: 12px;">Invert Mask (change everything except selection)</span>
              </label>
            </div>
            <div v-if="hasMask" style="font-size: 11px; color: var(--accent); margin-top: 4px;">
              Mask drawn ({{ maskHistory.length }} strokes){{ invertMask ? ' [inverted]' : '' }}
            </div>
            <div v-else style="font-size: 11px; color: var(--text-dim); margin-top: 4px;">
              Paint over areas to regenerate
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

          <div v-if="mode !== 'remove'" class="form-group" style="margin-top: 12px;">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px;">
              <label class="form-label" style="margin-bottom: 0;">Extracted Tags</label>
              <div style="display: flex; gap: 6px;">
                <button class="btn btn-sm" :class="analyzeMode === 'quick' ? 'btn-primary' : 'btn-secondary'" @click="analyzeMode = 'quick'" style="font-size: 11px; padding: 2px 8px;">Quick</button>
                <button class="btn btn-sm" :class="analyzeMode === 'deep' ? 'btn-primary' : 'btn-secondary'" @click="analyzeMode = 'deep'" style="font-size: 11px; padding: 2px 8px;">Deep</button>
              </div>
            </div>
            <div style="display: flex; gap: 8px; margin-bottom: 6px;">
              <button class="btn btn-sm btn-secondary" @click="analyzeImage" :disabled="analyzing || !uploadedImage">
                <template v-if="analyzing">
                  <span v-if="analyzeMode === 'deep' && chainStep > 0">Step {{ chainStep }}/{{ chainTotal }} &mdash; {{ analyzeElapsed }}s</span>
                  <span v-else>Analyzing...</span>
                </template>
                <span v-else>Analyze</span>
              </button>
              <button v-if="!kidsModeActive" class="btn btn-sm btn-secondary" @click="tags = ''" :disabled="!tags || generatingImage" title="Clear tags">&#10005;</button>
              <button v-if="tags" class="btn btn-sm btn-secondary" @click="transferToGenerate" :disabled="generatingImage" title="Transfer tags to Generate tab">&#8594; Generate</button>
            </div>
            <textarea class="form-textarea" v-model="tags" rows="4" placeholder="Extracted tags appear here. You can also type your own description to guide generation." :disabled="generatingImage || kidsModeActive"></textarea>
            <div v-if="kidsModeActive" style="margin-top: 4px; padding: 6px; background: var(--bg-secondary); border-radius: 4px; text-align: center; font-size: 11px; color: var(--text-dim);">
              &#128274; Tags editing restricted in Kids Mode
            </div>
          </div>

          <div v-if="mode !== 'remove' && recommendation" class="fi-recommendation">
            <div style="font-weight: 600; margin-bottom: 6px;">Recommended: {{ recommendation.preset_name }}</div>
            <div v-if="recommendation.reasoning" style="font-size: 12px; color: var(--text-dim); margin-bottom: 8px;">
              {{ recommendation.reasoning }}
            </div>
            <div v-if="recommendation.extra_prompt" style="font-size: 12px; color: var(--text-dim); margin-bottom: 8px;">
              <span style="font-weight: 600;">Extra tags:</span> {{ recommendation.extra_prompt }}
            </div>
            <div style="display: flex; gap: 8px;">
              <button class="btn btn-sm btn-primary" @click="applyRecommendation">Apply</button>
              <button class="btn btn-sm btn-secondary" @click="recommendation = null">Dismiss</button>
            </div>
          </div>

          <div v-if="mode !== 'remove' && recommending" style="margin-top: 8px; display: flex; align-items: center; gap: 8px; color: var(--text-dim); font-size: 13px;">
            <span class="spinner" style="width: 14px; height: 14px; border-width: 2px;"></span>
            Recommending preset...
          </div>

          <div v-if="!kidsModeActive" class="form-group">
            <label class="form-label">Extra Negative</label>
            <textarea class="form-textarea" v-model="extraNegativePrompt" rows="2" placeholder="Additional negative prompt..." :disabled="generatingImage"></textarea>
          </div>

          <button class="btn btn-primary" :class="{ 'btn-generating': generatingImage }" style="width: 100%; justify-content: center; padding: 12px;" @click="generate" :disabled="generatingImage || !uploadedImage || ((mode === 'inpaint' || mode === 'remove') && !hasMask) || (mode !== 'remove' && (genMode === 'preset' ? !selectedPresetId : !selectedCompoundPresetId))">
            <span v-if="generatingImage" style="display: inline-flex; align-items: center; gap: 6px;">
              <span class="spinner" style="width: 14px; height: 14px; border-width: 2px;"></span>
              {{ removeStage === 'analyzing' ? 'Analyzing context...' : generationStage === 'analyzing' ? 'Analyzing image...' : 'Generating image...' }}
            </span>
            <span v-else>Generate</span>
          </button>
        </div>
      </div>

      <div class="generate-section">
        <div class="generate-image-area">
          <div v-if="generatingImage" style="text-align: center;">
            <span class="spinner" style="width: 32px; height: 32px; border-width: 3px;"></span>
            <p style="margin-top: 12px; color: var(--text-dim);">{{ removeStage === 'analyzing' ? 'Analyzing context...' : generationStage === 'analyzing' ? 'Analyzing image...' : 'Generating image...' }}</p>
          </div>
          <div v-else-if="generatedImage" style="width: 100%; padding: 12px;">
            <img :src="'data:image/png;base64,' + generatedImage" alt="Generated" class="img-fade-in" style="border-radius: var(--radius-sm); cursor: zoom-in;" @click="showViewer = true" />
            <div style="display: flex; gap: 8px; margin-top: 12px; justify-content: center;">
              <button class="btn btn-secondary btn-sm" @click="downloadImage">Download</button>
              <button class="btn btn-secondary btn-sm" @click="copyPrompt">Copy</button>
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

    <ImageViewer v-if="showViewer" :image-base64="generatedImage" @close="showViewer = false" />

    <div v-if="fullscreenMask" class="fs-mask-overlay">
      <div class="fs-mask-toolbar">
        <span style="font-weight: 600;">Mask Editor</span>
        <div style="display: flex; align-items: center; gap: 12px;">
          <label class="form-label" style="margin: 0; font-size: 12px;">Brush: {{ brushSize }}px</label>
          <input type="range" v-model.number="brushSize" min="5" max="100" step="1" style="width: 120px; accent-color: var(--accent);" />
          <button class="btn btn-sm btn-secondary" @click="fsClearMask">Clear</button>
          <button class="btn btn-sm btn-secondary" @click="fsUndoMask" :disabled="fsHistory.length === 0">Undo</button>
          <button class="btn btn-sm btn-primary" @click="closeFullscreenMask">Done (Esc)</button>
        </div>
      </div>
      <div class="fs-mask-canvas-wrap">
        <img :src="'data:' + uploadedImageMime + ';base64,' + uploadedImage" class="fs-mask-img" />
        <canvas
          ref="fsCanvasRef"
          class="fs-mask-canvas"
          @mousedown="fsStartDraw"
          @mousemove="fsDraw"
          @mouseup="fsStopDraw"
          @mouseleave="fsStopDraw"
          @touchstart="fsStartDraw"
          @touchmove="fsDraw"
          @touchend="fsStopDraw"
        />
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
.form-range {
  width: 100%;
  accent-color: var(--accent);
}
.fi-btn-row {
  display: flex;
  gap: 8px;
  margin-top: 8px;
  flex-wrap: wrap;
}
.fi-recommendation {
  margin-top: 12px;
  padding: 12px;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-left: 3px solid var(--accent);
  border-radius: var(--radius-sm);
}
.inpaint-canvas-container {
  position: relative;
  display: flex;
  justify-content: center;
  border-radius: var(--radius-sm);
  overflow: hidden;
  background: #000;
}
.inpaint-source-img {
  max-height: 400px;
  max-width: 100%;
  display: block;
  border-radius: var(--radius-sm);
}
.inpaint-mask-canvas {
  position: absolute;
  top: 0;
  left: 50%;
  transform: translateX(-50%);
  max-height: 400px;
  max-width: 100%;
  border-radius: var(--radius-sm);
}
.inpaint-controls {
  margin-top: 12px;
  padding: 12px;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}
.fs-mask-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  z-index: 9999;
  background: rgba(0, 0, 0, 0.95);
  display: flex;
  flex-direction: column;
}
.fs-mask-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  background: var(--bg-primary, #1a1a2e);
  color: var(--text-primary, #e0e0e0);
  border-bottom: 1px solid var(--border, #333);
  flex-shrink: 0;
}
.fs-mask-canvas-wrap {
  flex: 1;
  display: flex;
  justify-content: center;
  align-items: center;
  position: relative;
  overflow: hidden;
}
.fs-mask-img {
  max-width: 95vw;
  max-height: calc(100vh - 60px);
  object-fit: contain;
  display: block;
}
.fs-mask-canvas {
  position: absolute;
  cursor: crosshair;
  max-width: 95vw;
  max-height: calc(100vh - 60px);
  object-fit: contain;
}
</style>
