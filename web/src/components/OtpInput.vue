<template>
  <div class="otp-wrapper">
    <div class="otp-inputs">
      <input
        v-for="(digit, index) in digits"
        :key="index"
        :ref="(el) => (inputRefs[index] = el as HTMLInputElement)"
        v-model="digits[index]"
        type="text"
        inputmode="numeric"
        maxlength="1"
        autocomplete="one-time-code"
        class="otp-slot"
        :class="{ 'otp-slot--filled': digits[index], 'otp-slot--focus': focusedIndex === index }"
        @input="(e) => onInput(index, e)"
        @keydown="(e) => onKeydown(index, e)"
        @paste="onPaste"
        @focus="focusedIndex = index"
        @blur="focusedIndex = -1"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'

const props = withDefaults(defineProps<{
  modelValue?: string
  length?: number
}>(), {
  modelValue: '',
  length: 6,
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'complete', value: string): void
}>()

const digits = ref<string[]>(Array.from({ length: props.length }, () => ''))
const inputRefs = ref<(HTMLInputElement | null)[]>([])
const focusedIndex = ref(-1)

// 初始化 digits
function initFromValue(val: string) {
  const chars = (val || '').slice(0, props.length).split('')
  digits.value = Array.from({ length: props.length }, (_, i) => chars[i] || '')
}

initFromValue(props.modelValue)

watch(() => props.modelValue, (val) => {
  initFromValue(val || '')
})

// 更新 v-model
function emitValue() {
  const val = digits.value.join('')
  emit('update:modelValue', val)
  if (val.length === props.length) {
    emit('complete', val)
  }
}

function focusInput(index: number) {
  nextTick(() => {
    const el = inputRefs.value[index]
    if (el) {
      el.focus()
      el.select()
    }
  })
}

function onInput(index: number, e: Event) {
  const target = e.target as HTMLInputElement
  const raw = target.value

  // 过滤非数字字符
  const cleaned = raw.replace(/\D/g, '')
  if (cleaned !== raw) {
    target.value = cleaned
  }

  // 只保留第一个数字字符
  if (cleaned.length > 0) {
    digits.value[index] = cleaned[0]
    target.value = cleaned[0]
  }

  emitValue()

  // 自动跳转到下一个输入框
  if (cleaned.length > 0 && index < props.length - 1) {
    focusInput(index + 1)
  }
}

function onKeydown(index: number, e: KeyboardEvent) {
  if (e.key === 'Backspace' || e.key === 'Delete') {
    e.preventDefault()
    if (digits.value[index]) {
      digits.value[index] = ''
      emitValue()
    } else if (index > 0) {
      digits.value[index - 1] = ''
      emitValue()
      focusInput(index - 1)
    }
  } else if (e.key === 'ArrowLeft' && index > 0) {
    e.preventDefault()
    focusInput(index - 1)
  } else if (e.key === 'ArrowRight' && index < props.length - 1) {
    e.preventDefault()
    focusInput(index + 1)
  } else if (e.key === 'ArrowUp' || e.key === 'ArrowDown') {
    e.preventDefault()
  }
}

function onPaste(e: ClipboardEvent) {
  e.preventDefault()
  const pasteData = e.clipboardData?.getData('text') || ''
  const cleaned = pasteData.replace(/\D/g, '').slice(0, props.length)

  if (!cleaned) return

  for (let i = 0; i < props.length; i++) {
    digits.value[i] = cleaned[i] || ''
  }

  emitValue()

  // 聚焦到最后一个已填充的位置或末尾
  const lastFilled = Math.min(cleaned.length, props.length) - 1
  focusInput(lastFilled >= 0 ? lastFilled : 0)
}
</script>

<style scoped>
.otp-wrapper {
  display: flex;
  justify-content: center;
}

.otp-inputs {
  display: flex;
  gap: 10px;
  justify-content: center;
}

.otp-slot {
  width: 44px;
  height: 52px;
  text-align: center;
  font-size: 22px;
  font-weight: 500;
  font-family: 'Roboto Mono', 'Fira Code', 'Noto Sans SC', monospace;
  color: var(--color-text);
  background: #fff;
  border: 1px solid #ddd;
  border-radius: var(--radius-md);
  outline: none;
  caret-color: var(--color-primary);
  transition: border-color 0.2s, box-shadow 0.2s;
  -webkit-appearance: none;
  -moz-appearance: textfield;
}

.otp-slot::-webkit-outer-spin-button,
.otp-slot::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}

.otp-slot:hover {
  border-color: #bbb;
}

.otp-slot--focus {
  border-color: var(--color-primary) !important;
  box-shadow: 0 0 0 2px rgba(0, 0, 0, 0.1) !important;
}

.otp-slot--filled {
  border-color: #aaa;
  background: #fafafa;
}

@media (max-width: 768px) {
  .otp-inputs {
    gap: 8px;
  }

  .otp-slot {
    width: 42px;
    height: 48px;
    font-size: 20px;
    border-radius: var(--radius-sm);
  }
}
</style>
