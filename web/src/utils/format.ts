/**
 * Format file size in bytes to human-readable string.
 */
export function formatSize(bytes: number): string {
  if (!bytes || bytes < 0) return '未知'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  return `${size.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

/**
 * Format a date string to locale date/time.
 */
export function formatDate(date: string | null | undefined): string {
  if (!date) return '未知'
  return new Date(date).toLocaleString('zh-CN')
}

/**
 * Format a date string to relative time (e.g. "3小时前", "2天前").
 */
export function formatRelativeTime(date: string | null | undefined): string {
  if (!date) return '未知'
  const now = Date.now()
  const then = new Date(date).getTime()
  const diff = now - then

  if (diff < 0) return '刚刚'

  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`

  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}小时前`

  const days = Math.floor(hours / 24)
  if (days < 30) return `${days}天前`

  return formatDate(date)
}
