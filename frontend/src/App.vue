<script setup>
import { ref, computed } from 'vue'
import PresetsPage from './components/PresetsPage.vue'
import GeneratePage from './components/GeneratePage.vue'
import SettingsPage from './components/SettingsPage.vue'

const page = ref('generate')

const currentPage = computed(() => {
  switch (page.value) {
    case 'presets': return PresetsPage
    case 'generate': return GeneratePage
    case 'settings': return SettingsPage
    default: return GeneratePage
  }
})
</script>

<template>
  <div class="app">
    <aside class="sidebar">
      <div class="sidebar-logo">
        <span>&#9670;</span> SD Studio
      </div>
      <nav class="sidebar-nav">
        <a class="sidebar-link" :class="{ active: page === 'generate' }" @click="page = 'generate'">
          &#9733; Generate
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
      <component :is="currentPage" />
    </main>
  </div>
</template>
