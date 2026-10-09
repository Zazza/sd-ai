import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { clearEventMocks } from './mocks/runtime'

vi.mock('../wailsjs/runtime/runtime', () => import('./mocks/runtime'))
vi.mock('../wailsjs/go/main/App.js', async () => (await import('./mocks/wails-app')).default)

import GenerateFromImagePage from '../components/GenerateFromImagePage.vue'
import ResolutionSelector from '../components/ResolutionSelector.vue'
import { GetSettings, ListResolutions } from '../wailsjs/go/main/App.js'
import { t } from '../i18n/index.js'

const resolution = { id: 7, name: 'HiRes', width: 1024, height: 1024, is_builtin: false }

function seedInput(wrapper) {
  return wrapper.find('[data-testid="fi-seed-input"]')
}

function qualityCheckbox(wrapper) {
  return wrapper.find('[data-testid="fi-quality-checkbox"]')
}

async function mountPage(settings = {}) {
  GetSettings.mockReturnValue(Promise.resolve(settings))
  const wrapper = mount(GenerateFromImagePage)
  await flushPromises()
  return wrapper
}

enableAutoUnmount(afterEach)

describe('From Image: seed and quality mode fields by genMode', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    clearEventMocks()
  })

  it('compound mode: seed and quality checkbox visible but disabled, preset-only hint rendered', async () => {
    const wrapper = await mountPage({ fi_mode: 'img2img', fi_gen_mode: 'compound' })

    expect(seedInput(wrapper).exists()).toBe(true)
    expect(seedInput(wrapper).element.disabled).toBe(true)
    expect(qualityCheckbox(wrapper).exists()).toBe(true)
    expect(qualityCheckbox(wrapper).element.disabled).toBe(true)
    expect(wrapper.text()).toContain(t('fi.quality_mode'))
    expect(wrapper.text()).toContain(t('fi.preset_only'))
    expect(wrapper.text()).not.toContain(t('fi.quality_hint'))
  })

  it('preset mode with resolution: seed and quality checkbox enabled, no hints', async () => {
    ListResolutions.mockReturnValueOnce(Promise.resolve([resolution]))
    const wrapper = await mountPage({ fi_mode: 'img2img', fi_gen_mode: 'preset' })

    wrapper.findComponent(ResolutionSelector).vm.$emit('update:modelValue', resolution.id)
    await nextTick()

    expect(seedInput(wrapper).element.disabled).toBe(false)
    expect(qualityCheckbox(wrapper).element.disabled).toBe(false)
    expect(wrapper.text()).toContain(t('fi.quality_hint'))
    expect(wrapper.text()).not.toContain(t('fi.preset_only'))
    expect(wrapper.text()).not.toContain(t('fi.quality_needs_resolution'))
  })

  it('preset mode without resolution: quality checkbox disabled, needs-resolution hint rendered', async () => {
    const wrapper = await mountPage({ fi_mode: 'img2img', fi_gen_mode: 'preset' })

    expect(seedInput(wrapper).element.disabled).toBe(false)
    expect(qualityCheckbox(wrapper).element.disabled).toBe(true)
    expect(wrapper.text()).toContain(t('fi.quality_hint'))
    expect(wrapper.text()).toContain(t('fi.quality_needs_resolution'))
    expect(wrapper.text()).not.toContain(t('fi.preset_only'))
  })

  it('preset inpaint mode: seed enabled, quality checkbox absent', async () => {
    const wrapper = await mountPage({ fi_mode: 'inpaint', fi_gen_mode: 'preset' })

    expect(seedInput(wrapper).exists()).toBe(true)
    expect(seedInput(wrapper).element.disabled).toBe(false)
    expect(qualityCheckbox(wrapper).exists()).toBe(false)
  })

  it('compound inpaint mode: seed disabled with preset-only hint, quality checkbox absent', async () => {
    const wrapper = await mountPage({ fi_mode: 'inpaint', fi_gen_mode: 'compound' })

    expect(seedInput(wrapper).exists()).toBe(true)
    expect(seedInput(wrapper).element.disabled).toBe(true)
    expect(wrapper.text()).toContain(t('fi.preset_only'))
    expect(qualityCheckbox(wrapper).exists()).toBe(false)
  })

  it('remove mode: seed and resolution selector hidden', async () => {
    const wrapper = await mountPage({ fi_mode: 'remove' })

    expect(seedInput(wrapper).exists()).toBe(false)
    expect(wrapper.findComponent(ResolutionSelector).exists()).toBe(false)
  })
})
