<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { api } from '../api.js'
import { t } from '../i18n/index.js'
import ImageViewer from './ImageViewer.vue'
import { FolderUp, Folder, FolderOpen, Image, Send, Download } from 'lucide-vue-next'

const emit = defineEmits(['navigate'])

const currentPath = ref('')
const entries = ref([])
const loading = ref(false)
const error = ref('')

const thumbnails = ref({})
const loadedSet = new Set()
let loadGeneration = 0

const viewerIndex = ref(-1)
const imageEntries = computed(() => entries.value.filter(e => !e.is_dir))
const imageIndexMap = computed(() => {
  const map = new Map()
  imageEntries.value.forEach((e, i) => map.set(e.path, i))
  return map
})

const fullImages = ref({})
const fullLoading = ref(new Set())

const viewerImage = computed(() => {
  if (viewerIndex.value < 0 || viewerIndex.value >= imageEntries.value.length) return ''
  const entry = imageEntries.value[viewerIndex.value]
  return fullImages.value[entry.path] || thumbnails.value[entry.path] || ''
})

let observer = null

function setupObserver() {
  observer = new IntersectionObserver((items) => {
    for (const item of items) {
      if (item.isIntersecting) {
        const path = item.target.dataset.path
        if (path && !loadedSet.has(path)) {
          loadedSet.add(path)
          loadSingleThumbnail(path)
        }
        observer.unobserve(item.target)
      }
    }
  }, { rootMargin: '200px' })
}

function observeCards() {
  if (!observer) return
  const cards = document.querySelectorAll('.fb-card-img[data-path]')
  cards.forEach(el => observer.observe(el))
}

async function loadSingleThumbnail(path) {
  try {
    const b64 = await api.readThumbnail(path)
    if (b64) thumbnails.value[path] = b64
  } catch {}
}

async function loadPath() {
  if (!currentPath.value) return
  loading.value = true
  error.value = ''
  thumbnails.value = {}
  fullImages.value = {}
  loadedSet.clear()
  viewerIndex.value = -1
  loadGeneration++
  const gen = loadGeneration
  try {
    entries.value = await api.browseDirectory(currentPath.value)
    if (gen !== loadGeneration) return
    requestAnimationFrame(() => observeCards())
  } catch (e) {
    error.value = String(e)
    entries.value = []
  } finally {
    loading.value = false
  }
}

async function browseFolder() {
  try {
    const path = await api.selectBrowserFolder()
    if (path) {
      currentPath.value = path
      await loadPath()
    }
  } catch (e) {
    error.value = String(e)
  }
}

function goUp() {
  if (!currentPath.value) return
  const p = currentPath.value.replace(/[/\\]+$/, '')
  const sep = p.includes('\\') ? '\\' : '/'
  const idx = p.lastIndexOf(sep)
  if (idx <= 0) return
  currentPath.value = p.substring(0, idx) || sep
  loadPath()
}

function openDir(entry) {
  currentPath.value = entry.path
  loadPath()
}

function openViewer(index) {
  viewerIndex.value = index
  loadFullImage(imageEntries.value[index])
}

function viewerPrev() {
  if (viewerIndex.value > 0) {
    viewerIndex.value--
    loadFullImage(imageEntries.value[viewerIndex.value])
  }
}

function viewerNext() {
  if (viewerIndex.value < imageEntries.value.length - 1) {
    viewerIndex.value++
    loadFullImage(imageEntries.value[viewerIndex.value])
  }
}

async function loadFullImage(entry) {
  if (!entry || fullImages.value[entry.path] || fullLoading.value.has(entry.path)) return
  fullLoading.value.add(entry.path)
  try {
    const b64 = await api.readFileAsBase64(entry.path)
    if (b64) fullImages.value[entry.path] = b64
  } catch {}
  fullLoading.value.delete(entry.path)
}

async function sendToFromImage(entry) {
  try {
    const b64 = await api.readFileAsBase64(entry.path)
    if (!b64) return
    await api.setLastImage(b64)
    emit('navigate', { page: 'generate', tab: 'from-image' })
  } catch (e) {
    error.value = String(e)
  }
}

async function sendToExport(entry) {
  try {
    const b64 = await api.readFileAsBase64(entry.path)
    if (!b64) return
    await api.setLastImage(b64)
    emit('navigate', { page: 'export' })
  } catch (e) {
    error.value = String(e)
  }
}

function formatSize(bytes) {
  if (!bytes) return ''
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function mimeForEntry(entry) {
  const ext = (entry.name.split('.').pop() || '').toLowerCase()
  if (ext === 'jpg' || ext === 'jpeg') return 'image/jpeg'
  if (ext === 'webp') return 'image/webp'
  return 'image/png'
}

function thumbSrc(entry) {
  const b64 = thumbnails.value[entry.path]
  if (!b64) return ''
  return `data:${mimeForEntry(entry)};base64,` + b64
}

function getEntryIndex(entry) {
  return imageIndexMap.value.get(entry.path) ?? -1
}

let lastClickTime = 0
let lastClickPath = ''

function handleCardClick(entry) {
  if (entry.is_dir) {
    openDir(entry)
  } else {
    openViewer(getEntryIndex(entry))
  }
}

onMounted(async () => {
  setupObserver()
  try {
    const settings = await api.getSettings()
    if (settings?.file_browser_path) {
      currentPath.value = settings.file_browser_path
      await loadPath()
    }
  } catch {}
})

onUnmounted(() => {
  if (observer) observer.disconnect()
  loadGeneration++
})
</script>

<template>
  <div class="file-browser">
    <div class="fb-path-bar">
      <input
        v-model="currentPath"
        class="fb-path-input"
        :placeholder="t('browser.placeholder_path')"
        @keydown.enter="loadPath"
      />
      <button class="btn btn-secondary" @click="browseFolder">
        <FolderOpen :size="14" /> {{ t('browser.btn_browse') }}
      </button>
      <button class="btn btn-secondary" @click="goUp" :disabled="!currentPath">
        <FolderUp :size="14" />
      </button>
      <button class="btn btn-primary" @click="loadPath" :disabled="!currentPath || loading">
        {{ t('browser.btn_go') }}
      </button>
    </div>

    <div v-if="error" class="fb-error">{{ error }}</div>

    <div v-if="loading" class="fb-loading">{{ t('browser.loading') }}</div>

    <div v-else-if="entries.length" class="fb-grid">
      <div
        v-for="entry in entries"
        :key="entry.path"
        class="fb-card"
        :class="{ 'fb-card-dir': entry.is_dir }"
        @click="handleCardClick(entry)"
      >
        <template v-if="entry.is_dir">
          <div class="fb-thumb fb-thumb-dir">
            <Folder :size="32" />
          </div>
          <div class="fb-card-info">
            <div class="fb-card-name">{{ entry.name }}</div>
          </div>
        </template>
        <template v-else>
          <div class="fb-thumb fb-card-img" :data-path="entry.path">
            <img v-if="thumbnails[entry.path]" :src="thumbSrc(entry)" :alt="entry.name" />
            <Image v-else :size="32" class="fb-thumb-placeholder" />
          </div>
          <div class="fb-card-info">
            <div class="fb-card-name">{{ entry.name }}</div>
            <div class="fb-card-meta">{{ formatSize(entry.size) }}</div>
          </div>
          <div class="fb-card-actions" @click.stop>
            <button class="fb-action-btn" :title="t('browser.send_to_from_image')" @click="sendToFromImage(entry)">
              <Send :size="12" />
            </button>
            <button class="fb-action-btn" :title="t('browser.send_to_export')" @click="sendToExport(entry)">
              <Download :size="12" />
            </button>
          </div>
        </template>
      </div>
    </div>

    <div v-else-if="currentPath && !loading" class="fb-empty">
      {{ t('browser.no_images') }}
    </div>

    <div v-else-if="!currentPath" class="fb-empty">
      {{ t('browser.select_folder') }}
    </div>

    <ImageViewer
      v-if="viewerIndex >= 0 && viewerImage"
      :imageBase64="viewerImage"
      :hasPrev="viewerIndex > 0"
      :hasNext="viewerIndex < imageEntries.length - 1"
      @close="viewerIndex = -1"
      @prev="viewerPrev"
      @next="viewerNext"
    />
  </div>
</template>

<style scoped>
.file-browser {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.fb-path-bar {
  display: flex;
  gap: 8px;
  align-items: center;
}

.fb-path-input {
  flex: 1;
  padding: 8px 12px;
  background: var(--surface-1);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text);
  font-size: 13px;
}

.fb-path-input:focus {
  outline: none;
  border-color: var(--primary, #7c5cfc);
}

.fb-error {
  color: var(--error, #e55);
  font-size: 13px;
  padding: 8px;
}

.fb-loading, .fb-empty {
  text-align: center;
  padding: 40px 16px;
  color: var(--text-dim);
  font-size: 14px;
}

.fb-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 12px;
}

.fb-card {
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
  background: var(--surface-1);
  cursor: pointer;
  transition: border-color 0.2s;
  position: relative;
}

.fb-card:hover {
  border-color: var(--primary, #7c5cfc);
}

.fb-card-dir:hover {
  border-color: var(--text-dim);
}

.fb-thumb {
  width: 100%;
  aspect-ratio: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--surface-2);
  overflow: hidden;
}

.fb-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.fb-thumb-dir {
  background: var(--surface-1);
  color: var(--text-dim);
}

.fb-thumb-placeholder {
  color: var(--text-dim);
}

.fb-card-info {
  padding: 8px 10px;
}

.fb-card-name {
  font-size: 12px;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.fb-card-meta {
  font-size: 11px;
  color: var(--text-dim);
  margin-top: 2px;
}

.fb-card-actions {
  position: absolute;
  top: 6px;
  right: 6px;
  display: flex;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.2s;
}

.fb-card:hover .fb-card-actions {
  opacity: 1;
}

.fb-action-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text);
  cursor: pointer;
  font-size: 12px;
}

.fb-action-btn:hover {
  background: var(--primary, #7c5cfc);
  color: #fff;
  border-color: var(--primary, #7c5cfc);
}
</style>
