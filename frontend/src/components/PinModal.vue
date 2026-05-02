<script setup>
import { ref, computed } from 'vue'
import { t } from '../i18n/index.js'

const props = defineProps({
  mode: { type: String, required: true },
  error: { type: String, default: '' },
})
const emit = defineEmits(['confirm', 'cancel'])

const pin = ref('')
const confirmPin = ref('')
const pinError = ref('')

const isSetMode = computed(() => props.mode === 'set')
const title = computed(() => isSetMode.value ? t('pin.set_pin') : t('pin.enter_pin'))

function handleSubmit() {
  pinError.value = ''
  if (!/^\d{4}$/.test(pin.value)) {
    pinError.value = t('pin.error_digits')
    return
  }
  if (isSetMode.value && pin.value !== confirmPin.value) {
    pinError.value = t('pin.error_mismatch')
    return
  }
  emit('confirm', pin.value)
}

function handleCancel() {
  pinError.value = ''
  pin.value = ''
  confirmPin.value = ''
  emit('cancel')
}
</script>

<template>
  <div class="modal-overlay" @click.self="handleCancel">
    <div class="modal" style="max-width: 360px;">
      <div class="modal-header">
        <h2 class="modal-title">{{ title }}</h2>
        <button class="modal-close" @click="handleCancel">&times;</button>
      </div>

      <div style="padding: 16px 0;">
        <div v-if="isSetMode" style="color: var(--text-dim); margin-bottom: 16px; font-size: 13px;">
          {{ t('pin.set_description') }}
        </div>
        <div v-else style="color: var(--text-dim); margin-bottom: 16px; font-size: 13px;">
          {{ t('pin.disable_description') }}
        </div>

        <div v-if="pinError || error" class="status status-error" style="margin-bottom: 12px;">
          {{ pinError || error }}
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('pin.label_pin') }}</label>
          <input
            class="form-input"
            type="password"
            inputmode="numeric"
            maxlength="4"
            v-model="pin"
            :placeholder="t('pin.placeholder_pin')"
            @keyup.enter="handleSubmit"
            autofocus
          />
        </div>

        <div v-if="isSetMode" class="form-group">
          <label class="form-label">{{ t('pin.label_confirm') }}</label>
          <input
            class="form-input"
            type="password"
            inputmode="numeric"
            maxlength="4"
            v-model="confirmPin"
            :placeholder="t('pin.placeholder_confirm')"
            @keyup.enter="handleSubmit"
          />
        </div>
      </div>

      <div style="display: flex; gap: 8px; justify-content: flex-end;">
        <button class="btn btn-secondary" @click="handleCancel">{{ t('pin.btn_cancel') }}</button>
        <button class="btn btn-primary" @click="handleSubmit">{{ t('pin.btn_confirm') }}</button>
      </div>
    </div>
  </div>
</template>
