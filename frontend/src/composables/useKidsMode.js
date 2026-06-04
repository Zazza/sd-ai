import { ref } from 'vue'
import { api } from '../api.js'

const kidsModeActive = ref(false)

export function useKidsMode() {
  async function loadKidsMode() {
    try {
      kidsModeActive.value = await api.isKidsModeActive()
    } catch (e) {
      console.error(e)
    }
  }

  return { kidsModeActive, loadKidsMode }
}
