import api from './index'

export interface ConvertibleFile {
  task_id: string
  app_id: number
  pubfile_id: number
  title: string
  video_files: string[]
  preview_name: string
  file_size: number
  converted: boolean
  mpkg_name?: string
  mpkg_size?: number
  download_url?: string
  expires_at?: string
}

export interface ConvertResult {
  status: string
  task_id: string
  title: string
  filename: string
  file_size: number
  download_url: string
}

export async function listConvertible(): Promise<{ files: ConvertibleFile[], total: number }> {
  const { data } = await api.get('/convert/list')
  return data
}

export async function convert(taskId: string): Promise<ConvertResult> {
  const { data } = await api.post('/convert', { task_id: taskId })
  return data
}

/**
 * 直链下载 URL — 直接从 /static/ 公开目录下载，无验证，sendfile 零拷贝。
 */
export function getConvertDownloadUrl(taskId: string, filename: string): string {
  return `/static/${taskId}/${encodeURIComponent(filename)}`
}
