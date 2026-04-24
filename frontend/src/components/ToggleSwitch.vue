<script setup>
defineProps({
  modelValue: Boolean,
  disabled: { type: Boolean, default: false },
})
defineEmits(['update:modelValue'])
</script>

<template>
  <label class="toggle-switch" :class="{ disabled }">
    <input type="checkbox" :checked="modelValue" :disabled="disabled" @change="$emit('update:modelValue', $event.target.checked)" />
    <span class="toggle-slider"></span>
  </label>
</template>

<style scoped>
.toggle-switch {
  position: relative;
  display: inline-block;
  width: 48px;
  height: 26px;
  cursor: pointer;
}
.toggle-switch.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}
.toggle-slider {
  position: absolute;
  inset: 0;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 13px;
  transition: background 0.2s, border-color 0.2s;
}
.toggle-slider::before {
  content: '';
  position: absolute;
  width: 20px;
  height: 20px;
  left: 2px;
  top: 2px;
  background: var(--text-dim);
  border-radius: 50%;
  transition: transform 0.2s, background 0.2s;
}
.toggle-switch input:checked + .toggle-slider {
  background: var(--accent);
  border-color: var(--accent);
}
.toggle-switch input:checked + .toggle-slider::before {
  transform: translateX(22px);
  background: #fff;
}
</style>
