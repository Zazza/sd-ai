<script setup>
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { t } from '../i18n/index.js'

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

const imgSrc = computed(() => {
  if (props.imageBase64.startsWith('/api/')) return props.imageBase64
  return 'data:image/png;base64,' + props.imageBase64
})

function onKeydown(e) {
  if (e.key === 'Escape') emit('close')
  if (e.key === 'ArrowLeft' && props.hasPrev) emit('prev')
  if (e.key === 'ArrowRight' && props.hasNext) emit('next')
  if (e.key === 'Delete') emit('delete')
}

onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))
</script>

<template>
  <div class="image-viewer-overlay" @click="$emit('close')">
    <img :src="imgSrc" :alt="t('viewer.full_size')" :class="{ 'with-bar': item }" @click.stop />
    <button class="image-viewer-close" @click="$emit('close')">&times;</button>
    <button v-if="hasPrev" class="image-viewer-nav image-viewer-prev" @click.stop="$emit('prev')">&lsaquo;</button>
    <button v-if="hasNext" class="image-viewer-nav image-viewer-next" @click.stop="$emit('next')">&rsaquo;</button>
    <div v-if="item" class="image-viewer-bar" @click.stop>
      <div class="viewer-bar-content">
        <div v-if="item.prompt" class="viewer-prompt" :class="{ expanded: promptExpanded }" @click="promptExpanded = !promptExpanded">
          {{ item.prompt }}
        </div>
        <div class="viewer-params">
          <span v-if="item.sampler">{{ item.sampler }}</span>
          <span v-if="item.steps">Steps: {{ item.steps }}</span>
          <span v-if="item.cfg_scale">CFG: {{ item.cfg_scale }}</span>
          <span v-if="item.seed">Seed: {{ item.seed }}</span>
          <span v-if="item.denoising">D: {{ item.denoising }}</span>
          <span v-if="item.width && item.height">{{ item.width }}&times;{{ item.height }}</span>
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
