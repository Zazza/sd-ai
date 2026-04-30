<script setup>
import { onMounted, onUnmounted } from 'vue'

const props = defineProps({
  imageBase64: { type: String, required: true },
})
const emit = defineEmits(['close'])

function onKeydown(e) {
  if (e.key === 'Escape') emit('close')
}

onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))
</script>

<template>
  <div class="image-viewer-overlay" @click="$emit('close')">
    <img :src="'data:image/png;base64,' + imageBase64" alt="Full size" @click.stop />
    <button class="image-viewer-close" @click="$emit('close')">&times;</button>
  </div>
</template>
