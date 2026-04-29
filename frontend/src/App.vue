<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { WindowSetSystemDefaultTheme, EventsOn, EventsOff } from './wailsjs/runtime/runtime'
import PresetsPage from './components/PresetsPage.vue'
import GeneratePage from './components/GeneratePage.vue'
import SettingsPage from './components/SettingsPage.vue'
import BatchPage from './components/BatchPage.vue'
import TestPage from './components/TestPage.vue'
import CompoundPresetsPage from './components/CompoundPresetsPage.vue'
import GenerateFromImagePage from './components/GenerateFromImagePage.vue'
import SceneEditorPage from './components/SceneEditorPage.vue'
import AppFooter from './components/AppFooter.vue'

const page = ref('generate')
const batchProps = ref({})

const currentPage = computed(() => {
  switch (page.value) {
    case 'presets': return PresetsPage
    case 'generate': return GeneratePage
    case 'batch': return BatchPage
    case 'test': return TestPage
    case 'pipelines': return CompoundPresetsPage
    case 'from-image': return GenerateFromImagePage
    case 'scene': return SceneEditorPage
    case 'settings': return SettingsPage
    default: return GeneratePage
  }
})

function onNavigateToBatch(data) {
  batchProps.value = {
    prefillPrompt: data?.prefillPrompt || '',
    prefillNegative: data?.prefillNegative || '',
    prefillPresetId: data?.prefillPresetId || null,
    prefillCompoundPresetId: data?.prefillCompoundPresetId || null,
  }
  page.value = 'batch'
}

onMounted(() => {
  WindowSetSystemDefaultTheme()
  EventsOn('navigate:batch', onNavigateToBatch)
})

onUnmounted(() => {
  EventsOff('navigate:batch')
})
</script>

<template>
  <div class="app">
    <div class="app-body">
    <aside class="sidebar">
      <div class="sidebar-logo">
        <span>&#9670;</span> SD Studio
      </div>
      <nav class="sidebar-nav">
        <a class="sidebar-link" :class="{ active: page === 'generate' }" @click="page = 'generate'">
          &#9733; Generate
        </a>
        <a class="sidebar-link" :class="{ active: page === 'from-image' }" @click="page = 'from-image'">
          &#9678; From Image
        </a>
        <a class="sidebar-link" :class="{ active: page === 'batch' }" @click="page = 'batch'">
          &#9638; Batch
        </a>
        <a class="sidebar-link" :class="{ active: page === 'test' }" @click="page = 'test'">
          &#9888; Test
        </a>
        <a class="sidebar-link" :class="{ active: page === 'pipelines' }" @click="page = 'pipelines'">
          &#10227; Pipelines
        </a>
        <a class="sidebar-link" :class="{ active: page === 'scene' }" @click="page = 'scene'">
          &#9674; Multi-Scene
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
      <KeepAlive v-if="page !== 'batch' && page !== 'test' && page !== 'pipelines' && page !== 'from-image' && page !== 'scene'">
        <component :is="currentPage" />
      </KeepAlive>
      <BatchPage v-else-if="page === 'batch'" v-bind="batchProps" :key="JSON.stringify(batchProps)" />
      <TestPage v-else-if="page === 'test'" />
      <CompoundPresetsPage v-else-if="page === 'pipelines'" />
      <GenerateFromImagePage v-else-if="page === 'from-image'" />
      <SceneEditorPage v-else-if="page === 'scene'" />
    </main>
    </div>
    <AppFooter />
  </div>
</template>
