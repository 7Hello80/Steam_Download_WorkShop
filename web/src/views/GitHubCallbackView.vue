<template>
  <div class="auth-page">
    <div class="callback-message">
      <el-icon :size="48" v-if="loading" class="loading-icon is-loading"><Loading /></el-icon>
      <h2 v-if="loading">正在处理 GitHub 登录...</h2>
      <h2 v-else-if="error" class="error-text">{{ error }}</h2>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { ElMessage } from 'element-plus'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

const loading = ref(true)
const error = ref('')

onMounted(async () => {
  const token = route.query.token as string
  if (token) {
    // Store token first
    localStorage.setItem('token', token)
    userStore.setAuth(token, null as any)
    try {
      // Fetch the full user profile
      await userStore.fetchProfile()
      ElMessage.success('GitHub 登录成功')
      router.push('/download')
    } catch {
      error.value = 'GitHub 登录失败 - 获取用户信息失败'
      loading.value = false
      setTimeout(() => router.push('/login'), 3000)
    }
  } else {
    error.value = 'GitHub 登录失败 - 未收到令牌'
    loading.value = false
    setTimeout(() => router.push('/login'), 3000)
  }
})
</script>

<style scoped>
.error-text {
  color: var(--color-danger);
}
.loading-icon.is-loading {
  animation: rotating 2s linear infinite;
}
@keyframes rotating {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
