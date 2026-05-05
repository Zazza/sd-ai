<script setup>
import { ref, computed, onMounted, onUnmounted, reactive } from 'vue'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'
import { api } from '../api.js'
import { t } from '../i18n/index.js'
import { useGenerationProgress } from '../composables/useGenerationProgress.js'

const presets = ref([])
const selectedPresetId = ref(null)
const description = ref('')
const negativePrompt = ref('')
const scene = ref(null)
const generating = ref(false)
const { sdProgress, preview, interrupt: interruptGeneration, reset: resetProgress } = useGenerationProgress()
const decomposing = ref(false)
const progress = ref(null)
const error = ref('')
const result = ref(null)
const resultImage = ref('')
const llmAvailable = ref(false)
const sdAvailable = ref(false)
const savedScenes = ref([])
const batchCount = ref(1)
const batchResults = ref([])
const batchCurrent = ref(0)

const presetOptions = computed(() =>
  presets.value.map(p => ({ id: p.id, name: p.name }))
)

function updateCharPosition(index, axis, value) {
  if (!scene.value || !scene.value.characters[index]) return
  scene.value.characters[index].position[axis] = parseFloat(value)
}

function updateCharScale(index, value) {
  if (!scene.value || !scene.value.characters[index]) return
  scene.value.characters[index].scale = parseFloat(value)
}

function removeCharacter(index) {
  if (!scene.value) return
  scene.value.characters.splice(index, 1)
}

function addCharacter() {
  if (!scene.value) return
  const count = scene.value.characters.length
  scene.value.characters.push({
    name: `character_${count + 1}`,
    prompt: '',
    position: { x: 0.5, y: 0.55 },
    scale: 0.4
  })
}

async function loadPresets() {
  try {
    const list = await api.listPresets()
    presets.value = list || []
  } catch (e) {
    error.value = 'Failed to load presets: ' + e
  }
}

async function checkServices() {
  try {
    const status = await api.checkServices()
    llmAvailable.value = status.llm?.available || false
    sdAvailable.value = status.sd?.available || false
  } catch {}
}

async function loadSavedScenes() {
  try {
    const list = await api.listSavedScenes()
    savedScenes.value = list || []
  } catch {}
}

async function loadScene(id) {
  try {
    const saved = await api.getSavedScene(id)
    const parsed = JSON.parse(saved.scene_json)
    scene.value = parsed
    selectedPresetId.value = parsed.preset_id || null
    result.value = null
    resultImage.value = ''
  } catch (e) {
    error.value = 'Failed to load scene: ' + e
  }
}

async function deleteScene(id) {
  try {
    await api.deleteSavedScene(id)
    savedScenes.value = savedScenes.value.filter(s => s.id !== id)
  } catch (e) {
    error.value = 'Delete failed: ' + e
  }
}

async function decompose() {
  if (!description.value.trim()) {
    error.value = t('scene.error_enter_description')
    return
  }
  if (!selectedPresetId.value) {
    error.value = t('scene.error_select_preset')
    return
  }

  decomposing.value = true
  error.value = ''
  scene.value = null
  result.value = null
  resultImage.value = ''

  try {
    const s = await api.decomposeScene({
      description: description.value,
      preset_id: selectedPresetId.value
    })
    if (negativePrompt.value.trim()) {
      s.negative_prompt = negativePrompt.value.trim()
    }
    scene.value = s
  } catch (e) {
    error.value = 'Decomposition failed: ' + e
  } finally {
    decomposing.value = false
  }
}

async function generate() {
  if (!scene.value) return

  const count = Math.max(1, Math.min(batchCount.value || 1, 20))

  generating.value = true
  error.value = ''
  progress.value = null
  result.value = null
  resultImage.value = ''
  batchResults.value = []
  batchCurrent.value = 0
  resetProgress()

  EventsOn('multipass:progress', (p) => {
    progress.value = p
  })

  for (let i = 0; i < count; i++) {
    batchCurrent.value = i + 1
    try {
      const r = await api.generateMultiPass(scene.value)
      if (i === 0) {
        result.value = r
        resultImage.value = r.image
      }
      batchResults.value.push(r)
    } catch (e) {
      error.value = `Image ${i + 1}/${count} failed: ${e}`
      break
    }
  }

  generating.value = false
  EventsOff('multipass:progress')
}

function progressLabel() {
  if (!progress.value) return ''
  const p = progress.value
  const prefix = batchCount.value > 1 ? `[${batchCurrent.value}/${batchCount.value}] ` : ''
  if (p.step === 'background') return prefix + t('scene.generating_background')
  if (p.step === 'character') return prefix + t('scene.generating_character', { current: p.character, total: p.total })
  if (p.step === 'rembg') return prefix + t('scene.removing_background', { current: p.character, total: p.total })
  if (p.step === 'refine') return prefix + t('scene.refining')
  if (p.step === 'done') return prefix + t('scene.done')
  return prefix + p.step
}

async function saveImage() {
  if (!resultImage.value) return
  try {
    await api.saveImage(resultImage.value, 'scene')
  } catch {}
}

async function downloadImage(imageBase64, index) {
  try {
    await api.saveImage(imageBase64, `scene-${index + 1}`)
  } catch {}
}

async function saveScene() {
  if (!scene.value) return
  const name = prompt('Scene name:', description.value.slice(0, 50))
  if (!name) return
  try {
    await api.saveScene({
      name,
      scene_json: JSON.stringify(scene.value)
    })
    loadSavedScenes()
  } catch (e) {
    error.value = 'Save failed: ' + e
  }
}

onMounted(() => {
  loadPresets()
  checkServices()
  loadSavedScenes()
})
</script>

<template>
  <div class="scene-editor">
    <h2>{{ t('scene.title') }}</h2>

    <div v-if="error" class="status" :class="error === 'interrupted' ? 'status-warning' : 'status-error'">{{ error }}</div>

    <!-- Step 1: Description + Preset -->
    <div class="section" v-if="!scene">
      <div class="form-group">
        <label>{{ t('scene.label_preset') }}</label>
        <select v-model="selectedPresetId">
          <option :value="null" disabled>{{ t('scene.select_preset') }}</option>
          <option v-for="p in presetOptions" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
      </div>

      <div class="form-group">
        <label>{{ t('scene.label_description') }}</label>
        <textarea v-model="description" rows="4" :placeholder="t('scene.placeholder_description')"></textarea>
      </div>

      <div class="form-group">
        <label>{{ t('scene.label_negative') }}</label>
        <input v-model="negativePrompt" type="text" :placeholder="t('scene.placeholder_negative')" />
      </div>

      <button @click="decompose" :disabled="decomposing || !llmAvailable || !selectedPresetId" class="btn-primary">
        {{ decomposing ? t('scene.decomposing') : t('scene.btn_decompose') }}
      </button>

      <div v-if="savedScenes.length > 0" class="saved-scenes">
        <h4>{{ t('scene.saved_scenes') }}</h4>
        <div v-for="s in savedScenes" :key="s.id" class="saved-scene-item">
          <div class="saved-scene-info">
            <span class="saved-scene-name" @click="loadScene(s.id)">{{ s.name }}</span>
            <span class="saved-scene-date">{{ s.created_at?.slice(0, 10) }}</span>
          </div>
          <button @click="deleteScene(s.id)" class="btn-danger btn-sm">{{ t('scene.btn_delete') }}</button>
        </div>
      </div>
    </div>

    <!-- Step 2: Scene Editor -->
    <div class="section" v-if="scene">
      <div class="editor-header">
        <h3>{{ t('scene.scene_editor') }}</h3>
        <button @click="scene = null; result = null" class="btn-secondary btn-sm">{{ t('scene.btn_back') }}</button>
      </div>

      <div class="form-group">
        <label>{{ t('scene.label_preset') }}</label>
        <select v-model="selectedPresetId" @change="scene.preset_id = selectedPresetId">
          <option v-for="p in presetOptions" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
      </div>

      <div class="form-group">
        <label>{{ t('scene.label_background') }}</label>
        <textarea v-model="scene.background_prompt" rows="2"></textarea>
      </div>

      <div class="form-group">
        <label>{{ t('scene.label_negative') }}</label>
        <input v-model="scene.negative_prompt" type="text" />
      </div>

      <div class="characters-list">
        <h4>{{ t('scene.characters', { count: scene.characters.length }) }}</h4>
        <button @click="addCharacter" class="btn-secondary btn-sm">{{ t('scene.btn_add_character') }}</button>

        <div v-for="(char, i) in scene.characters" :key="i" class="character-card">
          <div class="char-header">
            <input v-model="char.name" class="char-name" :placeholder="t('scene.placeholder_char_name')" />
            <button @click="removeCharacter(i)" class="btn-danger btn-sm">{{ t('scene.btn_remove') }}</button>
          </div>

          <div class="form-group">
            <label>{{ t('scene.label_char_prompt') }}</label>
            <textarea v-model="char.prompt" rows="2"></textarea>
          </div>

          <div class="char-controls">
            <div class="control-group">
              <label>X Position: {{ char.position.x.toFixed(2) }}</label>
              <input type="range" min="0" max="1" step="0.05" :value="char.position.x"
                @input="updateCharPosition(i, 'x', $event.target.value)" />
            </div>
            <div class="control-group">
              <label>Y Position: {{ char.position.y.toFixed(2) }}</label>
              <input type="range" min="0" max="1" step="0.05" :value="char.position.y"
                @input="updateCharPosition(i, 'y', $event.target.value)" />
            </div>
            <div class="control-group">
              <label>Scale: {{ char.scale.toFixed(2) }}</label>
              <input type="range" min="0.1" max="0.8" step="0.05" :value="char.scale"
                @input="updateCharScale(i, $event.target.value)" />
            </div>
          </div>

          <!-- Visual preview -->
          <div class="preview-box">
            <div class="preview-canvas">
              <div class="preview-char"
                :style="{
                  left: (char.position.x * 100) + '%',
                  top: (char.position.y * 100) + '%',
                  transform: 'translate(-50%, -50%)',
                  width: (char.scale * 100) + 'px',
                  height: (char.scale * 150) + 'px'
                }">
                {{ char.name }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="editor-actions">
        <div class="form-group" style="margin-bottom: 0;">
          <label>{{ t('scene.label_count') }}</label>
          <input type="number" v-model.number="batchCount" min="1" max="20" style="width: 70px;" />
        </div>
        <button @click="generate" :disabled="generating || !sdAvailable" class="btn-primary">
          {{ generating ? progressLabel() : `Generate${batchCount > 1 ? ' x' + batchCount : ''}` }}
        </button>
        <button @click="saveScene" :disabled="!scene" class="btn-secondary">{{ t('scene.btn_save_scene') }}</button>
        <button @click="scene = null; result = null" class="btn-secondary">{{ t('scene.btn_cancel') }}</button>
      </div>

      <div v-if="generating && progress" class="progress-info">
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: progress ? ((progress.step === 'done' ? 1 : (batchCurrent / batchCount || 0)) * 100) + '%' : '0%' }"></div>
        </div>
        <span>{{ progressLabel() }}</span>
        <div v-if="sdProgress && sdProgress.progress > 0" style="margin-top: 8px;">
          <div style="display: flex; justify-content: space-between; margin-bottom: 4px;">
            <span style="color: var(--text-dim); font-size: 12px;">SD: {{ Math.round(sdProgress.progress * 100) }}%</span>
            <span style="color: var(--text-dim); font-size: 12px;">{{ t('progress.sd_step', { current: Math.round(sdProgress.progress * sdProgress.steps), total: sdProgress.steps }) }}</span>
          </div>
          <div style="background: var(--surface-2); border-radius: 4px; overflow: hidden; height: 4px;">
            <div :style="{ width: (sdProgress.progress * 100) + '%', background: 'var(--accent)', height: '100%', transition: 'width 0.3s' }"></div>
          </div>
          <button class="btn btn-sm btn-secondary" @click="interruptGeneration" style="margin-top: 6px; font-size: 11px;">{{ t('progress.btn_interrupt') }}</button>
        </div>
      </div>
    </div>

    <!-- Step 3: Result -->
    <div class="section" v-if="resultImage && batchResults.length <= 1">
      <h3>{{ t('scene.result') }}</h3>
      <div class="result-image">
        <img :src="'data:image/png;base64,' + resultImage" alt="Generated scene" />
      </div>
      <div class="result-actions">
        <button @click="saveImage" class="btn-secondary">{{ t('scene.btn_save_image') }}</button>
        <button @click="scene = null; result = null; resultImage = ''" class="btn-secondary">{{ t('scene.btn_new_scene') }}</button>
      </div>
    </div>

    <!-- Step 3b: Batch Results -->
    <div class="section" v-if="batchResults.length > 1">
      <h3>{{ t('scene.results', { count: batchResults.length }) }}</h3>
      <div class="batch-results-grid">
        <div v-for="(r, i) in batchResults" :key="i" class="batch-result-card">
          <img class="batch-result-image" :src="'data:image/png;base64,' + r.image" :alt="'Result ' + (i + 1)" />
          <div class="batch-result-meta">
            <button @click="downloadImage(r.image, i)" class="btn-secondary btn-sm" style="width: 100%;">{{ t('scene.btn_save') }}</button>
          </div>
        </div>
      </div>
      <div class="result-actions" style="margin-top: 12px;">
        <button @click="scene = null; result = null; resultImage = ''; batchResults = []" class="btn-secondary">{{ t('scene.btn_new_scene') }}</button>
      </div>
    </div>
  </div>
</template>

<script>
export default { name: 'SceneEditorPage' }
</script>

<style scoped>
.scene-editor {
  padding: 20px;
  max-width: 900px;
  margin: 0 auto;
}

h2 { margin-bottom: 20px; }

.status-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.online { color: #4caf50; font-weight: bold; }
.offline { color: #f44336; }

.section {
  background: var(--color-bg-soft, #1e1e2e);
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 16px;
}

.form-group {
  margin-bottom: 12px;
}

.form-group label {
  display: block;
  margin-bottom: 4px;
  font-size: 0.9em;
  color: var(--color-text-secondary, #aaa);
}

.form-group input[type="text"],
.form-group textarea,
.form-group select {
  width: 100%;
  padding: 8px;
  border: 1px solid var(--color-border, #444);
  border-radius: 4px;
  background: var(--color-input-bg, #2a2a3a);
  color: var(--color-text, #eee);
  font-size: 0.95em;
  box-sizing: border-box;
}

textarea { resize: vertical; font-family: inherit; }

.error {
  color: #f44336;
  padding: 8px 12px;
  background: rgba(244, 67, 54, 0.1);
  border-radius: 4px;
  margin-bottom: 12px;
}

.editor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.characters-list {
  margin-top: 16px;
}

.characters-list h4 {
  display: inline;
  margin-right: 12px;
}

.character-card {
  background: var(--color-bg-mute, #252535);
  border-radius: 6px;
  padding: 12px;
  margin: 12px 0;
}

.char-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.char-name {
  font-weight: bold;
  background: transparent;
  border: none;
  color: var(--color-text, #eee);
  font-size: 1em;
  border-bottom: 1px solid var(--color-border, #444);
  padding: 2px 4px;
}

.char-controls {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
}

.control-group {
  flex: 1;
  min-width: 120px;
}

.control-group label {
  display: block;
  font-size: 0.8em;
  color: var(--color-text-secondary, #aaa);
  margin-bottom: 2px;
}

.control-group input[type="range"] {
  width: 100%;
}

.preview-box {
  margin-top: 8px;
}

.preview-canvas {
  position: relative;
  width: 200px;
  height: 120px;
  background: var(--color-bg, #333);
  border: 1px dashed var(--color-border, #555);
  border-radius: 4px;
  overflow: hidden;
}

.preview-char {
  position: absolute;
  background: rgba(100, 150, 255, 0.3);
  border: 1px solid rgba(100, 150, 255, 0.6);
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.7em;
  color: var(--color-text, #ccc);
  overflow: hidden;
}

.editor-actions {
  display: flex;
  gap: 10px;
  margin-top: 16px;
  flex-wrap: wrap;
}

.progress-info {
  margin-top: 12px;
  display: flex;
  align-items: center;
  gap: 10px;
}

.progress-bar {
  flex: 1;
  height: 6px;
  background: var(--color-bg-mute, #333);
  border-radius: 3px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: #4caf50;
  transition: width 0.3s;
}

.result-image img {
  max-width: 100%;
  border-radius: 6px;
}

.result-actions {
  display: flex;
  gap: 10px;
  margin-top: 12px;
}

.btn-primary {
  padding: 8px 20px;
  background: #4caf50;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.95em;
}

.btn-primary:disabled { background: #666; cursor: not-allowed; }

.btn-secondary {
  padding: 8px 16px;
  background: transparent;
  color: var(--color-text, #eee);
  border: 1px solid var(--color-border, #555);
  border-radius: 4px;
  cursor: pointer;
}

.btn-sm { padding: 4px 10px; font-size: 0.85em; }

.btn-danger {
  padding: 4px 10px;
  background: #f44336;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.85em;
}

.saved-scenes {
  margin-top: 20px;
  border-top: 1px solid var(--color-border, #444);
  padding-top: 16px;
}

.saved-scenes h4 {
  margin-bottom: 10px;
  color: var(--color-text-secondary, #aaa);
}

.saved-scene-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid var(--color-border, #333);
}

.saved-scene-info {
  display: flex;
  gap: 12px;
  align-items: center;
}

.saved-scene-name {
  color: var(--color-text, #eee);
  cursor: pointer;
  font-size: 0.9em;
}

.saved-scene-name:hover {
  text-decoration: underline;
}

.saved-scene-date {
  color: var(--color-text-secondary, #888);
  font-size: 0.8em;
}

.batch-results-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 12px;
}

.batch-result-card {
  border: 1px solid var(--color-border, #444);
  border-radius: 8px;
  overflow: hidden;
  background: var(--color-bg-mute, #252535);
}

.batch-result-image {
  width: 100%;
  display: block;
  object-fit: cover;
  background: var(--color-bg, #333);
}

.batch-result-meta {
  padding: 8px;
}
</style>
