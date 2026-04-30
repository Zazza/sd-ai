<script setup>
import { onMounted, onUnmounted } from 'vue'

const props = defineProps({
  imageBase64: { type: String, required: true },
  hasPrev: { type: Boolean, default: false },
  hasNext: { type: Boolean, default: false },
})
const emit = defineEmits(['close', 'prev', 'next'])

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
    <img :src="'data:image/png;base64,' + imageBase64" alt="Full size" @click.stop />
    <button class="image-viewer-close" @click="$emit('close')">&times;</button>
    <button v-if="hasPrev" class="image-viewer-nav image-viewer-prev" @click.stop="$emit('prev')">&lsaquo;</button>
    <button v-if="hasNext" class="image-viewer-nav image-viewer-next" @click.stop="$emit('next')">&rsaquo;</button>
  </div>
</template>
