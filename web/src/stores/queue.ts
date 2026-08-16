import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { QueueInfo, WSMessage } from '@/types'

export const useQueueStore = defineStore('queue', () => {
  const queueInfo = ref<QueueInfo>({
    active_count: 0,
    queue_length: 0,
    your_position: 0,
    your_task_id: '',
    total_ahead: 0,
  })

  const wsStatus = ref<'disconnected' | 'connecting' | 'connected'>('disconnected')
  let ws: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let reconnectAttempts = 0
  const maxReconnectDelay = 30000 // 30 seconds max

  // Message callbacks registered by views
  const listeners: Map<string, Set<(data: any) => void>> = new Map()

  function on(type: string, callback: (data: any) => void) {
    if (!listeners.has(type)) {
      listeners.set(type, new Set())
    }
    listeners.get(type)!.add(callback)
  }

  function off(type: string, callback: (data: any) => void) {
    listeners.get(type)?.delete(callback)
  }

  function emit(type: string, data: any) {
    listeners.get(type)?.forEach(cb => cb(data))
  }

  function getReconnectDelay(): number {
    // Exponential backoff: 1s, 2s, 4s, 8s, 16s, 30s, 30s...
    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), maxReconnectDelay)
    // Add jitter (±20%)
    const jitter = delay * 0.2 * (Math.random() * 2 - 1)
    return Math.floor(delay + jitter)
  }

  function connect() {
    const token = localStorage.getItem('token')
    if (!token) return

    // Don't stack reconnect attempts
    if (ws && (ws.readyState === WebSocket.CONNECTING || ws.readyState === WebSocket.OPEN)) {
      return
    }

    wsStatus.value = 'connecting'

    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${location.host}/api/ws?token=${token}`

    ws = new WebSocket(wsUrl)

    ws.onopen = () => {
      wsStatus.value = 'connected'
      reconnectAttempts = 0
    }

    ws.onmessage = (event) => {
      try {
        const msg: WSMessage = JSON.parse(event.data)
        emit(msg.type, msg.data)

        if (msg.type === 'queue_update') {
          queueInfo.value = { ...queueInfo.value, ...msg.data }
        }

        // Handle batched output: emit individual lines for compatibility
        if (msg.type === 'pty_output_batch' && msg.data?.lines) {
          for (const line of msg.data.lines) {
            emit('pty_output', { task_id: msg.data.task_id, output: line })
          }
        }
      } catch {
        // Ignore parse errors
      }
    }

    ws.onclose = () => {
      wsStatus.value = 'disconnected'
      ws = null
      // Reconnect with exponential backoff
      const delay = getReconnectDelay()
      reconnectAttempts++
      reconnectTimer = setTimeout(connect, delay)
    }

    ws.onerror = () => {
      ws?.close()
    }
  }

  function disconnect() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    reconnectAttempts = 0
    ws?.close()
    ws = null
    wsStatus.value = 'disconnected'
  }

  function send(type: string, data: any) {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type, data }))
    }
  }

  function sendPtyInput(taskId: string, input: string) {
    send('pty_input', { task_id: taskId, input })
  }

  function setQueueInfo(info: QueueInfo) {
    queueInfo.value = info
  }

  return {
    queueInfo,
    wsStatus,
    connect,
    disconnect,
    on,
    off,
    send,
    sendPtyInput,
    setQueueInfo,
  }
})
