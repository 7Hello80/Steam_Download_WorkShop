<template>
  <div class="not-found-page">
    <div class="not-found-content">
      <h1 class="not-found-code">404</h1>
      <p class="not-found-text">页面不存在</p>
      <el-button type="primary" @click="$router.push('/download')" size="large">
        返回下载中心
      </el-button>
    </div>
  </div>
</template>

<style scoped>
.not-found-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
}
.not-found-content {
  text-align: center;
}
.not-found-code {
  font-size: 6rem;
  font-weight: 700;
  color: var(--color-text);
  opacity: 0.3;
  line-height: 1;
  margin-bottom: 8px;
}
.not-found-text {
  font-size: 1.2rem;
  color: var(--color-text-muted);
  margin-bottom: 24px;
}
</style>
