<template>
  <div class="sponsors-view">
    <div class="sponsors-header">
      <h2>
        <font-awesome-icon icon="heart" class="header-icon" />
        赞助名单
      </h2>
      <p class="sponsors-subtitle">感谢以下小伙伴对项目的支持 ❤️</p>
    </div>

    <div v-if="loading" class="loading-wrap">
      <el-skeleton :rows="3" animated />
    </div>

    <div v-else-if="sponsors.length === 0" class="empty-wrap">
      <font-awesome-icon icon="mug-hot" class="empty-icon" />
      <p>还没有赞助记录</p>
      <p class="empty-hint">成为第一个支持本项目的人吧~</p>
    </div>

    <div v-else class="sponsors-grid">
      <div v-for="sp in sponsors" :key="sp.id" class="sponsor-card">
        <div class="sp-card-left">
          <div class="sp-card-avatar">
            <font-awesome-icon icon="user" />
          </div>
          <div class="sp-card-method-line">
            <span v-if="sp.method === 'wechat'" class="sp-method-dot wechat"></span>
            <span v-else class="sp-method-dot alipay"></span>
            <span class="sp-method-text">{{ sp.method === 'wechat' ? '微信' : '支付宝' }}</span>
          </div>
        </div>
        <div class="sp-card-body">
          <div class="sp-card-name">{{ sp.name }}</div>
          <div v-if="sp.amount" class="sp-card-amount">
            <font-awesome-icon icon="circle-dollar-to-slot" class="sp-info-icon" />
            <span class="sp-amount-value">¥{{ sp.amount }}</span>
          </div>
          <div v-if="sp.message" class="sp-card-message">
            <font-awesome-icon icon="comment-dots" class="sp-info-icon" />
            <span>{{ sp.message }}</span>
          </div>
          <div class="sp-card-footer">
            <span class="sp-card-time">{{ formatTime(sp.created_at) }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Sponsor support CTA -->
    <div class="sponsor-cta">
      <el-button type="primary" size="large" round @click="showSponsorModal = true">
        <font-awesome-icon icon="heart" style="margin-right:6px;" />
        我也要赞助
      </el-button>
    </div>

    <!-- Sponsor Modal (QR codes) -->
    <SponsorModal v-model:visible="showSponsorModal" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getVisibleSponsors } from '@/api/sponsor'
import type { Sponsor } from '@/types'
import SponsorModal from '@/components/SponsorModal.vue'

defineOptions({ name: 'SponsorsView' })

const sponsors = ref<Sponsor[]>([])
const loading = ref(true)
const showSponsorModal = ref(false)

function formatTime(ts: string): string {
  const d = new Date(ts)
  const month = d.getMonth() + 1
  const day = d.getDate()
  return `${month}月${day}日`
}

onMounted(async () => {
  try {
    sponsors.value = await getVisibleSponsors()
  } catch {
    // ignore
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.sponsors-view {
  max-width: 720px;
  margin: 0 auto;
  padding: 0 16px;
}

.sponsors-header {
  text-align: center;
  margin-bottom: 32px;
  padding-top: 12px;
}

.sponsors-header h2 {
  font-size: 22px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.header-icon {
  color: #d32f2f;
}

.sponsors-subtitle {
  margin-top: 8px;
  font-size: 14px;
  color: var(--text-secondary);
}

.loading-wrap {
  padding: 20px 0;
}

.empty-wrap {
  text-align: center;
  padding: 60px 20px;
  color: var(--text-secondary);
}

.empty-icon {
  font-size: 48px;
  color: #ccc;
  margin-bottom: 16px;
}

.empty-hint {
  font-size: 13px;
  color: #bbb;
  margin-top: 4px;
}

.sponsors-grid {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.sponsor-card {
  display: flex;
  align-items: stretch;
  gap: 16px;
  background: var(--bg-card);
  border-radius: 16px;
  padding: 20px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);
  border: 1px solid var(--border-color, #f0f0f0);
  transition: all 0.25s ease;
}

.sponsor-card:hover {
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
  border-color: #ddd;
  transform: translateY(-1px);
}

/* ---- Left column: avatar + method ---- */
.sp-card-left {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
  min-width: 52px;
}

.sp-card-avatar {
  width: 46px;
  height: 46px;
  border-radius: 50%;
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: #fff;
  font-size: 21px;
}

.sp-card-method-line {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 3px;
}

.sp-method-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.sp-method-dot.wechat {
  background: #07c160;
}

.sp-method-dot.alipay {
  background: #1677ff;
}

.sp-method-text {
  font-size: 11px;
  color: var(--text-secondary, #999);
  white-space: nowrap;
}

/* ---- Right column: body ---- */
.sp-card-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.sp-card-name {
  font-size: 16px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1.3;
}

/* Amount row */
.sp-card-amount {
  display: flex;
  align-items: center;
  gap: 6px;
}

.sp-info-icon {
  font-size: 14px;
  color: var(--text-secondary, #b0b0b0);
  flex-shrink: 0;
  width: 16px;
  text-align: center;
}

.sp-amount-value {
  font-size: 17px;
  font-weight: 700;
  color: #e53935;
  letter-spacing: 0.5px;
}

/* Message row */
.sp-card-message {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  font-size: 14px;
  color: var(--text-secondary, #666);
  line-height: 1.5;
  word-break: break-word;
}

/* Footer */
.sp-card-footer {
  display: flex;
  align-items: center;
  padding-top: 2px;
}

.sp-card-time {
  font-size: 12px;
  color: #c0c0c0;
}

/* CTA */
.sponsor-cta {
  text-align: center;
  margin-top: 36px;
  padding-bottom: 20px;
}

/* ===== Responsive ===== */
@media (max-width: 768px) {
  .sponsors-view {
    max-width: 100%;
    padding: 0 10px;
  }

  .sponsors-grid {
    gap: 12px;
  }

  .sponsor-card {
    padding: 14px;
    gap: 12px;
    border-radius: 12px;
  }

  .sp-card-left {
    min-width: 40px;
    gap: 8px;
  }

  .sp-card-avatar {
    width: 38px;
    height: 38px;
    font-size: 17px;
  }

  .sp-card-name {
    font-size: 15px;
  }

  .sp-amount-value {
    font-size: 15px;
  }

  .sp-card-message {
    font-size: 13px;
  }
}
</style>
