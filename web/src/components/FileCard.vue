<template>
  <div class="file-card">
    <div class="file-icon">
      <el-icon :size="32"><FolderOpened /></el-icon>
    </div>
    <div class="file-info">
      <h4 class="file-name">{{ task.zip_filename || `workshop_${task.id}.zip` }}</h4>
      <p class="file-meta">
        应用 {{ task.app_id }} · 资源 {{ task.pubfile_id }}
      </p>
      <p class="file-meta">
        大小: {{ formatSize(task.file_size) }}
        <span v-if="task.expires_at" class="expires">
          · 过期时间: {{ formatDate(task.expires_at) }}
        </span>
      </p>
      <p v-if="isExpired" class="expired-tag">已过期</p>
    </div>
    <div class="file-actions">
      <el-button
        size="small"
        type="primary"
        :disabled="isExpired"
        @click="download"
      >
        <el-icon><Download /></el-icon>
        下载
      </el-button>
      <el-button size="small" type="danger" plain @click="$emit('delete', task.id)">
        <el-icon><Delete /></el-icon>
        删除
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { DownloadTask } from '@/types'
import { getFileDownloadUrl } from '@/api/files'
import { formatSize, formatDate } from '@/utils/format'

const props = defineProps<{
  task: DownloadTask
}>()

defineEmits<{
  delete: [taskId: string]
}>()

const isExpired = computed(() => {
  if (!props.task.expires_at) return false
  return new Date(props.task.expires_at) < new Date()
})

function download() {
  window.open(getFileDownloadUrl(props.task.id), '_blank')
}
</script>
