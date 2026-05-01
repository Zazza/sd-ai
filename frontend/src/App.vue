<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { WindowSetSystemDefaultTheme } from './wailsjs/runtime/runtime'
import { Diamond, Sparkles, LayoutGrid, Sliders, Settings, RotateCcw, Download, FolderOpen, Sun, Moon } from 'lucide-vue-next'
import { api } from './api.js'
import UnifiedPresetsPage from './components/UnifiedPresetsPage.vue'
import UnifiedGeneratePage from './components/UnifiedGeneratePage.vue'
import ExportPage from './components/ExportPage.vue'
import SettingsPage from './components/SettingsPage.vue'
import SceneEditorPage from './components/SceneEditorPage.vue'
import AppFooter from './components/AppFooter.vue'
import FileBrowserPage from './components/FileBrowserPage.vue'

const page = ref('generate')
const resetKey = ref(0)
const confirmReset = ref(false)
const resetting = ref(false)
let resetTimer = null

const generateTab = ref('')
const generateKey = ref(0)
const theme = ref('dark')

function toggleTheme() {
  theme.value = theme.value === 'dark' ? 'light' : 'dark'
  document.documentElement.setAttribute('data-theme', theme.value)
  api.updateSettings({ theme: theme.value }).catch(() => {})
}

async function resetAll() {
  if (resetting.value) return
  if (!confirmReset.value) {
    confirmReset.value = true
    resetTimer = setTimeout(() => { confirmReset.value = false }, 3000)
    return
  }
  clearTimeout(resetTimer)
  confirmReset.value = false
  resetting.value = true
  try {
    await api.updateSettings({
      gen_preset_id: '',
      gen_type_id: '',
      gen_description: '',
      gen_negative: '',
      gen_extra_prompt: '',
      gen_extra_negative: '',
      gen_mode: 'preset',
      gen_compound_preset_id: '',
      fi_mode: 'img2img',
      fi_preset_id: '',
      fi_compound_preset_id: '',
      fi_gen_mode: 'preset',
      fi_denoising: '0.5',
      fi_extra_negative: '',
      fi_analyze_mode: 'quick',
      batch_preset_id: '',
      batch_compound_preset_id: '',
      batch_mode: 'preset',
      batch_prompt: '',
      batch_negative: '',
      batch_count: '',
      batch_output_folder: '',
      test_prompt: '',
      test_negative: '',
      test_mode: 'preset',
      test_sampler: '',
      test_schedule_type: '',
      test_steps: '',
      test_cfg_scale: '',
      test_width: '',
      test_height: '',
    })
    api.clearLastImage()
    resetKey.value++
  } catch (e) {
    console.error('Reset failed:', e)
  } finally {
    resetting.value = false
  }
}

function onBrowserNavigate(target) {
  if (target.tab) {
    generateTab.value = target.tab
    generateKey.value++
  }
  page.value = target.page
}

const currentPage = computed(() => {
  switch (page.value) {
    case 'generate': return UnifiedGeneratePage
    case 'export': return ExportPage
    case 'scene': return SceneEditorPage
    case 'presets': return UnifiedPresetsPage
    case 'settings': return SettingsPage
    case 'browser': return FileBrowserPage
    default: return UnifiedGeneratePage
  }
})

onMounted(async () => {
  WindowSetSystemDefaultTheme()
  try {
    const s = await api.getSettings()
    if (s?.theme) {
      theme.value = s.theme
      document.documentElement.setAttribute('data-theme', s.theme)
    }
  } catch {}
})

onUnmounted(() => {
  clearTimeout(resetTimer)
})
</script>

<template>
  <div class="app">
    <div class="app-body">
    <aside class="sidebar">
      <div class="sidebar-logo">
        <Diamond :size="20" class="icon" /> SD Studio
      </div>
      <nav class="sidebar-nav">
        <div class="sidebar-group">
          <div class="sidebar-group-label">Generation</div>
          <a class="sidebar-link" :class="{ active: page === 'generate' }" @click="page = 'generate'">
            <Sparkles :size="16" class="icon" /> Generate
          </a>
          <a class="sidebar-link" :class="{ active: page === 'export' }" @click="page = 'export'">
            <Download :size="16" class="icon" /> Export
          </a>
          <a class="sidebar-link" :class="{ active: page === 'scene' }" @click="page = 'scene'">
            <LayoutGrid :size="16" class="icon" /> Multi-Scene
          </a>
        </div>
        <div class="sidebar-group">
          <div class="sidebar-group-label">Tools</div>
          <a class="sidebar-link" :class="{ active: page === 'browser' }" @click="page = 'browser'">
            <FolderOpen :size="16" class="icon" /> File Browser
          </a>
        </div>
        <div class="sidebar-group">
          <div class="sidebar-group-label">Management</div>
          <a class="sidebar-link" :class="{ active: page === 'presets' }" @click="page = 'presets'">
            <Sliders :size="16" class="icon" /> Presets
          </a>
          <a class="sidebar-link" :class="{ active: page === 'settings' }" @click="page = 'settings'">
            <Settings :size="16" class="icon" /> Settings
          </a>
        </div>
      </nav>
      <div class="sidebar-footer">
        <button class="sidebar-theme-btn" @click="toggleTheme" :title="theme === 'dark' ? 'Light mode' : 'Dark mode'">
          <Sun v-if="theme === 'dark'" :size="14" class="icon" />
          <Moon v-else :size="14" class="icon" />
        </button>
        <button class="sidebar-reset-btn" :class="{ confirm: confirmReset }" @click="resetAll">
          <RotateCcw :size="14" class="icon" /> {{ confirmReset ? 'Confirm?' : 'Reset All' }}
        </button>
      </div>
    </aside>
    <main class="main">
      <UnifiedGeneratePage v-if="page === 'generate'" :key="resetKey + '-' + generateKey" :initial-tab="generateTab" />
      <ExportPage v-else-if="page === 'export'" />
      <FileBrowserPage v-else-if="page === 'browser'" @navigate="onBrowserNavigate" />
      <component v-else :is="currentPage" />
    </main>
    </div>
    <AppFooter @navigate="onBrowserNavigate" />
  </div>
</template>
