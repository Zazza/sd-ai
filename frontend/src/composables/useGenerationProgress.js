import { ref, onMounted, onUnmounted } from 'vue'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'
import { api } from '../api.js'

export function useGenerationProgress() {
  const llmStatus = ref('')
  const sdProgress = ref(null)
  const preview = ref('')

  function onLLMStatus(data) {
    llmStatus.value = data?.status || ''
  }

  function onSDProgress(data) {
    sdProgress.value = data
    if (data?.preview) {
      preview.value = data.preview.startsWith('data:') ? data.preview : 'data:image/png;base64,' + data.preview
    }
  }

  async function interrupt() {
    try {
      await api.interruptGeneration()
    } catch {}
  }

  function reset() {
    llmStatus.value = ''
    sdProgress.value = null
    preview.value = ''
  }

  const isGenerating = () => llmStatus.value === 'thinking' || (sdProgress.value && sdProgress.value.progress > 0)

  onMounted(() => {
    EventsOn('llm:status', onLLMStatus)
    EventsOn('sd:progress', onSDProgress)
  })

  onUnmounted(() => {
    EventsOff('llm:status')
    EventsOff('sd:progress')
  })

  return { llmStatus, sdProgress, preview, interrupt, reset, isGenerating }
}
