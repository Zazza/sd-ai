<script setup>
import { ref, computed, onMounted, onUnmounted, reactive } from 'vue'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'
import { api } from '../api.js'

const presets = ref([])
const selectedPresetId = ref(null)
const description = ref('')
const scene = ref(null)
const generating = ref(false)
const decomposing = ref(false)
const progress = ref(null)
const error = ref('')
const result = ref(null)
const resultImage = ref('')
const llmAvailable = ref(false)
const sdAvailable = ref(false)

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

async function decompose() {
  if (!description.value.trim()) {
    error.value = 'Enter a scene description'
    return
  }
  if (!selectedPresetId.value) {
    error.value = 'Select a preset'
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
    scene.value = s
  } catch (e) {
    error.value = 'Decomposition failed: ' + e
  } finally {
    decomposing.value = false
  }
}

async function generate() {
  if (!scene.value) return

  generating.value = true
  error.value = ''
  progress.value = null
  result.value = null
  resultImage.value = ''

  EventsOn('multipass:progress', (p) => {
    progress.value = p
  })

  try {
    const r = await api.generateMultiPass(scene.value)
    result.value = r
    resultImage.value = r.image
  } catch (e) {
    error.value = 'Generation failed: ' + e
  } finally {
    generating.value = false
    EventsOff('multipass:progress')
  }
}

function progressLabel() {
  if (!progress.value) return ''
  const p = progress.value
  if (p.step === 'background') return 'Generating background...'
  if (p.step === 'character') return `Generating character ${p.character}/${p.total}...`
  if (p.step === 'done') return 'Compositing complete!'
  return p.step
}

async function saveImage() {
  if (!resultImage.value) return
  try {
    await api.saveImage(resultImage.value, 'scene')
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
  } catch (e) {
    error.value = 'Save failed: ' + e
  }
}

onMounted(() => {
  loadPresets()
  checkServices()
})
</script>

<template>
  <div class="scene-editor">
    <h2>Multi-Pass Scene Generator</h2>

    <div v-if="error" class="error">{{ error }}</div>

    <div class="status-bar">
      <span :class="{ online: llmAvailable, offline: !llmAvailable }">LLM</span>
      <span :class="{ online: sdAvailable, offline: !sdAvailable }">SD</span>
    </div>

    <!-- Step 1: Description + Preset -->
    <div class="section" v-if="!scene">
      <div class="form-group">
        <label>Preset</label>
        <select v-model="selectedPresetId">
          <option :value="null" disabled>Select preset...</option>
          <option v-for="p in presetOptions" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
      </div>

      <div class="form-group">
        <label>Scene Description</label>
        <textarea v-model="description" rows="4" placeholder="Describe the scene with all characters, e.g.: A warrior and a mage standing in a dark forest clearing. The warrior is on the left with a sword, the mage on the right casting fire."></textarea>
      </div>

      <button @click="decompose" :disabled="decomposing || !llmAvailable || !selectedPresetId" class="btn-primary">
        {{ decomposing ? 'Decomposing...' : 'Decompose Scene' }}
      </button>
    </div>

    <!-- Step 2: Scene Editor -->
    <div class="section" v-if="scene">
      <div class="editor-header">
        <h3>Scene Editor</h3>
        <button @click="scene = null; result = null" class="btn-secondary btn-sm">Back</button>
      </div>

      <div class="form-group">
        <label>Background Prompt</label>
        <textarea v-model="scene.background_prompt" rows="2"></textarea>
      </div>

      <div class="form-group">
        <label>Negative Prompt</label>
        <input v-model="scene.negative_prompt" type="text" />
      </div>

      <div class="characters-list">
        <h4>Characters ({{ scene.characters.length }})</h4>
        <button @click="addCharacter" class="btn-secondary btn-sm">+ Add Character</button>

        <div v-for="(char, i) in scene.characters" :key="i" class="character-card">
          <div class="char-header">
            <input v-model="char.name" class="char-name" placeholder="Character name" />
            <button @click="removeCharacter(i)" class="btn-danger btn-sm">Remove</button>
          </div>

          <div class="form-group">
            <label>Character Prompt</label>
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
        <button @click="generate" :disabled="generating || !sdAvailable" class="btn-primary">
          {{ generating ? progressLabel() : 'Generate Multi-Pass' }}
        </button>
        <button @click="saveScene" :disabled="!scene" class="btn-secondary">Save Scene</button>
        <button @click="scene = null; result = null" class="btn-secondary">Cancel</button>
      </div>

      <div v-if="generating && progress" class="progress-info">
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: progressWidth }"></div>
        </div>
        <span>{{ progressLabel() }}</span>
      </div>
    </div>

    <!-- Step 3: Result -->
    <div class="section" v-if="resultImage">
      <h3>Result</h3>
      <div class="result-image">
        <img :src="'data:image/png;base64,' + resultImage" alt="Generated scene" />
      </div>
      <div class="result-actions">
        <button @click="saveImage" class="btn-secondary">Save Image</button>
        <button @click="scene = null; result = null; resultImage = ''" class="btn-secondary">New Scene</button>
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
</style>
