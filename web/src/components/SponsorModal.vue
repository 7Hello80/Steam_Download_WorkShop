<template>
  <Teleport to="body">
    <Transition name="sponsor-fade">
      <div v-if="visible" class="sponsor-overlay" @click.self="close">
        <div class="sponsor-dialog">
          <div class="sponsor-header">
            <font-awesome-icon icon="heart" class="sponsor-heart" />
            <h2>赞赏支持</h2>
            <p class="sponsor-subtitle">如果这个工具帮到了你，请开发者喝杯咖啡</p>
            <button class="sponsor-close-btn" @click="close" aria-label="关闭">
              <font-awesome-icon icon="xmark" />
            </button>
          </div>

          <div class="sponsor-qr-row">
            <div class="sponsor-qr-card">
              <img
                :src="alipayQr"
                alt="支付宝赞赏码"
                class="sponsor-qr-img"
                @error="onImgError"
              />
              <p class="sponsor-qr-label">支付宝扫一扫</p>
            </div>
            <div class="sponsor-qr-card">
              <img
                :src="wechatQr"
                alt="微信赞赏码"
                class="sponsor-qr-img"
                @error="onImgError"
              />
              <p class="sponsor-qr-label">微信扫一扫</p>
            </div>
          </div>

          <p class="sponsor-footer-text">感谢每一份支持，这是项目持续维护的动力</p>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import alipayQr from '@/assets/alipay.png'
import wechatQr from '@/assets/vx.png'

defineOptions({ name: 'SponsorModal' })

defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
}>()

const STORAGE_KEY = 'sponsor_dismissed'

function close() {
  localStorage.setItem(STORAGE_KEY, '1')
  emit('update:visible', false)
}

function onImgError(e: Event) {
  const img = e.target as HTMLImageElement
  img.style.display = 'none'
}
</script>

<script lang="ts">
export function shouldAutoShowSponsor(): boolean {
  return !localStorage.getItem('sponsor_dismissed')
}
</script>

<style scoped>
.sponsor-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  padding: 20px;
}

.sponsor-dialog {
  background: #fff;
  border-radius: 16px;
  width: 400px;
  max-width: 100%;
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.15);
  overflow: hidden;
  position: relative;
}

.sponsor-header {
  padding: 24px 24px 0;
  text-align: center;
  position: relative;
}

.sponsor-heart {
  font-size: 22px;
  color: #d32f2f;
  display: block;
  margin-bottom: 8px;
}

.sponsor-header h2 {
  font-size: 18px;
  font-weight: 600;
  color: #111;
  margin: 0;
}

.sponsor-subtitle {
  margin-top: 4px;
  font-size: 12px;
  color: #999;
  line-height: 1.5;
}

.sponsor-close-btn {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: none;
  background: #f0f0f0;
  color: #888;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
}

.sponsor-close-btn:hover {
  background: #e0e0e0;
  color: #333;
}

/* Horizontal QR row */
.sponsor-qr-row {
  display: flex;
  gap: 16px;
  padding: 18px 24px;
  justify-content: center;
}

.sponsor-qr-card {
  text-align: center;
  flex: 1;
}

.sponsor-qr-img {
  width: 100%;
  max-width: 150px;
  aspect-ratio: 1;
  object-fit: contain;
  border-radius: 10px;
  border: 1px solid #eee;
  padding: 8px;
  background: #fafafa;
}

.sponsor-qr-label {
  margin-top: 8px;
  font-size: 12px;
  color: #aaa;
  font-weight: 500;
}

.sponsor-footer-text {
  padding: 0 24px 20px;
  text-align: center;
  font-size: 12px;
  color: #bbb;
  line-height: 1.6;
}

/* Transitions */
.sponsor-fade-enter-active {
  transition: opacity 0.3s ease;
}
.sponsor-fade-leave-active {
  transition: opacity 0.2s ease;
}
.sponsor-fade-enter-from,
.sponsor-fade-leave-to {
  opacity: 0;
}

.sponsor-fade-enter-active .sponsor-dialog {
  animation: sponsorScaleIn 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.sponsor-fade-leave-active .sponsor-dialog {
  animation: sponsorScaleOut 0.15s ease;
}

@keyframes sponsorScaleIn {
  from { transform: scale(0.9); opacity: 0; }
  to { transform: scale(1); opacity: 1; }
}

@keyframes sponsorScaleOut {
  from { transform: scale(1); opacity: 1; }
  to { transform: scale(0.95); opacity: 0; }
}
</style>
