<script setup>
import { ref, computed, reactive, onMounted } from 'vue'
import { api } from '../api.js'

const props = defineProps({
  modelValue: { type: Number, default: null },
})
const emit = defineEmits(['update:modelValue'])

const profiles = ref([])
const showAdd = ref(false)
const editing = ref(false)
const newProfile = reactive({ name: '', upscale: 2.0, denoising_strength: 0.5, upscaler: '' })

const sorted = computed(() => {
  const builtin = profiles.value.filter(h => h.is_builtin)
  const custom = profiles.value.filter(h => !h.is_builtin).sort((a, b) => a.name.localeCompare(b.name))
  return [...builtin, ...custom]
})

const selectedId = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})

const selectedProfile = computed(() => {
  if (!selectedId.value) return null
  return profiles.value.find(h => h.id === selectedId.value) || null
})

async function load() {
  try {
    const items = await api.listHiresProfiles()
    profiles.value = items || []
  } catch (e) {
    console.error('ListHiresProfiles:', e)
  }
}

function emitChange() {
  emit('update:modelValue', selectedId.value)
}

async function addProfile() {
  if (!newProfile.name.trim()) return
  try {
    const created = await api.createHiresProfile({
      name: newProfile.name.trim(),
      upscale: Number(newProfile.upscale),
      denoising_strength: Number(newProfile.denoising_strength),
      upscaler: newProfile.upscaler.trim(),
    })
    await load()
    if (created && created.id) {
      selectedId.value = created.id
      emitChange()
    }
    resetForm()
  } catch (e) {
    console.error('CreateHiresProfile:', e)
  }
}

async function removeProfile() {
  if (!selectedProfile.value || selectedProfile.value.is_builtin) return
  try {
    await api.deleteHiresProfile(selectedProfile.value.id)
    selectedId.value = null
    emitChange()
    await load()
  } catch (e) {
    console.error('DeleteHiresProfile:', e)
  }
}

function resetForm() {
  newProfile.name = ''
  newProfile.upscale = 2.0
  newProfile.denoising_strength = 0.5
  newProfile.upscaler = ''
  showAdd.value = false
  editing.value = false
}

function startEdit() {
  if (!selectedProfile.value) return
  newProfile.name = selectedProfile.value.name
  newProfile.upscale = selectedProfile.value.upscale
  newProfile.denoising_strength = selectedProfile.value.denoising_strength
  newProfile.upscaler = selectedProfile.value.upscaler
  editing.value = true
  showAdd.value = true
}

async function saveEdit() {
  if (!selectedProfile.value || !newProfile.name.trim()) return
  try {
    await api.updateHiresProfile({
      id: selectedProfile.value.id,
      name: newProfile.name.trim(),
      upscale: Number(newProfile.upscale),
      denoising_strength: Number(newProfile.denoising_strength),
      upscaler: newProfile.upscaler.trim(),
    })
    await load()
    resetForm()
  } catch (e) {
    console.error('UpdateHiresProfile:', e)
  }
}

onMounted(load)
</script>

<template>
  <div class="selector-wrap">
    <div class="selector-header">
      <label class="form-label">Hires Profile</label>
      <button @click="showAdd = true; editing = false; newProfile.name = ''; newProfile.upscale = 2.0; newProfile.denoising_strength = 0.5; newProfile.upscaler = ''" class="btn-icon" title="Add custom">+</button>
    </div>
    <select v-model="selectedId" @change="emitChange" class="form-select">
      <option :value="null">No hires</option>
      <option v-for="h in sorted" :key="h.id" :value="h.id">
        {{ h.name }} ({{ h.upscale }}x, denoise {{ h.denoising_strength }})
      </option>
    </select>
    <div v-if="showAdd" class="selector-inline">
      <input v-model="newProfile.name" placeholder="Name" class="form-input" />
      <input v-model.number="newProfile.upscale" type="number" placeholder="Upscale" class="form-input" step="0.5" min="1" max="4" />
      <input v-model.number="newProfile.denoising_strength" type="number" placeholder="Denoise" class="form-input" step="0.05" min="0" max="1" />
      <input v-model="newProfile.upscaler" placeholder="Upscaler" class="form-input" />
      <button v-if="!editing" @click="addProfile" class="btn btn-sm">Add</button>
      <button v-if="editing" @click="saveEdit" class="btn btn-sm">Save</button>
      <button @click="resetForm" class="btn btn-sm btn-secondary">Cancel</button>
    </div>
    <div v-if="selectedProfile && !selectedProfile.is_builtin && !showAdd" class="selector-manage">
      <button @click="startEdit" class="btn btn-sm">Edit</button>
      <button @click="removeProfile" class="btn btn-sm btn-danger">Delete</button>
    </div>
  </div>
</template>

<style scoped>
.selector-wrap {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.selector-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.btn-icon {
  background: none;
  border: 1px solid var(--border);
  color: var(--text);
  font-size: 16px;
  width: 28px;
  height: 28px;
  border-radius: var(--radius);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
}

.btn-icon:hover {
  background: var(--surface-2);
}

.selector-inline {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  align-items: center;
}

.selector-inline .form-input {
  flex: 1;
  min-width: 80px;
}

.btn.sm,
.btn-sm {
  padding: 4px 12px;
  font-size: 12px;
}

.selector-manage {
  display: flex;
  gap: 6px;
}
</style>
