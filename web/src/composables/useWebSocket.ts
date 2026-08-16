import { onMounted, onUnmounted } from 'vue'
import { useQueueStore } from '@/stores/queue'

export function useWebSocket() {
  const queueStore = useQueueStore()

  onMounted(() => {
    queueStore.connect()
  })

  onUnmounted(() => {
    queueStore.disconnect()
  })

  return {
    wsStatus: queueStore.wsStatus,
    on: queueStore.on.bind(queueStore),
    off: queueStore.off.bind(queueStore),
    sendPtyInput: queueStore.sendPtyInput.bind(queueStore),
  }
}
