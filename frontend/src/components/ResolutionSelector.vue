<script setup>
import { ref, computed, reactive, onMounted } from 'vue'
import { api } from '../api.js'

const props = defineProps({
  modelValue: { type: Number, default: null },
})
const emit = defineEmits(['update:modelValue'])

const resolutions = ref([])
const showAdd = ref(false)
const editing = ref(false)
const newRes = reactive({ name: '', width: 512, height: 512 })

const sorted = computed(() => {
  const builtin = resolutions.value.filter(r => r.is_builtin)
  const custom = resolutions.value.filter(r => !r.is_builtin).sort((a, b) => a.name.localeCompare(b.name))
  return [...builtin, ...custom]
})

const selectedId = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})

const selectedResolution = computed(() => {
  if (!selectedId.value) return null
  return resolutions.value.find(r => r.id === selectedId.value) || null
})

async function load() {
  try {
    const items = await api.listResolutions()
    resolutions.value = items || []
  } catch (e) {
    console.error('ListResolutions:', e)
  }
}

function emitChange() {
  emit('update:modelValue', selectedId.value)
}

async function addResolution() {
  if (!newRes.name.trim() || !newRes.width || !newRes.height) return
  try {
    const created = await api.createResolution({
      name: newRes.name.trim(),
      width: Number(newRes.width),
      height: Number(newRes.height),
    })
    await load()
    if (created && created.id) {
      selectedId.value = created.id
      emitChange()
    }
    resetForm()
  } catch (e) {
    console.error('CreateResolution:', e)
  }
}

async function removeResolution() {
  if (!selectedResolution.value || selectedResolution.value.is_builtin) return
  try {
    await api.deleteResolution(selectedResolution.value.id)
    selectedId.value = null
    emitChange()
    await load()
  } catch (e) {
    console.error('DeleteResolution:', e)
  }
}

function resetForm() {
  newRes.name = ''
  newRes.width = 512
  newRes.height = 512
  showAdd.value = false
  editing.value = false
}

function startEdit() {
  if (!selectedResolution.value) return
  newRes.name = selectedResolution.value.name
  newRes.width = selectedResolution.value.width
  newRes.height = selectedResolution.value.height
  editing.value = true
  showAdd.value = true
}

async function saveEdit() {
  if (!selectedResolution.value || !newRes.name.trim()) return
  try {
    await api.updateResolution({
      id: selectedResolution.value.id,
      name: newRes.name.trim(),
      width: Number(newRes.width),
      height: Number(newRes.height),
    })
    await load()
    resetForm()
  } catch (e) {
    console.error('UpdateResolution:', e)
  }
}

onMounted(load)
</script>

<template>
  <div class="selector-wrap">
    <div class="selector-header">
      <label class="form-label">Resolution</label>
      <button @click="showAdd = true; editing = false; newRes.name = ''; newRes.width = 512; newRes.height = 512" class="btn-icon" title="Add custom">+</button>
    </div>
    <select v-model="selectedId" @change="emitChange" class="form-select">
      <option :value="null">From preset</option>
      <option v-for="r in sorted" :key="r.id" :value="r.id">
        {{ r.name }} ({{ r.width }}x{{ r.height }})
      </option>
    </select>
    <div v-if="showAdd" class="selector-inline">
      <input v-model="newRes.name" placeholder="Name" class="form-input" />
      <input v-model.number="newRes.width" type="number" placeholder="Width" class="form-input" step="64" min="64" max="4096" />
      <input v-model.number="newRes.height" type="number" placeholder="Height" class="form-input" step="64" min="64" max="4096" />
      <button v-if="!editing" @click="addResolution" class="btn btn-sm">Add</button>
      <button v-if="editing" @click="saveEdit" class="btn btn-sm">Save</button>
      <button @click="resetForm" class="btn btn-sm btn-secondary">Cancel</button>
    </div>
    <div v-if="selectedResolution && !selectedResolution.is_builtin && !showAdd" class="selector-manage">
      <button @click="startEdit" class="btn btn-sm">Edit</button>
      <button @click="removeResolution" class="btn btn-sm btn-danger">Delete</button>
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
