<script setup>
import { ref } from 'vue'
import { t } from '../i18n/index.js'
import PresetsPage from './PresetsPage.vue'
import CompoundPresetsPage from './CompoundPresetsPage.vue'

const activeTab = ref('presets')
</script>

<template>
  <div class="page-enter">
    <div class="tabs">
      <button class="tab" :class="{ active: activeTab === 'presets' }" @click="activeTab = 'presets'">{{ t('unified.tab_presets') }}</button>
      <button class="tab" :class="{ active: activeTab === 'pipelines' }" @click="activeTab = 'pipelines'">{{ t('unified.tab_pipelines') }}</button>
    </div>
    <PresetsPage v-if="activeTab === 'presets'" />
    <CompoundPresetsPage v-else />
  </div>
</template>
