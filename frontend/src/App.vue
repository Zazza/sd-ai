<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { WindowSetSystemDefaultTheme } from './wailsjs/runtime/runtime'
import { Diamond, Sparkles, LayoutGrid, Sliders, Settings, RotateCcw, Download, FolderOpen, Sun, Moon, ImagePlus, Columns } from 'lucide-vue-next'
import { api } from './api.js'
import { t } from './i18n/index.js'
import UnifiedPresetsPage from './components/UnifiedPresetsPage.vue'
import UnifiedGeneratePage from './components/UnifiedGeneratePage.vue'
import GenerateFromImagePage from './components/GenerateFromImagePage.vue'
import ComparePage from './components/ComparePage.vue'
import ExportPage from './components/ExportPage.vue'
import SettingsPage from './components/SettingsPage.vue'
import SceneEditorPage from './components/SceneEditorPage.vue'
import AppFooter from './components/AppFooter.vue'
import FileBrowserPage from './components/FileBrowserPage.vue'

const page = ref('generate')
const resetKey = ref(0)
const confirmReset = ref(false)
const resetting = ref(false)
const isResetting = ref(false)
let resetTimer = null

const generateTab = ref('')
const generateKey = ref(0)
const settingsTab = ref('')
const settingsKey = ref(0)
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
  isResetting.value = true
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
      gen_resolution_id: '',
      gen_hires_profile_id: '',
      gen_count: '',
      fi_mode: 'img2img',
      fi_preset_id: '',
      fi_compound_preset_id: '',
      fi_gen_mode: 'preset',
      fi_denoising: '0.5',
      fi_extra_negative: '',
      fi_analyze_mode: 'quick',
      fi_mask_padding: '',
      fi_mask_feather: '',
      test_prompt: '',
      test_negative: '',
      test_mode: 'preset',
      test_sampler: '',
      test_schedule_type: '',
      test_steps: '',
      test_cfg_scale: '',
      test_width: '',
      test_height: '',
      test_resolution_id: '',
      test_hires_profile_id: '',
    })
    await api.clearLastImage()
    resetKey.value++
    await nextTick()
  } catch (e) {
    console.error('Reset failed:', e)
  } finally {
    isResetting.value = false
    resetting.value = false
  }
}

function onNavigate(target) {
  if (target.page === 'settings' && target.tab) {
    settingsTab.value = target.tab
  } else if (target.page === 'generate' && target.tab) {
    generateTab.value = target.tab
    generateKey.value++
  } else if (target.page === 'remix') {
    // handled directly
  }
  page.value = target.page
  if (target.page === 'settings') {
    setTimeout(() => { settingsTab.value = '' }, 100)
  }
}

const currentPage = computed(() => {
  switch (page.value) {
    case 'generate': return UnifiedGeneratePage
    case 'remix': return GenerateFromImagePage
    case 'compare': return ComparePage
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
  window.addEventListener('beforeunload', () => {
    api.saveWindowLayout(0).catch(() => {})
  })
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
        <Diamond :size="20" class="icon" /> {{ t('app.title') }}
      </div>
      <nav class="sidebar-nav">
        <div class="sidebar-group">
          <div class="sidebar-group-label">{{ t('app.nav_create') }}</div>
          <a class="sidebar-link" :class="{ active: page === 'generate' }" @click="page = 'generate'">
            <Sparkles :size="16" class="icon" /> {{ t('app.nav_generate') }}
          </a>
          <a class="sidebar-link" :class="{ active: page === 'remix' }" @click="page = 'remix'">
            <ImagePlus :size="16" class="icon" /> {{ t('app.nav_remix') }}
          </a>
          <a class="sidebar-link" :class="{ active: page === 'compare' }" @click="page = 'compare'">
            <Columns :size="16" class="icon" /> {{ t('app.nav_compare') }}
          </a>
          <a class="sidebar-link" :class="{ active: page === 'scene' }" @click="page = 'scene'">
            <LayoutGrid :size="16" class="icon" /> {{ t('app.nav_multi_scene') }}
          </a>
        </div>
        <div class="sidebar-group">
          <div class="sidebar-group-label">{{ t('app.nav_library') }}</div>
          <a class="sidebar-link" :class="{ active: page === 'browser' }" @click="page = 'browser'">
            <FolderOpen :size="16" class="icon" /> {{ t('app.nav_my_images') }}
          </a>
          <a class="sidebar-link" :class="{ active: page === 'export' }" @click="page = 'export'">
            <Download :size="16" class="icon" /> {{ t('app.nav_export') }}
          </a>
        </div>
        <div class="sidebar-group">
          <div class="sidebar-group-label">{{ t('app.nav_setup') }}</div>
          <a class="sidebar-link" :class="{ active: page === 'presets' }" @click="page = 'presets'">
            <Sliders :size="16" class="icon" /> {{ t('app.nav_styles') }}
          </a>
          <a class="sidebar-link" :class="{ active: page === 'settings' }" @click="page = 'settings'">
            <Settings :size="16" class="icon" /> {{ t('app.nav_settings') }}
          </a>
        </div>
      </nav>
      <div class="sidebar-footer">
        <button class="sidebar-theme-btn" @click="toggleTheme" :title="theme === 'dark' ? t('app.theme_light') : t('app.theme_dark')">
          <Sun v-if="theme === 'dark'" :size="14" class="icon" />
          <Moon v-else :size="14" class="icon" />
        </button>
        <button class="sidebar-reset-btn" :class="{ confirm: confirmReset }" @click="resetAll">
          <RotateCcw :size="14" class="icon" /> {{ confirmReset ? t('app.confirm') : t('app.reset_all') }}
        </button>
      </div>
    </aside>
    <main class="main">
      <SettingsPage v-show="page === 'settings'" :initial-tab="settingsTab" @navigate="onNavigate" />
      <template v-if="page !== 'settings'">
        <UnifiedGeneratePage v-if="page === 'generate'" :key="resetKey + '-' + generateKey" :initial-tab="generateTab" :resetting="isResetting" />
        <GenerateFromImagePage v-else-if="page === 'remix'" />
        <ExportPage v-else-if="page === 'export'" />
        <FileBrowserPage v-else-if="page === 'browser'" @navigate="onNavigate" />
        <component v-else-if="page !== 'compare'" :is="currentPage" />
      </template>
      <ComparePage v-show="page === 'compare'" :active="page === 'compare'" />
    </main>
    </div>
    <AppFooter @navigate="onNavigate" />
  </div>
</template>
