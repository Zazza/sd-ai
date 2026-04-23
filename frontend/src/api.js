import {
  ListPresets, ListPresetsByType, CreatePreset, UpdatePreset, DeletePreset,
  GenerateSDPrompt, GenerateImage, SaveImage,
  GetSDModels, GetSDSamplers, GetLLMModels,
  GetSettings, UpdateSettings,
  ListDescriptions, CreateDescription, DeleteDescription,
  ListPrompts, CreatePrompt, DeletePrompt,
} from './wailsjs/go/main/App.js'

export const api = {
  listPresets: () => ListPresets(),
  listPresetsByType: (type) => ListPresetsByType(type),
  createPreset: (data) => CreatePreset(data),
  updatePreset: (id, data) => UpdatePreset({ ...data, id }),
  deletePreset: (id) => DeletePreset(id),
  generateSdPrompt: (description, presetType) => GenerateSDPrompt(description, presetType),
  generateImage: (presetId, extraPrompt, extraNegativePrompt) =>
    GenerateImage({ preset_id: presetId, extra_prompt: extraPrompt, extra_negative_prompt: extraNegativePrompt }),
  saveImage: (base64Data, defaultName) => SaveImage(base64Data, defaultName),
  getModels: () => GetSDModels(),
  getSamplers: () => GetSDSamplers(),
  getLLMModels: () => GetLLMModels(),
  getSettings: () => GetSettings(),
  updateSettings: (data) => UpdateSettings(data),
  listDescriptions: () => ListDescriptions(),
  createDescription: (text) => CreateDescription(text),
  deleteDescription: (id) => DeleteDescription(id),
  listPrompts: () => ListPrompts(),
  createPrompt: (text) => CreatePrompt(text),
  deletePrompt: (id) => DeletePrompt(id),
}
