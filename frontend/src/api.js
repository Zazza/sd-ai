import {
  ListPresets, ListPresetsByType, CreatePreset, UpdatePreset, DeletePreset,
  GenerateSDPrompt, GenerateImage, UpscaleImage, UpscalePreview, SaveImage, FastSaveImage,
  GetSDModels, GetSDSamplers, GetSDSchedulers, GetSDUpscalers, GetSDVAEs, GetLLMModels,
  GetSettings, UpdateSettings, CheckServices,
  ListDescriptions, CreateDescription, CreateDescriptionFull, UpdateDescription, DeleteDescription,
  SetKidsMode, IsKidsModeActive, GetKidsCategories, SetKidsCategory,
  ExportPresets, PreparePresetsExport, OpenImportFile, ImportPresets,
  ExportCompoundPresets, PrepareCompoundPresetsExport, OpenImportCompoundFile, ImportCompoundPresets,
  AnalyzeImage, ReadImageFile, ReadClipboardImage,
  ListPresetTypes, GetPresetType, CreatePresetType, UpdatePresetType, DeletePresetType,
  GetAllTags, GetSDLoRAs, ValidateImportModels,
  RecommendPreset, GetDefaultPromptInstruction, GetDefaultAnalyzePrompts,
  TestGenerate,
  ListCompoundPresets, GetCompoundPreset, CreateCompoundPreset, UpdateCompoundPreset, DeleteCompoundPreset,
  GenerateCompoundImage, TestCompoundGenerate,
  GenerateFromImage,
  DecomposeScene, GenerateMultiPass,
  ListSavedScenes, GetSavedScene, SaveScene, UpdateSavedScene, DeleteSavedScene,
  CheckRembg,
  ClearLastImage,
  ListExportPresets, SaveExportPreset, DeleteExportPreset, ExportImage,
  BrowseDirectory, ReadFileAsBase64, ReadThumbnail, SelectBrowserFolder, SetLastImage,
  CreateSession, ListSessions, SwitchSession, RenameSession, DeleteSession,
  GetSessionItems, GetActiveSessionItem, SetActiveSessionItem, DeleteSessionItem,
  ClearSession, GetSessionImage, GetSessionThumb,
  InterruptGeneration,
  DiscoverServers, GetServerStatus, GetServerModels, GetServerLLMModels,
  DownloadServerModel, DeleteServerModel, PullServerLLMModel, DeleteServerLLMModel,
  GetServerBackends, SwitchServerBackend, GetModelCatalog,
  GetPresetsInstallStatus, InstallPresetDeps,
  StartServerProcess, StopServerProcess, RestartServerProcess,
  Version,
  ListResolutions, CreateResolution, UpdateResolution, DeleteResolution,
  ListHiresProfiles, CreateHiresProfile, UpdateHiresProfile, DeleteHiresProfile,
  EnqueueTxt2Img, EnqueueFromImage, EnqueueCompound, EnqueueCompareItem,
  GetQueue, RemoveQueueJob, CancelQueueJob, PauseQueue, ResumeQueue, CancelQueue,
  IsQueuePaused, ClearCompletedQueueJobs, ResumePausedQueueJobs,
  SaveWindowLayout, GetFooterHeight,
} from './wailsjs/go/main/App.js'

export const api = {
  listPresets: () => ListPresets(),
  listPresetsByType: (type) => ListPresetsByType(type),
  createPreset: (data) => CreatePreset(data),
  updatePreset: (id, data) => UpdatePreset({ ...data, id }),
  deletePreset: (id) => DeletePreset(id),
  generateSdPrompt: (params) => GenerateSDPrompt(params),
  generateImage: (presetId, extraPrompt, extraNegativePrompt, resolutionId, hiresProfileId) =>
    GenerateImage({ preset_id: presetId, extra_prompt: extraPrompt, extra_negative_prompt: extraNegativePrompt, resolution_id: resolutionId || null, hires_profile_id: hiresProfileId || null }),
  upscalePreview: (previewImageBase64, presetId, seed, resolutionId, hiresProfileId) =>
    UpscalePreview({ preview_image_base64: previewImageBase64, preset_id: presetId, seed, resolution_id: resolutionId || null, hires_profile_id: hiresProfileId || null }),
  upscaleImage: (imageBase64, genInfo, presetId) =>
    UpscaleImage({ image_base64: imageBase64, gen_info: typeof genInfo === 'string' ? genInfo : JSON.stringify(genInfo || {}), preset_id: presetId || 0 }),
  saveImage: (base64Data, defaultName) => SaveImage(base64Data, defaultName),
  fastSaveImage: (base64Data, filename, format) => FastSaveImage(base64Data, filename, format),
  getModels: () => GetSDModels(),
  getSamplers: () => GetSDSamplers(),
  getSchedulers: () => GetSDSchedulers(),
  getUpscalers: () => GetSDUpscalers(),
  getVAEs: () => GetSDVAEs(),
  getLLMModels: () => GetLLMModels(),
  getSettings: () => GetSettings(),
  updateSettings: (data) => UpdateSettings(data),
  checkServices: () => CheckServices(),
  listDescriptions: () => ListDescriptions(),
  createDescription: (text) => CreateDescription(text),
  createDescriptionFull: (data) => CreateDescriptionFull(data),
  updateDescription: (data) => UpdateDescription(data),
  deleteDescription: (id) => DeleteDescription(id),
  setKidsMode: (enabled, pin) => SetKidsMode(enabled, pin),
  isKidsModeActive: () => IsKidsModeActive(),
  getKidsCategories: () => GetKidsCategories(),
  setKidsCategory: (name, enabled, pin) => SetKidsCategory(name, enabled, pin),
  exportPresets: (ids) => ExportPresets(ids),
  preparePresetsExport: (ids) => PreparePresetsExport(ids),
  openImportFile: () => OpenImportFile(),
  importPresets: (items) => ImportPresets(items),
  analyzeImage: (imageBase64) => AnalyzeImage(imageBase64),
  readImageFile: () => ReadImageFile(),
  readClipboardImage: () => ReadClipboardImage(),
  listPresetTypes: () => ListPresetTypes(),
  getPresetType: (id) => GetPresetType(id),
  createPresetType: (data) => CreatePresetType(data),
  updatePresetType: (data) => UpdatePresetType(data),
  deletePresetType: (id) => DeletePresetType(id),
  getAllTags: () => GetAllTags(),
  getLoRAs: () => GetSDLoRAs(),
  validateImportModels: (items) => ValidateImportModels(items),
  exportCompoundPresets: (ids) => ExportCompoundPresets(ids),
  prepareCompoundPresetsExport: (ids) => PrepareCompoundPresetsExport(ids),
  openImportCompoundFile: () => OpenImportCompoundFile(),
  importCompoundPresets: (items) => ImportCompoundPresets(items),
  recommendPreset: (description) => RecommendPreset(description),
  getDefaultPromptInstruction: () => GetDefaultPromptInstruction(),
  getDefaultAnalyzePrompts: () => GetDefaultAnalyzePrompts(),
  testGenerate: (params) => TestGenerate(params),
  listCompoundPresets: () => ListCompoundPresets(),
  getCompoundPreset: (id) => GetCompoundPreset(id),
  createCompoundPreset: (data) => CreateCompoundPreset(data),
  updateCompoundPreset: (data) => UpdateCompoundPreset(data),
  deleteCompoundPreset: (id) => DeleteCompoundPreset(id),
  generateCompoundImage: (params) => GenerateCompoundImage(params),
  testCompoundGenerate: (params) => TestCompoundGenerate(params),
  generateFromImage: (params) => GenerateFromImage(params),
  decomposeScene: (params) => DecomposeScene(params),
  generateMultiPass: (scene) => GenerateMultiPass(scene),
  listSavedScenes: () => ListSavedScenes(),
  getSavedScene: (id) => GetSavedScene(id),
  saveScene: (data) => SaveScene(data),
  updateSavedScene: (data) => UpdateSavedScene(data),
  deleteSavedScene: (id) => DeleteSavedScene(id),
  checkRembg: () => CheckRembg(),
  clearLastImage: () => ClearLastImage(),
  listExportPresets: () => ListExportPresets(),
  saveExportPreset: (data) => SaveExportPreset(data),
  deleteExportPreset: (id) => DeleteExportPreset(id),
  exportImage: (params) => ExportImage(params),
  browseDirectory: (path) => BrowseDirectory(path),
  readFileAsBase64: (path) => ReadFileAsBase64(path),
  readThumbnail: (path) => ReadThumbnail(path),
  selectBrowserFolder: () => SelectBrowserFolder(),
  setLastImage: (base64) => SetLastImage(base64),

  createSession: (name) => CreateSession(name),
  listSessions: () => ListSessions(),
  switchSession: (id) => SwitchSession(id),
  renameSession: (id, name) => RenameSession(id, name),
  deleteSession: (id) => DeleteSession(id),
  getSessionItems: () => GetSessionItems(),
  getActiveSessionItem: () => GetActiveSessionItem(),
  setActiveSessionItem: (id) => SetActiveSessionItem(id),
  deleteSessionItem: (id) => DeleteSessionItem(id),
  clearSession: () => ClearSession(),
  getSessionImage: (id) => GetSessionImage(id),
  getSessionThumb: (id) => GetSessionThumb(id),
  sessionImageUrl: (id) => `/api/img/${id}.jpg`,
  sessionThumbUrl: (id) => `/api/thumb/${id}.jpg`,
  interruptGeneration: () => InterruptGeneration(),
  version: () => Version(),
  discoverServers: () => DiscoverServers(),
  getServerStatus: () => GetServerStatus(),
  getServerModels: (type) => GetServerModels(type),
  getServerLLMModels: () => GetServerLLMModels(),
  downloadServerModel: (type, url, filename) => DownloadServerModel(type, url, filename),
  downloadServerModelStream: (serverURL, type, downloadURL, filename, onProgress) => {
    return new Promise((resolve, reject) => {
      const url = `${serverURL}/api/server/models/download/stream?type=${encodeURIComponent(type)}&url=${encodeURIComponent(downloadURL)}&filename=${encodeURIComponent(filename)}`
      const es = new EventSource(url)
      es.onmessage = (e) => {
        if (e.data === '[DONE]') {
          es.close()
          resolve()
        } else if (e.data.startsWith('[ERROR]')) {
          es.close()
          reject(new Error(e.data.slice(7).trim()))
        } else {
          try {
            const p = JSON.parse(e.data)
            const pct = p.total > 0 ? ` (${p.percent.toFixed(1)}%)` : ''
            onProgress(`${formatBytes(p.downloaded)} / ${p.total > 0 ? formatBytes(p.total) : '???'}${pct}`)
          } catch {
            onProgress(e.data)
          }
        }
      }
      es.onerror = () => {
        es.close()
        reject(new Error('Connection lost'))
      }
    })
  },
  deleteServerModel: (type, filename) => DeleteServerModel(type, filename),
  pullServerLLMModel: (name) => PullServerLLMModel(name),
  pullServerLLMModelStream: (serverURL, name, onProgress) => {
    return new Promise((resolve, reject) => {
      const url = `${serverURL}/api/server/models/llm/pull?name=${encodeURIComponent(name)}`
      const es = new EventSource(url)
      es.onmessage = (e) => {
        if (e.data === '[DONE]') {
          es.close()
          resolve()
        } else if (e.data.startsWith('[ERROR]')) {
          es.close()
          reject(new Error(e.data.slice(7).trim()))
        } else {
          onProgress(e.data)
        }
      }
      es.onerror = (e) => {
        es.close()
        reject(new Error('Connection lost'))
      }
    })
  },
  deleteServerLLMModel: (name) => DeleteServerLLMModel(name),
  getServerBackends: () => GetServerBackends(),
  switchServerBackend: (backend) => SwitchServerBackend(backend),
  getModelCatalog: () => GetModelCatalog(),
  getPresetsInstallStatus: () => GetPresetsInstallStatus(),
  installPresetDeps: (id) => InstallPresetDeps(id),
  startServerProcess: (name) => StartServerProcess(name),
  stopServerProcess: (name) => StopServerProcess(name),
  restartServerProcess: (name) => RestartServerProcess(name),

  listResolutions: () => ListResolutions(),
  createResolution: (data) => CreateResolution(data),
  updateResolution: (data) => UpdateResolution(data),
  deleteResolution: (id) => DeleteResolution(id),
  listHiresProfiles: () => ListHiresProfiles(),
  createHiresProfile: (data) => CreateHiresProfile(data),
  updateHiresProfile: (data) => UpdateHiresProfile(data),
  deleteHiresProfile: (id) => DeleteHiresProfile(id),

  enqueueTxt2Img: (params) => EnqueueTxt2Img(params),
  enqueueFromImage: (params) => EnqueueFromImage(params),
  enqueueCompound: (params) => EnqueueCompound(params),
  enqueueCompareItem: (params, modelIndex) => EnqueueCompareItem(params, modelIndex),
  getQueue: () => GetQueue(),
  removeQueueJob: (id) => RemoveQueueJob(id),
  cancelQueueJob: (id) => CancelQueueJob(id),
  pauseQueue: () => PauseQueue(),
  resumeQueue: () => ResumeQueue(),
  cancelQueue: () => CancelQueue(),
  isQueuePaused: () => IsQueuePaused(),
  clearCompletedQueueJobs: () => ClearCompletedQueueJobs(),
  resumePausedQueueJobs: () => ResumePausedQueueJobs(),
  saveWindowLayout: (footerHeight) => SaveWindowLayout(footerHeight),
  getFooterHeight: () => GetFooterHeight(),
}

function formatBytes(b) {
  if (b >= 1073741824) return (b / 1073741824).toFixed(1) + ' GB'
  if (b >= 1048576) return (b / 1048576).toFixed(1) + ' MB'
  if (b >= 1024) return (b / 1024).toFixed(1) + ' KB'
  return b + ' B'
}
