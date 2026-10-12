import { parseLoras, buildLorasOverride } from './loraOverride.js'

export const MAX_CUSTOM_STEPS = 8

export function newCustomStep() {
  return { preset_id: null, denoising_strength: 0.5, loraRows: [], loraIgnoreAll: false }
}

export function normalizeCustomStep(raw) {
  const step = newCustomStep()
  if (!raw || typeof raw !== 'object') return step
  const pid = Number(raw.preset_id)
  if (Number.isFinite(pid) && pid > 0) step.preset_id = pid
  const d = Number(raw.denoising_strength)
  step.denoising_strength = Number.isFinite(d) ? Math.min(1, Math.max(0, d)) : 0.5
  if (Array.isArray(raw.loraRows)) {
    step.loraRows = raw.loraRows
      .filter(r => r && typeof r === 'object' && String(r.name || '').trim())
      .map(r => ({ name: String(r.name).trim(), weight: Number(r.weight) }))
  }
  step.loraIgnoreAll = !!raw.loraIgnoreAll
  return step
}

export function customPresetLoras(presets, presetId) {
  const p = (presets || []).find(x => x.id === presetId)
  return p ? parseLoras(p.loras) : []
}

export function customStepLoras(step, presets) {
  if (!step) return null
  return buildLorasOverride(customPresetLoras(presets, step.preset_id), step.loraRows || [], !!step.loraIgnoreAll)
}

export function customStepsPayload(steps, presets) {
  return (steps || []).map(s => {
    const item = { preset_id: s.preset_id, denoising_strength: s.denoising_strength }
    const loras = customStepLoras(s, presets)
    if (loras !== null) item.loras = loras
    return item
  })
}

export function persistCustomSteps(steps, presets) {
  return (steps || []).map(s => {
    const item = { preset_id: s.preset_id, denoising_strength: s.denoising_strength }
    if (customStepLoras(s, presets) !== null) {
      item.loraRows = s.loraRows || []
      item.loraIgnoreAll = !!s.loraIgnoreAll
    }
    return item
  })
}

export function restoreCustomSteps(raw, presets) {
  if (!Array.isArray(raw)) return [newCustomStep()]
  const ids = (presets || []).map(p => p.id)
  const steps = raw
    .slice(0, MAX_CUSTOM_STEPS)
    .map((item, i) => {
      const s = normalizeCustomStep(item)
      if (i > 0 && s.denoising_strength < 0.05) s.denoising_strength = 0.05
      if (!item || !Array.isArray(item.loraRows)) {
        s.loraRows = customPresetLoras(presets, s.preset_id).map(l => ({ name: l.name, weight: l.weight }))
      }
      return s
    })
    .filter(s => s.preset_id !== null)
    .filter(s => ids.includes(s.preset_id))
  return steps.length ? steps : [newCustomStep()]
}
