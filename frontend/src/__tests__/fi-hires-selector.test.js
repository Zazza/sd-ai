import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { clearEventMocks } from './mocks/runtime'

vi.mock('../wailsjs/runtime/runtime', () => import('./mocks/runtime'))
vi.mock('../wailsjs/go/main/App.js', async () => (await import('./mocks/wails-app')).default)

import GenerateFromImagePage from '../components/GenerateFromImagePage.vue'
import HiresProfileSelector from '../components/HiresProfileSelector.vue'
import { GetSettings, ListHiresProfiles, ListPresets, ListCompoundPresets, EnqueueFromImage } from '../wailsjs/go/main/App.js'
import { t } from '../i18n/index.js'

const hiresProfile = { id: 7, name: 'Test Hires', upscale: 2, denoising_strength: 0.5, upscaler: 'R-ESRGAN 4x+', is_builtin: false }

function hireSelector(wrapper) {
  return wrapper.findComponent(HiresProfileSelector)
}

async function mountPage(settings = {}) {
  GetSettings.mockReturnValue(Promise.resolve(settings))
  const wrapper = mount(GenerateFromImagePage)
  await flushPromises()
  return wrapper
}

async function mountWithImage(settings = {}) {
  const wrapper = await mountPage(settings)
  await wrapper.setProps({ droppedImage: 'data:image/png;base64,abc' })
  await flushPromises()
  return wrapper
}

function generateButton(wrapper) {
  return wrapper.findAll('button').find(b => b.text().includes(t('fi.btn_generate')))
}

enableAutoUnmount(afterEach)

describe('From Image: hires profile selector', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ListHiresProfiles.mockImplementation(() => Promise.resolve([]))
    clearEventMocks()
  })

  it('compound img2img: selector mounted with hint', async () => {
    const wrapper = await mountPage({ fi_mode: 'img2img', fi_gen_mode: 'compound' })

    expect(hireSelector(wrapper).exists()).toBe(true)
    expect(wrapper.text()).toContain(t('fi.hires_hint'))
  })

  it('preset img2img: selector absent', async () => {
    const wrapper = await mountPage({ fi_mode: 'img2img', fi_gen_mode: 'preset' })

    expect(hireSelector(wrapper).exists()).toBe(false)
    expect(wrapper.text()).not.toContain(t('fi.hires_hint'))
  })

  it('remove mode: selector absent even in compound genMode', async () => {
    const wrapper = await mountPage({ fi_mode: 'remove', fi_gen_mode: 'compound' })

    expect(hireSelector(wrapper).exists()).toBe(false)
    expect(wrapper.text()).not.toContain(t('fi.hires_hint'))
  })

  it('restore: valid fi_hires_profile_id selects profile', async () => {
    ListHiresProfiles.mockReturnValue(Promise.resolve([hiresProfile]))
    const wrapper = await mountPage({ fi_mode: 'img2img', fi_gen_mode: 'compound', fi_hires_profile_id: '7' })

    expect(hireSelector(wrapper).props('modelValue')).toBe(7)
  })

  it('restore: unknown fi_hires_profile_id is dropped', async () => {
    ListHiresProfiles.mockReturnValue(Promise.resolve([hiresProfile]))
    const wrapper = await mountPage({ fi_mode: 'img2img', fi_gen_mode: 'compound', fi_hires_profile_id: '999' })

    expect(hireSelector(wrapper).props('modelValue')).toBe(null)
  })

  it('generate in compound mode: enqueue receives hires_profile_id', async () => {
    ListHiresProfiles.mockReturnValue(Promise.resolve([hiresProfile]))
    ListCompoundPresets.mockReturnValueOnce(Promise.resolve([{ id: 3, name: 'Pipeline', steps: [] }]))
    const wrapper = await mountWithImage({ fi_mode: 'img2img', fi_gen_mode: 'compound', fi_compound_preset_id: '3', fi_hires_profile_id: '7' })

    const btn = generateButton(wrapper)
    expect(btn.element.disabled).toBe(false)
    await btn.trigger('click')
    await flushPromises()

    expect(EnqueueFromImage).toHaveBeenCalledTimes(1)
    expect(EnqueueFromImage.mock.calls[0][0]).toEqual(expect.objectContaining({
      gen_mode: 'compound',
      compound_preset_id: 3,
      hires_profile_id: 7,
    }))
  })

  it('generate in preset mode: enqueue receives hires_profile_id null even when restored', async () => {
    ListHiresProfiles.mockReturnValue(Promise.resolve([hiresProfile]))
    ListPresets.mockReturnValueOnce(Promise.resolve([{ id: 5, name: 'Style', type_id: 1 }]))
    const wrapper = await mountWithImage({ fi_mode: 'img2img', fi_gen_mode: 'preset', fi_preset_id: '5', fi_hires_profile_id: '7' })

    expect(hireSelector(wrapper).exists()).toBe(false)
    const btn = generateButton(wrapper)
    expect(btn.element.disabled).toBe(false)
    await btn.trigger('click')
    await flushPromises()

    expect(EnqueueFromImage).toHaveBeenCalledTimes(1)
    expect(EnqueueFromImage.mock.calls[0][0]).toEqual(expect.objectContaining({
      gen_mode: 'preset',
      preset_id: 5,
      hires_profile_id: null,
    }))
  })
})
