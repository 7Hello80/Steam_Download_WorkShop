<template>
  <div class="page-container">
    <div v-if="files.length > 0" class="files-list">
      <FileCard
        v-for="file in files"
        :key="file.id"
        :task="file"
        @delete="handleDelete"
      />
    </div>

    <el-empty v-else description="还没有下载完成的文件，先去下载中心创建一个吧！" />

    <div v-if="total > limit" class="pagination">
      <el-pagination
        v-model:current-page="page"
        :page-size="limit"
        :total="total"
        layout="prev, pager, next"
        @current-change="fetchFiles"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { DownloadTask } from '@/types'
import { listFiles, deleteFile } from '@/api/files'
import { ElMessage, ElMessageBox } from 'element-plus'
import FileCard from '@/components/FileCard.vue'

const files = ref<DownloadTask[]>([])
const page = ref(1)
const limit = ref(12)
const total = ref(0)

async function fetchFiles(p = 1) {
  page.value = p
  try {
    const result = await listFiles({ page: p, limit: limit.value })
    files.value = result.files || []
    total.value = result.total
  } catch {
    // Ignore
  }
}

async function handleDelete(taskId: string) {
  try {
    await ElMessageBox.confirm('确定要永久删除此文件吗？', '确认删除', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await deleteFile(taskId)
    ElMessage.success('文件已删除')
    await fetchFiles(page.value)
  } catch {
    // Cancelled or error
  }
}

onMounted(() => {
  fetchFiles()
})
</script>
