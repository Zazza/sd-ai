<script setup>
import { ref, computed, nextTick } from 'vue'

const props = defineProps({
  modelValue: { type: [Number, String], default: null },
  items: { type: Array, default: () => [] },
  placeholder: { type: String, default: '' },
  emptyText: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue'])

const open = ref(false)
const query = ref('')
const activeIndex = ref(0)
const rootEl = ref(null)
const inputEl = ref(null)
const listEl = ref(null)

const selectedItem = computed(() => props.items.find(i => i.value === props.modelValue) || null)
const hasValue = computed(() => props.modelValue !== null && props.modelValue !== undefined)

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return props.items
  return props.items.filter(i =>
    String(i.label || '').toLowerCase().includes(q) ||
    (i.hint ? String(i.hint).toLowerCase().includes(q) : false)
  )
})

const displayValue = computed(() => (open.value ? query.value : (selectedItem.value ? selectedItem.value.label : '')))

function openList() {
  if (open.value) return
  open.value = true
  query.value = selectedItem.value ? selectedItem.value.label : ''
  activeIndex.value = Math.max(0, filtered.value.findIndex(i => i.value === props.modelValue))
  nextTick(() => inputEl.value && inputEl.value.select())
}

function closeList() {
  open.value = false
  query.value = ''
}

function onInput(e) {
  query.value = e.target.value
  open.value = true
  activeIndex.value = 0
}

function onBlur(e) {
  if (rootEl.value && e.relatedTarget && rootEl.value.contains(e.relatedTarget)) return
  closeList()
}

function select(item) {
  emit('update:modelValue', item.value)
  closeList()
}

function clear() {
  if (!hasValue.value) return
  emit('update:modelValue', null)
  closeList()
}

function onKeydown(e) {
  if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    e.preventDefault()
    if (!open.value) { openList(); return }
    const n = filtered.value.length
    if (!n) return
    activeIndex.value = e.key === 'ArrowDown'
      ? (activeIndex.value + 1) % n
      : (activeIndex.value - 1 + n) % n
    scrollActiveIntoView()
  } else if (e.key === 'Enter') {
    e.preventDefault()
    if (!open.value) return
    const item = filtered.value[activeIndex.value]
    if (item) select(item)
  } else if (e.key === 'Escape') {
    if (open.value) {
      e.stopPropagation()
      closeList()
    }
  }
}

function scrollActiveIntoView() {
  nextTick(() => {
    const el = listEl.value && listEl.value.children[activeIndex.value]
    if (el && el.scrollIntoView) el.scrollIntoView({ block: 'nearest' })
  })
}
</script>

<template>
  <div ref="rootEl" class="searchable-select">
    <input
      ref="inputEl"
      class="form-input ss-input"
      type="text"
      :value="displayValue"
      :placeholder="placeholder"
      autocomplete="off"
      @input="onInput"
      @focus="openList"
      @click="openList"
      @blur="onBlur"
      @keydown="onKeydown"
    />
    <button v-if="hasValue" type="button" class="ss-clear" @mousedown.prevent @click="clear">&times;</button>
    <ul v-if="open" ref="listEl" class="ss-list">
      <li v-if="filtered.length === 0" class="ss-empty">{{ emptyText }}</li>
      <li
        v-for="(item, i) in filtered"
        :key="item.value"
        class="ss-option"
        :class="{ 'ss-option-active': i === activeIndex, 'ss-option-selected': item.value === modelValue }"
        @mousedown.prevent
        @click="select(item)"
        @mousemove="activeIndex = i"
      >
        <span class="ss-option-label">{{ item.label }}</span>
        <span v-if="item.hint" class="ss-option-hint">{{ item.hint }}</span>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.searchable-select {
  position: relative;
}

.ss-input {
  padding-right: 30px;
}

.ss-clear {
  position: absolute;
  right: 4px;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  color: var(--text-dim);
  font-size: 16px;
  line-height: 1;
  padding: 4px 6px;
  cursor: pointer;
}

.ss-clear:hover {
  color: var(--text);
}

.ss-list {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  max-height: 260px;
  overflow-y: auto;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.25);
  z-index: 1000;
  list-style: none;
  padding: 4px 0;
}

.ss-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  padding: 7px 12px;
  cursor: pointer;
}

.ss-option-active {
  background: var(--accent-bg);
}

.ss-option-selected .ss-option-label {
  color: var(--accent);
  font-weight: 600;
}

.ss-option-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ss-option-hint {
  flex-shrink: 0;
  max-width: 45%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 11px;
  color: var(--text-dim);
}

.ss-empty {
  padding: 8px 12px;
  color: var(--text-dim);
  font-size: 13px;
}
</style>
