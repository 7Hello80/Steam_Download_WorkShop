import api from './index'
import type { Announcement } from '@/types'

export function getActiveAnnouncements(): Promise<Announcement[]> {
  return api.get('/announcements').then((r) => r.data)
}

export function getAllAnnouncements(): Promise<Announcement[]> {
  return api.get('/admin/announcements').then((r) => r.data)
}

export function createAnnouncement(data: { title: string; content: string }): Promise<Announcement> {
  return api.post('/admin/announcements', data).then((r) => r.data)
}

export function updateAnnouncement(
  id: string,
  data: { title: string; content: string; is_active?: boolean }
): Promise<Announcement> {
  return api.put(`/admin/announcements/${id}`, data).then((r) => r.data)
}

export function deleteAnnouncement(id: string): Promise<void> {
  return api.delete(`/admin/announcements/${id}`).then((r) => r.data)
}
