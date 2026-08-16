<template>
  <div class="apps-page">
    <div class="page-header">
      <h2 class="page-title">应用中心</h2>
      <p class="page-desc">
        选择下方应用开始使用工具，点击卡片进入对应功能页面。
      </p>
    </div>

    <div class="apps-grid">
      <!-- 壁纸转换 -->
      <div class="app-card" @click="$router.push('/apps/convert')">
        <div class="app-card-icon convert-icon">
          <font-awesome-icon icon="exchange-alt" :size="'2x'" />
        </div>
        <div class="app-card-body">
          <h3 class="app-card-title">壁纸转换</h3>
          <p class="app-card-desc">
            将已下载的 zip 文件转换为 mpkg 格式，方便导入 Wallpaper Engine 使用。
          </p>
        </div>
        <div class="app-card-arrow">
          <font-awesome-icon icon="chevron-right" />
        </div>
      </div>

      <!-- 壁纸制作 -->
      <div class="app-card" @click="$router.push('/apps/wallpaper-maker')">
        <div class="app-card-icon maker-icon">
          <font-awesome-icon icon="wand-magic-sparkles" :size="'2x'" />
        </div>
        <div class="app-card-body">
          <h3 class="app-card-title">壁纸制作</h3>
          <p class="app-card-desc">
            上传视频和预览图，一键生成 Wallpaper Engine 壁纸文件，所有处理在浏览器本地完成。
          </p>
        </div>
        <div class="app-card-arrow">
          <font-awesome-icon icon="chevron-right" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
</script>

<style scoped>
.apps-page {
  max-width: 720px;
  margin: 0 auto;
}

.page-header {
  margin-bottom: 28px;
}

.page-header .page-title {
  font-size: 20px;
  font-weight: 500;
  color: var(--color-text);
  letter-spacing: 0.15px;
  margin-bottom: 6px;
}

.page-header .page-desc {
  font-size: 13px;
  color: var(--color-text-muted);
  line-height: 1.5;
}

/* ---- App Cards Grid ---- */
.apps-grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.app-card {
  display: flex;
  align-items: center;
  gap: 18px;
  padding: 20px 22px;
  background: #fff;
  border-radius: var(--radius-lg);
  box-shadow: var(--elevation-1);
  cursor: pointer;
  transition: box-shadow 0.25s, transform 0.2s;
  user-select: none;
  -webkit-tap-highlight-color: transparent;
}

.app-card:hover {
  box-shadow: var(--elevation-3);
  transform: translateY(-2px);
}

.app-card:active {
  transform: scale(0.985);
  box-shadow: var(--elevation-1);
}

/* ---- Icon ---- */
.app-card-icon {
  flex-shrink: 0;
  width: 60px;
  height: 60px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
}

.convert-icon {
  background: linear-gradient(135deg, #4a90d9 0%, #357abd 100%);
  box-shadow: 0 4px 12px rgba(74, 144, 217, 0.3);
}

.maker-icon {
  background: linear-gradient(135deg, #e8894b 0%, #d4722b 100%);
  box-shadow: 0 4px 12px rgba(232, 137, 75, 0.3);
}

/* ---- Body ---- */
.app-card-body {
  flex: 1;
  min-width: 0;
}

.app-card-title {
  font-size: 16px;
  font-weight: 500;
  color: var(--color-text);
  letter-spacing: 0.15px;
  margin-bottom: 4px;
}

.app-card-desc {
  font-size: 12px;
  color: var(--color-text-muted);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* ---- Arrow ---- */
.app-card-arrow {
  flex-shrink: 0;
  color: #ccc;
  font-size: 14px;
  transition: color 0.2s, transform 0.2s;
}

.app-card:hover .app-card-arrow {
  color: var(--color-text-secondary);
  transform: translateX(3px);
}

/* ---- Responsive ---- */
@media (max-width: 768px) {
  .app-card {
    padding: 16px 18px;
    gap: 14px;
  }

  .app-card-icon {
    width: 50px;
    height: 50px;
    border-radius: 12px;
  }

  .app-card-title {
    font-size: 15px;
  }
}
</style>
