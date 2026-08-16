<template>
  <div class="page-container">
    <div class="page-header">
      <h2 class="page-title">壁纸转换</h2>
      <p class="page-desc">
        将已下载完成的 zip 文件转换为 mpkg 格式，所有已完成的下载均可转换。
      </p>
    </div>

    <!-- Loading state -->
    <div v-if="loading" class="loading-wrap">
      <font-awesome-icon icon="spinner" spin :size="'2x'" />
      <span>正在加载可转换的文件...</span>
    </div>

    <!-- Empty state -->
    <el-empty
      v-else-if="files.length === 0"
      description="暂无可转换的文件"
    >
      <template #extra>
        <p class="empty-hint">
          还没有已下载完成的文件。<br />
          请先在<a href="/download">下载中心</a>下载创意工坊文件，完成后即可在此转换。
        </p>
      </template>
    </el-empty>

    <!-- File list -->
    <div v-else class="convert-list">
      <div
        v-for="file in files"
        :key="file.task_id"
        class="convert-card"
      >
        <div class="convert-card-body">
          <div class="convert-card-icon">
            <font-awesome-icon icon="file-zipper" />
          </div>
          <div class="convert-card-info">
            <h4 class="convert-card-title">{{ truncateTitle(file.title) }}</h4>
            <div class="convert-card-meta">
              <span class="meta-tag">
                <font-awesome-icon icon="gamepad" />
                AppID: {{ file.app_id }}
              </span>
              <span v-if="file.video_files.length > 0" class="meta-tag">
                <font-awesome-icon icon="video" />
                {{ file.video_files.join(', ') }}
              </span>
              <span v-if="file.preview_name" class="meta-tag">
                <font-awesome-icon icon="image" />
                {{ file.preview_name }}
              </span>
              <span class="meta-tag">
                <font-awesome-icon icon="weight-hanging" />
                {{ formatSize(file.file_size) }}
              </span>
            </div>
          </div>
        </div>
        <div class="convert-card-actions">
          <!-- Converting in progress -->
          <template v-if="converting[file.task_id]">
            <el-button type="primary" loading>转换中...</el-button>
          </template>

          <!-- Already converted (from list) or just converted -->
          <template v-else-if="file.converted || results[file.task_id]">
            <el-tag type="success" class="done-tag">已转换</el-tag>
            <span v-if="mpkgSizes[file.task_id] || file.mpkg_size" class="mpkg-size">
              {{ formatSize(mpkgSizes[file.task_id] || file.mpkg_size || 0) }}
            </span>
            <el-button type="primary" @click="downloadMpkg(file.task_id)">
              <font-awesome-icon icon="download" />
              下载 mpkg
            </el-button>
          </template>

          <!-- Not yet converted -->
          <template v-else>
            <el-button type="primary" @click="startConvert(file.task_id)">
              <font-awesome-icon icon="cog" />
              转换为 mpkg
            </el-button>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  listConvertible,
  convert,
  getConvertDownloadUrl,
  type ConvertibleFile,
  type ConvertResult,
} from '@/api/convert'
import { formatSize } from '@/utils/format'

const files = ref<ConvertibleFile[]>([])
const loading = ref(true)
const converting = ref<Record<string, boolean>>({})
const results = ref<Record<string, ConvertResult>>({})
const mpkgSizes = ref<Record<string, number>>({})

async function fetchFiles() {
  loading.value = true
  try {
    const res = await listConvertible()
    files.value = res.files || []
  } catch {
    ElMessage.error('加载可转换文件列表失败')
  } finally {
    loading.value = false
  }
}

async function startConvert(taskId: string) {
  converting.value[taskId] = true
  try {
    const result = await convert(taskId)
    results.value[taskId] = result
    mpkgSizes.value[taskId] = result.file_size
    ElMessage.success(`"${result.title}" 转换完成！`)
  } catch (e: any) {
    const msg = e?.response?.data?.error || '转换失败'
    ElMessage.error(msg)
  } finally {
    converting.value[taskId] = false
  }
}

function truncateTitle(title: string): string {
  if (!title) return '未命名文件'
  if (title.length > 10) return title.slice(0, 10) + '...'
  return title
}

function downloadMpkg(taskId: string) {
  const file = files.value.find(f => f.task_id === taskId)
  const result = results.value[taskId]

  let url = ''
  if (result) {
    url = getConvertDownloadUrl(taskId, result.filename)
  } else if (file?.download_url) {
    url = file.download_url
  }
  if (url) window.open(url, '_blank')
}

onMounted(() => {
  fetchFiles()
})
</script>

<style scoped>
.page-header {
  margin-bottom: 24px;
}

.page-title {
  font-size: 22px;
  font-weight: 600;
  margin: 0 0 8px;
  color: var(--el-text-color-primary);
}

.page-desc {
  margin: 0;
  color: var(--el-text-color-regular);
  font-size: 14px;
}

.loading-wrap {
  display: flex;
  align-items: center;
  gap: 12px;
  justify-content: center;
  padding: 60px 0;
  color: var(--el-text-color-secondary);
}

.empty-hint {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.8;
}

.empty-hint a {
  color: var(--el-color-primary);
  text-decoration: none;
}

.empty-hint a:hover {
  text-decoration: underline;
}

.convert-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.convert-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  border-radius: 12px;
  padding: 16px 20px;
  transition: box-shadow 0.2s;
}

.convert-card:hover {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
}

.convert-card-body {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
  flex: 1;
}

.convert-card-icon {
  width: 44px;
  height: 44px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  font-size: 20px;
}

.convert-card-info {
  min-width: 0;
  overflow: hidden;
}

.convert-card-title {
  margin: 0 0 6px;
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

.convert-card-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.meta-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
  padding: 2px 8px;
  border-radius: 4px;
}

.convert-card-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.done-tag {
  font-weight: 500;
}

.mpkg-size {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

@media (max-width: 640px) {
  .convert-card {
    flex-direction: column;
    align-items: flex-start;
  }

  .convert-card-actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
