<template>
  <div class="page-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <h2 class="page-title">壁纸制作</h2>
      <p class="page-desc">
        上传视频和预览图，一键生成 Wallpaper Engine 壁纸文件（mpkg 格式），
        所有处理均在浏览器本地完成，不会上传到服务器。
      </p>
    </div>

    <!-- 主体双栏 -->
    <div class="maker-layout">
      <!-- ========== 左侧主表单 ========== -->
      <div class="maker-main">
        <el-card shadow="hover" class="maker-card">
          <template #header>
            <div class="card-header">
              <span class="card-title">
                <font-awesome-icon icon="video" class="card-title-icon" />
                视频文件
              </span>
              <el-tag v-if="!videoFile" type="danger" size="small" effect="plain">必填</el-tag>
              <el-tag v-else type="success" size="small" effect="plain">已选择</el-tag>
            </div>
          </template>

          <!-- 未选择视频 -->
          <el-upload
            v-if="!videoFile"
            ref="videoUploadRef"
            drag
            :auto-upload="false"
            :show-file-list="false"
            accept="video/*"
            :on-change="handleVideoChange"
            class="upload-drag-area"
          >
            <font-awesome-icon icon="cloud-upload-alt" :size="'3x'" class="upload-drag-icon" />
            <div class="upload-drag-text">
              <em>点击选择</em> 或拖拽视频文件到此处
            </div>
            <div class="upload-drag-hint">
              支持 MP4, AVI, MKV, WebM, MOV, WMV 等格式
            </div>
          </el-upload>

          <!-- 已选择视频 -->
          <div v-else class="file-selected-box">
            <div class="file-selected-left">
              <div class="file-icon-wrap">
                <font-awesome-icon icon="file-video" :size="'2x'" />
              </div>
              <div class="file-selected-info">
                <span class="file-selected-name" :title="videoFile.name">{{ videoFile.name }}</span>
                <div class="file-selected-meta">
                  <el-tag size="small" type="info">{{ formatFileSize(videoFile.size) }}</el-tag>
                  <el-tag v-if="videoResolution" size="small" type="info">{{ videoResolution }}</el-tag>
                  <el-tag v-if="videoDuration" size="small" type="info">{{ formatDuration(videoDuration) }}</el-tag>
                </div>
              </div>
            </div>
            <el-button type="danger" plain size="small" @click="removeVideo">
              <font-awesome-icon icon="xmark" /> 移除
            </el-button>
          </div>
        </el-card>

        <!-- 预览图 -->
        <el-card shadow="hover" class="maker-card">
          <template #header>
            <div class="card-header">
              <span class="card-title">
                <font-awesome-icon icon="image" class="card-title-icon" />
                预览图
              </span>
              <el-tooltip content="用作壁纸封面展示，可从视频自动提取" placement="top">
                <font-awesome-icon icon="circle-question" class="help-icon" />
              </el-tooltip>
              <el-tag v-if="previewBlob" type="success" size="small" effect="plain">已选择</el-tag>
            </div>
          </template>

          <div class="preview-upload-row">
            <!-- 预览图缩略图 or 上传区 -->
            <div class="preview-left">
              <div v-if="previewBlob && previewObjectUrl" class="preview-img-wrap">
                <img :src="previewObjectUrl" class="preview-img" alt="预览图" />
                <div class="preview-img-overlay">
                  <el-button type="danger" circle :icon="Close" size="small" @click="removePreview" />
                </div>
              </div>
              <el-upload
                v-else
                :auto-upload="false"
                :show-file-list="false"
                accept="image/*"
                :on-change="handlePreviewChange"
                class="preview-upload-btn"
              >
                <div class="preview-upload-placeholder">
                  <font-awesome-icon icon="cloud-upload-alt" :size="'2x'" />
                  <span>上传预览图</span>
                </div>
              </el-upload>
            </div>

            <div class="preview-right">
              <p class="preview-hint-text">用作壁纸在软件中的封面展示</p>
              <el-button
                v-if="videoFile"
                type="primary"
                plain
                :loading="extracting"
                :disabled="!videoFile"
                @click="handleExtractFrame"
              >
                <font-awesome-icon icon="camera" />
                {{ extracting ? '正在提取...' : '从视频自动提取' }}
              </el-button>
              <p v-if="!videoFile" class="preview-hint-text muted">请先上传视频后再使用自动提取</p>
            </div>
          </div>
        </el-card>

        <!-- 壁纸信息 -->
        <el-card shadow="hover" class="maker-card">
          <template #header>
            <span class="card-title">
              <font-awesome-icon icon="info-circle" class="card-title-icon" />
              壁纸信息
            </span>
          </template>

          <el-form label-position="top" :model="formData" class="wallpaper-form">
            <el-form-item label="壁纸标题" required>
              <el-input
                v-model="title"
                placeholder="为你的壁纸起一个名字"
                maxlength="100"
                show-word-limit
                clearable
              />
            </el-form-item>

            <el-form-item label="标签">
              <el-select
                v-model="tags"
                multiple
                filterable
                allow-create
                default-first-option
                placeholder="添加标签（回车确认）"
                :reserve-keyword="false"
                clearable
                class="tag-select"
              >
                <el-option
                  v-for="tag in suggestedTags"
                  :key="tag"
                  :label="tag"
                  :value="tag"
                />
              </el-select>
              <div class="tag-suggestions">
                <span class="suggest-label">常用:</span>
                <el-tag
                  v-for="tag in suggestedTags"
                  :key="tag"
                  size="small"
                  class="suggest-tag"
                  :type="tags.includes(tag) ? 'primary' : 'info'"
                  effect="plain"
                  @click="toggleTag(tag)"
                >
                  {{ tag }}
                </el-tag>
              </div>
            </el-form-item>

            <el-form-item label="内容分级">
              <el-radio-group v-model="contentRating" class="rating-group">
                <el-radio-button value="Everyone">
                  <font-awesome-icon icon="circle-check" />
                  全年龄
                </el-radio-button>
                <el-radio-button value="Questionable">
                  <font-awesome-icon icon="triangle-exclamation" />
                  敏感内容
                </el-radio-button>
                <el-radio-button value="Mature">
                  <font-awesome-icon icon="shield-halved" />
                  成人内容
                </el-radio-button>
              </el-radio-group>
            </el-form-item>
          </el-form>
        </el-card>

        <!-- 生成操作区 -->
        <div class="generate-area">
          <el-alert
            v-if="!canGenerate && (videoFile || title)"
            title="请确保已上传视频文件并填写壁纸标题"
            type="warning"
            :closable="false"
            show-icon
            class="generate-alert"
          />

          <el-button
            type="primary"
            size="large"
            :disabled="!canGenerate"
            :loading="generating"
            @click="handleGenerate"
            class="generate-btn"
          >
            <font-awesome-icon v-if="!generating" icon="wand-magic-sparkles" />
            {{ generating ? '正在生成 mpkg 文件...' : '生成并下载 mpkg 壁纸' }}
          </el-button>

          <!-- 生成成功 -->
          <transition name="el-fade-in">
            <el-result
              v-if="mpkgSize"
              icon="success"
              title="壁纸生成成功！"
              :sub-title="`文件大小: ${formatFileSize(mpkgSize)}，下载已自动开始`"
              class="generate-result"
            >
              <template #extra>
                <el-button type="primary" @click="handleGenerate">
                  <font-awesome-icon icon="download" /> 重新下载
                </el-button>
                <el-button @click="resetAll">制作新壁纸</el-button>
              </template>
            </el-result>
          </transition>
        </div>
      </div>

      <!-- ========== 右侧面板 ========== -->
      <div class="maker-sidebar">
        <!-- 视频预览 -->
        <el-card shadow="hover" class="sidebar-card">
          <template #header>
            <span class="card-title">
              <font-awesome-icon icon="play" class="card-title-icon" />
              实时预览
            </span>
          </template>
          <div v-if="videoObjectUrl" class="video-preview-wrap">
            <video
              ref="videoPlayerRef"
              :src="videoObjectUrl"
              controls
              loop
              class="video-preview"
            />
            <el-descriptions :column="2" size="small" border class="video-metadata">
              <el-descriptions-item label="分辨率">
                {{ videoResolution || '--' }}
              </el-descriptions-item>
              <el-descriptions-item label="时长">
                {{ videoDuration ? formatDuration(videoDuration) : '--' }}
              </el-descriptions-item>
            </el-descriptions>
          </div>
          <el-empty
            v-else
            description="上传视频后即可预览"
            :image-size="80"
          />
        </el-card>

        <!-- 使用步骤 -->
        <el-card shadow="hover" class="sidebar-card">
          <template #header>
            <span class="card-title">
              <font-awesome-icon icon="book" class="card-title-icon" />
              操作步骤
            </span>
          </template>
          <el-steps direction="vertical" :space="28" class="help-steps">
            <el-step title="上传视频" description="选择或拖拽壁纸视频文件" />
            <el-step title="设置预览图" description="上传图片或从视频自动提取" />
            <el-step title="填写信息" description="设置标题、标签和分级" />
            <el-step title="生成下载" description="一键生成 mpkg 并下载到本地" />
            <el-step title="导入使用" description="将 .mpkg 放入 Wallpaper Engine projects 目录" />
          </el-steps>

          <el-divider />
          <el-alert
            title="隐私安全"
            description="所有处理在浏览器内完成，视频和图片不会上传到服务器。"
            type="success"
            :closable="false"
            show-icon
          />
        </el-card>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted, nextTick, reactive } from 'vue'
import { Close } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type { UploadFile, UploadInstance } from 'element-plus'
import {
  buildMpkg,
  downloadBlob,
  extractVideoFrame,
  formatFileSize,
  type WallpaperConfig,
} from '@/utils/mpkg-builder'

// ========== 视频 ==========
const videoFile = ref<File | null>(null)
const videoObjectUrl = ref<string | null>(null)
const videoUploadRef = ref<UploadInstance | null>(null)
const videoPlayerRef = ref<HTMLVideoElement | null>(null)
const videoDuration = ref(0)
const videoResolution = ref('')

function handleVideoChange(file: UploadFile) {
  const raw = file.raw
  if (raw) {
    setVideoFile(raw)
  }
}

function setVideoFile(file: File) {
  if (videoObjectUrl.value) URL.revokeObjectURL(videoObjectUrl.value)
  videoFile.value = file
  videoObjectUrl.value = URL.createObjectURL(file)
  videoDuration.value = 0
  videoResolution.value = ''

  // 读取元数据
  const video = document.createElement('video')
  video.preload = 'metadata'
  video.onloadedmetadata = () => {
    videoDuration.value = video.duration
    videoResolution.value = `${video.videoWidth}×${video.videoHeight}`
    URL.revokeObjectURL(video.src)
  }
  video.src = videoObjectUrl.value
}

function removeVideo() {
  if (videoObjectUrl.value) URL.revokeObjectURL(videoObjectUrl.value)
  videoFile.value = null
  videoObjectUrl.value = null
  videoDuration.value = 0
  videoResolution.value = ''
}

// ========== 预览图 ==========
const previewBlob = ref<Blob | null>(null)
const previewObjectUrl = ref<string | null>(null)
const previewFileName = ref('')
const extracting = ref(false)

function handlePreviewChange(file: UploadFile) {
  const raw = file.raw
  if (raw) {
    setPreviewBlob(raw, raw.name)
  }
}

function setPreviewBlob(blob: Blob, name: string) {
  if (previewObjectUrl.value) URL.revokeObjectURL(previewObjectUrl.value)
  previewBlob.value = blob
  previewFileName.value = name
  previewObjectUrl.value = blob.type.startsWith('image/') ? URL.createObjectURL(blob) : null
}

function removePreview() {
  if (previewObjectUrl.value) URL.revokeObjectURL(previewObjectUrl.value)
  previewBlob.value = null
  previewObjectUrl.value = null
  previewFileName.value = ''
}

async function handleExtractFrame() {
  if (!videoFile.value) return
  extracting.value = true
  try {
    const frame = await extractVideoFrame(videoFile.value)
    if (frame) {
      setPreviewBlob(frame, 'preview.jpg')
      ElMessage.success('预览图提取成功')
    } else {
      ElMessage.error('预览图提取失败，请手动上传')
    }
  } finally {
    extracting.value = false
  }
}

// ========== 壁纸信息 ==========
const formData = reactive({})
const title = ref('')
const tags = ref<string[]>([])
const contentRating = ref<'Everyone' | 'Questionable' | 'Mature'>('Everyone')

const suggestedTags = ['Anime', 'Game', 'Nature', 'Abstract', 'Sci-Fi', 'Fantasy', 'Music', 'Movie']

function toggleTag(tag: string) {
  const idx = tags.value.indexOf(tag)
  if (idx >= 0) {
    tags.value.splice(idx, 1)
  } else {
    tags.value.push(tag)
  }
}

// ========== 生成 ==========
const generating = ref(false)
const mpkgSize = ref(0)

const canGenerate = computed(() => {
  return !!(videoFile.value && title.value.trim())
})

async function handleGenerate() {
  if (!videoFile.value || !title.value.trim()) return

  generating.value = true
  mpkgSize.value = 0

  try {
    await nextTick()

    const config: WallpaperConfig = {
      title: title.value.trim(),
      videoFileName: videoFile.value.name,
      previewFileName: previewBlob.value
        ? (previewBlob.value instanceof File ? (previewBlob.value as File).name : 'preview.jpg')
        : 'preview.jpg',
      tags: tags.value.length > 0 ? tags.value : undefined,
      contentRating: contentRating.value,
    }

    const mpkgBlob = buildMpkg(config, videoFile.value, previewBlob.value)
    mpkgSize.value = mpkgBlob.size

    const safeTitle = config.title.replace(/[/\\:*?"<>|]/g, '_').trim() || 'wallpaper'
    downloadBlob(mpkgBlob, `${safeTitle}.mpkg`)

    ElMessage.success(`壁纸 "${config.title}" 生成成功！`)
  } catch (e: any) {
    ElMessage.error(`生成失败: ${e?.message || '未知错误'}`)
  } finally {
    generating.value = false
  }
}

function resetAll() {
  removeVideo()
  removePreview()
  title.value = ''
  tags.value = []
  contentRating.value = 'Everyone'
  mpkgSize.value = 0
}

// ========== 工具函数 ==========
function formatDuration(seconds: number): string {
  if (!seconds || !isFinite(seconds)) return '--'
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${m}:${s.toString().padStart(2, '0')}`
}

// ========== 清理 ==========
onUnmounted(() => {
  if (videoObjectUrl.value) URL.revokeObjectURL(videoObjectUrl.value)
  if (previewObjectUrl.value) URL.revokeObjectURL(previewObjectUrl.value)
})
</script>

<style scoped>
/* ======== 页面头部 ======== */
.page-header {
  margin-bottom: 24px;
}

.page-title {
  font-size: 22px;
  font-weight: 700;
  margin: 0 0 6px;
  color: var(--el-text-color-primary);
}

.page-desc {
  margin: 0;
  color: var(--el-text-color-secondary);
  font-size: 14px;
  line-height: 1.7;
}

/* ======== 双栏布局 ======== */
.maker-layout {
  display: grid;
  grid-template-columns: 1fr 380px;
  gap: 24px;
  align-items: start;
}

@media (max-width: 960px) {
  .maker-layout {
    grid-template-columns: 1fr;
  }
}

/* ======== 卡片通用 ======== */
.maker-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.card-title {
  font-size: 15px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 6px;
}

.card-title-icon {
  color: var(--el-color-primary);
}

.help-icon {
  color: var(--el-text-color-placeholder);
  cursor: help;
  font-size: 14px;
}

/* ======== 上传拖拽区（无文件） ======== */
.upload-drag-area {
  width: 100%;
}

.upload-drag-area :deep(.el-upload-dragger) {
  padding: 48px 20px;
  border-radius: 10px;
}

.upload-drag-icon {
  color: var(--el-color-primary);
  margin-bottom: 12px;
}

.upload-drag-text {
  font-size: 15px;
  color: var(--el-text-color-regular);
  margin-bottom: 6px;
}

.upload-drag-text em {
  color: var(--el-color-primary);
  font-style: normal;
  font-weight: 600;
}

.upload-drag-hint {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

/* ======== 已选文件展示 ======== */
.file-selected-box {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 4px 0;
}

.file-selected-left {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
  flex: 1;
}

.file-icon-wrap {
  width: 48px;
  height: 48px;
  flex-shrink: 0;
  border-radius: 10px;
  background: var(--el-color-success-light-9);
  color: var(--el-color-success);
  display: flex;
  align-items: center;
  justify-content: center;
}

.file-selected-info {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.file-selected-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  word-break: break-all;
  line-height: 1.4;
}

.file-selected-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

/* ======== 预览图区域 ======== */
.preview-upload-row {
  display: flex;
  gap: 20px;
  align-items: center;
}

.preview-left {
  flex-shrink: 0;
}

.preview-upload-btn :deep(.el-upload) {
  display: block;
}

.preview-upload-placeholder {
  width: 160px;
  height: 100px;
  border: 2px dashed var(--el-border-color);
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  transition: all 0.2s;
  font-size: 13px;
}

.preview-upload-placeholder:hover {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.preview-img-wrap {
  position: relative;
  width: 160px;
  height: 100px;
  border-radius: 10px;
  overflow: hidden;
}

.preview-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.preview-img-overlay {
  position: absolute;
  top: 4px;
  right: 4px;
}

.preview-right {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}

.preview-hint-text {
  margin: 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.preview-hint-text.muted {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}

/* ======== 表单 ======== */
.wallpaper-form {
  margin-top: 4px;
}

.wallpaper-form :deep(.el-form-item) {
  margin-bottom: 18px;
}

.wallpaper-form :deep(.el-form-item:last-child) {
  margin-bottom: 0;
}

.tag-select {
  width: 100%;
}

.tag-suggestions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
}

.suggest-label {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  margin-right: 4px;
}

.suggest-tag {
  cursor: pointer;
  transition: all 0.15s;
}

.suggest-tag:hover {
  opacity: 0.8;
}

.rating-group {
  width: 100%;
}

.rating-group :deep(.el-radio-button__inner) {
  display: flex;
  align-items: center;
  gap: 6px;
}

/* ======== 生成区域 ======== */
.generate-area {
  padding-top: 8px;
}

.generate-alert {
  margin-bottom: 16px;
}

.generate-btn {
  width: 100%;
  height: 48px;
  font-size: 16px;
  font-weight: 600;
}

.generate-result {
  margin-top: 16px;
  padding: 24px;
}

/* ======== 右侧边栏 ======== */
.maker-sidebar {
  position: sticky;
  top: 20px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.sidebar-card {
  /* el-card covers */
}

.sidebar-card :deep(.el-card__body) {
  padding: 16px;
}

/* 视频预览 */
.video-preview-wrap {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.video-preview {
  width: 100%;
  border-radius: 8px;
  background: #000;
  display: block;
}

.video-metadata {
  font-size: 12px;
}

/* 步骤 */
.help-steps {
  margin-top: 4px;
}

.help-steps :deep(.el-step__title) {
  font-size: 13px;
  font-weight: 500;
}

.help-steps :deep(.el-step__description) {
  font-size: 12px;
}
</style>
