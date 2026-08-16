<template>
  <div class="page-container">
    <div class="banner-tip">
      <span class="banner-icon">💡</span>
      <span style="flex:1;">后台系统自动下载，您可以离开过段时间再来查看</span>
      <el-button plain round size="small" @click="showSponsor = true">
        <font-awesome-icon icon="heart" style="margin-right:4px;color:#d32f2f;" />
        赞赏
      </el-button>
    </div>

    <QueueStatus />
    <DownloadForm @submit="handleStartDownload" />

    <!-- Active tasks (pending / queued / downloading) -->
    <div v-if="activeTasks.length > 0" class="active-tasks">
      <h3 class="section-title">
        进行中的任务
        <span class="task-count-badge">{{ activeTasks.length }}</span>
      </h3>
      <TaskProgress
        v-for="task in activeTasks"
        :key="task.id"
        :task="task"
        @cancel="handleCancel"
      />
    </div>

    <!-- Completed / Failed / Cancelled history -->
    <div v-if="historyTasks.length > 0" class="history-tasks">
      <h3 class="section-title">
        历史任务
        <span class="task-count-badge history">{{ historyTasks.length }}</span>
      </h3>
      <TaskProgress
        v-for="task in historyTasks"
        :key="task.id"
        :task="task"
        @cancel="handleCancel"
      />
    </div>

    <el-empty v-if="!tasks.length && !loading" description="暂无下载任务">
      <template #description>
        <p style="color: var(--color-text-muted);">还没有下载任务</p>
        <p style="color: var(--color-text-muted); font-size: 12px; margin-top: 4px;">
          填写 App ID 和资源 ID 开始下载创意工坊内容
        </p>
      </template>
    </el-empty>

    <SponsorModal v-model:visible="showSponsor" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import type { DownloadTask, StartDownloadRequest } from '@/types'
import { startDownload, listDownloads, cancelDownload } from '@/api/download'
import { useWebSocket } from '@/composables/useWebSocket'
import { ElMessage } from 'element-plus'
import DownloadForm from '@/components/DownloadForm.vue'
import QueueStatus from '@/components/QueueStatus.vue'
import TaskProgress from '@/components/TaskProgress.vue'
import SponsorModal from '@/components/SponsorModal.vue'

const { on, off } = useWebSocket()

const tasks = ref<DownloadTask[]>([])
const loading = ref(false)
const showSponsor = ref(false)

const activeTasks = computed(() =>
  tasks.value.filter(t => ['pending', 'queued', 'downloading'].includes(t.status))
)

const historyTasks = computed(() =>
  tasks.value.filter(t => ['completed', 'failed', 'cancelled', 'expired'].includes(t.status))
)

async function handleStartDownload(req: StartDownloadRequest) {
  try {
    const resp = await startDownload(req)
    ElMessage.success(`已加入下载队列！前方还有 ${resp.queue_position - 1} 个任务`)
    await fetchTasks()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || '启动下载失败')
  }
}

async function handleCancel(taskId: string) {
  try {
    await cancelDownload(taskId)
    ElMessage.success('任务已取消')
    await fetchTasks()
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || '取消失败')
  }
}

async function fetchTasks() {
  try {
    const result = await listDownloads({ limit: 50 })
    tasks.value = result.tasks || []
  } catch {
    // Ignore
  }
}

function onTaskUpdate(data: any) {
  // Find existing task
  const idx = tasks.value.findIndex(t => t.id === data.task_id)
  if (idx >= 0) {
    // Merge update - only overwrite fields that are present in data
    const updated = { ...tasks.value[idx] }
    for (const key of Object.keys(data)) {
      if (data[key] !== undefined) {
        ;(updated as any)[key] = data[key]
      }
    }
    tasks.value[idx] = updated
  } else if (data.status && ['queued', 'downloading'].includes(data.status)) {
    // New task appeared via WebSocket (from another tab), add it
    fetchTasks()
  }

  // Refresh on terminal states to get final data (file_size, expires_at etc.)
  if (data.status && ['completed', 'failed', 'cancelled'].includes(data.status)) {
    // Use a short delay to let backend finish updating
    setTimeout(fetchTasks, 500)
  }
}

function onQueueUpdate() {
  // Refresh task list to get updated queue positions
  fetchTasks()
}

onMounted(() => {
  fetchTasks()
  on('task_update', onTaskUpdate)
  on('queue_update', onQueueUpdate)
})

onUnmounted(() => {
  off('task_update', onTaskUpdate)
  off('queue_update', onQueueUpdate)
})
</script>

<style scoped>
.banner-tip {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  margin-bottom: 20px;
  font-size: 13px;
  color: #666;
  background: #f5f5f5;
  border-left: 3px solid #999;
  border-radius: 4px;
  line-height: 1.5;
}

.banner-icon {
  font-size: 15px;
  flex-shrink: 0;
}

.task-count-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 22px;
  height: 22px;
  padding: 0 7px;
  font-size: 12px;
  font-weight: 600;
  color: #fff;
  background: var(--color-primary-gradient);
  border-radius: var(--radius-full);
  margin-left: 8px;
  vertical-align: middle;
}

.task-count-badge.history {
  background: var(--color-text-muted);
}

.history-tasks {
  margin-top: 32px;
  opacity: 0.85;
}
</style>
