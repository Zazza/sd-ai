import { ref } from 'vue'
import { api } from '../api.js'

const presets = ref([])
const presetTypes = ref([])
const compoundPresets = ref([])

export function usePresets() {
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

  return { presets, presetTypes, compoundPresets, loadPresets }
}
