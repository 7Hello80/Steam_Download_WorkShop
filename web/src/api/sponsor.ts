import api from './index'
import type { Sponsor } from '@/types'

export function getVisibleSponsors(): Promise<Sponsor[]> {
  return api.get('/sponsors').then((r) => r.data)
}

export function getAllSponsors(): Promise<Sponsor[]> {
  return api.get('/admin/sponsors').then((r) => r.data)
}

export function createSponsor(data: {
  name: string
  method: string
  amount: string
  message: string
}): Promise<Sponsor> {
  return api.post('/admin/sponsors', data).then((r) => r.data)
}

export function updateSponsor(
  id: string,
  data: {
    name: string
    method: string
    amount: string
    message: string
    is_visible?: boolean
  }
): Promise<Sponsor> {
  return api.put(`/admin/sponsors/${id}`, data).then((r) => r.data)
}

export function deleteSponsor(id: string): Promise<void> {
  return api.delete(`/admin/sponsors/${id}`).then((r) => r.data)
}
