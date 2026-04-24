<script setup>
import { ref, onMounted, reactive, computed } from 'vue'
import { api } from '../api.js'

const props = defineProps({
  preset: Object,
})
const emit = defineEmits(['save', 'close'])

const models = ref([])
const samplers = ref([])
const upscalers = ref([])
const vaes = ref([])

const STYLE_MARKERS = [
  { id: 'realistic', label: 'Realistic', tags: 'realistic, photorealistic, photograph, raw photo, 8k uhd' },
  { id: '3d', label: '3D', tags: '3d render, octane render, unreal engine, highly detailed 3d' },
  { id: 'anime', label: 'Anime', tags: 'anime style, anime art, illustration, cel shading' },
  { id: 'cartoon', label: 'Cartoon', tags: 'cartoon style, cartoon art, colorful, bold outlines' },
  { id: 'painting', label: 'Painting', tags: 'oil painting, digital painting, concept art, artstation' },
  { id: 'pixel', label: 'Pixel Art', tags: 'pixel art, 16-bit, retro game style, sprite art' },
  { id: 'watercolor', label: 'Watercolor', tags: 'watercolor painting, soft colors, delicate brush strokes' },
  { id: 'comic', label: 'Comic', tags: 'comic book style, graphic novel, bold lines, halftone' },
  { id: 'dark', label: 'Dark Fantasy', tags: 'dark fantasy, gothic, moody, dramatic lighting, dark atmosphere' },
  { id: 'cinematic', label: 'Cinematic', tags: 'cinematic, movie still, dramatic composition, volumetric lighting' },
]

const selectedStyles = ref([])

const form = reactive({
  name: props.preset?.name || '',
  preset_type: props.preset?.preset_type || '',
  prompt: props.preset?.prompt || '',
  negative_prompt: props.preset?.negative_prompt || '',
  sampler: props.preset?.sampler || 'Euler a',
  steps: props.preset?.steps || 20,
  cfg_scale: props.preset?.cfg_scale || 7.0,
  width: props.preset?.width || 512,
  height: props.preset?.height || 512,
  model_name: props.preset?.model_name || '',
  seed: props.preset?.seed ?? null,
  denoising_strength: props.preset?.denoising_strength ?? null,
  clip_skip: props.preset?.clip_skip ?? null,
  batch_size: props.preset?.batch_size ?? 1,
  batch_count: props.preset?.batch_count ?? 1,
  hires_fix: props.preset?.hires_fix ?? false,
  hires_upscale: props.preset?.hires_upscale ?? 2.0,
  hires_denoising_strength: props.preset?.hires_denoising_strength ?? 0.5,
  hires_upscaler: props.preset?.hires_upscaler ?? '',
  vae: props.preset?.vae ?? '',
})

const saving = ref(false)
const generatingPrompt = ref(false)

async function loadModels() {
  try { models.value = await api.getModels() } catch {}
}

async function loadSamplers() {
  try { samplers.value = await api.getSamplers() } catch {}
}

async function loadUpscalers() {
  try { upscalers.value = await api.getUpscalers() } catch {}
}

async function loadVAEs() {
  try { vaes.value = await api.getVAEs() } catch {}
}

function buildDescription() {
  let parts = []
  if (form.name) parts.push(`Subject: ${form.name}`)
  if (form.preset_type) parts.push(`Type: ${form.preset_type}`)
  if (form.prompt) parts.push(`Current prompt to improve: ${form.prompt}`)
  if (form.width && form.height) parts.push(`Resolution: ${form.width}x${form.height}`)
  const activeStyles = STYLE_MARKERS.filter(s => selectedStyles.value.includes(s.id))
  if (activeStyles.length > 0) {
    parts.push(`Style: ${activeStyles.map(s => s.label).join(', ')}`)
  }
  return parts.join('. ') || 'Generate a generic high-quality image prompt'
}

async function generatePrompt() {
  generatingPrompt.value = true
  try {
    const description = buildDescription()
    const result = await api.generateSdPrompt(description, form.preset_type)
    let prompt = result.prompt
    const activeStyles = STYLE_MARKERS.filter(s => selectedStyles.value.includes(s.id))
    if (activeStyles.length > 0) {
      const styleTags = activeStyles.map(s => s.tags).join(', ')
      prompt = styleTags + ', ' + prompt
    }
    form.prompt = prompt
  } catch (e) {
    alert('Prompt generation failed: ' + e.message)
  } finally {
    generatingPrompt.value = false
  }
}

function toggleStyle(styleId) {
  const idx = selectedStyles.value.indexOf(styleId)
  if (idx === -1) {
    selectedStyles.value.push(styleId)
  } else {
    selectedStyles.value.splice(idx, 1)
  }
}

async function save() {
  saving.value = true
  try {
    await emit('save', {
      name: form.name,
      preset_type: form.preset_type,
      prompt: form.prompt,
      negative_prompt: form.negative_prompt,
      sampler: form.sampler,
      steps: Number(form.steps),
      cfg_scale: Number(form.cfg_scale),
      width: Number(form.width),
      height: Number(form.height),
      model_name: form.model_name,
      seed: form.seed ? Number(form.seed) : null,
      denoising_strength: form.denoising_strength != null ? Number(form.denoising_strength) : null,
      clip_skip: form.clip_skip != null ? Number(form.clip_skip) : null,
      batch_size: form.batch_size != null ? Number(form.batch_size) : null,
      batch_count: form.batch_count != null ? Number(form.batch_count) : null,
      hires_fix: form.hires_fix || false,
      hires_upscale: form.hires_upscale ? Number(form.hires_upscale) : null,
      hires_denoising_strength: form.hires_denoising_strength != null ? Number(form.hires_denoising_strength) : null,
      hires_upscaler: form.hires_upscaler || '',
      vae: form.vae || '',
    })
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadModels()
  loadSamplers()
  loadUpscalers()
  loadVAEs()
})
</script>

<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal">
      <div class="modal-header">
        <h2 class="modal-title">{{ preset ? 'Edit Preset' : 'New Preset' }}</h2>
        <button class="modal-close" @click="$emit('close')">&times;</button>
      </div>

      <form @submit.prevent="save">
        <div class="form-group">
          <label class="form-label">Name</label>
          <input class="form-input" v-model="form.name" required />
        </div>

        <div class="form-group">
          <label class="form-label">Type</label>
          <input class="form-input" v-model="form.preset_type" placeholder="weapon, character, scene..." />
        </div>

        <div class="form-group">
          <label class="form-label">Prompt</label>
          <textarea class="form-textarea" v-model="form.prompt" rows="4" placeholder="masterpiece, best quality, ..."></textarea>
          <button type="button" class="btn btn-secondary btn-sm" style="margin-top: 6px;" @click="generatePrompt" :disabled="generatingPrompt">
            {{ generatingPrompt ? 'Generating...' : 'AI Generate Prompt' }}
          </button>
        </div>

        <div class="form-group">
          <label class="form-label">Negative Prompt</label>
          <textarea class="form-textarea" v-model="form.negative_prompt" rows="2"></textarea>
        </div>

        <div class="form-group">
          <label class="form-label">Style</label>
          <div class="style-markers">
            <span
              v-for="s in STYLE_MARKERS"
              :key="s.id"
              class="style-chip"
              :class="{ active: selectedStyles.includes(s.id) }"
              @click="toggleStyle(s.id)"
            >{{ s.label }}</span>
          </div>
        </div>

        <div class="form-group">
          <label class="form-label">Model</label>
          <select class="form-select" v-model="form.model_name">
            <option value="">Default</option>
            <option v-for="m in models" :key="m.model_name" :value="m.model_name">{{ m.title || m.model_name }}</option>
          </select>
        </div>

        <div class="form-group">
          <label class="form-label">Sampler</label>
          <select class="form-select" v-model="form.sampler">
            <option v-for="s in samplers" :key="s.name" :value="s.name">{{ s.name }}</option>
            <option v-if="samplers.length === 0" value="Euler a">Euler a</option>
          </select>
        </div>

        <div class="form-row">
          <div class="form-group">
            <label class="form-label">Steps</label>
            <input class="form-input" type="number" v-model.number="form.steps" min="1" max="150" />
          </div>
          <div class="form-group">
            <label class="form-label">CFG Scale</label>
            <input class="form-input" type="number" v-model.number="form.cfg_scale" step="0.5" min="1" max="30" />
          </div>
          <div class="form-group">
            <label class="form-label">Seed</label>
            <input class="form-input" type="number" v-model="form.seed" placeholder="Random" />
          </div>
        </div>

        <div class="form-row">
          <div class="form-group">
            <label class="form-label">Width</label>
            <input class="form-input" type="number" v-model.number="form.width" step="64" min="64" max="2048" />
          </div>
          <div class="form-group">
            <label class="form-label">Height</label>
            <input class="form-input" type="number" v-model.number="form.height" step="64" min="64" max="2048" />
          </div>
          <div></div>
        </div>

        <div class="form-row">
          <div class="form-group">
            <label class="form-label">Batch Size</label>
            <input class="form-input" type="number" v-model.number="form.batch_size" min="1" max="8" />
          </div>
          <div class="form-group">
            <label class="form-label">Batch Count</label>
            <input class="form-input" type="number" v-model.number="form.batch_count" min="1" max="8" />
          </div>
          <div class="form-group">
            <label class="form-label">Clip Skip</label>
            <input class="form-input" type="number" v-model.number="form.clip_skip" min="1" max="12" placeholder="1" />
          </div>
        </div>

        <div class="form-row">
          <div class="form-group">
            <label class="form-label">Denoising Strength</label>
            <input class="form-input" type="number" v-model.number="form.denoising_strength" step="0.05" min="0" max="1" placeholder="0.75" />
          </div>
          <div class="form-group">
            <label class="form-label">VAE</label>
            <select class="form-select" v-model="form.vae">
              <option value="">Default</option>
              <option v-for="v in vaes" :key="v.model_name" :value="v.model_name">{{ v.model_name }}</option>
            </select>
          </div>
          <div></div>
        </div>

        <div class="form-group">
          <label class="form-label" style="display: flex; align-items: center; gap: 8px; cursor: pointer;">
            <input type="checkbox" v-model="form.hires_fix" />
            Hires Fix
          </label>
        </div>

        <template v-if="form.hires_fix">
          <div class="form-row">
            <div class="form-group">
              <label class="form-label">Upscale Factor</label>
              <input class="form-input" type="number" v-model.number="form.hires_upscale" step="0.5" min="1" max="4" />
            </div>
            <div class="form-group">
              <label class="form-label">Hires Denoising</label>
              <input class="form-input" type="number" v-model.number="form.hires_denoising_strength" step="0.05" min="0" max="1" />
            </div>
            <div class="form-group">
              <label class="form-label">Hires Upscaler</label>
              <select class="form-select" v-model="form.hires_upscaler">
                <option value="">Default</option>
                <option v-for="u in upscalers" :key="u.name" :value="u.name">{{ u.name }}</option>
              </select>
            </div>
          </div>
        </template>

        <div style="display: flex; gap: 10px; justify-content: flex-end; margin-top: 20px;">
          <button type="button" class="btn btn-secondary" @click="$emit('close')">Cancel</button>
          <button type="submit" class="btn btn-primary" :disabled="saving">
            {{ saving ? 'Saving...' : 'Save' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
