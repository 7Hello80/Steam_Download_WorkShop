<template>
  <teleport to="body">
    <transition name="ann-fade">
      <div v-if="show && current" class="ann-overlay" @click.self="dismissCurrent">
        <div class="ann-dialog">
          <div class="ann-dialog-header">
            <div class="ann-dialog-icon">
              <font-awesome-icon icon="bullhorn" />
            </div>
            <span class="ann-dialog-title">{{ current.title }}</span>
            <button class="ann-dialog-close" @click="dismissCurrent" title="关闭">
              <font-awesome-icon icon="xmark" />
            </button>
          </div>
          <div class="ann-dialog-body">
            <p class="ann-dialog-content">{{ current.content }}</p>
          </div>
          <div class="ann-dialog-footer">
            <div v-if="visibleAnnouncements.length > 1" class="ann-dialog-dots">
              <span
                v-for="(_, i) in visibleAnnouncements"
                :key="i"
                class="dot"
                :class="{ active: i === currentIndex }"
                @click="currentIndex = i"
              ></span>
            </div>
            <button class="ann-dialog-btn" @click="dismissCurrent">
              {{ visibleAnnouncements.length > 1 ? '下一条' : '我知道了' }}
            </button>
          </div>
        </div>
      </div>
    </transition>
  </teleport>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { getActiveAnnouncements } from '@/api/announcement'
import type { Announcement } from '@/types'

const DISMISSED_KEY = 'dismissed_announcements'

const announcements = ref<Announcement[]>([])
const dismissedIds = ref<Set<string>>(new Set())
const currentIndex = ref(0)
const show = ref(false)

// Load dismissed IDs from localStorage on init
try {
  const saved = JSON.parse(localStorage.getItem(DISMISSED_KEY) || '[]')
  if (Array.isArray(saved)) {
    dismissedIds.value = new Set(saved)
  }
} catch { /* ignore */ }

function saveDismissed() {
  localStorage.setItem(DISMISSED_KEY, JSON.stringify([...dismissedIds.value]))
}

// Reactive computed — depends on refs, not localStorage
const visibleAnnouncements = computed(() => {
  return announcements.value.filter((a) => !dismissedIds.value.has(a.id))
})

const current = computed(() => {
  if (visibleAnnouncements.value.length === 0) return null
  return visibleAnnouncements.value[currentIndex.value]
})

function dismissCurrent() {
  if (visibleAnnouncements.value.length === 0) return
  const id = visibleAnnouncements.value[currentIndex.value].id
  dismissedIds.value.add(id)
  saveDismissed()
  // Vue reactivity: visibleAnnouncements now shrinks by 1
  if (currentIndex.value >= visibleAnnouncements.value.length) {
    show.value = false
  }
}

onMounted(async () => {
  try {
    announcements.value = await getActiveAnnouncements()
    if (visibleAnnouncements.value.length > 0) {
      show.value = true
    }
  } catch {
    // Silently ignore
  }
})
</script>

<style scoped>
.ann-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  padding: 20px;
}

.ann-fade-enter-active,
.ann-fade-leave-active {
  transition: opacity 0.25s ease;
}
.ann-fade-enter-from,
.ann-fade-leave-to {
  opacity: 0;
}

.ann-dialog {
  background: #fff;
  border-radius: 16px;
  width: 100%;
  max-width: 420px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2);
  overflow: hidden;
}

.ann-dialog-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 20px 20px 0;
}

.ann-dialog-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  flex-shrink: 0;
}

.ann-dialog-title {
  flex: 1;
  font-size: 17px;
  font-weight: 700;
  color: #1a1a1a;
  line-height: 1.3;
}

.ann-dialog-close {
  background: none;
  border: none;
  color: #aaa;
  cursor: pointer;
  font-size: 18px;
  padding: 4px;
  line-height: 1;
  border-radius: 6px;
  transition: color 0.2s, background 0.2s;
  flex-shrink: 0;
}

.ann-dialog-close:hover {
  color: #333;
  background: #f0f0f0;
}

.ann-dialog-body {
  padding: 16px 20px;
}

.ann-dialog-content {
  margin: 0;
  font-size: 15px;
  color: #555;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-word;
}

.ann-dialog-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px 20px;
}

.ann-dialog-dots {
  display: flex;
  gap: 6px;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #ddd;
  cursor: pointer;
  transition: background 0.2s;
}

.dot.active {
  background: #6366f1;
}

.ann-dialog-btn {
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  color: #fff;
  border: none;
  padding: 10px 28px;
  border-radius: 20px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.2s;
  margin-left: auto;
}

.ann-dialog-btn:hover {
  opacity: 0.9;
}

@media (max-width: 768px) {
  .ann-dialog {
    max-width: 100%;
    border-radius: 14px;
  }

  .ann-dialog-header {
    padding: 16px 16px 0;
  }

  .ann-dialog-body {
    padding: 12px 16px;
  }

  .ann-dialog-footer {
    padding: 0 16px 16px;
  }
}
</style>
