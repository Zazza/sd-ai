<script setup>
import { ref, computed } from 'vue'
import { api } from '../api.js'
import { t, locale } from '../i18n/index.js'
import { splitHighlight } from '../highlight.js'
import { Pencil, Trash2 } from 'lucide-vue-next'

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
const sortBy = ref('new')
const editingId = ref(null)
const editName = ref('')
const editType = ref('')
const editNegative = ref('')
const editText = ref('')
const newText = ref('')
const newNegative = ref('')
const newName = ref('')
const newType = ref('')
const showCreate = ref(false)

const types = computed(() => {
  const set = new Set(props.descriptions.map(d => d.type).filter(Boolean))
  return [...set].sort()
})

function cmpTime(a, b) {
  const ca = a.created_at || ''
  const cb = b.created_at || ''
  if (ca !== cb) return ca < cb ? -1 : 1
  return (a.id || 0) - (b.id || 0)
}

const filtered = computed(() => {
  let list = props.descriptions
  const q = search.value.trim().toLowerCase()
  if (q) {
    list = list.filter(d =>
      d.text.toLowerCase().includes(q) ||
      (d.name || '').toLowerCase().includes(q) ||
      (d.type || '').toLowerCase().includes(q) ||
      (d.negative_prompt || '').toLowerCase().includes(q))
  }
  if (typeFilter.value) {
    list = list.filter(d => d.type === typeFilter.value)
  }
  const sorted = [...list]
  if (sortBy.value === 'name') {
    sorted.sort((a, b) => {
      const an = (a.name || '').trim()
      const bn = (b.name || '').trim()
      if (!an && bn) return 1
      if (an && !bn) return -1
      const c = an.localeCompare(bn, locale.value)
      return c !== 0 ? c : cmpTime(b, a)
    })
  } else {
    sorted.sort((a, b) => (sortBy.value === 'old' ? cmpTime(a, b) : cmpTime(b, a)))
  }
  return sorted
})

function displayName(desc) {
  if (desc.name) return desc.name
  const text = desc.text || ''
  return text.length > 40 ? text.slice(0, 40) + '…' : text
}

function fmtDate(iso) {
  if (!iso) return ''
  const d = new Date(String(iso).replace(' ', 'T'))
  if (isNaN(d.getTime())) return ''
  return d.toLocaleDateString(locale.value === 'en' ? 'en-US' : 'ru-RU', { day: 'numeric', month: 'short' })
}

function useDesc(desc) {
  emit('use', desc)
  emit('close')
}

function searchEnter() {
  if (filtered.value.length) useDesc(filtered.value[0])
}

function startEdit(desc) {
  editingId.value = desc.id
  editText.value = desc.text || ''
  editName.value = desc.name || ''
  editType.value = desc.type || ''
  editNegative.value = desc.negative_prompt || ''
}

function saveEdit(desc) {
  emit('update', { ...desc, text: editText.value, name: editName.value, type: editType.value, negative_prompt: editNegative.value })
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
    <div class="modal" style="max-width: 860px;">
      <div class="modal-header">
        <h2 class="modal-title">{{ t('descriptions.title') }}</h2>
        <button class="modal-close" @click="$emit('close')">&times;</button>
      </div>

      <div class="sdm-toolbar">
        <input class="form-input" v-model="search" :placeholder="t('descriptions.placeholder_search')"
          style="flex: 1; min-width: 160px;" @keydown.enter="searchEnter" />
        <span v-if="descriptions.length" class="sdm-count">
          {{ t('descriptions.found_of', { found: filtered.length, total: descriptions.length }) }}
        </span>
        <select class="form-select sdm-sort" v-model="sortBy">
          <option value="new">{{ t('descriptions.sort_new') }}</option>
          <option value="old">{{ t('descriptions.sort_old') }}</option>
          <option value="name">{{ t('descriptions.sort_name') }}</option>
        </select>
        <button class="btn btn-primary btn-sm" @click="showCreate = !showCreate">{{ showCreate ? t('descriptions.btn_cancel') : t('descriptions.btn_new') }}</button>
      </div>

      <div v-if="types.length > 0" class="style-markers" style="margin-bottom: 12px;">
        <span class="style-chip" :class="{ active: !typeFilter }" @click="typeFilter = ''">{{ t('descriptions.all') }}</span>
        <span v-for="ty in types" :key="ty" class="style-chip" :class="{ active: typeFilter === ty }" @click="typeFilter = ty">{{ ty }}</span>
      </div>

      <div v-if="showCreate" style="background: var(--surface-2); padding: 12px; border-radius: var(--radius-sm); margin-bottom: 12px;">
        <div class="form-group">
          <input class="form-input" v-model="newText" :placeholder="t('descriptions.placeholder_text')" />
        </div>
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 8px;">
          <input class="form-input" v-model="newName" :placeholder="t('descriptions.placeholder_name')" />
          <input class="form-input" v-model="newType" :placeholder="t('descriptions.placeholder_tag')" />
        </div>
        <div class="form-group" style="margin-top: 8px;">
          <textarea class="form-textarea" v-model="newNegative" :placeholder="t('descriptions.placeholder_exclude')" rows="2"></textarea>
        </div>
        <button class="btn btn-primary btn-sm" @click="handleCreate" :disabled="!newText.trim()">{{ t('descriptions.btn_save') }}</button>
      </div>

      <div class="saved-modal-list">
        <div v-for="desc in filtered" :key="desc.id" class="saved-modal-item">
          <div class="sdm-main" @click="editingId !== desc.id && useDesc(desc)">
            <div class="sdm-head">
              <span class="sdm-name">
                <template v-for="(seg, i) in splitHighlight(displayName(desc), search)" :key="i">
                  <mark v-if="seg.hit" class="sdm-mark">{{ seg.text }}</mark>
                  <template v-else>{{ seg.text }}</template>
                </template>
              </span>
              <span v-if="desc.type" class="preset-type">{{ desc.type }}</span>
              <span v-if="fmtDate(desc.created_at)" class="sdm-date">{{ fmtDate(desc.created_at) }}</span>
              <span class="sdm-actions">
                <button v-if="editingId !== desc.id" class="btn btn-secondary btn-sm" :aria-label="t('descriptions.btn_edit')"
                  :title="t('descriptions.btn_edit')" @click.stop="startEdit(desc)">
                  <Pencil :size="12" />
                </button>
                <button class="btn btn-danger btn-sm" :aria-label="t('descriptions.btn_del')"
                  :title="t('descriptions.btn_del')" @click.stop="handleDelete(desc.id)">
                  <Trash2 :size="12" />
                </button>
              </span>
            </div>
            <div v-if="editingId !== desc.id" class="sdm-text">
              <template v-for="(seg, i) in splitHighlight(desc.text, search)" :key="i">
                <mark v-if="seg.hit" class="sdm-mark">{{ seg.text }}</mark>
                <template v-else>{{ seg.text }}</template>
              </template>
            </div>
            <div v-else class="sdm-edit">
              <textarea class="form-textarea" v-model="editText" :placeholder="t('descriptions.placeholder_text')" rows="3"></textarea>
              <input class="form-input" v-model="editName" :placeholder="t('descriptions.placeholder_name_edit')" />
              <input class="form-input" v-model="editType" :placeholder="t('descriptions.placeholder_type_edit')" />
              <textarea class="form-textarea" v-model="editNegative" :placeholder="t('descriptions.placeholder_exclude_edit')" rows="2"></textarea>
              <div style="display: flex; gap: 6px;">
                <button class="btn btn-primary btn-sm" @click.stop="saveEdit(desc)">{{ t('descriptions.btn_save') }}</button>
                <button class="btn btn-secondary btn-sm" @click.stop="editingId = null">{{ t('descriptions.btn_cancel') }}</button>
              </div>
            </div>
            <div v-if="desc.negative_prompt && editingId !== desc.id" class="sdm-neg">
              Neg: {{ desc.negative_prompt }}
            </div>
          </div>
        </div>
        <div v-if="filtered.length === 0" class="empty-state">
          <div class="empty-state-icon">&#128196;</div>
          <div>{{ t('descriptions.no_saved') }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.saved-modal-list {
  max-height: 60vh;
}

.sdm-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 12px;
}

.sdm-count {
  font-size: 11px;
  color: var(--text-dim);
  white-space: nowrap;
}

.sdm-sort {
  width: auto;
  min-width: 110px;
}

.sdm-main {
  flex: 1;
  min-width: 0;
  cursor: pointer;
}

.sdm-head {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.sdm-name {
  font-weight: 600;
  color: var(--text-bright);
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sdm-date {
  margin-left: auto;
  font-size: 11px;
  color: var(--text-dim);
  white-space: nowrap;
}

.sdm-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

.sdm-text {
  font-size: 13px;
  line-height: 1.5;
  color: var(--text);
  margin-top: 3px;
  word-break: break-word;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.sdm-main:hover .sdm-text {
  color: var(--accent);
}

.sdm-neg {
  font-size: 11px;
  color: var(--text-dim);
  margin-top: 3px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sdm-edit {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 6px;
  cursor: default;
}

.sdm-mark {
  background: var(--accent-bg);
  color: inherit;
  border-radius: 2px;
}
</style>
