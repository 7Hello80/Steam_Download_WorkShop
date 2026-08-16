<template>
  <div class="pty-container">
    <div class="terminal-header">
      <div class="terminal-dots">
        <span class="dot dot-red"></span>
        <span class="dot dot-yellow"></span>
        <span class="dot dot-green"></span>
      </div>
      <span class="terminal-title">DepotDownloader 输出</span>
      <span class="terminal-line-count">{{ lines.length }} 行</span>
    </div>
    <div class="terminal" ref="terminalEl">
      <div v-if="lines.length === 0" class="terminal-empty">
        <span>等待 DepotDownloader 输出...</span>
      </div>
      <div
        v-for="(line, i) in visibleLines"
        :key="i"
        class="terminal-line"
        :class="getLineClass(line)"
        v-html="formatLine(line)"
      ></div>
      <div v-if="waitingForInput" class="terminal-line prompt-line">
        <span class="prompt-arrow">❯</span>
        <span class="prompt-text">{{ promptText || '请输入验证码:' }}</span>
        <span class="cursor">▊</span>
      </div>
    </div>
    <div v-if="waitingForInput" class="pty-input-row">
      <el-input
        v-model="inputValue"
        placeholder="输入 Steam 验证码后按回车..."
        @keyup.enter="submitInput"
        size="default"
        :disabled="submitting"
        ref="inputRef"
      >
        <template #append>
          <el-button @click="submitInput" :loading="submitting">
            发送
          </el-button>
        </template>
      </el-input>
      <div v-if="inputStatus" class="input-status" :class="'status-' + inputStatus">
        <el-icon v-if="inputStatus === 'sent'" :size="14"><SuccessFilled /></el-icon>
        <el-icon v-else-if="inputStatus === 'error'" :size="14"><CircleCloseFilled /></el-icon>
        <span>{{ inputStatus === 'sent' ? '验证码已发送' : '发送失败' }}</span>
      </div>
      <p class="input-hint">5 分钟内未输入将自动取消下载</p>
    </div>
    <div class="terminal-footer">
      <el-button size="small" text @click="toggleCollapse" v-if="lines.length > 0">
        <el-icon :size="14"><component :is="collapsed ? 'ArrowDown' : 'ArrowUp'" /></el-icon>
        {{ collapsed ? `展开 (${lines.length} 行)` : '收起' }}
      </el-button>
      <el-button size="small" text @click="lines = []" v-if="lines.length > 0">
        <el-icon :size="14"><Delete /></el-icon>
        清空
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, watch, onMounted, onUnmounted } from 'vue'
import { useQueueStore } from '@/stores/queue'
import { getTaskOutput } from '@/api/download'
import type { DownloadTask } from '@/types'
const props = defineProps<{
  task: DownloadTask
}>()

const queueStore = useQueueStore()

const terminalEl = ref<HTMLElement>()
const inputRef = ref()
const lines = ref<string[]>([])
const waitingForInput = ref(false)
const promptText = ref('')
const inputValue = ref('')
const submitting = ref(false)
const inputStatus = ref<'sent' | 'error' | ''>('')
const collapsed = ref(false)

const visibleLines = computed(() => {
  if (collapsed.value) return lines.value.slice(-8)
  return lines.value
})

function scrollToBottom() {
  nextTick(() => {
    if (terminalEl.value) {
      terminalEl.value.scrollTop = terminalEl.value.scrollHeight
    }
  })
}

function formatLine(line: string): string {
  // Escape HTML
  const escaped = line
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
  return escaped
}

function getLineClass(line: string): string {
  const lower = line.toLowerCase()
  if (lower.includes('error') || lower.includes('fail') || lower.includes('failed')) return 'line-error'
  if (lower.includes('warn')) return 'line-warn'
  if (lower.includes('steam guard') || lower.includes('2fa') || lower.includes('code')) return 'line-prompt'
  if (lower.includes('downloading') || lower.includes('progress')) return 'line-info'
  if (lower.includes('success') || lower.includes('complete') || lower.includes('finished')) return 'line-success'
  return ''
}

function onPtyOutput(data: any) {
  if (data.task_id === props.task.id) {
    lines.value.push(data.output)
    // Keep only last 1000 lines to prevent memory issues
    if (lines.value.length > 1000) {
      lines.value = lines.value.slice(-500)
    }
    scrollToBottom()
  }
}

function onPtyPrompt(data: any) {
  if (data.task_id === props.task.id) {
    waitingForInput.value = true
    promptText.value = data.prompt || '请输入 Steam 验证码:'
    scrollToBottom()
    nextTick(() => {
      inputRef.value?.focus()
    })
  }
}

function onPtyInputAck(data: any) {
  if (data.task_id === props.task.id) {
    inputStatus.value = data.status || 'sent'
    if (data.status === 'error') {
      waitingForInput.value = true
      nextTick(() => inputRef.value?.focus())
    }
    setTimeout(() => { inputStatus.value = '' }, 3000)
  }
}

function submitInput() {
  const val = inputValue.value.trim()
  if (!val || submitting.value) return

  submitting.value = true
  queueStore.sendPtyInput(props.task.id, val)
  lines.value.push(`>>> ${val}`)
  inputValue.value = ''
  waitingForInput.value = false
  submitting.value = false
  scrollToBottom()
}

function toggleCollapse() {
  collapsed.value = !collapsed.value
  nextTick(scrollToBottom)
}

// Load historical output on mount (survives page refresh)
onMounted(async () => {
  if (props.task.status === 'downloading' || props.task.status === 'queued') {
    try {
      const result = await getTaskOutput(props.task.id)
      if (result.lines && result.lines.length > 0) {
        lines.value = result.lines
        scrollToBottom()
      }
    } catch {
      // No log yet or task doesn't exist
    }
  }
})

// Register listeners
queueStore.on('pty_output', onPtyOutput)
queueStore.on('pty_prompt', onPtyPrompt)
queueStore.on('pty_input_ack', onPtyInputAck)

// Auto-open when task is downloading
watch(() => props.task.status, (status) => {
  if (status === 'downloading' && lines.value.length === 0) {
    // Will fill as output comes in
  }
  if (status !== 'downloading') {
    waitingForInput.value = false
  }
}, { immediate: true })

onUnmounted(() => {
  queueStore.off('pty_output', onPtyOutput)
  queueStore.off('pty_prompt', onPtyPrompt)
  queueStore.off('pty_input_ack', onPtyInputAck)
})
</script>

<style scoped>
.pty-container {
  margin-top: 12px;
  border: 1px solid var(--color-border-light);
  overflow: hidden;
}

.terminal-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 14px;
  background: #1e293b;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.terminal-dots {
  display: flex;
  gap: 6px;
}

.dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.dot-red { background: #ef4444; }
.dot-yellow { background: #f59e0b; }
.dot-green { background: #10b981; }

.terminal-title {
  flex: 1;
  font-size: 11px;
  color: rgba(255, 255, 255, 0.5);
  font-weight: 500;
}

.terminal-line-count {
  font-size: 10px;
  color: rgba(255, 255, 255, 0.3);
}

.terminal {
  background: #0f172a;
  color: #cbd5e1;
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', 'Courier New', monospace;
  font-size: 12.5px;
  padding: 12px 14px;
  max-height: 360px;
  overflow-y: auto;
  line-height: 1.6;
  tab-size: 4;
}

.terminal:empty {
  display: none;
}

.terminal-empty {
  color: rgba(255, 255, 255, 0.25);
  font-style: italic;
  padding: 20px 0;
  text-align: center;
}

.terminal-line {
  white-space: pre-wrap;
  word-break: break-all;
}

.terminal-line.line-error {
  color: #fca5a5;
  font-weight: 500;
}

.terminal-line.line-warn {
  color: #fde68a;
}

.terminal-line.line-prompt {
  color: #fbbf24;
  font-weight: 600;
}

.terminal-line.line-info {
  color: #93c5fd;
}

.terminal-line.line-success {
  color: #86efac;
}

.prompt-line {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 0;
  color: #fbbf24;
  font-weight: 600;
}

.prompt-arrow {
  color: var(--color-primary-light);
}

.cursor {
  animation: blink 1s step-end infinite;
  color: var(--color-primary-light);
}

@keyframes blink {
  50% { opacity: 0; }
}

.pty-input-row {
  padding: 10px 14px;
  background: var(--color-bg-secondary);
  border-top: 1px solid var(--color-border-light);
}

.input-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  margin-top: 6px;
  font-weight: 500;
}

.input-status.status-sent {
  color: var(--color-success);
}

.input-status.status-error {
  color: var(--color-danger);
}

.input-hint {
  font-size: 11px;
  color: var(--color-text-muted);
  margin-top: 6px;
}

.terminal-footer {
  display: flex;
  gap: 8px;
  padding: 6px 10px;
  background: var(--color-bg-secondary);
  border-top: 1px solid var(--color-border-light);
}
</style>
