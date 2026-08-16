/**
 * mpkg-builder.ts — 纯浏览器端 Wallpaper Engine mpkg 壁纸文件构建工具
 *
 * 参考 mpkg.java 和 internal/mpkg/mpkg.go 的二进制格式实现。
 *
 * MPKG 二进制格式 (PKGM0018):
 *   [4 bytes LE: version_string_length]
 *   [version_string_bytes]
 *   [4 bytes LE: file_count]
 *   For each file:
 *     [4 bytes LE: name_length]
 *     [name_bytes]
 *     [4 bytes LE: index]       — 当前文件数据的累计偏移量
 *     [4 bytes LE: size]        — 当前文件数据大小
 *   [所有文件数据按顺序拼接]
 *
 * 所有操作在浏览器端完成，无需后端支持。大文件 (100MB+) 使用 Blob 拼接，
 * 浏览器底层会将 Blob 映射到文件系统缓存而非完全加载到内存。
 */

const MPKG_VERSION = 'PKGM0018'

export interface WallpaperConfig {
  /** 壁纸标题 */
  title: string
  /** 视频文件名 (在 mpkg 内的路径，如 "myvideo.mp4") */
  videoFileName: string
  /** 预览图文件名 (在 mpkg 内的路径，如 "preview.jpg") */
  previewFileName: string
  /** 标签 */
  tags?: string[]
  /** 内容分级 */
  contentRating?: 'Everyone' | 'Mature' | 'Questionable'
}

/**
 * 将字符串编码为 UTF-8 字节数组
 */
function encodeUTF8(str: string): ArrayBuffer {
  const encoder = new TextEncoder()
  const encoded = encoder.encode(str)
  // 确保返回 ArrayBuffer 而非 SharedArrayBuffer，兼容 BlobPart 类型
  return encoded.buffer.slice(encoded.byteOffset, encoded.byteOffset + encoded.byteLength)
}

/**
 * 向 DataView 写入 32 位小端整数
 */
function writeInt32LE(view: DataView, offset: number, value: number): void {
  view.setInt32(offset, value, true)
}

/**
 * 构建 project.json 内容
 */
export function buildProjectJson(config: WallpaperConfig): string {
  const project: Record<string, unknown> = {
    contentrating: config.contentRating || 'Everyone',
    file: config.videoFileName,
    preview: config.previewFileName,
    tags: config.tags && config.tags.length > 0 ? config.tags : ['Anime'],
    title: config.title,
    type: 'video',
  }
  return JSON.stringify(project, null, '\t')
}

/**
 * 计算 mpkg 头部大小 (字节数)
 */
function calculateHeaderSize(entries: Array<{ name: string; size: number }>): number {
  // version length (4) + version string (8) + file count (4)
  let size = 4 + MPKG_VERSION.length + 4
  for (const entry of entries) {
    const nameBytes = encodeUTF8(entry.name)
    size += 4 + nameBytes.byteLength + 4 + 4 // name_len(4) + name + index(4) + size(4)
  }
  return size
}

/**
 * 构建 mpkg 头部二进制数据
 * 返回 { headerBuffer, cumulativeIndex }
 */
function buildMpkgHeader(entries: Array<{ name: string; size: number }>): ArrayBuffer {
  const headerSize = calculateHeaderSize(entries)
  const buffer = new ArrayBuffer(headerSize)
  const view = new DataView(buffer)
  const u8 = new Uint8Array(buffer)
  let offset = 0

  // 写入版本号长度
  writeInt32LE(view, offset, MPKG_VERSION.length)
  offset += 4

  // 写入版本号字符串
  const verBytes = new Uint8Array(encodeUTF8(MPKG_VERSION))
  u8.set(verBytes, offset)
  offset += verBytes.length

  // 写入文件数量
  writeInt32LE(view, offset, entries.length)
  offset += 4

  // 写入每个文件的头部信息
  let cumulativeIndex = 0
  for (const entry of entries) {
    const nameBytes = new Uint8Array(encodeUTF8(entry.name))

    // 文件名长度
    writeInt32LE(view, offset, nameBytes.length)
    offset += 4

    // 文件名
    u8.set(nameBytes, offset)
    offset += nameBytes.length

    // 累计偏移量 (index)
    writeInt32LE(view, offset, cumulativeIndex)
    offset += 4

    // 文件大小
    writeInt32LE(view, offset, entry.size)
    offset += 4

    cumulativeIndex += entry.size
  }

  return buffer
}

/**
 * 从视频文件中提取第一帧作为预览图。
 * 使用 <video> + <canvas> 在浏览器内完成，无需后端。
 *
 * @param videoFile 视频文件
 * @param maxWidth 预览图最大宽度 (等比缩放)
 * @returns JPEG Blob，失败时返回 null
 */
export function extractVideoFrame(
  videoFile: File,
  maxWidth: number = 1920
): Promise<Blob | null> {
  return new Promise((resolve) => {
    const video = document.createElement('video')
    const canvas = document.createElement('canvas')
    const ctx = canvas.getContext('2d')

    if (!ctx) {
      resolve(null)
      return
    }

    // 设置超时，避免无限等待
    const timeout = setTimeout(() => {
      URL.revokeObjectURL(video.src)
      resolve(null)
    }, 10000)

    video.onloadedmetadata = () => {
      // 跳转到 1 秒处或视频 10% 位置，取一个不太可能是黑屏的帧
      const seekTime = Math.min(1.0, video.duration * 0.1)
      video.currentTime = seekTime
    }

    video.onseeked = () => {
      clearTimeout(timeout)

      // 计算缩放后的尺寸
      let w = video.videoWidth
      let h = video.videoHeight
      if (w > maxWidth) {
        h = Math.round(h * (maxWidth / w))
        w = maxWidth
      }

      canvas.width = w
      canvas.height = h
      ctx.drawImage(video, 0, 0, w, h)

      canvas.toBlob(
        (blob) => {
          URL.revokeObjectURL(video.src)
          resolve(blob)
        },
        'image/jpeg',
        0.85
      )
    }

    video.onerror = () => {
      clearTimeout(timeout)
      URL.revokeObjectURL(video.src)
      resolve(null)
    }

    video.preload = 'metadata'
    video.muted = true
    video.src = URL.createObjectURL(videoFile)
  })
}

/**
 * 构建 mpkg 文件并返回可下载的 Blob。
 *
 * 文件在 mpkg 内的顺序：
 *   1. project.json
 *   2. 预览图
 *   3. 视频文件
 *
 * @param config 壁纸配置
 * @param videoFile 视频文件
 * @param previewBlob 预览图 Blob (可以是 JPEG/PNG，或 null 使用默认)
 * @returns mpkg Blob
 */
export function buildMpkg(
  config: WallpaperConfig,
  videoFile: File | Blob,
  previewBlob: Blob | null
): Blob {
  // 获取视频文件名
  const videoName = videoFile instanceof File ? videoFile.name : 'video.mp4'
  config.videoFileName = sanitizeFileName(videoName)

  // 预览图文件名
  const previewName = previewBlob
    ? (previewBlob instanceof File ? previewBlob.name : 'preview.jpg')
    : 'preview.jpg'
  config.previewFileName = sanitizeFileName(previewName)

  // 构建 project.json
  const projectJsonStr = buildProjectJson(config)
  const projectJsonBytes = encodeUTF8(projectJsonStr)

  // 构建文件条目列表 (顺序: project.json, preview, video)
  const entries: Array<{ name: string; size: number; blob: Blob }> = []

  // 1. project.json
  entries.push({
    name: 'project.json',
    size: projectJsonBytes.byteLength,
    blob: new Blob([projectJsonBytes], { type: 'application/json' }),
  })

  // 2. 预览图
  const effectivePreview = previewBlob || createDefaultPreview()
  entries.push({
    name: config.previewFileName,
    size: effectivePreview.size,
    blob: effectivePreview,
  })

  // 获取视频大小
  const videoSize = videoFile.size

  // 3. 视频
  entries.push({
    name: config.videoFileName,
    size: videoSize,
    blob: videoFile,
  })

  // 构建头部
  const headerMeta = entries.map((e) => ({ name: e.name, size: e.size }))
  const headerBuffer = buildMpkgHeader(headerMeta)

  // 拼接所有部分: 头部 + project.json + preview + video
  const parts: BlobPart[] = [headerBuffer]
  for (const entry of entries) {
    parts.push(entry.blob)
  }

  // 生成安全的文件名
  const safeTitle = sanitizeFileName(config.title || 'wallpaper')
  const mpkgBlob = new Blob(parts, { type: 'application/octet-stream' })

  return mpkgBlob
}

/**
 * 触发浏览器下载
 */
export function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  // 延迟释放 URL，确保下载已开始
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

/**
 * 创建一个默认的占位预览图 (1x1 黑色 JPEG)
 * 当用户未提供预览图时使用。
 */
function createDefaultPreview(): Blob {
  // 最小 JPEG (1x1 黑色像素)
  const defaultJpegBase64 =
    '/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAMCAgMCAgMDAwMEAwMEBQgFBQQEBQoHBwYIDAoMCwsKCwsM' +
    'Dw4SEA4ODg8MERMREQ4RERERFBUUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQU' +
    'FBQUFBQUFBQUFBQUFBQUFBT/wAARCAABAAEDAREAAhEBAxEB/8QAFAABAAAAAAAAAAAAAAAAAAAACf/E' +
    'ABQQAQAAAAAAAAAAAAAAAAAAAAD/xAAUAQEAAAAAAAAAAAAAAAAAAAAA/8QAFBEBAAAAAAAAAAAAAAAA' +
    'AAAAAP/aAAwDAQACEQMRAD8AKwA//9k='
  const binaryStr = atob(defaultJpegBase64)
  const bytes = new Uint8Array(binaryStr.length)
  for (let i = 0; i < binaryStr.length; i++) {
    bytes[i] = binaryStr.charCodeAt(i)
  }
  return new Blob([bytes], { type: 'image/jpeg' })
}

/**
 * 清理文件名中的不安全字符
 */
function sanitizeFileName(name: string): string {
  // 移除路径和特殊字符
  let cleaned = name.replace(/[/\\:*?"<>|]/g, '_')
  // 去除控制字符
  cleaned = cleaned.replace(/[\x00-\x1f\x7f]/g, '')
  // 去除首尾空格和点
  cleaned = cleaned.trim().replace(/^\.+|\.+$/g, '')
  if (!cleaned) cleaned = 'file'
  return cleaned
}

/**
 * 格式化文件大小
 */
export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const k = 1024
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  const size = (bytes / Math.pow(k, i)).toFixed(i === 0 ? 0 : 1)
  return `${size} ${units[i]}`
}
