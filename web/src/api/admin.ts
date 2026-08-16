import api from './index'
import type { User } from '@/types'

export interface DashboardStats {
  total_users: number
  total_tasks: number
  pending_tasks: number
  running_tasks: number
  completed_tasks: number
  expired_tasks: number
  total_files_size: number
}

export interface PaginatedUsersResponse {
  users: User[]
  total: number
  page: number
  page_size: number
}

export function getUsers(page?: number, pageSize?: number): Promise<PaginatedUsersResponse> {
  const params: Record<string, string> = {}
  if (page && page > 0) params.page = String(page)
  if (pageSize && pageSize > 0) params.page_size = String(pageSize)
  return api.get('/admin/users', { params }).then((r) => r.data)
}

export function getUser(userId: string): Promise<User> {
  return api.get(`/admin/users/${userId}`).then((r) => r.data)
}

export function updateUserRole(userId: string, role: string): Promise<User> {
  return api.put(`/admin/users/${userId}/role`, { role }).then((r) => r.data)
}

export function banUser(userId: string): Promise<User> {
  return api.post(`/admin/users/${userId}/ban`).then((r) => r.data)
}

export function unbanUser(userId: string): Promise<User> {
  return api.post(`/admin/users/${userId}/unban`).then((r) => r.data)
}

export function deleteUser(userId: string): Promise<void> {
  return api.delete(`/admin/users/${userId}`).then((r) => r.data)
}

export function getDashboard(): Promise<DashboardStats> {
  return api.get('/admin/dashboard').then((r) => r.data)
}
