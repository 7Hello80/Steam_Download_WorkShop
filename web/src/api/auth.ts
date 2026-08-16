import api from './index'
import type { AuthResponse, RegisterResponse, User } from '@/types'

export async function register(email: string, username: string, password: string): Promise<RegisterResponse> {
  const { data } = await api.post('/auth/register', { email, username, password })
  return data
}

export async function verifyEmail(email: string, code: string): Promise<AuthResponse> {
  const { data } = await api.post('/auth/verify-email', { email, code })
  return data
}

export async function resendCode(email: string): Promise<void> {
  await api.post('/auth/resend-code', { email })
}

export async function login(email: string, password: string): Promise<AuthResponse> {
  const { data } = await api.post('/auth/login', { email, password })
  return data
}

export async function getProfile(): Promise<User> {
  const { data } = await api.get('/user/profile')
  return data
}

export async function updateProfile(username: string, avatarUrl?: string): Promise<User> {
  const { data } = await api.put('/user/profile', { username, avatar_url: avatarUrl })
  return data
}

export async function uploadAvatar(file: File): Promise<User> {
  const formData = new FormData()
  formData.append('avatar', file)
  const { data } = await api.post('/user/avatar', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return data
}

export function getGitHubLoginUrl(): string {
  return '/api/auth/github'
}
