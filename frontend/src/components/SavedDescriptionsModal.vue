<script setup>
import { ref, computed } from 'vue'
import { api } from '../api.js'

const props = defineProps({
  visible: { type: Boolean, default: false },
  descriptions: { type: Array, default: () => [] },
  createMode: { type: Boolean, default: false },
  createText: { type: String, default: '' },
  createNegative: { type: String, default: '' },
})
const emit = defineEmits(['close', 'use', 'create', 'update', 'delete'])

const search = ref('')
const typeFilter = ref('')
const editingId = ref(null)
const editName = ref('')
const editType = ref('')
const editNegative = ref('')
const newText = ref('')
const newNegative = ref('')
const newName = ref('')
const newType = ref('')
const showCreate = ref(false)

const types = computed(() => {
  const set = new Set(props.descriptions.map(d => d.type).filter(Boolean))
  return [...set].sort()
})

const filtered = computed(() => {
  let list = props.descriptions
  if (search.value) {
    const q = search.value.toLowerCase()
    list = list.filter(d => d.text.toLowerCase().includes(q) || (d.name || '').toLowerCase().includes(q))
  }
  if (typeFilter.value) {
    list = list.filter(d => d.type === typeFilter.value)
  }
  return list
})

function useDesc(desc) {
  emit('use', desc)
  emit('close')
}

function startEdit(desc) {
  editingId.value = desc.id
  editName.value = desc.name || ''
  editType.value = desc.type || ''
  editNegative.value = desc.negative_prompt || ''
}

function saveEdit(desc) {
  emit('update', { ...desc, name: editName.value, type: editType.value, negative_prompt: editNegative.value })
  editingId.value = null
}

function handleCreate() {
  if (!newText.value.trim()) return
  emit('create', {
    text: newText.value,
    name: newName.value,
    negative_prompt: newNegative.value,
    type: newType.value,
  })
  newText.value = ''
  newNegative.value = ''
  newName.value = ''
  newType.value = ''
  showCreate.value = false
}

function handleDelete(id) {
  emit('delete', id)
}
</script>

<template>
  <div v-if="visible" class="modal-overlay" @click.self="$emit('close')">
    <div class="modal" style="max-width: 640px;">
      <div class="modal-header">
        <h2 class="modal-title">Saved Descriptions</h2>
        <button class="modal-close" @click="$emit('close')">&times;</button>
      </div>

      <div style="display: flex; gap: 8px; margin-bottom: 12px;">
        <input class="form-input" v-model="search" placeholder="Search..." style="flex: 1;" />
        <button class="btn btn-primary btn-sm" @click="showCreate = !showCreate">{{ showCreate ? 'Cancel' : 'New' }}</button>
      </div>

      <div v-if="types.length > 0" class="style-markers" style="margin-bottom: 12px;">
        <span class="style-chip" :class="{ active: !typeFilter }" @click="typeFilter = ''">All</span>
        <span v-for="t in types" :key="t" class="style-chip" :class="{ active: typeFilter === t }" @click="typeFilter = t">{{ t }}</span>
      </div>

      <div v-if="showCreate" style="background: var(--surface-2); padding: 12px; border-radius: var(--radius-sm); margin-bottom: 12px;">
        <div class="form-group">
          <input class="form-input" v-model="newText" placeholder="Description text" />
        </div>
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 8px;">
          <input class="form-input" v-model="newName" placeholder="Name (optional)" />
          <input class="form-input" v-model="newType" placeholder="Tag/type (optional)" />
        </div>
        <div class="form-group" style="margin-top: 8px;">
          <textarea class="form-textarea" v-model="newNegative" placeholder="Negative prompt (optional)" rows="2"></textarea>
        </div>
        <button class="btn btn-primary btn-sm" @click="handleCreate" :disabled="!newText.trim()">Save</button>
      </div>

      <div class="saved-modal-list">
        <div v-for="desc in filtered" :key="desc.id" class="saved-modal-item">
          <div style="flex: 1; cursor: pointer;" @click="useDesc(desc)">
            <div style="display: flex; align-items: center; gap: 8px;">
              <span v-if="desc.name" style="font-weight: 500; color: var(--text-bright);">{{ desc.name }}</span>
              <span v-if="desc.type" class="preset-type">{{ desc.type }}</span>
            </div>
            <div v-if="editingId !== desc.id" class="saved-modal-text">{{ desc.text }}</div>
            <div v-else style="display: flex; flex-direction: column; gap: 6px;">
              <input class="form-input" v-model="editName" placeholder="Name" />
              <input class="form-input" v-model="editType" placeholder="Type/tag" />
              <textarea class="form-textarea" v-model="editNegative" placeholder="Negative prompt" rows="2"></textarea>
              <div style="display: flex; gap: 6px;">
                <button class="btn btn-primary btn-sm" @click.stop="saveEdit(desc)">Save</button>
                <button class="btn btn-secondary btn-sm" @click.stop="editingId = null">Cancel</button>
              </div>
            </div>
            <div v-if="desc.negative_prompt" style="font-size: 11px; color: var(--text-dim); margin-top: 4px;">
              Neg: {{ desc.negative_prompt.substring(0, 80) }}{{ desc.negative_prompt.length > 80 ? '...' : '' }}
            </div>
          </div>
          <div style="display: flex; flex-direction: column; gap: 4px; flex-shrink: 0;">
            <button v-if="editingId !== desc.id" class="btn btn-secondary btn-sm" @click.stop="startEdit(desc)">Edit</button>
            <button class="btn btn-danger btn-sm" @click.stop="handleDelete(desc.id)">Del</button>
          </div>
        </div>
        <div v-if="filtered.length === 0" class="empty-state">
          <div class="empty-state-icon">&#128196;</div>
          <div>No saved descriptions</div>
        </div>
      </div>
    </div>
  </div>
</template>
