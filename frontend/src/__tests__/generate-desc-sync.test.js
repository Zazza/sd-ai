import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { clearEventMocks } from './mocks/runtime'

vi.mock('../wailsjs/runtime/runtime', () => import('./mocks/runtime'))
vi.mock('../wailsjs/go/main/App.js', async () => (await import('./mocks/wails-app')).default)

import GeneratePage from '../components/GeneratePage.vue'
import SavedDescriptionsModal from '../components/SavedDescriptionsModal.vue'
import { UpdateDescription } from '../wailsjs/go/main/App.js'
import { t } from '../i18n/index.js'

enableAutoUnmount(afterEach)

function descField(wrapper) {
  return wrapper.findAll('textarea').find(ta => ta.attributes('placeholder') === t('generate.placeholder_description'))
}

function negField(wrapper) {
  return wrapper.findAll('textarea').find(ta => ta.attributes('placeholder') === t('generate.placeholder_exclude'))
}

async function openSavedIdeas(wrapper) {
  const label = t('generate.saved_ideas')
  const toggle = wrapper.findAll('button').find(b => b.text().includes(label))
  await toggle.trigger('click')
  return wrapper.findComponent(SavedDescriptionsModal)
}

async function mountPage() {
  const wrapper = mount(GeneratePage)
  await flushPromises()
  return wrapper
}

async function emitAndClose(modal, event, payload) {
  modal.vm.$emit(event, payload)
  await flushPromises()
  modal.vm.$emit('close')
  await flushPromises()
}

describe('GeneratePage description sync', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    clearEventMocks()
  })

  it('updates description and negative when the loaded idea is edited in the modal', async () => {
    const wrapper = await mountPage()

    const modal = await openSavedIdeas(wrapper)
    await emitAndClose(modal, 'use', { id: 7, text: 'old idea', negative_prompt: 'old neg' })
    expect(descField(wrapper).element.value).toBe('old idea')
    expect(negField(wrapper).element.value).toBe('old neg')

    const reopened = await openSavedIdeas(wrapper)
    await emitAndClose(reopened, 'update', { id: 7, text: 'new idea', negative_prompt: 'new neg' })

    expect(UpdateDescription).toHaveBeenCalledWith({ id: 7, text: 'new idea', negative_prompt: 'new neg' })
    expect(descField(wrapper).element.value).toBe('new idea')
    expect(negField(wrapper).element.value).toBe('new neg')
  })

  it('keeps description when a different idea is updated', async () => {
    const wrapper = await mountPage()

    const modal = await openSavedIdeas(wrapper)
    await emitAndClose(modal, 'use', { id: 7, text: 'idea seven', negative_prompt: '' })

    const reopened = await openSavedIdeas(wrapper)
    await emitAndClose(reopened, 'update', { id: 99, text: 'other idea', negative_prompt: '' })

    expect(UpdateDescription).toHaveBeenCalledTimes(1)
    expect(descField(wrapper).element.value).toBe('idea seven')
  })

  it('does not overwrite manually edited description on same-id update', async () => {
    const wrapper = await mountPage()

    const modal = await openSavedIdeas(wrapper)
    await emitAndClose(modal, 'use', { id: 7, text: 'idea', negative_prompt: '' })
    expect(descField(wrapper).element.value).toBe('idea')

    await descField(wrapper).setValue('my own edit')

    const reopened = await openSavedIdeas(wrapper)
    await emitAndClose(reopened, 'update', { id: 7, text: 'updated elsewhere', negative_prompt: '' })

    expect(UpdateDescription).toHaveBeenCalledTimes(1)
    expect(descField(wrapper).element.value).toBe('my own edit')
  })
})

describe('saved idea negative edge branches', () => {
  it('manual negative edit survives same-id update', async () => {
    const wrapper = await mountPage()
    await flushPromises()
    await openSavedIdeas(wrapper)
    const modal = wrapper.findComponent(SavedDescriptionsModal)
    await emitAndClose(modal, 'use', { id: 7, text: 'idea seven', negative_prompt: 'old neg' })
    negField(wrapper).setValue('my own neg')
    await flushPromises()
    await openSavedIdeas(wrapper)
    const modal2 = wrapper.findComponent(SavedDescriptionsModal)
    await emitAndClose(modal2, 'update', { id: 7, text: 'edited seven', negative_prompt: 'new neg' })
    expect(UpdateDescription).toHaveBeenCalled()
    expect(descField(wrapper).element.value).toBe('edited seven')
    expect(negField(wrapper).element.value).toBe('my own neg')
  })

  it('update adding negative to neg-less idea fills empty field', async () => {
    const wrapper = await mountPage()
    await flushPromises()
    await openSavedIdeas(wrapper)
    const modal = wrapper.findComponent(SavedDescriptionsModal)
    await emitAndClose(modal, 'use', { id: 9, text: 'idea nine', negative_prompt: '' })
    expect(negField(wrapper).element.value).toBe('')
    await openSavedIdeas(wrapper)
    const modal2 = wrapper.findComponent(SavedDescriptionsModal)
    await emitAndClose(modal2, 'update', { id: 9, text: 'edited nine', negative_prompt: 'fresh neg' })
    expect(descField(wrapper).element.value).toBe('edited nine')
    expect(negField(wrapper).element.value).toBe('fresh neg')
  })
})

describe('manual description edit vs idea-sourced negative', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    clearEventMocks()
  })

  it('typing a new description clears the idea-sourced negative', async () => {
    const wrapper = await mountPage()

    const modal = await openSavedIdeas(wrapper)
    await emitAndClose(modal, 'use', { id: 5, text: 'scene five', negative_prompt: 'idea neg' })
    expect(negField(wrapper).element.value).toBe('idea neg')

    await descField(wrapper).setValue('совершенно другая сцена')
    await flushPromises()

    expect(descField(wrapper).element.value).toBe('совершенно другая сцена')
    expect(negField(wrapper).element.value).toBe('')
  })

  it('hand-written negative survives manual description edit', async () => {
    const wrapper = await mountPage()

    const modal = await openSavedIdeas(wrapper)
    await emitAndClose(modal, 'use', { id: 6, text: 'scene six', negative_prompt: 'idea neg six' })

    await negField(wrapper).setValue('my own neg')
    await descField(wrapper).setValue('другая сцена')
    await flushPromises()

    expect(descField(wrapper).element.value).toBe('другая сцена')
    expect(negField(wrapper).element.value).toBe('my own neg')
  })
})
