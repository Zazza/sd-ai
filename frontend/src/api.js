import {
  ListPresets, ListPresetsByType, CreatePreset, UpdatePreset, DeletePreset,
  GenerateSDPrompt, GenerateImage,
  GetSDModels, GetSDSamplers, GetLLMModels,
  GetSettings, UpdateSettings,
  ListDescriptions, CreateDescription, DeleteDescription,
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
  getModels: () => GetSDModels(),
  getSamplers: () => GetSDSamplers(),
  getLLMModels: () => GetLLMModels(),
  getSettings: () => GetSettings(),
  updateSettings: (data) => UpdateSettings(data),
  listDescriptions: () => ListDescriptions(),
  createDescription: (text) => CreateDescription(text),
  deleteDescription: (id) => DeleteDescription(id),
}
