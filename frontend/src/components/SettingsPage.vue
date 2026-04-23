<script setup>
import { ref, reactive, onMounted } from 'vue'
import { api } from '../api.js'

const activeTab = ref('connection')

const connectionForm = reactive({
  llm_url: '',
  sd_url: '',
  llm_model: '',
  sd_prompt_model: '',
})
const connectionSaved = ref(false)
const connectionError = ref('')

const sdModels = ref([])
const sdSamplers = ref([])
const sdLoading = ref(false)
const sdError = ref('')

const llmModels = ref([])
const llmLoading = ref(false)
const llmError = ref('')

async function loadSettings() {
  try {
    const settings = await api.getSettings()
    connectionForm.llm_url = settings.llm_url || ''
    connectionForm.sd_url = settings.sd_url || ''
    connectionForm.llm_model = settings.llm_model || ''
    connectionForm.sd_prompt_model = settings.sd_prompt_model || ''
  } catch {}
}

async function saveConnection() {
  connectionSaved.value = false
  connectionError.value = ''
  try {
    await api.updateSettings({
      llm_url: connectionForm.llm_url,
      sd_url: connectionForm.sd_url,
      llm_model: connectionForm.llm_model,
      sd_prompt_model: connectionForm.sd_prompt_model,
    })
    connectionSaved.value = true
  } catch (e) {
    connectionError.value = e.message
  }
}

async function loadSD() {
  sdLoading.value = true
  sdError.value = ''
  try {
    const [m, s] = await Promise.allSettled([api.getModels(), api.getSamplers()])
    if (m.status === 'fulfilled') sdModels.value = m.value
    else sdError.value = 'Cannot load models — is Stable Diffusion running?'
    if (s.status === 'fulfilled') sdSamplers.value = s.value
  } finally {
    sdLoading.value = false
  }
}

async function loadLLM() {
  llmLoading.value = true
  llmError.value = ''
  try {
    llmModels.value = await api.getLLMModels() || []
  } catch (e) {
    llmError.value = 'Cannot load models — is LM Studio running?'
  } finally {
    llmLoading.value = false
  }
}

function switchTab(tab) {
  activeTab.value = tab
  if (tab === 'sd' && sdModels.value.length === 0 && !sdError.value) loadSD()
  if (tab === 'llm' && llmModels.value.length === 0 && !llmError.value) loadLLM()
}

onMounted(loadSettings)
</script>

<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">Settings</h1>
    </div>

    <div class="tabs">
      <button class="tab" :class="{ active: activeTab === 'connection' }" @click="switchTab('connection')">Connection</button>
      <button class="tab" :class="{ active: activeTab === 'sd' }" @click="switchTab('sd')">Stable Diffusion</button>
      <button class="tab" :class="{ active: activeTab === 'llm' }" @click="switchTab('llm')">LLM</button>
    </div>

    <!-- Connection Tab -->
    <div v-if="activeTab === 'connection'" class="card">
      <div v-if="connectionSaved" class="status status-success" style="margin-bottom: 16px;">Settings saved. Changes apply immediately.</div>
      <div v-if="connectionError" class="status status-error" style="margin-bottom: 16px;">{{ connectionError }}</div>

      <div class="form-group">
        <label class="form-label">LLM URL</label>
        <input class="form-input" v-model="connectionForm.llm_url" placeholder="http://localhost:1234" />
      </div>

      <div class="form-group">
        <label class="form-label">LLM Model (for chat)</label>
        <input class="form-input" v-model="connectionForm.llm_model" placeholder="openai/gpt-oss-20b" />
      </div>

      <div class="form-group">
        <label class="form-label">SD Prompt Model (model for prompt generation)</label>
        <input class="form-input" v-model="connectionForm.sd_prompt_model" placeholder="default" />
      </div>

      <div class="form-group">
        <label class="form-label">Stable Diffusion URL</label>
        <input class="form-input" v-model="connectionForm.sd_url" placeholder="http://localhost:7860" />
      </div>

      <button class="btn btn-primary" @click="saveConnection">Save Connection Settings</button>
    </div>

    <!-- SD Tab -->
    <div v-if="activeTab === 'sd'">
      <div v-if="sdError" class="status status-error">{{ sdError }}</div>

      <div class="form-row-2" style="margin-top: 16px;">
        <div class="card">
          <h3 style="color: var(--text-bright); margin-bottom: 16px;">Available Models</h3>
          <div v-if="sdLoading" style="text-align: center; padding: 20px;"><span class="spinner"></span></div>
          <div v-else-if="sdModels.length === 0" style="color: var(--text-dim);">No models loaded</div>
          <div v-for="m in sdModels" :key="m.model_name" style="padding: 8px 0; border-bottom: 1px solid var(--border);">
            <div style="color: var(--text-bright); font-size: 13px;">{{ m.title }}</div>
            <div style="color: var(--text-dim); font-size: 11px;">{{ m.model_name }}</div>
          </div>
        </div>

        <div class="card">
          <h3 style="color: var(--text-bright); margin-bottom: 16px;">Samplers</h3>
          <div v-if="sdLoading" style="text-align: center; padding: 20px;"><span class="spinner"></span></div>
          <div v-else style="display: flex; flex-wrap: wrap; gap: 6px;">
            <span v-for="s in sdSamplers" :key="s.name"
              style="padding: 4px 10px; background: var(--surface-2); border: 1px solid var(--border); border-radius: 4px; font-size: 12px; color: var(--text);">
              {{ s.name }}
            </span>
          </div>
        </div>
      </div>

      <button class="btn btn-secondary" style="margin-top: 16px;" @click="loadSD" :disabled="sdLoading">
        {{ sdLoading ? 'Loading...' : 'Refresh' }}
      </button>
    </div>

    <!-- LLM Tab -->
    <div v-if="activeTab === 'llm'">
      <div v-if="llmError" class="status status-error">{{ llmError }}</div>

      <div class="card" style="margin-top: 16px;">
        <h3 style="color: var(--text-bright); margin-bottom: 16px;">Available Models</h3>
        <div v-if="llmLoading" style="text-align: center; padding: 20px;"><span class="spinner"></span></div>
        <div v-else-if="llmModels.length === 0" style="color: var(--text-dim);">No models available</div>
        <div v-for="m in llmModels" :key="m.id" style="padding: 8px 0; border-bottom: 1px solid var(--border);">
          <div style="color: var(--text-bright); font-size: 13px;">{{ m.id }}</div>
          <div style="color: var(--text-dim); font-size: 11px;">{{ m.object }}</div>
        </div>
      </div>

      <button class="btn btn-secondary" style="margin-top: 16px;" @click="loadLLM" :disabled="llmLoading">
        {{ llmLoading ? 'Loading...' : 'Refresh' }}
      </button>
    </div>
  </div>
</template>
