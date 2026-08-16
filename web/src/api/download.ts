import api from './index'
import type { DownloadTask, QueueInfo, StartDownloadRequest, StartDownloadResponse, SteamAccount } from '@/types'

export async function startDownload(req: StartDownloadRequest): Promise<StartDownloadResponse> {
  const { data } = await api.post('/downloads', req)
  return data
}

export async function listDownloads(params: {
  page?: number
  limit?: number
  status?: string
} = {}): Promise<{ tasks: DownloadTask[], total: number, page: number, limit: number }> {
  const { data } = await api.get('/downloads', { params })
  return data
}

export async function getDownload(id: string): Promise<DownloadTask> {
  const { data } = await api.get(`/downloads/${id}`)
  return data
}

export async function cancelDownload(id: string): Promise<{ status: string }> {
  const { data } = await api.post(`/downloads/${id}/cancel`)
  return data
}

export async function saveSteamCredentials(username: string, password: string): Promise<SteamAccount> {
  const { data } = await api.post('/steam/credentials', { steam_username: username, steam_password: password })
  return data
}

export async function getSteamCredentials(): Promise<SteamAccount[]> {
  const { data } = await api.get('/steam/credentials')
  return data
}

export async function getQueueInfo(): Promise<QueueInfo> {
  const { data } = await api.get('/queue')
  return data
}

export async function getTaskOutput(id: string): Promise<{ task_id: string, lines: string[] }> {
  const { data } = await api.get(`/downloads/${id}/output`)
  return data
}

export async function deleteSteamCredentials(id: string): Promise<void> {
  await api.delete(`/steam/credentials/${id}`)
}
