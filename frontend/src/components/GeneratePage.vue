<script setup>
import { ref, computed, onMounted, onUnmounted, watch, inject } from 'vue'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'
import { api } from '../api.js'
import { t } from '../i18n/index.js'
import { useGenerationProgress } from '../composables/useGenerationProgress.js'
import { useKidsMode } from '../composables/useKidsMode.js'
import { usePresets } from '../composables/usePresets.js'
import SavedDescriptionsModal from './SavedDescriptionsModal.vue'
import ImageViewer from './ImageViewer.vue'
import ResolutionSelector from './ResolutionSelector.vue'
import HiresProfileSelector from './HiresProfileSelector.vue'

const { presets, presetTypes, compoundPresets, loadPresets } = usePresets()

const props = defineProps({
  resetting: { type: Boolean, default: false },
})
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
const hiresSkipped = ref(false)
const savedPreview = ref(null)
const upscaling = ref(false)
const upscalingX2 = ref(false)
const previewMode = ref(false)
const effectivePrompt = ref('')
const effectiveNegative = ref('')

const showFastSaveModal = ref(false)
const fastSaveFilename = ref('')
const fastSaveFormat = ref('jpg')
const fastSaveLoading = ref(false)

const generatingImage = ref(false)
const { llmStatus, sdProgress, preview, interrupt: interruptGeneration, reset: resetProgress } = useGenerationProgress()
const generationStage = ref('')
const error = ref('')
let promptDirty = true

const { kidsModeActive, loadKidsMode } = useKidsMode()

const selectedResolutionId = ref(null)
const selectedHiresProfileId = ref(null)

const shared = inject('sharedGenState', null)

const recommendDesc = ref('')
const recommending = ref(false)
const recommendResult = ref(null)
const showSuggestPopover = ref(false)

const genCount = ref(1)
const enqueuedJobIds = ref(new Set())
const batchTotal = ref(0)
const batchDone = ref(0)

const savedDescs = ref([])
const showSavedDescs = ref(false)
const showViewer = ref(false)

const moreOptionsOpen = ref(false)
const showWorkflowLink = ref(false)

const isDesktop = ref(window.innerWidth > 1024)

const filteredPresets = computed(() => {
  if (!selectedTypeId.value) return presets.value
  return presets.value.filter(p => p.type_id === selectedTypeId.value)
})

const imageSrc = computed(() => {
  if (!generatedImage.value) return ''
  if (generatedImage.value.startsWith('/api/')) return generatedImage.value
  return 'data:image/png;base64,' + generatedImage.value
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

watch([description, negative, selectedPresetId, selectedCompoundPresetId], () => {
  promptDirty = true
})

watch(selectedTypeId, () => {
  const filtered = filteredPresets.value
  if (selectedPresetId.value && !filtered.find(p => p.id === selectedPresetId.value)) {
    selectedPresetId.value = null
  }
  api.updateSettings({ gen_type_id: String(selectedTypeId.value || '') }).catch(() => {})
})

watch([selectedResolutionId, selectedHiresProfileId], () => {
  api.updateSettings({
    gen_resolution_id: String(selectedResolutionId.value || ''),
    gen_hires_profile_id: String(selectedHiresProfileId.value || ''),
  }).catch(() => {})
})

function onResize() {
  isDesktop.value = window.innerWidth > 1024
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
    gen_resolution_id: String(selectedResolutionId.value || ''),
    gen_hires_profile_id: String(selectedHiresProfileId.value || ''),
    gen_count: String(genCount.value || 1),
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
    error.value = t('generate.error_recommend', { error: String(e) })
  } finally {
    recommending.value = false
  }
}

async function sendToSD() {
  if (genMode.value === 'compound' && !selectedCompoundPresetId.value) {
    error.value = t('generate.error_select_workflow')
    return
  }
  if (genMode.value === 'preset' && !selectedPresetId.value) {
    error.value = t('generate.error_select_style')
    return
  }
  saveGenState()
  generationStage.value = 'image'
  generatingImage.value = true
  generatedImage.value = ''
  genInfo.value = null
  isPreview.value = false
  hiresSkipped.value = false
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
        resolution_id: selectedResolutionId.value,
        hires_profile_id: selectedHiresProfileId.value,
      })
    } else {
      result = await api.generateImage(selectedPresetId.value, extraPrompt.value, extraNegativePrompt.value, selectedResolutionId.value, selectedHiresProfileId.value)
    }
    if (!result || !result.image) {
      error.value = t('generate.error_no_image')
    } else {
      error.value = ''
      generatedImage.value = result.image
      genInfo.value = result.info
      sourceImage.value = result.image
      sourceGenInfo.value = result.info
      isPreview.value = result.is_preview || false
      hiresSkipped.value = result.hires_fix_skipped || false
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
    error.value = t('generate.error_select_workflow')
    return
  }
  if (genMode.value === 'preset' && !selectedPresetId.value) {
    error.value = t('generate.error_select_style')
    return
  }
  saveGenState()
  resetProgress()
  generatingImage.value = true
  generatedImage.value = ''
  genInfo.value = null
  isPreview.value = false
  hiresSkipped.value = false
  savedPreview.value = null
  effectivePrompt.value = ''
  effectiveNegative.value = ''
  enqueuedJobIds.value = new Set()
  batchTotal.value = 0
  batchDone.value = 0

  if (promptDirty) {
    let llmPresetId = selectedPresetId.value
    if (genMode.value === 'compound') {
      const cp = compoundPresets.value.find(c => c.id === selectedCompoundPresetId.value)
      if (cp && cp.steps && cp.steps.length > 0) {
        llmPresetId = cp.steps[0].preset_id
      }
    }
    if (llmPresetId && (genMode.value === 'preset' || genMode.value === 'compound')) {
      generationStage.value = 'prompt'
      error.value = ''
      try {
        const promptResult = await api.generateSdPrompt({
          preset_id: llmPresetId,
          description: description.value,
          negative: negative.value,
        })
        if (promptResult && promptResult.prompt) {
          extraPrompt.value = promptResult.prompt
          extraNegativePrompt.value = promptResult.negative_prompt || ''
          promptDirty = false
        } else {
          error.value = t('generate.error_empty_llm')
          generatingImage.value = false
          generationStage.value = ''
          return
        }
      } catch (e) {
        error.value = t('generate.error_prompt_gen', { error: String(e) })
        generatingImage.value = false
        generationStage.value = ''
        return
      }
      generatingImage.value = false
      generationStage.value = ''
    }
  }

  generatingImage.value = true
  generationStage.value = 'image'
  try {
    const count = Math.max(1, Math.min(100, genCount.value || 1))
    batchTotal.value = count
    for (let i = 0; i < count; i++) {
      let jobId
      if (genMode.value === 'compound') {
        jobId = await api.enqueueCompound({
          compound_preset_id: selectedCompoundPresetId.value,
          extra_prompt: extraPrompt.value,
          extra_negative_prompt: extraNegativePrompt.value,
          resolution_id: selectedResolutionId.value,
          hires_profile_id: selectedHiresProfileId.value,
        })
      } else {
        jobId = await api.enqueueTxt2Img({
          preset_id: selectedPresetId.value,
          extra_prompt: extraPrompt.value,
          extra_negative_prompt: extraNegativePrompt.value,
          resolution_id: selectedResolutionId.value,
          hires_profile_id: selectedHiresProfileId.value,
        })
      }
      if (jobId) enqueuedJobIds.value.add(jobId)
    }
    error.value = ''
  } catch (e) {
    error.value = String(e)
    if (enqueuedJobIds.value.size === 0) {
      generatingImage.value = false
      generationStage.value = ''
    }
  }
}

async function upscalePreview() {
  if (!generatedImage.value || !selectedPresetId.value || !genInfo.value) return
  const b64 = await resolveImageBase64()
  if (!b64) return
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
    const result = await api.upscalePreview(b64, selectedPresetId.value, seed)
    if (!result || !result.image) {
      error.value = t('generate.error_upscale')
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
  const b64 = await resolveImageBase64()
  if (!b64) return
  upscalingX2.value = true
  error.value = ''
  try {
    const result = await api.upscaleImage(b64, genInfo.value, selectedPresetId.value)
    if (!result || !result.image) {
      error.value = t('generate.error_upscale_x2')
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
    error.value = t('generate.error_save_desc', { error: String(e) })
  }
}

async function deleteDescription(id) {
  try {
    await api.deleteDescription(id)
    await loadSavedDescs()
  } catch (e) {
    error.value = t('generate.error_delete_desc', { error: String(e) })
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
    error.value = t('generate.error_create', { error: String(e) })
  }
}

async function handleUpdateDesc(data) {
  try {
    await api.updateDescription(data)
    await loadSavedDescs()
  } catch (e) {
    error.value = t('generate.error_update', { error: String(e) })
  }
}

async function resolveImageBase64() {
  if (!generatedImage.value) return null
  if (!generatedImage.value.startsWith('/api/')) return generatedImage.value
  const match = generatedImage.value.match(/\/api\/img\/(\d+)\.jpg/)
  if (match) {
    try { return await api.getSessionImage(Number(match[1])) } catch { return null }
  }
  return generatedImage.value
}

async function downloadImage() {
  if (!generatedImage.value) return
  try {
    const b64 = await resolveImageBase64()
    if (!b64) return
    const defaultName = 'sd-studio-' + Date.now() + '.png'
    const savedPath = await api.saveImage(b64, defaultName)
    if (savedPath) {
      error.value = ''
    }
  } catch (e) {
    error.value = t('generate.error_save', { error: String(e) })
  }
}

function openFastSaveModal() {
  fastSaveFilename.value = 'sd-studio-' + Date.now()
  showFastSaveModal.value = true
}

async function confirmFastSave() {
  if (!fastSaveFilename.value.trim() || !generatedImage.value) return
  fastSaveLoading.value = true
  try {
    const b64 = await resolveImageBase64()
    if (!b64) return
    const savedPath = await api.fastSaveImage(b64, fastSaveFilename.value.trim(), fastSaveFormat.value)
    if (savedPath) {
      showFastSaveModal.value = false
      error.value = ''
    }
  } catch (e) {
    error.value = t('generate.error_quick_save', { error: String(e) })
  } finally {
    fastSaveLoading.value = false
  }
}

function copyPrompt() {
  const parts = []
  if (effectivePrompt.value) parts.push(effectivePrompt.value)
  if (effectiveNegative.value) parts.push('Negative: ' + effectiveNegative.value)
  if (parts.length) navigator.clipboard.writeText(parts.join('\n'))
}

function onQueueCompleted(data) {
  if (!data) return

  if (data.job_id && enqueuedJobIds.value.has(data.job_id)) {
    enqueuedJobIds.value.delete(data.job_id)
    batchDone.value++
  }

  if (data.result) {
    try {
      const r = typeof data.result === 'string' ? JSON.parse(data.result) : data.result
      if (r.image_base64) {
        generatedImage.value = r.image_base64
        sourceImage.value = r.image_base64
        let info = null
        if (r.info) {
          try { info = typeof r.info === 'string' ? JSON.parse(r.info) : r.info } catch { info = r.info }
        }
        genInfo.value = info
        sourceGenInfo.value = info
        isPreview.value = false
        hiresSkipped.value = false
      }
    } catch {}
  }

  if (enqueuedJobIds.value.size === 0) {
    generatingImage.value = false
    generationStage.value = ''
  }
}

function onQueueFailed(data) {
  if (!data) return

  if (data.job_id && enqueuedJobIds.value.has(data.job_id)) {
    enqueuedJobIds.value.delete(data.job_id)
    batchDone.value++
  }

  if (data.error) error.value = data.error

  if (enqueuedJobIds.value.size === 0) {
    generatingImage.value = false
    generationStage.value = ''
  }
}

async function onSessionAdded() {
  if (!generatingImage.value && generatedImage.value) return
  try {
    const item = await api.getActiveSessionItem()
    if (!item) return
    generatedImage.value = api.sessionImageUrl(item.id)
    sourceImage.value = generatedImage.value
    genInfo.value = item.info || null
    sourceGenInfo.value = item.info || null
    isPreview.value = item.is_preview || false
    generatingImage.value = false
    generationStage.value = ''
  } catch {}
}

function enableWorkflowMode() {
  genMode.value = 'compound'
  showWorkflowLink.value = false
}

onMounted(async () => {
  const isReset = props.resetting
  await loadPresets()
  loadKidsMode()
  loadSavedDescs()
  let resolutionLoaded = false
  try {
    const s = await api.getSettings()
    previewMode.value = s.preview_mode === 'true'
    if (s.gen_type_id) {
      const tid = Number(s.gen_type_id) || null
      if (tid && presetTypes.value.find(pt => pt.id === tid)) {
        selectedTypeId.value = tid
      }
    }
    if (s.gen_extra_prompt) extraPrompt.value = s.gen_extra_prompt
    if (s.gen_extra_negative) extraNegativePrompt.value = s.gen_extra_negative
    if (s.gen_preset_id) {
      const pid = Number(s.gen_preset_id)
      if (presets.value.find(p => p.id === pid)) {
        selectedPresetId.value = pid
      }
    }
    if (s.gen_description) description.value = s.gen_description
    if (s.gen_negative) negative.value = s.gen_negative
    if (s.gen_mode) genMode.value = s.gen_mode
    if (s.gen_compound_preset_id) {
      const cpid = Number(s.gen_compound_preset_id)
      if (compoundPresets.value.find(c => c.id === cpid)) {
        selectedCompoundPresetId.value = cpid
      }
    }
    if (s.gen_resolution_id) {
      selectedResolutionId.value = Number(s.gen_resolution_id)
      resolutionLoaded = true
    }
    if (s.gen_hires_profile_id) selectedHiresProfileId.value = Number(s.gen_hires_profile_id)
    if (s.gen_count) genCount.value = Math.max(1, Math.min(100, Number(s.gen_count) || 1))
  } catch {}
  if (shared) {
    if (shared.selectedPresetId) selectedPresetId.value = shared.selectedPresetId
    if (shared.selectedCompoundPresetId) selectedCompoundPresetId.value = shared.selectedCompoundPresetId
    if (shared.genMode) genMode.value = shared.genMode
    if (shared.description) description.value = shared.description
    if (shared.negative) negative.value = shared.negative
    if (shared.selectedResolutionId) {
      selectedResolutionId.value = shared.selectedResolutionId
      resolutionLoaded = true
    }
    if (shared.selectedHiresProfileId !== undefined) selectedHiresProfileId.value = shared.selectedHiresProfileId
  }
  if (!resolutionLoaded) {
    try {
      const resolutions = await api.listResolutions()
      if (resolutions && resolutions.length > 0) selectedResolutionId.value = resolutions[0].id
    } catch {}
  }
  if (!isReset) {
    try {
      const item = await api.getActiveSessionItem()
      if (item) {
        generatedImage.value = api.sessionImageUrl(item.id)
        genInfo.value = item.info || null
        sourceImage.value = generatedImage.value
        sourceGenInfo.value = item.info || null
        isPreview.value = item.is_preview || false
      }
    } catch {}
  }

  try {
    const queue = await api.getQueue()
    const activeJobs = (queue || []).filter(j => j.status === 'pending' || j.status === 'running')
    if (activeJobs.length > 0) {
      generatingImage.value = true
      generationStage.value = 'image'
      enqueuedJobIds.value = new Set(activeJobs.map(j => j.id))
      batchTotal.value = activeJobs.length
      batchDone.value = 0
    }
  } catch {}

  EventsOn('queue:completed', onQueueCompleted)
  EventsOn('queue:failed', onQueueFailed)
  EventsOn('session:added', onSessionAdded)
  document.addEventListener('keydown', onKeydown)
  window.addEventListener('resize', onResize)
})

onUnmounted(() => {
  if (props.resetting) {
    EventsOff('queue:completed')
    EventsOff('queue:failed')
    EventsOff('session:added')
    document.removeEventListener('keydown', onKeydown)
    window.removeEventListener('resize', onResize)
    return
  }
  EventsOff('queue:completed')
  EventsOff('queue:failed')
  EventsOff('session:added')
  document.removeEventListener('keydown', onKeydown)
  window.removeEventListener('resize', onResize)
  saveGenState()
  if (shared) {
    shared.selectedPresetId = selectedPresetId.value
    shared.selectedCompoundPresetId = selectedCompoundPresetId.value
    shared.genMode = genMode.value
    shared.description = description.value
    shared.negative = negative.value
    shared.selectedResolutionId = selectedResolutionId.value
    shared.selectedHiresProfileId = selectedHiresProfileId.value
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
      <h1 class="page-title">{{ t('generate.title') }}</h1>
    </div>

    <div v-if="kidsModeActive" class="service-status">
      <div class="status-badge status-ok">
        &#9679; {{ t('generate.kids_mode') }}
      </div>
    </div>

    <div v-if="error && error !== 'interrupted'" class="status status-error">
      &#9888; {{ error }}
    </div>
    <div v-if="error === 'interrupted'" class="status status-warning">{{ error }}</div>

    <div class="generate-layout">
      <div class="generate-section">
        <div class="card">
          <!-- Zone 1: Main flow (always visible) -->

          <!-- Type & Style selector -->
          <div v-if="genMode === 'preset'" class="form-group">
            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px;">
              <div class="form-group">
                <label class="form-label">{{ t('generate.label_type') }}</label>
                <select class="form-select" v-model="selectedTypeId">
                  <option :value="null">{{ t('generate.all_types') }}</option>
                  <option v-for="pt in presetTypes" :key="pt.id" :value="pt.id">{{ pt.name }}</option>
                </select>
              </div>
              <div class="form-group">
                <div style="display: flex; align-items: center; gap: 8px;">
                  <div class="form-group" style="flex: 1; margin-bottom: 0;">
                    <label class="form-label">{{ t('generate.label_style') }}</label>
                    <select class="form-select" v-model="selectedPresetId">
                      <option :value="null" disabled>{{ t('generate.select_style') }}</option>
                      <option v-for="p in filteredPresets" :key="p.id" :value="p.id">{{ p.name }}</option>
                    </select>
                  </div>
                  <button class="btn-icon-action" style="margin-top: 20px; width: 32px; height: 32px; display: flex; align-items: center; justify-content: center; background: var(--surface-2); border: 1px solid var(--border); border-radius: var(--radius-sm); color: var(--text-dim); cursor: pointer;" @click="showSuggestPopover = !showSuggestPopover" title="Suggest Style">&#10024;</button>
                </div>
              </div>
            </div>
            <div v-if="showSuggestPopover" style="margin-top: 8px; padding: 12px; background: var(--surface-2); border: 1px solid var(--border); border-radius: var(--radius-sm);">
              <div style="display: flex; gap: 8px;">
                <input class="form-input" v-model="recommendDesc" :placeholder="t('generate.placeholder_suggest')" :disabled="recommending" style="flex: 1;" />
                <button class="btn btn-secondary" @click="recommendPreset" :disabled="recommending || !recommendDesc.trim()">
                  {{ recommending ? '...' : t('generate.btn_suggest') }}
                </button>
              </div>
              <div v-if="recommendResult" style="margin-top: 8px; padding: 8px; background: var(--surface); border-radius: 6px; font-size: 13px;">
                <div style="color: var(--text-bright);">{{ recommendResult.preset_name }}</div>
                <div v-if="recommendResult.reasoning" style="color: var(--text-dim); margin-top: 4px;">{{ recommendResult.reasoning }}</div>
              </div>
              <button style="margin-top: 6px; background: none; border: none; color: var(--text-dim); font-size: 11px; cursor: pointer;" @click="showSuggestPopover = false; recommendResult = null">Close</button>
            </div>
            <div v-if="genMode === 'preset'" class="workflow-link" @click="enableWorkflowMode">{{ t('generate.using_workflows') }}</div>
          </div>

          <div v-if="genMode === 'compound'" class="form-group">
            <label class="form-label">{{ t('generate.label_workflow') }}</label>
            <select class="form-select" v-model="selectedCompoundPresetId">
              <option :value="null" disabled>{{ t('generate.select_workflow') }}</option>
              <option v-for="c in compoundPresets" :key="c.id" :value="c.id">{{ c.name }} ({{ c.steps.length }} steps)</option>
            </select>
            <div class="workflow-link" @click="genMode = 'preset'">{{ t('generate.btn_style') }} mode</div>
          </div>

          <!-- Description -->
          <div class="form-group">
            <label class="form-label">{{ t('generate.label_description') }}</label>
            <textarea class="form-textarea" v-model="description" rows="4" :placeholder="t('generate.placeholder_description')"></textarea>
            <div style="display: flex; gap: 8px; margin-top: 6px;">
              <button class="btn btn-sm btn-secondary" @click="saveDescription" :disabled="!description.trim()">{{ t('generate.btn_save') }}</button>
              <button class="btn btn-sm btn-secondary" @click="showSavedDescs = !showSavedDescs">
                {{ t('generate.saved_ideas') }} {{ savedDescs.length ? '(' + savedDescs.length + ')' : '' }}
              </button>
            </div>
          </div>

          <!-- Generate button -->
          <div style="display: flex; gap: 8px; width: 100%;">
            <button class="btn btn-primary btn-generate" :class="{ 'btn-generating': generatingImage }" style="flex: 1; justify-content: center;" @click="generateImage" :disabled="genMode === 'preset' ? !selectedPresetId : !selectedCompoundPresetId">
              <span v-if="generatingImage" style="display: inline-flex; align-items: center; gap: 6px;">
                <span class="spinner" style="width: 14px; height: 14px; border-width: 2px;"></span>
                {{ generationStage === 'prompt' ? t('generate.generating_prompt') : t('generate.generating_image') }}
              </span>
              <span v-else>{{ genCount <= 1 ? t('generate.btn_generate') : t('generate.btn_generate_n', { count: genCount }) }}</span>
            </button>
            <input class="form-input" type="number" v-model.number="genCount" min="1" max="100" style="width: 70px; text-align: center;" @change="genCount = Math.max(1, Math.min(100, genCount || 1))" />
          </div>
          <div v-if="isDesktop" class="shortcut-hint">{{ t('generate.shortcut_hint') }}</div>

          <!-- Zone 2: More Options (collapsible) -->
          <div style="margin-top: 12px;">
            <button class="more-options-toggle" @click="moreOptionsOpen = !moreOptionsOpen">
              <span class="more-options-arrow" :class="{ open: moreOptionsOpen }">&#9654;</span>
              {{ t('generate.more_options') }}
            </button>
            <div class="more-options-content" :class="{ open: moreOptionsOpen }">
              <div class="more-options-inner">
                <div style="padding-top: 12px;">
                  <div class="form-group">
                    <label class="form-label">{{ t('generate.label_exclude') }}</label>
                    <textarea class="form-textarea" v-model="negative" rows="2" :placeholder="t('generate.placeholder_exclude')"></textarea>
                  </div>
                  <div style="display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 12px;">
                    <ResolutionSelector v-model="selectedResolutionId" />
                  </div>
                  <HiresProfileSelector v-model="selectedHiresProfileId" />
                </div>
              </div>
            </div>
          </div>

          <!-- Zone 3: Advanced (existing details) -->
          <details v-if="!kidsModeActive" style="margin-top: 8px;" class="card">
            <summary style="cursor: pointer; color: var(--text-dim); font-size: 13px; padding: 8px;">{{ t('generate.advanced_options') }}</summary>
            <div style="margin-top: 8px;">
              <div class="form-group">
                <label class="form-label">{{ t('generate.label_positive_prompt') }}</label>
                <textarea class="form-textarea" v-model="extraPrompt" rows="4" :placeholder="t('generate.placeholder_positive_prompt')"></textarea>
              </div>
              <div class="form-group">
                <label class="form-label">{{ t('generate.label_negative_prompt') }}</label>
                <textarea class="form-textarea" v-model="extraNegativePrompt" rows="2" :placeholder="t('generate.placeholder_negative_prompt')"></textarea>
              </div>
              <button class="btn btn-secondary" style="width: 100%; justify-content: center;" @click="sendToSD" :disabled="(genMode === 'preset' ? !selectedPresetId : !selectedCompoundPresetId) || !extraPrompt">
                {{ t('generate.btn_generate_directly') }}
              </button>
            </div>
          </details>
          <div v-else style="margin-top: 8px; padding: 8px; background: var(--surface-2); border-radius: 6px; text-align: center; font-size: 12px; color: var(--text-dim);">
            {{ t('generate.safe_prompt_disabled') }}
          </div>
        </div>
      </div>

      <div class="generate-section">
        <div class="generate-image-area" :class="{ 'loading': generatingImage && !generatedImage, 'empty': !generatedImage && !generatingImage }">
          <div v-if="generatingImage && !generatedImage" style="text-align: center; padding: 24px;">
            <img v-if="preview && sdProgress && sdProgress.progress > 0 && sdProgress.progress < 1" :src="preview" alt="preview" style="max-width: 100%; border-radius: var(--radius-sm); opacity: 0.6; image-rendering: pixelated;" />
            <span v-else class="spinner" style="width: 32px; height: 32px; border-width: 3px;"></span>
            <p style="margin-top: 12px; color: var(--text-dim);">{{ batchTotal > 1 ? (batchDone + 1) + ' / ' + batchTotal + ' — ' : '' }}{{ llmStatus === 'thinking' ? t('generate.generating_prompt') : upscalingX2 ? t('generate.upscaling_x2') : upscaling ? t('generate.upscaling_full') : t('generate.generating_image') }}</p>
            <div v-if="sdProgress && sdProgress.progress > 0" style="margin-top: 12px; max-width: 300px; margin-left: auto; margin-right: auto;">
              <div style="display: flex; justify-content: space-between; margin-bottom: 4px;">
                <span style="color: var(--text-dim); font-size: 12px;">{{ Math.round(sdProgress.progress * 100) }}%</span>
                <span style="color: var(--text-dim); font-size: 12px;">{{ t('progress.sd_step', { current: Math.round(sdProgress.progress * sdProgress.steps), total: sdProgress.steps }) }}
                  <span v-if="sdProgress.etaRelative > 0"> — ~{{ Math.ceil(sdProgress.etaRelative) }}s</span>
                </span>
              </div>
              <div style="background: var(--surface-2); border-radius: 4px; overflow: hidden; height: 6px;">
                <div :style="{ width: (sdProgress.progress * 100) + '%', background: 'var(--accent)', height: '100%', transition: 'width 0.3s' }"></div>
              </div>
              <button class="btn btn-sm btn-secondary" @click="interruptGeneration" style="margin-top: 8px; font-size: 11px;">{{ t('progress.btn_interrupt') }}</button>
            </div>
          </div>
          <div v-else-if="generatingImage && generatedImage && enqueuedJobIds.size > 0" style="width: 100%; padding: 12px; position: relative;">
            <img :src="imageSrc" alt="Generating..." style="border-radius: var(--radius-sm); opacity: 0.4; width: 100%;" />
            <div style="position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); text-align: center; background: rgba(0,0,0,0.6); padding: 16px 24px; border-radius: 8px;">
              <img v-if="preview && sdProgress && sdProgress.progress > 0 && sdProgress.progress < 1" :src="preview" alt="preview" style="max-width: 200px; border-radius: var(--radius-sm); opacity: 0.8; image-rendering: pixelated;" />
              <span v-else class="spinner" style="width: 24px; height: 24px; border-width: 2px;"></span>
              <p style="margin-top: 8px; color: var(--text-dim); font-size: 13px;">{{ (batchDone + 1) + ' / ' + batchTotal }}</p>
              <div v-if="sdProgress && sdProgress.progress > 0" style="margin-top: 8px; min-width: 180px;">
                <div style="display: flex; justify-content: space-between; margin-bottom: 4px;">
                  <span style="color: var(--text-dim); font-size: 11px;">{{ Math.round(sdProgress.progress * 100) }}%</span>
                  <span style="color: var(--text-dim); font-size: 11px;">
                    <span v-if="sdProgress.etaRelative > 0">~{{ Math.ceil(sdProgress.etaRelative) }}s</span>
                  </span>
                </div>
                <div style="background: var(--surface-2); border-radius: 4px; overflow: hidden; height: 4px;">
                  <div :style="{ width: (sdProgress.progress * 100) + '%', background: 'var(--accent)', height: '100%', transition: 'width 0.3s' }"></div>
                </div>
              </div>
            </div>
          </div>
          <div v-else-if="generatedImage" style="width: 100%; padding: 12px;">
            <div v-if="isPreview && previewMode" class="status status-info" style="margin-bottom: 8px; text-align: center;">
              {{ t('generate.preview_info') }}
            </div>
            <div v-if="hiresSkipped" class="status status-warning" style="margin-bottom: 8px; text-align: center;">
              {{ t('generate.hires_skipped') }}
            </div>
            <img :src="imageSrc" alt="Generated" class="img-fade-in" style="border-radius: var(--radius-sm); cursor: zoom-in;" @click="showViewer = true" />
            <div class="action-row" style="margin-top: 12px;">
              <button v-if="generatedImage" class="btn btn-primary btn-sm" @click="openFastSaveModal">{{ t('generate.btn_quick_save') }}</button>
              <button v-if="isPreview && previewMode" class="btn btn-sm btn-secondary" @click="upscalePreview">{{ t('generate.btn_upscale_full') }}</button>
              <button v-if="!isPreview && savedPreview" class="btn btn-sm btn-secondary" @click="backToPreview">{{ t('generate.btn_back_preview') }}</button>
              <button v-if="generatedImage && !isPreview" class="btn btn-sm btn-secondary" @click="upscaleImageX2" :disabled="upscalingX2">{{ t('generate.btn_upscale_x2') }}</button>
              <button class="btn btn-sm btn-secondary" @click="downloadImage" data-tooltip="Download">{{ t('generate.btn_download') }}</button>
              <button class="btn btn-sm btn-secondary" @click="copyPrompt" data-tooltip="Copy prompt">{{ t('generate.btn_copy') }}</button>
              <button class="btn btn-sm btn-secondary" @click="generateImage" data-tooltip="Regenerate">{{ t('generate.btn_regenerate') }}</button>
            </div>
          </div>
          <div v-else class="generate-placeholder">
            <div class="generate-placeholder-icon">&#10024;</div>
            <div class="generate-placeholder-title">{{ t('generate.placeholder_image') }}</div>
            <div class="generate-placeholder-hint">{{ t('generate.placeholder_image_hint') }}</div>
          </div>
        </div>

        <details v-if="genInfo" class="gen-info card">
          <summary>{{ t('generate.technical_details') }}</summary>
          <pre style="white-space: pre-wrap; word-break: break-word; overflow-wrap: break-word;">{{ formattedGenInfo }}</pre>
        </details>

        <details v-if="effectivePrompt" class="gen-info card">
          <summary>{{ t('generate.final_prompt') }}</summary>
          <div style="margin-bottom: 8px;">
            <div style="color: var(--text-dim); font-size: 11px; margin-bottom: 4px;">{{ t('generate.positive') }}</div>
            <div style="font-size: 12px; line-height: 1.5; word-break: break-word;">{{ effectivePrompt }}</div>
          </div>
          <div v-if="effectiveNegative" style="margin-top: 8px; padding-top: 8px; border-top: 1px solid var(--border);">
            <div style="color: var(--text-dim); font-size: 11px; margin-bottom: 4px;">{{ t('generate.negative') }}</div>
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

    <div v-if="showFastSaveModal" class="modal-overlay">
      <div class="modal-content" style="max-width: 400px;">
        <h3 style="margin: 0 0 12px;">{{ t('generate.quick_save_title') }}</h3>
        <div class="form-group">
          <label class="form-label">{{ t('generate.quick_save_filename') }}</label>
          <input class="form-input" v-model="fastSaveFilename" @keydown.enter="confirmFastSave" :disabled="fastSaveLoading" autofocus />
        </div>
        <div class="form-group" style="margin-top: 8px;">
          <label class="form-label">{{ t('generate.quick_save_format') }}</label>
          <div style="display: flex; gap: 8px;">
            <button class="btn btn-sm" :class="fastSaveFormat === 'jpg' ? 'btn-primary' : 'btn-secondary'" @click="fastSaveFormat = 'jpg'">JPG</button>
            <button class="btn btn-sm" :class="fastSaveFormat === 'png' ? 'btn-primary' : 'btn-secondary'" @click="fastSaveFormat = 'png'">PNG</button>
          </div>
        </div>
        <div style="display: flex; gap: 8px; justify-content: flex-end; margin-top: 12px;">
          <button class="btn btn-secondary" @click="showFastSaveModal = false" :disabled="fastSaveLoading">{{ t('generate.quick_save_cancel') }}</button>
          <button class="btn btn-primary" @click="confirmFastSave" :disabled="!fastSaveFilename.trim() || fastSaveLoading">
            <span v-if="fastSaveLoading" class="spinner" style="width: 14px; height: 14px; border-width: 2px; display: inline-block; vertical-align: middle; margin-right: 4px;"></span>
            {{ t('generate.quick_save_btn') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
