import api from './index'
import type { DownloadTask } from '@/types'

export async function listFiles(params: {
  page?: number
  limit?: number
} = {}): Promise<{ files: DownloadTask[], total: number, page: number, limit: number }> {
  const { data } = await api.get('/files', { params })
  return data
}

export async function deleteFile(id: string): Promise<void> {
  await api.delete(`/files/${id}`)
}

/**
 * 直链下载 URL — 直接从 /static/ 公开目录下载，无验证，sendfile 零拷贝。
 */
export function getFileDownloadUrl(id: string): string {
  return `/static/${id}/workshop_${id}.zip`
}
