<script setup>
import { ref } from 'vue'
import PresetsPage from './PresetsPage.vue'
import CompoundPresetsPage from './CompoundPresetsPage.vue'

const activeTab = ref('presets')
</script>

<template>
  <div class="page-enter">
    <div class="tabs">
      <button class="tab" :class="{ active: activeTab === 'presets' }" @click="activeTab = 'presets'">Presets</button>
      <button class="tab" :class="{ active: activeTab === 'pipelines' }" @click="activeTab = 'pipelines'">Pipelines</button>
    </div>
    <PresetsPage v-if="activeTab === 'presets'" />
    <CompoundPresetsPage v-else />
  </div>
</template>
