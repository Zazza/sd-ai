<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { WindowSetSystemDefaultTheme, EventsOn, EventsOff } from './wailsjs/runtime/runtime'
import { Diamond, Sparkles, LayoutGrid, Sliders, Settings } from 'lucide-vue-next'
import UnifiedPresetsPage from './components/UnifiedPresetsPage.vue'
import UnifiedGeneratePage from './components/UnifiedGeneratePage.vue'
import SettingsPage from './components/SettingsPage.vue'
import SceneEditorPage from './components/SceneEditorPage.vue'
import AppFooter from './components/AppFooter.vue'

const page = ref('generate')

const currentPage = computed(() => {
  switch (page.value) {
    case 'generate': return UnifiedGeneratePage
    case 'scene': return SceneEditorPage
    case 'presets': return UnifiedPresetsPage
    case 'settings': return SettingsPage
    default: return UnifiedGeneratePage
  }
})

onMounted(() => {
  WindowSetSystemDefaultTheme()
})

onUnmounted(() => {
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
          <a class="sidebar-link" :class="{ active: page === 'scene' }" @click="page = 'scene'">
            <LayoutGrid :size="16" class="icon" /> Multi-Scene
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
    </aside>
    <main class="main">
      <component :is="currentPage" />
    </main>
    </div>
    <AppFooter />
  </div>
</template>
