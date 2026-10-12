<script setup>
import { computed, ref, watch, onMounted, onUnmounted } from 'vue'
import { t } from '../i18n/index.js'
import { parseLoras } from '../loraOverride.js'

const props = defineProps({
  imageBase64: { type: String, required: true },
  hasPrev: { type: Boolean, default: false },
  hasNext: { type: Boolean, default: false },
  item: { type: Object, default: null },
  showActions: { type: Boolean, default: false },
  confirmDelete: { type: Boolean, default: false },
})
const emit = defineEmits(['close', 'prev', 'next', 'remix', 'export', 'delete'])

const promptExpanded = ref(false)
const infoExpanded = ref(false)

const ZOOM_MIN = 10
const ZOOM_MAX = 500
const ZOOM_PRESETS = [30, 50, 100]

const naturalW = ref(0)
const naturalH = ref(0)
const zoomMode = ref('fit')
const zoomPct = ref(100)
const panX = ref(0)
const panY = ref(0)
const dragging = ref(false)
const zoomMenuOpen = ref(false)
let dragLastX = 0
let dragLastY = 0
let dragMovedDist = 0
let suppressDbl = false
let suppressDblTimer = null

const imgSrc = computed(() => {
  if (props.imageBase64.startsWith('/api/')) return props.imageBase64
  return 'data:image/png;base64,' + props.imageBase64
})

const chain = computed(() => {
  const c = props.item?.meta?.chain
  return Array.isArray(c) ? c.filter(s => s && typeof s === 'object') : []
})
const chainLabel = computed(() => props.item?.meta?.label || '')
const metaModel = computed(() => props.item?.meta?.model || '')

const chainHeader = computed(() => {
  if (!chain.value.length) return ''
  return chainLabel.value
    ? t('viewer.chain_title') + ': ' + chainLabel.value
    : t('viewer.chain_custom', { count: chain.value.length })
})

const metaOnlyLine = computed(() => {
  if (chain.value.length) return ''
  return [chainLabel.value, metaModel.value].filter(Boolean).join(' · ')
})

function opLabel(op) {
  if (op === 'base') return t('viewer.chain_op_base')
  if (op === 'upscale') return t('viewer.chain_op_upscale')
  if (op === 'redenoise') return t('viewer.chain_op_redenoise')
  return op || ''
}

function formatDenoise(v) {
  if (v === null || v === undefined || v === '') return ''
  const n = Number(v)
  if (!Number.isFinite(n) || n < 0 || n > 1) return ''
  return n.toFixed(2)
}

function loraSummary(json) {
  const parts = parseLoras(json).map(r => `${r.name} (${r.weight})`)
  if (!parts.length) return null
  const full = parts.join(', ')
  if (full.length <= 80) return { text: full, full }
  const kept = []
  let len = 0
  for (const p of parts) {
    if (kept.length && len + p.length > 60) break
    kept.push(p)
    len += p.length + 2
  }
  return { text: kept.join(', ') + ', …', full }
}

const chainRows = computed(() => chain.value.map((s, i) => {
  const denoise = formatDenoise(s.denoise)
  const loras = loraSummary(s.loras)
  return {
    num: s.step || i + 1,
    name: s.preset || opLabel(s.op),
    model: s.model || t('viewer.chain_default_model'),
    denoise,
    loraText: loras ? loras.text : '',
    loraFull: loras ? loras.full : '',
    hasLoras: !!loras,
  }
}))

const isFit = computed(() => zoomMode.value === 'fit')
const zoomLabel = computed(() => isFit.value ? t('viewer.zoom_fit') : zoomPct.value + '%')

const viewportW = ref(window.innerWidth)
const viewportH = ref(window.innerHeight)
const availW = computed(() => viewportW.value * 0.95)
const availH = computed(() => viewportH.value * (infoExpanded.value ? 0.8 : 0.95))

const renderedW = computed(() => naturalW.value ? (naturalW.value * zoomPct.value) / 100 : 0)
const renderedH = computed(() => naturalH.value ? (naturalH.value * zoomPct.value) / 100 : 0)

const panEnabled = computed(() => {
  if (isFit.value || !naturalW.value) return false
  return renderedW.value > availW.value || renderedH.value > availH.value
})

const imgStyle = computed(() => {
  if (isFit.value || !naturalW.value) return null
  return {
    width: Math.round(renderedW.value) + 'px',
    transform: `translate(${Math.round(panX.value)}px, ${Math.round(panY.value)}px)`,
  }
})

function clampPan() {
  const maxX = Math.max(0, (renderedW.value - availW.value) / 2)
  const maxY = Math.max(0, (renderedH.value - availH.value) / 2)
  panX.value = Math.min(maxX, Math.max(-maxX, panX.value))
  panY.value = Math.min(maxY, Math.max(-maxY, panY.value))
}

function fitPercent() {
  if (!naturalW.value || !naturalH.value) return 100
  const scale = Math.min(availW.value / naturalW.value, availH.value / naturalH.value)
  return Math.floor(scale * 100)
}

function enterZoom(pct) {
  zoomMode.value = 'pct'
  zoomPct.value = Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, pct))
  clampPan()
  zoomMenuOpen.value = false
}

function setFit() {
  zoomMode.value = 'fit'
  zoomPct.value = 100
  panX.value = 0
  panY.value = 0
  dragging.value = false
  zoomMenuOpen.value = false
}

function zoomBy(delta) {
  const base = isFit.value ? Math.floor(fitPercent() / 10) * 10 : zoomPct.value
  enterZoom(base + delta)
}

function onZoomIndicatorClick() {
  if (!isFit.value) {
    setFit()
    return
  }
  zoomMenuOpen.value = !zoomMenuOpen.value
}

function toggleFit100() {
  if (suppressDbl) return
  if (isFit.value) enterZoom(100)
  else setFit()
}

function onImgLoad(e) {
  naturalW.value = e.target.naturalWidth
  naturalH.value = e.target.naturalHeight
  if (!isFit.value) clampPan()
}

function onPointerDown(e) {
  if (!panEnabled.value || e.button !== 0) return
  dragging.value = true
  dragMovedDist = 0
  dragLastX = e.clientX
  dragLastY = e.clientY
  e.target.setPointerCapture(e.pointerId)
}

function onPointerMove(e) {
  if (!dragging.value) return
  panX.value += e.clientX - dragLastX
  panY.value += e.clientY - dragLastY
  dragMovedDist += Math.abs(e.clientX - dragLastX) + Math.abs(e.clientY - dragLastY)
  dragLastX = e.clientX
  dragLastY = e.clientY
  clampPan()
}

function onPointerUp() {
  if (dragging.value && dragMovedDist > 4) {
    suppressDbl = true
    clearTimeout(suppressDblTimer)
    suppressDblTimer = setTimeout(() => { suppressDbl = false }, 300)
  }
  dragging.value = false
}

function onWheel(e) {
  if (!e.ctrlKey) return
  e.preventDefault()
  zoomBy(e.deltaY < 0 ? 10 : -10)
}

function onResize() {
  viewportW.value = window.innerWidth
  viewportH.value = window.innerHeight
  if (!isFit.value) clampPan()
}

watch(() => props.imageBase64, () => {
  naturalW.value = 0
  naturalH.value = 0
  setFit()
})

watch(infoExpanded, () => {
  if (!isFit.value) clampPan()
})

function onKeydown(e) {
  if (e.key === 'Escape') emit('close')
  if (e.key === 'ArrowLeft' && props.hasPrev) emit('prev')
  if (e.key === 'ArrowRight' && props.hasNext) emit('next')
  if (e.key === 'Delete') emit('delete')
}

onMounted(() => {
  document.addEventListener('keydown', onKeydown)
  window.addEventListener('resize', onResize)
})
onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
  window.removeEventListener('resize', onResize)
  clearTimeout(suppressDblTimer)
})
</script>

<template>
  <div class="image-viewer-overlay" @click="$emit('close')">
    <div class="image-viewer-stage" @wheel="onWheel" @pointerdown="zoomMenuOpen = false">
      <img
        :src="imgSrc"
        :alt="t('viewer.full_size')"
        :class="{ 'with-bar': item && infoExpanded, zoomed: !isFit && naturalW > 0, pannable: panEnabled, panning: dragging }"
        :style="imgStyle"
        @click.stop
        @dblclick.stop="toggleFit100"
        @dragstart.prevent
        @load="onImgLoad"
        @pointerdown="onPointerDown"
        @pointermove="onPointerMove"
        @pointerup="onPointerUp"
        @pointercancel="onPointerUp"
      />
    </div>
    <button class="image-viewer-close" @click="$emit('close')">&times;</button>
    <div v-if="item" class="image-viewer-tools" @click.stop>
      <button
        class="image-viewer-tool-btn"
        :class="{ active: infoExpanded }"
        :title="t('viewer.info_toggle')"
        :aria-label="t('viewer.info_toggle')"
        :aria-expanded="infoExpanded"
        @click="infoExpanded = !infoExpanded"
      >&#9432;</button>
      <template v-if="showActions && !infoExpanded">
        <button class="image-viewer-tool-btn" @click="$emit('remix')">{{ t('viewer.remix') }}</button>
        <button class="image-viewer-tool-btn" @click="$emit('export')">{{ t('viewer.export') }}</button>
        <button
          class="image-viewer-tool-btn tool-danger"
          :class="{ confirm: confirmDelete }"
          @click="$emit('delete')"
        >{{ confirmDelete ? t('viewer.confirm_delete') : t('viewer.delete') }}</button>
      </template>
    </div>
    <div class="image-viewer-zoom" @click.stop>
      <button
        class="image-viewer-zoom-btn"
        :title="t('viewer.zoom_out')"
        :aria-label="t('viewer.zoom_out')"
        :disabled="isFit && fitPercent() <= ZOOM_MIN"
        @click="zoomBy(-10)"
      >&minus;</button>
      <button class="image-viewer-zoom-btn image-viewer-zoom-label" :title="t('viewer.zoom_presets')" :aria-expanded="zoomMenuOpen" aria-haspopup="menu" @click="onZoomIndicatorClick">{{ zoomLabel }}</button>
      <button
        class="image-viewer-zoom-btn"
        :title="t('viewer.zoom_in')"
        :aria-label="t('viewer.zoom_in')"
        @click="zoomBy(10)"
      >&plus;</button>
      <div v-if="zoomMenuOpen" class="image-viewer-zoom-menu" role="menu">
        <button :class="{ active: isFit }" @click="setFit">{{ t('viewer.zoom_fit') }}</button>
        <button
          v-for="p in ZOOM_PRESETS"
          :key="p"
          :class="{ active: !isFit && zoomPct === p }"
          @click="enterZoom(p)"
        >{{ p }}%</button>
      </div>
    </div>
    <button v-if="hasPrev" class="image-viewer-nav image-viewer-prev" @click.stop="$emit('prev')">&lsaquo;</button>
    <button v-if="hasNext" class="image-viewer-nav image-viewer-next" @click.stop="$emit('next')">&rsaquo;</button>
    <div v-if="item && infoExpanded" class="image-viewer-bar" @click.stop>
      <div class="viewer-bar-content">
        <div v-if="item.prompt" class="viewer-prompt" :class="{ expanded: promptExpanded }" @click="promptExpanded = !promptExpanded">
          {{ item.prompt }}
        </div>
        <div class="viewer-params">
          <span v-if="item.sampler">{{ item.sampler }}</span>
          <span v-if="item.steps">Steps: {{ item.steps }}</span>
          <span v-if="item.cfg_scale">CFG: {{ item.cfg_scale }}</span>
          <span v-if="item.seed != null">Seed: {{ item.seed }}</span>
          <span v-if="item.denoising">D: {{ item.denoising }}</span>
          <span v-if="item.width && item.height">{{ item.width }}&times;{{ item.height }}</span>
        </div>
        <div v-if="chainRows.length" class="viewer-chain">
          <div class="viewer-chain-title">{{ chainHeader }}</div>
          <div v-for="row in chainRows" :key="row.num" class="viewer-chain-row">
            <span class="viewer-chain-num">{{ t('viewer.chain_step', { num: row.num }) }}</span>
            <span v-if="row.name" class="viewer-chain-name">{{ row.name }}</span>
            <span v-if="row.model">{{ row.model }}</span>
            <span v-if="row.denoise">{{ t('viewer.chain_denoise', { value: row.denoise }) }}</span>
            <span v-if="row.hasLoras" :title="row.loraFull">{{ t('viewer.chain_loras') }}: {{ row.loraText }}</span>
          </div>
        </div>
        <div v-else-if="metaOnlyLine" class="viewer-params">
          <span>{{ metaOnlyLine }}</span>
        </div>
        <div v-if="showActions" class="viewer-actions">
          <button class="btn btn-primary btn-sm" @click="$emit('remix')">{{ t('viewer.remix') }}</button>
          <button class="btn btn-secondary btn-sm" @click="$emit('export')">{{ t('viewer.export') }}</button>
          <button class="btn btn-danger btn-sm" @click="$emit('delete')">{{ confirmDelete ? t('viewer.confirm_delete') : t('viewer.delete') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>
