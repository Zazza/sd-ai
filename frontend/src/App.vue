<script setup>
import { ref, computed } from 'vue'
import { WindowMinimise, Quit } from './wailsjs/runtime/runtime'
import PresetsPage from './components/PresetsPage.vue'
import GeneratePage from './components/GeneratePage.vue'
import AnalyzePage from './components/AnalyzePage.vue'
import SettingsPage from './components/SettingsPage.vue'

const page = ref('generate')

const currentPage = computed(() => {
  switch (page.value) {
    case 'presets': return PresetsPage
    case 'generate': return GeneratePage
    case 'analyze': return AnalyzePage
    case 'settings': return SettingsPage
    default: return GeneratePage
  }
})

const minimize = () => WindowMinimise()
const close = () => Quit()
</script>

<template>
  <div class="app">
    <div class="titlebar">
      <div class="titlebar-drag">
        <span class="titlebar-logo">&#9670;</span> SD Studio
      </div>
      <div class="titlebar-controls">
        <button class="titlebar-btn" @click="minimize">&#8722;</button>
        <button class="titlebar-btn titlebar-btn-close" @click="close">&#10005;</button>
      </div>
    </div>
    <div class="app-body">
    <aside class="sidebar">
      <div class="sidebar-logo">
        <span>&#9670;</span> SD Studio
      </div>
      <nav class="sidebar-nav">
        <a class="sidebar-link" :class="{ active: page === 'generate' }" @click="page = 'generate'">
          &#9733; Generate
        </a>
        <a class="sidebar-link" :class="{ active: page === 'analyze' }" @click="page = 'analyze'">
          &#9673; Analyze
        </a>
        <a class="sidebar-link" :class="{ active: page === 'presets' }" @click="page = 'presets'">
          &#9776; Presets
        </a>
        <a class="sidebar-link" :class="{ active: page === 'settings' }" @click="page = 'settings'">
          &#9881; Settings
        </a>
      </nav>
    </aside>
    <main class="main">
      <KeepAlive>
        <component :is="currentPage" />
      </KeepAlive>
    </main>
    </div>
  </div>
</template>
