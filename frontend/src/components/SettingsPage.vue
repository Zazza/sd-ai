<script setup>
import { ref, reactive, watch, onMounted } from 'vue'
import { api } from '../api.js'
import ToggleSwitch from './ToggleSwitch.vue'
import PinModal from './PinModal.vue'

const props = defineProps({
  initialTab: { type: String, default: 'connection' }
})

const activeTab = ref(props.initialTab || 'connection')

const defaultURLs = {
  lmstudio: 'http://localhost:1234',
  ollama: 'http://localhost:11434',
  llamacpp: 'http://localhost:8081',
}

const backendLabel = {
  lmstudio: 'LM Studio',
  ollama: 'Ollama',
  llamacpp: 'llama.cpp',
}

const connectionForm = reactive({
  llm_url: '',
  sd_url: '',
  llm_backend: 'lmstudio',
  llm_keep_alive: '5m',
  llm_num_gpu: '0',
  llm_max_tokens: '256',
})
const connectionSaved = ref(false)
const connectionError = ref('')

const sdModels = ref([])
const sdLoRAs = ref([])
const sdLoading = ref(false)
const sdError = ref('')

const llmModels = ref([])
const llmLoading = ref(false)
const llmError = ref('')

const connectionLLMModels = ref([])
const connectionLLMLoading = ref(false)

const generateModel = ref('')
const analyzeModel = ref('')

const generateParams = reactive({
  temperature: 0.4,
  num_ctx: 4096,
  num_predict: 256,
  top_p: 0.9,
  num_thread: 0,
})

const analyzeParams = reactive({
  temperature: 0.4,
  num_ctx: 4096,
  num_predict: 256,
  top_p: 0.9,
  num_thread: 0,
})

const llmParamsSaved = ref(false)
const llmParamsError = ref('')

const kidsMode = ref(false)
const showPinModal = ref(false)
const pinMode = ref('set')
const pinError = ref('')
const kidsCategories = ref([])
const kidsCatPinModal = ref(false)
const kidsCatPinError = ref('')
const kidsCatPending = ref(null)

const generationForm = reactive({
  preview_mode: false,
  preview_width: 512,
  preview_height: 512,
})
const generationSaved = ref(false)
const generationError = ref('')

const promptInstruction = ref('')
const promptInstructionSaved = ref(false)
const promptInstructionError = ref('')
const defaultPromptInstruction = ref('')

const analyzeSystemPrompt = ref('')
const analyzeSinglePrompt = ref('')
const analyzeChainPrompts = reactive(['', '', '', ''])
const analyzeUseChain = ref(true)
const analyzeSaved = ref(false)
const analyzeError = ref('')
const defaultAnalyzePrompts = ref(null)

const rembgForm = reactive({
  rembg_url: '',
})
const rembgSaved = ref(false)
const rembgError = ref('')
const rembgTesting = ref(false)
const rembgStatus = ref('')

watch(() => connectionForm.llm_backend, (newVal, oldVal) => {
  if (oldVal && defaultURLs[oldVal] && connectionForm.llm_url === defaultURLs[oldVal]) {
    connectionForm.llm_url = defaultURLs[newVal] || defaultURLs.lmstudio
  }
  loadConnectionLLMModels()
})

async function loadSettings() {
  try {
    const settings = await api.getSettings()
    connectionForm.llm_url = settings.llm_url || ''
    connectionForm.sd_url = settings.sd_url || ''
    connectionForm.llm_backend = settings.llm_backend || 'lmstudio'
    connectionForm.llm_keep_alive = settings.llm_keep_alive || '5m'
    connectionForm.llm_num_gpu = settings.llm_num_gpu || '0'
    connectionForm.llm_max_tokens = settings.llm_max_tokens || '256'
    generateModel.value = settings.llm_generate_model || settings.sd_prompt_model || ''
    analyzeModel.value = settings.llm_analyze_model || settings.vision_model || ''
    generateParams.temperature = parseFloat(settings.llm_generate_temperature) || 0.4
    generateParams.num_ctx = parseInt(settings.llm_generate_num_ctx) || 4096
    generateParams.num_predict = parseInt(settings.llm_generate_num_predict) || 256
    generateParams.top_p = parseFloat(settings.llm_generate_top_p) || 0.9
    generateParams.num_thread = parseInt(settings.llm_generate_num_thread) || 0
    analyzeParams.temperature = parseFloat(settings.llm_analyze_temperature) || 0.4
    analyzeParams.num_ctx = parseInt(settings.llm_analyze_num_ctx) || 4096
    analyzeParams.num_predict = parseInt(settings.llm_analyze_num_predict) || 256
    analyzeParams.top_p = parseFloat(settings.llm_analyze_top_p) || 0.9
    analyzeParams.num_thread = parseInt(settings.llm_analyze_num_thread) || 0
    loadConnectionLLMModels()
    kidsMode.value = await api.isKidsModeActive()
    loadKidsCategories()
    generationForm.preview_mode = settings.preview_mode === 'true'
    generationForm.preview_width = parseInt(settings.preview_width) || 512
    generationForm.preview_height = parseInt(settings.preview_height) || 512
    promptInstruction.value = settings.sd_prompt_instruction || ''
    rembgForm.rembg_url = settings.rembg_url || ''
  } catch (e) {
    console.error('loadSettings:', e)
  }

  try {
    defaultPromptInstruction.value = await api.getDefaultPromptInstruction()
  } catch (e) {
    console.error('getDefaultPromptInstruction:', e)
  }

  if (!promptInstruction.value && defaultPromptInstruction.value) {
    promptInstruction.value = defaultPromptInstruction.value
  }

  try {
    const settings = await api.getSettings()
    analyzeSystemPrompt.value = settings.analyze_system_prompt || ''
    analyzeSinglePrompt.value = settings.analyze_prompt || ''
    analyzeUseChain.value = settings.analyze_use_chain !== 'false'
    for (let i = 0; i < 4; i++) {
      analyzeChainPrompts[i] = settings['analyze_chain_' + (i + 1)] || ''
    }
  } catch (e) {
    console.error('loadAnalyzeSettings:', e)
  }

  try {
    defaultAnalyzePrompts.value = await api.getDefaultAnalyzePrompts()
  } catch (e) {
    console.error('getDefaultAnalyzePrompts:', e)
  }

  if (!analyzeSystemPrompt.value && defaultAnalyzePrompts.value) {
    analyzeSystemPrompt.value = defaultAnalyzePrompts.value.system_prompt
  }
  if (!analyzeSinglePrompt.value && defaultAnalyzePrompts.value) {
    analyzeSinglePrompt.value = defaultAnalyzePrompts.value.single_prompt
  }
  for (let i = 0; i < 4; i++) {
    if (!analyzeChainPrompts[i] && defaultAnalyzePrompts.value) {
      analyzeChainPrompts[i] = defaultAnalyzePrompts.value.chain_prompts[i] || ''
    }
  }
}

async function loadConnectionLLMModels() {
  if (connectionForm.llm_backend === 'llamacpp') return
  connectionLLMLoading.value = true
  try {
    connectionLLMModels.value = await api.getLLMModels() || []
  } catch {
    connectionLLMModels.value = []
  } finally {
    connectionLLMLoading.value = false
  }
}

async function saveConnection() {
  connectionSaved.value = false
  connectionError.value = ''
  try {
    await api.updateSettings({
      llm_url: connectionForm.llm_url,
      sd_url: connectionForm.sd_url,
      llm_backend: connectionForm.llm_backend,
      llm_keep_alive: String(connectionForm.llm_keep_alive),
      llm_num_gpu: String(connectionForm.llm_num_gpu),
      llm_max_tokens: String(connectionForm.llm_max_tokens),
      llm_generate_model: generateModel.value,
      llm_analyze_model: analyzeModel.value,
    })
    connectionSaved.value = true
  } catch (e) {
    connectionError.value = String(e)
  }
}

async function saveLLMParams() {
  llmParamsSaved.value = false
  llmParamsError.value = ''
  try {
    await api.updateSettings({
      llm_generate_temperature: String(generateParams.temperature),
      llm_generate_num_ctx: String(generateParams.num_ctx),
      llm_generate_num_predict: String(generateParams.num_predict),
      llm_generate_top_p: String(generateParams.top_p),
      llm_generate_num_thread: String(generateParams.num_thread),
      llm_analyze_temperature: String(analyzeParams.temperature),
      llm_analyze_num_ctx: String(analyzeParams.num_ctx),
      llm_analyze_num_predict: String(analyzeParams.num_predict),
      llm_analyze_top_p: String(analyzeParams.top_p),
      llm_analyze_num_thread: String(analyzeParams.num_thread),
    })
    llmParamsSaved.value = true
  } catch (e) {
    llmParamsError.value = String(e)
  }
}

async function loadSD() {
  sdLoading.value = true
  sdError.value = ''
  try {
    const [m, l] = await Promise.allSettled([api.getModels(), api.getLoRAs()])
    if (m.status === 'fulfilled') sdModels.value = m.value
    else sdError.value = 'Cannot load models — is Stable Diffusion running?'
    if (l.status === 'fulfilled') sdLoRAs.value = l.value
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
    const label = backendLabel[connectionForm.llm_backend] || 'LLM backend'
    llmError.value = `Cannot load models — is ${label} running?`
  } finally {
    llmLoading.value = false
  }
}

function switchTab(tab) {
  activeTab.value = tab
  if (tab === 'advanced' && sdModels.value.length === 0 && !sdError.value) loadSD()
  if (tab === 'connection' && llmModels.value.length === 0 && !llmError.value) loadLLM()
}

function onKidsToggle(val) {
  pinError.value = ''
  if (val) {
    pinMode.value = 'set'
    showPinModal.value = true
  } else {
    pinMode.value = 'verify'
    showPinModal.value = true
  }
}

async function onPinConfirm(pin) {
  pinError.value = ''
  const enabled = pinMode.value === 'set'
  try {
    await api.setKidsMode(enabled, pin)
    kidsMode.value = enabled
    showPinModal.value = false
    if (enabled) loadKidsCategories()
  } catch (e) {
    pinError.value = String(e) || 'Error'
  }
}

function onPinCancel() {
  showPinModal.value = false
  pinError.value = ''
}

async function loadKidsCategories() {
  try {
    kidsCategories.value = await api.getKidsCategories()
  } catch {}
}

function onKidsCatToggle(cat) {
  if (cat.alwaysOn || !kidsMode.value) return
  kidsCatPending.value = { name: cat.name, enabled: !cat.enabled }
  kidsCatPinError.value = ''
  kidsCatPinModal.value = true
}

async function onKidsCatPinConfirm(pin) {
  if (!kidsCatPending.value) return
  kidsCatPinError.value = ''
  try {
    await api.setKidsCategory(kidsCatPending.value.name, kidsCatPending.value.enabled, pin)
    const cat = kidsCategories.value.find(c => c.name === kidsCatPending.value.name)
    if (cat) cat.enabled = kidsCatPending.value.enabled
    kidsCatPinModal.value = false
    kidsCatPending.value = null
  } catch (e) {
    kidsCatPinError.value = String(e) || 'Error'
  }
}

function onKidsCatPinCancel() {
  kidsCatPinModal.value = false
  kidsCatPending.value = null
  kidsCatPinError.value = ''
}

async function saveGeneration() {
  generationSaved.value = false
  generationError.value = ''
  try {
    await api.updateSettings({
      preview_mode: generationForm.preview_mode ? 'true' : 'false',
      preview_width: String(generationForm.preview_width),
      preview_height: String(generationForm.preview_height),
    })
    generationSaved.value = true
  } catch (e) {
    generationError.value = String(e)
  }
}

async function savePromptInstruction() {
  promptInstructionSaved.value = false
  promptInstructionError.value = ''
  try {
    await api.updateSettings({ sd_prompt_instruction: promptInstruction.value })
    promptInstructionSaved.value = true
  } catch (e) {
    promptInstructionError.value = String(e)
  }
}

async function saveAnalyzePrompts() {
  analyzeSaved.value = false
  analyzeError.value = ''
  try {
    const data = {
      analyze_system_prompt: analyzeSystemPrompt.value,
      analyze_prompt: analyzeSinglePrompt.value,
      analyze_use_chain: analyzeUseChain.value ? 'true' : 'false',
    }
    for (let i = 0; i < 4; i++) {
      data['analyze_chain_' + (i + 1)] = analyzeChainPrompts[i]
    }
    await api.updateSettings(data)
    analyzeSaved.value = true
  } catch (e) {
    analyzeError.value = String(e)
  }
}

function resetAnalyzePrompts() {
  if (!defaultAnalyzePrompts.value) return
  analyzeSystemPrompt.value = defaultAnalyzePrompts.value.system_prompt
  analyzeSinglePrompt.value = defaultAnalyzePrompts.value.single_prompt
  for (let i = 0; i < 4; i++) {
    analyzeChainPrompts[i] = defaultAnalyzePrompts.value.chain_prompts[i] || ''
  }
}

async function saveRembg() {
  rembgSaved.value = false
  rembgError.value = ''
  rembgStatus.value = ''
  try {
    await api.updateSettings({ rembg_url: rembgForm.rembg_url })
    rembgSaved.value = true
  } catch (e) {
    rembgError.value = String(e)
  }
}

async function testRembg() {
  rembgTesting.value = true
  rembgStatus.value = ''
  rembgError.value = ''
  try {
    await api.updateSettings({ rembg_url: rembgForm.rembg_url })
    await api.checkRembg()
    rembgStatus.value = 'ok'
  } catch (e) {
    rembgStatus.value = 'error'
    rembgError.value = String(e)
  } finally {
    rembgTesting.value = false
  }
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
      <button class="tab" :class="{ active: activeTab === 'generation' }" @click="switchTab('generation')">Generation</button>
      <button class="tab" :class="{ active: activeTab === 'analyze' }" @click="switchTab('analyze')">Analyze</button>
      <button class="tab" :class="{ active: activeTab === 'safety' }" @click="switchTab('safety')">Safety</button>
      <button class="tab" :class="{ active: activeTab === 'advanced' }" @click="switchTab('advanced')">Advanced</button>
    </div>

    <!-- Connection Tab -->
    <div v-if="activeTab === 'connection'" class="card">
      <div v-if="connectionSaved" class="status status-success" style="margin-bottom: 16px;">Settings saved. Changes apply immediately.</div>
      <div v-if="connectionError" class="status status-error" style="margin-bottom: 16px;">{{ connectionError }}</div>

      <div class="form-group">
        <label class="form-label">LLM Backend</label>
        <select class="form-input" v-model="connectionForm.llm_backend">
          <option value="lmstudio">LM Studio</option>
          <option value="ollama">Ollama</option>
          <option value="llamacpp">llama.cpp</option>
        </select>
      </div>

      <div class="form-group">
        <label class="form-label">LLM URL</label>
        <input class="form-input" v-model="connectionForm.llm_url" :placeholder="defaultURLs[connectionForm.llm_backend]" />
      </div>

      <div class="form-group" v-if="connectionForm.llm_backend !== 'llamacpp'">
        <label class="form-label">Model for Generate</label>
        <div style="display: flex; gap: 8px;">
          <select class="form-input" v-model="generateModel" style="flex: 1;">
            <option value="">default</option>
            <option v-for="m in connectionLLMModels" :key="m.id" :value="m.id">{{ m.id }}</option>
          </select>
          <button class="btn btn-secondary btn-sm" @click="loadConnectionLLMModels" :disabled="connectionLLMLoading">
            {{ connectionLLMLoading ? '...' : 'Refresh' }}
          </button>
        </div>
      </div>

      <div class="form-group" v-if="connectionForm.llm_backend !== 'llamacpp'">
        <label class="form-label">Model for Analyze</label>
        <div style="display: flex; gap: 8px;">
          <select class="form-input" v-model="analyzeModel" style="flex: 1;">
            <option value="">Same as Generate</option>
            <option v-for="m in connectionLLMModels" :key="m.id" :value="m.id">{{ m.id }}</option>
          </select>
        </div>
      </div>

      <div class="form-group" v-if="connectionForm.llm_backend === 'llamacpp'">
        <div style="color: var(--text-dim); font-size: 13px; padding: 8px; background: var(--surface-2); border-radius: 6px;">
          llama.cpp uses a single loaded model. Model selection is not available.
        </div>
      </div>

      <div class="form-group">
        <label class="form-label">Max Tokens (prompt generation)</label>
        <input class="form-input" type="number" v-model="connectionForm.llm_max_tokens" placeholder="256" min="64" max="8192" />
      </div>

      <div class="form-group">
        <label class="form-label">Stable Diffusion URL</label>
        <input class="form-input" v-model="connectionForm.sd_url" placeholder="http://localhost:7860" />
      </div>

      <!-- Ollama-specific -->
      <template v-if="connectionForm.llm_backend === 'ollama'">
        <div class="form-group">
          <label class="form-label">Keep Alive</label>
          <input class="form-input" v-model="connectionForm.llm_keep_alive" placeholder="5m" />
        </div>
        <div class="form-group">
          <label class="form-label">GPU Layers (num_gpu)</label>
          <input class="form-input" type="number" v-model="connectionForm.llm_num_gpu" placeholder="0" />
        </div>
      </template>

      <!-- llama.cpp-specific -->
      <template v-if="connectionForm.llm_backend === 'llamacpp'">
        <div class="form-group">
          <label class="form-label">GPU Layers (num_gpu)</label>
          <input class="form-input" type="number" v-model="connectionForm.llm_num_gpu" placeholder="0" />
        </div>
      </template>

      <button class="btn btn-primary" @click="saveConnection">Save Connection Settings</button>

      <div class="card" style="margin-top: 24px;">
        <h3 style="color: var(--text-bright); margin-bottom: 16px;">Rembg (Background Removal)</h3>
        <div v-if="rembgSaved" class="status status-success" style="margin-bottom: 16px;">URL saved.</div>
        <div v-if="rembgError" class="status status-error" style="margin-bottom: 16px;">{{ rembgError }}</div>

        <div style="color: var(--text-dim); font-size: 13px; margin-bottom: 12px; line-height: 1.5;">
          Rembg is an AI background removal service. Run it on any device with Python/CUDA:
          <code style="background: var(--surface-2); padding: 2px 6px; border-radius: 4px; font-size: 12px;">rembg s --host 0.0.0.0 --port 7000</code>
          <br>Then enter its URL below. Required for clean multi-character compositing.
        </div>

        <div class="form-group">
          <label class="form-label">Rembg Server URL</label>
          <div style="display: flex; gap: 8px;">
            <input class="form-input" v-model="rembgForm.rembg_url" placeholder="http://192.168.1.100:7000" style="flex: 1;" />
            <button class="btn btn-secondary btn-sm" @click="testRembg" :disabled="rembgTesting || !rembgForm.rembg_url">
              {{ rembgTesting ? 'Testing...' : 'Test' }}
            </button>
          </div>
        </div>

        <div v-if="rembgStatus === 'ok'" style="color: #4ade80; font-size: 13px; margin-bottom: 12px;">
          Connection successful — rembg is running.
        </div>
        <div v-if="rembgStatus === 'error'" style="color: #f87171; font-size: 13px; margin-bottom: 12px;">
          Connection failed — check URL and make sure rembg is running.
        </div>

        <div v-if="!rembgForm.rembg_url" style="color: var(--text-dim); font-size: 12px; padding: 8px; background: var(--surface-2); border-radius: 6px; margin-bottom: 12px;">
          Without rembg, Go-based white background removal will be used (lower quality, visible artifacts on edges).
        </div>

        <button class="btn btn-primary" @click="saveRembg">Save Rembg URL</button>
      </div>

      <div class="card" style="margin-top: 24px;">
        <h3 style="color: var(--text-bright); margin-bottom: 16px;">LLM Models</h3>
        <div v-if="llmError" class="status status-error" style="margin-bottom: 8px;">{{ llmError }}</div>
        <div v-if="llmLoading" style="text-align: center; padding: 20px;"><span class="spinner"></span></div>
        <div v-else-if="llmModels.length === 0" style="color: var(--text-dim);">No models available</div>
        <div v-for="m in llmModels" :key="m.id" style="padding: 8px 0; border-bottom: 1px solid var(--border);">
          <div style="color: var(--text-bright); font-size: 13px;">{{ m.id }}</div>
          <div style="color: var(--text-dim); font-size: 11px;">{{ m.object }}</div>
        </div>
        <button class="btn btn-secondary btn-sm" style="margin-top: 12px;" @click="loadLLM" :disabled="llmLoading">
          {{ llmLoading ? 'Loading...' : 'Refresh Models' }}
        </button>
      </div>

      <div class="card" style="margin-top: 16px;">
        <h3 style="color: var(--text-bright); margin-bottom: 16px;">LLM Parameters</h3>
        <div v-if="llmParamsSaved" class="status status-success" style="margin-bottom: 16px;">Parameters saved.</div>
        <div v-if="llmParamsError" class="status status-error" style="margin-bottom: 16px;">{{ llmParamsError }}</div>

        <h4 style="color: var(--text-bright); margin: 16px 0 8px; font-size: 14px;">Generate Parameters</h4>
        <div class="form-row-2">
          <div class="form-group">
            <label class="form-label">Temperature</label>
            <input class="form-input" type="number" v-model.number="generateParams.temperature" step="0.1" min="0" max="2" />
          </div>
          <div class="form-group">
            <label class="form-label">Context Size (num_ctx)</label>
            <input class="form-input" type="number" v-model.number="generateParams.num_ctx" step="512" min="512" max="32768" />
          </div>
        </div>
        <div class="form-row-2">
          <div class="form-group">
            <label class="form-label">Max Predict (num_predict)</label>
            <input class="form-input" type="number" v-model.number="generateParams.num_predict" min="32" max="8192" />
          </div>
          <div class="form-group">
            <label class="form-label">Top P</label>
            <input class="form-input" type="number" v-model.number="generateParams.top_p" step="0.05" min="0" max="1" />
          </div>
        </div>
        <div class="form-row-2">
          <div class="form-group">
            <label class="form-label">Threads (0=auto)</label>
            <input class="form-input" type="number" v-model.number="generateParams.num_thread" min="0" max="64" />
          </div>
        </div>

        <h4 style="color: var(--text-bright); margin: 16px 0 8px; font-size: 14px;">Analyze Parameters</h4>
        <div class="form-row-2">
          <div class="form-group">
            <label class="form-label">Temperature</label>
            <input class="form-input" type="number" v-model.number="analyzeParams.temperature" step="0.1" min="0" max="2" />
          </div>
          <div class="form-group">
            <label class="form-label">Context Size (num_ctx)</label>
            <input class="form-input" type="number" v-model.number="analyzeParams.num_ctx" step="512" min="512" max="32768" />
          </div>
        </div>
        <div class="form-row-2">
          <div class="form-group">
            <label class="form-label">Max Predict (num_predict)</label>
            <input class="form-input" type="number" v-model.number="analyzeParams.num_predict" min="32" max="8192" />
          </div>
          <div class="form-group">
            <label class="form-label">Top P</label>
            <input class="form-input" type="number" v-model.number="analyzeParams.top_p" step="0.05" min="0" max="1" />
          </div>
        </div>
        <div class="form-row-2">
          <div class="form-group">
            <label class="form-label">Threads (0=auto)</label>
            <input class="form-input" type="number" v-model.number="analyzeParams.num_thread" min="0" max="64" />
          </div>
        </div>

        <button class="btn btn-primary" @click="saveLLMParams" style="margin-top: 8px;">Save Parameters</button>
      </div>
    </div>

    <!-- Generation Tab (merged: Generation + Prompt) -->
    <div v-if="activeTab === 'generation'">
      <div class="card">
        <h3 style="color: var(--text-bright); margin-bottom: 16px;">Preview Generation</h3>
        <div v-if="generationSaved" class="status status-success" style="margin-bottom: 16px;">Settings saved.</div>
        <div v-if="generationError" class="status status-error" style="margin-bottom: 16px;">{{ generationError }}</div>

        <div style="display: flex; align-items: center; gap: 16px; margin-bottom: 16px;">
          <ToggleSwitch v-model="generationForm.preview_mode" />
          <div>
            <div style="color: var(--text-bright); font-weight: 500;">{{ generationForm.preview_mode ? 'Enabled' : 'Disabled' }}</div>
            <div style="color: var(--text-dim); font-size: 12px; margin-top: 2px;">
              Generate a small preview first, then upscale to full resolution
            </div>
          </div>
        </div>

        <template v-if="generationForm.preview_mode">
          <div class="form-row-2">
            <div class="form-group">
              <label class="form-label">Preview Width</label>
              <input class="form-input" type="number" v-model.number="generationForm.preview_width" step="64" min="64" max="2048" />
            </div>
            <div class="form-group">
              <label class="form-label">Preview Height</label>
              <input class="form-input" type="number" v-model.number="generationForm.preview_height" step="64" min="64" max="2048" />
            </div>
          </div>
        </template>

        <button class="btn btn-primary" @click="saveGeneration">Save Generation Settings</button>
      </div>

      <div class="card" style="margin-top: 16px;">
        <h3 style="color: var(--text-bright); margin-bottom: 16px;">SD Prompt Instruction</h3>
        <div v-if="promptInstructionSaved" class="status status-success" style="margin-bottom: 16px;">Instruction saved.</div>
        <div v-if="promptInstructionError" class="status status-error" style="margin-bottom: 16px;">{{ promptInstructionError }}</div>
        <div style="color: var(--text-dim); font-size: 13px; margin-bottom: 12px; line-height: 1.5;">
          This instruction is sent to the LLM when generating SD prompts. It defines how the LLM should merge your preset with your description into a valid Stable Diffusion prompt.
        </div>
        <div class="form-group">
          <textarea class="form-textarea" v-model="promptInstruction" rows="16" style="font-family: monospace; font-size: 12px; line-height: 1.5;"></textarea>
        </div>
        <div style="display: flex; gap: 8px;">
          <button class="btn btn-primary" @click="savePromptInstruction">Save Instruction</button>
          <button class="btn btn-secondary" @click="promptInstruction = defaultPromptInstruction">Reset to Default</button>
        </div>
      </div>
    </div>

    <!-- Analyze Tab (unchanged) -->
    <div v-if="activeTab === 'analyze'" class="card">
      <h3 style="color: var(--text-bright); margin-bottom: 16px;">Image Analysis Prompts</h3>
      <div v-if="analyzeSaved" class="status status-success" style="margin-bottom: 16px;">Prompts saved.</div>
      <div v-if="analyzeError" class="status status-error" style="margin-bottom: 16px;">{{ analyzeError }}</div>

      <div style="display: flex; align-items: center; gap: 16px; margin-bottom: 16px;">
        <ToggleSwitch v-model="analyzeUseChain" />
        <div>
          <div style="color: var(--text-bright); font-weight: 500;">{{ analyzeUseChain ? 'Chain Mode (4 steps)' : 'Single Prompt' }}</div>
          <div style="color: var(--text-dim); font-size: 12px; margin-top: 2px;">
            Chain mode runs 4 sequential vision calls for 30-50% more detail
          </div>
        </div>
      </div>

      <div class="form-group">
        <label class="form-label">System Prompt</label>
        <textarea class="form-textarea" v-model="analyzeSystemPrompt" rows="3" style="font-family: monospace; font-size: 12px; line-height: 1.5;"></textarea>
      </div>

      <template v-if="!analyzeUseChain">
        <div class="form-group">
          <label class="form-label">Single Analysis Prompt</label>
          <textarea class="form-textarea" v-model="analyzeSinglePrompt" rows="10" style="font-family: monospace; font-size: 12px; line-height: 1.5;"></textarea>
        </div>
      </template>

      <template v-if="analyzeUseChain">
        <div class="form-group">
          <label class="form-label">Step 1 — Main Subject</label>
          <textarea class="form-textarea" v-model="analyzeChainPrompts[0]" rows="3" style="font-family: monospace; font-size: 12px; line-height: 1.5;"></textarea>
        </div>
        <div class="form-group">
          <label class="form-label">Step 2 — Background & Setting</label>
          <textarea class="form-textarea" v-model="analyzeChainPrompts[1]" rows="3" style="font-family: monospace; font-size: 12px; line-height: 1.5;"></textarea>
        </div>
        <div class="form-group">
          <label class="form-label">Step 3 — Colors, Lighting & Style</label>
          <textarea class="form-textarea" v-model="analyzeChainPrompts[2]" rows="3" style="font-family: monospace; font-size: 12px; line-height: 1.5;"></textarea>
        </div>
        <div class="form-group">
          <label class="form-label">Step 4 — Details & Final Tags</label>
          <textarea class="form-textarea" v-model="analyzeChainPrompts[3]" rows="3" style="font-family: monospace; font-size: 12px; line-height: 1.5;"></textarea>
        </div>
      </template>

      <div style="display: flex; gap: 8px;">
        <button class="btn btn-primary" @click="saveAnalyzePrompts">Save Prompts</button>
        <button class="btn btn-secondary" @click="resetAnalyzePrompts">Reset to Default</button>
      </div>
    </div>

    <!-- Safety Tab -->
    <div v-if="activeTab === 'safety'" class="card">
      <h3 style="color: var(--text-bright); margin-bottom: 16px;">Kids Mode</h3>
      <div style="display: flex; align-items: center; gap: 16px; margin-bottom: 16px;">
        <ToggleSwitch :modelValue="kidsMode" @update:modelValue="onKidsToggle" />
        <div>
          <div style="color: var(--text-bright); font-weight: 500;">{{ kidsMode ? 'Enabled' : 'Disabled' }}</div>
          <div style="color: var(--text-dim); font-size: 12px; margin-top: 2px;">
            Content filter for child-safe image generation
          </div>
        </div>
      </div>

      <div v-if="kidsMode && kidsCategories.length > 0" style="margin-bottom: 16px; padding: 12px; background: var(--surface-2); border: 1px solid var(--border); border-radius: var(--radius-sm);">
        <div style="color: var(--text-bright); font-weight: 500; margin-bottom: 10px;">Filter Categories</div>
        <div style="display: flex; flex-direction: column; gap: 8px;">
          <div v-for="cat in kidsCategories" :key="cat.name" style="display: flex; align-items: center; justify-content: space-between; padding: 4px 0;">
            <div>
              <span style="color: var(--text-bright); font-size: 13px;">{{ cat.label }}</span>
              <span v-if="cat.alwaysOn" style="color: var(--text-dim); font-size: 11px; margin-left: 6px;">(always on)</span>
            </div>
            <ToggleSwitch :modelValue="cat.enabled" @update:modelValue="onKidsCatToggle(cat)" :disabled="cat.alwaysOn" />
          </div>
        </div>
        <div style="color: var(--text-dim); font-size: 11px; margin-top: 8px;">PIN required to change categories</div>
      </div>

      <div style="color: var(--text-dim); font-size: 13px; line-height: 1.6;">
        When enabled, Kids Mode applies multiple safety layers:
        <ul style="margin: 8px 0 0 16px; padding: 0;">
          <li>Filters user input for restricted content</li>
          <li>Instructs the LLM to generate only safe prompts</li>
          <li>Filters LLM output for inappropriate tags</li>
          <li>Forces negative prompt safety tags</li>
        </ul>
        <div style="margin-top: 8px;">Protected by 4-digit PIN to prevent children from disabling it.</div>
      </div>
    </div>

    <!-- Advanced Tab (SD Models + Rembg) -->
    <div v-if="activeTab === 'advanced'">
      <div v-if="sdError" class="status status-error">{{ sdError }}</div>

      <div class="form-row-2">
        <div class="card">
          <h3 style="color: var(--text-bright); margin-bottom: 16px;">SD Models</h3>
          <div v-if="sdLoading" style="text-align: center; padding: 20px;"><span class="spinner"></span></div>
          <div v-else-if="sdModels.length === 0" style="color: var(--text-dim);">No models loaded</div>
          <div v-for="m in sdModels" :key="m.model_name" style="padding: 8px 0; border-bottom: 1px solid var(--border);">
            <div style="color: var(--text-bright); font-size: 13px;">{{ m.title }}</div>
            <div style="color: var(--text-dim); font-size: 11px;">{{ m.model_name }}</div>
          </div>
        </div>

        <div class="card">
          <h3 style="color: var(--text-bright); margin-bottom: 16px;">SD LoRA</h3>
          <div v-if="sdLoading" style="text-align: center; padding: 20px;"><span class="spinner"></span></div>
          <div v-else-if="sdLoRAs.length === 0" style="color: var(--text-dim);">No LoRA models found</div>
          <div v-else style="display: flex; flex-wrap: wrap; gap: 6px;">
            <span v-for="l in sdLoRAs" :key="l.name"
              style="padding: 4px 10px; background: var(--surface-2); border: 1px solid var(--border); border-radius: 4px; font-size: 12px; color: var(--text);"
              :title="l.path">
              {{ l.name }}
            </span>
          </div>
        </div>
      </div>

      <button class="btn btn-secondary" style="margin-top: 16px;" @click="loadSD" :disabled="sdLoading">
        {{ sdLoading ? 'Loading...' : 'Refresh' }}
      </button>
    </div>

    <PinModal v-if="showPinModal" :mode="pinMode" :error="pinError" @confirm="onPinConfirm" @cancel="onPinCancel" />
    <PinModal v-if="kidsCatPinModal" mode="verify" :error="kidsCatPinError" @confirm="onKidsCatPinConfirm" @cancel="onKidsCatPinCancel" />
  </div>
</template>
