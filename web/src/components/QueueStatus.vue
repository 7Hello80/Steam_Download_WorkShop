<template>
  <div class="queue-status" v-if="info.total_ahead !== undefined">
    <div class="queue-card">
      <div class="queue-stat">
        <span class="stat-icon" style="background: var(--color-primary-bg); color: var(--color-primary);">
          <el-icon :size="20"><VideoPlay /></el-icon>
        </span>
        <div class="stat-body">
          <span class="stat-value">{{ info.active_count }}</span>
          <span class="stat-label">正在下载</span>
        </div>
      </div>
      <div class="queue-stat">
        <span class="stat-icon" style="background: var(--color-warning-bg); color: var(--color-warning);">
          <el-icon :size="20"><User /></el-icon>
        </span>
        <div class="stat-body">
          <span class="stat-value">{{ info.total_ahead }}</span>
          <span class="stat-label">排在前面</span>
        </div>
      </div>
      <div class="queue-stat">
        <span class="stat-icon" style="background: var(--color-info-bg); color: var(--color-info);">
          <el-icon :size="20"><List /></el-icon>
        </span>
        <div class="stat-body">
          <span class="stat-value">{{ info.queue_length }}</span>
          <span class="stat-label">队列总数</span>
        </div>
      </div>
    </div>

    <!-- Your task position -->
    <div v-if="info.your_position > 0" class="your-position">
      <el-icon :size="16"><Location /></el-icon>
      <span>您的任务排在 <strong>第 {{ info.your_position }} 位</strong></span>
      <span v-if="info.total_ahead === 0 && info.active_count > 0" class="position-note">（即将开始）</span>
    </div>

    <div class="ws-indicator" :class="wsStatus">
      <span class="ws-dot"></span>
      <span class="ws-text">{{ wsStatusText }}</span>
      <span v-if="wsStatus === 'disconnected'" class="ws-retry-hint">— 自动重连中</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useQueueStore } from '@/stores/queue'
import { storeToRefs } from 'pinia'
import { getQueueInfo } from '@/api/download'

const queueStore = useQueueStore()
const { queueInfo: info, wsStatus } = storeToRefs(queueStore)

const wsStatusText = computed(() => {
  switch (wsStatus.value) {
    case 'connected': return '实时连接已建立'
    case 'connecting': return '正在连接...'
    default: return '连接已断开'
  }
})

async function fetchQueueInfo() {
  try {
    const data = await getQueueInfo()
    queueStore.setQueueInfo(data)
  } catch {
    // ignore errors - WebSocket will update when available
  }
}

onMounted(() => {
  fetchQueueInfo()
  queueStore.on('queue_update', fetchQueueInfo)
})

onUnmounted(() => {
  queueStore.off('queue_update', fetchQueueInfo)
})
</script>

<style scoped>
.queue-status {
  margin-bottom: 24px;
}

.queue-card {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}

.queue-stat {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 18px 20px;
  background: var(--color-bg);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-xs);
  transition: all 0.25s;
}

.queue-stat:hover {
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
  border-color: var(--color-border);
}

.stat-icon {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-body {
  display: flex;
  flex-direction: column;
}

.stat-value {
  font-size: 1.65rem;
  font-weight: 800;
  line-height: 1;
  color: var(--color-text);
  background: var(--color-primary-gradient);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
}

.stat-label {
  margin-top: 3px;
  font-size: 11px;
  color: var(--color-text-muted);
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.your-position {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding: 10px 16px;
  background: var(--color-primary-bg);
  border-radius: var(--radius-md);
  font-size: 13px;
  color: var(--color-primary-dark);
}

.your-position strong {
  font-weight: 700;
}

.position-note {
  color: var(--color-success);
  font-weight: 600;
  font-size: 12px;
}

.ws-indicator {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  font-size: 12px;
  padding: 7px 14px;
  background: var(--color-bg);
  border-radius: var(--radius-full);
  border: 1px solid var(--color-border-light);
}

.ws-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #cbd5e1;
  transition: all 0.3s;
}

.ws-indicator.connected .ws-dot {
  background: var(--color-success);
  box-shadow: 0 0 8px rgba(16, 185, 129, 0.4);
}

.ws-indicator.connecting .ws-dot {
  background: var(--color-warning);
  animation: pulse-dot 1.5s ease-in-out infinite;
}

@keyframes pulse-dot {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.4; transform: scale(1.3); }
}

.ws-text {
  color: var(--color-text-muted);
}

.ws-retry-hint {
  color: var(--color-warning);
  font-size: 11px;
}

@media (max-width: 768px) {
  .queue-card {
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
  }

  .queue-stat {
    padding: 12px;
    gap: 8px;
  }

  .stat-icon {
    width: 36px;
    height: 36px;
  }

  .stat-value {
    font-size: 1.3rem;
  }
}
</style>
