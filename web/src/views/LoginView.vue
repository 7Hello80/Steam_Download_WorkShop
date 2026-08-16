<template>
  <div class="auth-page">
    <div class="auth-form">
      <h2>欢迎回来</h2>
      <p style="text-align:center;color:var(--color-text-muted);margin-bottom:28px;font-size:13px;">登录您的 Steam 下载工具账号</p>
      <el-form @submit.prevent="handleLogin">
        <el-form-item>
          <el-input v-model="email" placeholder="邮箱地址" size="large">
            <template #prefix>
              <font-awesome-icon icon="envelope" class="input-icon" />
            </template>
          </el-input>
        </el-form-item>
        <el-form-item>
          <el-input v-model="password" type="password" placeholder="登录密码" size="large" show-password>
            <template #prefix>
              <font-awesome-icon icon="lock" class="input-icon" />
            </template>
          </el-input>
        </el-form-item>
        <el-form-item>
          <el-button
            type="primary"
            size="large"
            :loading="loading"
            @click="handleLogin"
            style="width: 100%"
          >
            {{ loading ? '登录中...' : '登 录' }}
          </el-button>
        </el-form-item>
      </el-form>

      <el-divider>
        <span style="color: var(--color-text-muted); font-size: 12px;">其他登录方式</span>
      </el-divider>

      <el-button
        size="large"
        @click="githubLogin"
        style="width: 100%; font-weight: 500;"
      >
        <font-awesome-icon :icon="['fab', 'github']" style="margin-right:6px;" />
        GitHub 登录
      </el-button>

      <p class="auth-link">
        还没有账号？<router-link to="/register">立即注册</router-link>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'
import * as authApi from '@/api/auth'
import { ElMessage } from 'element-plus'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

const email = ref('')
const password = ref('')
const loading = ref(false)

async function handleLogin() {
  if (!email.value || !password.value) {
    ElMessage.warning('请输入邮箱和密码')
    return
  }

  loading.value = true
  try {
    const resp = await authApi.login(email.value, password.value)
    userStore.setAuth(resp.token, resp.user)
    ElMessage.success('登录成功')
    const redirect = (route.query.redirect as string) || '/download'
    router.push(redirect)
  } catch (err: any) {
    const errMsg: string = err.response?.data?.error || '登录失败'
    if (errMsg.includes('邮箱未验证')) {
      ElMessage.warning(errMsg)
      // 跳转到注册页的验证码步骤，并预填邮箱
      router.push({ path: '/register', query: { email: email.value } })
    } else {
      ElMessage.error(errMsg)
    }
  } finally {
    loading.value = false
  }
}

function githubLogin() {
  window.location.href = authApi.getGitHubLoginUrl()
}
</script>

<style scoped>
.input-icon {
  color: #aaa;
  font-size: 14px;
}
</style>
