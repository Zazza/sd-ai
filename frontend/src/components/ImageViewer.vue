<script setup>
import { computed, onMounted, onUnmounted } from 'vue'
import { t } from '../i18n/index.js'

const props = defineProps({
  imageBase64: { type: String, required: true },
  hasPrev: { type: Boolean, default: false },
  hasNext: { type: Boolean, default: false },
})
const emit = defineEmits(['close', 'prev', 'next'])

const imgSrc = computed(() => {
  if (props.imageBase64.startsWith('/api/')) return props.imageBase64
  return 'data:image/png;base64,' + props.imageBase64
})

function onKeydown(e) {
  if (e.key === 'Escape') emit('close')
  if (e.key === 'ArrowLeft' && props.hasPrev) emit('prev')
  if (e.key === 'ArrowRight' && props.hasNext) emit('next')
}

onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))
</script>

<template>
  <div class="image-viewer-overlay" @click="$emit('close')">
    <img :src="imgSrc" :alt="t('viewer.full_size')" @click.stop />
    <button class="image-viewer-close" @click="$emit('close')">&times;</button>
    <button v-if="hasPrev" class="image-viewer-nav image-viewer-prev" @click.stop="$emit('prev')">&lsaquo;</button>
    <button v-if="hasNext" class="image-viewer-nav image-viewer-next" @click.stop="$emit('next')">&rsaquo;</button>
  </div>
</template>
