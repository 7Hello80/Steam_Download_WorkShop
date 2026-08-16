export interface User {
  id: string
  email: string
  username: string
  github_id: string
  avatar_url: string
  role: string
  email_verified: boolean
  banned: boolean
  banned_at: string | null
  created_at: string
  updated_at: string
}

export interface RegisterResponse {
  message: string
  email: string
}

export interface SteamAccount {
  id: string
  user_id: string
  steam_username: string
  created_at: string
  updated_at: string
}

export interface DownloadTask {
  id: string
  user_id: string
  app_id: number
  pubfile_id: number
  steam_username: string
  status: string
  queue_position: number
  output_dir: string
  zip_path: string
  zip_filename: string
  error_message: string
  file_size: number
  created_at: string
  started_at: string | null
  completed_at: string | null
  expires_at: string | null
}

export interface AuthResponse {
  token: string
  user: User
}

export interface StartDownloadRequest {
  app_id: number
  pubfile_id: number
  steam_username: string
  steam_password: string
  save_credentials: boolean
}

export interface StartDownloadResponse {
  task_id: string
  queue_position: number
}

export interface QueueInfo {
  active_count: number
  queue_length: number
  your_position: number
  your_task_id: string
  total_ahead: number
}

export interface WSMessage {
  type: string
  data: any
}

export interface Announcement {
  id: string
  title: string
  content: string
  is_active: boolean
  created_by: string
  created_at: string
  updated_at: string
}

export interface Sponsor {
  id: string
  name: string
  method: 'wechat' | 'alipay'
  amount: string
  message: string
  is_visible: boolean
  created_at: string
  updated_at: string
}
