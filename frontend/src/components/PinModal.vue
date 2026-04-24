<script setup>
import { ref, computed } from 'vue'

const props = defineProps({
  mode: { type: String, required: true },
  error: { type: String, default: '' },
})
const emit = defineEmits(['confirm', 'cancel'])

const pin = ref('')
const confirmPin = ref('')
const pinError = ref('')

const isSetMode = computed(() => props.mode === 'set')
const title = computed(() => isSetMode.value ? 'Set PIN' : 'Enter PIN')

function handleSubmit() {
  pinError.value = ''
  if (!/^\d{4}$/.test(pin.value)) {
    pinError.value = 'PIN must be exactly 4 digits'
    return
  }
  if (isSetMode.value && pin.value !== confirmPin.value) {
    pinError.value = 'PINs do not match'
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
          Set a 4-digit PIN to protect Kids Mode settings. Your child will need this PIN to disable the safety filter.
        </div>
        <div v-else style="color: var(--text-dim); margin-bottom: 16px; font-size: 13px;">
          Enter the PIN to disable Kids Mode.
        </div>

        <div v-if="pinError || error" class="status status-error" style="margin-bottom: 12px;">
          {{ pinError || error }}
        </div>

        <div class="form-group">
          <label class="form-label">PIN</label>
          <input
            class="form-input"
            type="password"
            inputmode="numeric"
            maxlength="4"
            v-model="pin"
            placeholder="4 digits"
            @keyup.enter="handleSubmit"
            autofocus
          />
        </div>

        <div v-if="isSetMode" class="form-group">
          <label class="form-label">Confirm PIN</label>
          <input
            class="form-input"
            type="password"
            inputmode="numeric"
            maxlength="4"
            v-model="confirmPin"
            placeholder="Re-enter PIN"
            @keyup.enter="handleSubmit"
          />
        </div>
      </div>

      <div style="display: flex; gap: 8px; justify-content: flex-end;">
        <button class="btn btn-secondary" @click="handleCancel">Cancel</button>
        <button class="btn btn-primary" @click="handleSubmit">Confirm</button>
      </div>
    </div>
  </div>
</template>
