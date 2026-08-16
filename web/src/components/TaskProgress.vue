<template>
  <div class="task-progress" :class="'task-border-' + task.status">
    <div class="task-header">
      <div class="task-header-left">
        <span class="task-id">{{ task.id }}</span>
        <span v-if="elapsedText" class="elapsed">{{ elapsedText }}</span>
      </div>
      <span class="status-badge" :class="'status-' + task.status">
        {{ statusText }}
      </span>
    </div>

    <div class="task-info">
      <div class="task-info-item">
        <span class="info-label">App ID</span>
        <span class="info-value">{{ task.app_id }}</span>
      </div>
      <div class="task-info-item">
        <span class="info-label">资源 ID</span>
        <span class="info-value">{{ task.pubfile_id }}</span>
      </div>
      <div class="task-info-item">
        <span class="info-label">Steam 账号</span>
        <span class="info-value">{{ task.steam_username }}</span>
      </div>
    </div>

    <!-- Queued status -->
    <div v-if="task.status === 'queued'" class="queue-notice">
      <el-icon :size="16"><Clock /></el-icon>
      <span>排队中 — 前方还有 <strong>{{ task.queue_position > 0 ? task.queue_position - 1 : '?' }}</strong> 个任务</span>
    </div>

    <!-- Downloading: progress bar + terminal -->
    <template v-if="task.status === 'downloading'">
      <div class="download-progress-section">
        <div class="progress-bar-wrapper">
          <div class="progress-bar">
            <div class="progress-bar-fill" :style="{ width: progressPercent + '%' }"></div>
          </div>
          <span class="progress-text">{{ progressPercent }}%</span>
        </div>
        <div class="download-stats" v-if="downloadedBytes > 0">
          <span>{{ formatSize(downloadedBytes) }}<span v-if="totalBytes > 0"> / {{ formatSize(totalBytes) }}</span></span>
          <span v-if="downloadSpeed > 0" class="speed">{{ formatSize(downloadSpeed) }}/s</span>
        </div>
      </div>
      <PTYTerminal :task="task" />
    </template>

    <!-- Completed -->
    <div v-if="task.status === 'completed'" class="completed-info">
      <div class="completed-row">
        <el-icon :size="18" color="var(--color-success)"><SuccessFilled /></el-icon>
        <span>文件大小: <strong>{{ formatSize(task.file_size) }}</strong></span>
      </div>
      <div v-if="task.expires_at" class="expiry-row">
        <el-icon :size="14"><Clock /></el-icon>
        <span>过期时间: {{ formatDate(task.expires_at) }}</span>
        <span class="expiry-countdown">{{ expiryCountdown }}</span>
      </div>
    </div>

    <!-- Failed -->
    <div v-if="task.status === 'failed'" class="error-info">
      <el-alert
        :title="task.error_message || '下载失败'"
        type="error"
        :closable="false"
        show-icon
      />
    </div>

    <!-- Cancelled -->
    <div v-if="task.status === 'cancelled'" class="cancelled-info">
      <span class="text-muted">任务已取消</span>
    </div>

    <!-- Expired -->
    <div v-if="task.status === 'expired'" class="expired-info">
      <span class="text-muted">文件已过期并被清理</span>
    </div>

    <!-- Actions: only cancel queued/downloading -->
    <div class="task-actions" v-if="canCancel">
      <el-button size="small" type="danger" plain @click="handleCancel" :loading="cancelling">
        <el-icon :size="14"><Close /></el-icon>
        取消任务
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onUnmounted } from 'vue'
import type { DownloadTask } from '@/types'
import { formatSize, formatDate } from '@/utils/format'
import { useQueueStore } from '@/stores/queue'
import PTYTerminal from './PTYTerminal.vue'
const props = defineProps<{
  task: DownloadTask
}>()

const emit = defineEmits<{
  cancel: [taskId: string]
}>()

const cancelling = ref(false)
const progressPercent = ref(0)
const downloadedBytes = ref(0)
const totalBytes = ref(0)
const downloadSpeed = ref(0)
let lastBytes = 0
let lastTime = 0

// Watch WebSocket task updates for real progress data
function updateProgress(data: any) {
  if (data.task_id !== props.task.id) return
  if (data.progress_pct !== undefined) {
    progressPercent.value = Math.round(data.progress_pct)
  }
  if (data.downloaded_bytes !== undefined) {
    const now = Date.now()
    if (lastBytes > 0 && lastTime > 0) {
      const timeDiff = (now - lastTime) / 1000
      if (timeDiff > 0.5) {
        downloadSpeed.value = Math.round((data.downloaded_bytes - lastBytes) / timeDiff)
      }
    }
    lastBytes = data.downloaded_bytes
    lastTime = now
    downloadedBytes.value = data.downloaded_bytes
  }
  if (data.total_bytes !== undefined) {
    totalBytes.value = data.total_bytes
  }
}

// Register progress listener
const queueStore = useQueueStore()
queueStore.on('task_update', updateProgress)

onUnmounted(() => {
  queueStore.off('task_update', updateProgress)
})

const canCancel = computed(() => {
  return ['pending', 'queued', 'downloading'].includes(props.task.status)
})

const statusText = computed(() => {
  const map: Record<string, string> = {
    pending: '等待中',
    queued: '排队中',
    downloading: '下载中',
    completed: '已完成',
    failed: '失败',
    cancelled: '已取消',
    expired: '已过期',
  }
  return map[props.task.status] || props.task.status
})

const elapsedText = computed(() => {
  if (!props.task.started_at) return ''
  const start = new Date(props.task.started_at).getTime()
  const now = Date.now()
  const diff = Math.floor((now - start) / 1000)
  if (diff < 60) return `${diff}秒`
  if (diff < 3600) return `${Math.floor(diff / 60)}分${diff % 60}秒`
  return `${Math.floor(diff / 3600)}时${Math.floor((diff % 3600) / 60)}分`
})

const expiryCountdown = computed(() => {
  if (!props.task.expires_at) return ''
  const expiry = new Date(props.task.expires_at).getTime()
  const now = Date.now()
  const diff = expiry - now
  if (diff <= 0) return '（已过期）'
  const hours = Math.floor(diff / 3600000)
  if (hours < 24) return `（剩余 ${hours} 小时）`
  const days = Math.floor(hours / 24)
  return `（剩余 ${days} 天 ${hours % 24} 小时）`
})

function handleCancel() {
  cancelling.value = true
  emit('cancel', props.task.id)
  setTimeout(() => { cancelling.value = false }, 2000)
}
</script>

<style scoped>
.task-progress {
  background: var(--color-bg);
  border: 1px solid var(--color-border-light);
  border-left: 4px solid var(--color-border-light);
  border-radius: var(--radius-lg);
  padding: 20px 24px;
  margin-bottom: 14px;
  box-shadow: var(--shadow-xs);
  transition: all 0.3s;
}

.task-progress:hover {
  box-shadow: var(--shadow-md);
}

.task-border-downloading { border-left-color: var(--color-primary); }
.task-border-completed { border-left-color: var(--color-success); }
.task-border-failed { border-left-color: var(--color-danger); }
.task-border-queued { border-left-color: var(--color-info); }
.task-border-cancelled { border-left-color: var(--color-warning); }
.task-border-expired { border-left-color: var(--color-text-muted); }

.task-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.task-header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.task-id {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 11px;
  color: var(--color-text-muted);
  background: var(--color-bg-tertiary);
  padding: 2px 8px;
  border-radius: var(--radius-sm);
}

.elapsed {
  font-size: 12px;
  color: var(--color-text-muted);
}

.task-info {
  display: flex;
  gap: 20px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.task-info-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.info-label {
  font-size: 11px;
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.info-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
}

.queue-notice {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--color-info);
  margin: 10px 0;
  padding: 10px 14px;
  background: var(--color-info-bg);
  border-radius: var(--radius-md);
  border: 1px dashed var(--color-info);
}

.download-progress-section {
  margin: 12px 0;
}

.progress-bar-wrapper {
  display: flex;
  align-items: center;
  gap: 12px;
}

.progress-bar {
  flex: 1;
  height: 8px;
  background: var(--color-bg-tertiary);
  border-radius: 4px;
  overflow: hidden;
}

.progress-bar-fill {
  height: 100%;
  background: var(--color-primary-gradient);
  border-radius: 4px;
  transition: width 0.5s ease;
  position: relative;
}

.progress-bar-fill::after {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(
    90deg,
    transparent 0%,
    rgba(255, 255, 255, 0.3) 50%,
    transparent 100%
  );
  animation: shimmer 2s infinite;
}

@keyframes shimmer {
  0% { transform: translateX(-100%); }
  100% { transform: translateX(100%); }
}

.progress-text {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-primary);
  min-width: 40px;
  text-align: right;
}

.download-stats {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: var(--color-text-muted);
  margin-top: 6px;
}

.speed {
  color: var(--color-primary);
  font-weight: 500;
}

.completed-info {
  margin: 10px 0;
}

.completed-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  margin-bottom: 6px;
}

.expiry-row {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--color-text-muted);
}

.expiry-countdown {
  color: var(--color-warning);
  font-weight: 500;
}

.cancelled-info, .expired-info {
  margin: 10px 0;
}

.task-actions {
  margin-top: 14px;
  padding-top: 14px;
  border-top: 1px solid var(--color-border-light);
}

.error-info {
  margin: 10px 0;
}
</style>
