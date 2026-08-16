<template>
  <div class="auth-page">
    <div class="auth-form">
      <h2>创建账号</h2>

      <!-- Step 1: Registration form -->
      <template v-if="step === 1">
        <p style="text-align:center;color:var(--color-text-muted);margin-bottom:28px;font-size:13px;">注册一个新的 Steam 下载工具账号</p>
        <el-form @submit.prevent="handleRegister">
          <el-form-item>
            <el-input v-model="form.email" placeholder="邮箱地址" size="large">
              <template #prefix>
                <font-awesome-icon icon="envelope" class="input-icon" />
              </template>
            </el-input>
          </el-form-item>
          <el-form-item>
            <el-input v-model="form.username" placeholder="昵称（2-32个字符）" size="large">
              <template #prefix>
                <font-awesome-icon icon="user" class="input-icon" />
              </template>
            </el-input>
          </el-form-item>
          <el-form-item>
            <el-input v-model="form.password" type="password" placeholder="密码（至少6位）" size="large" show-password>
              <template #prefix>
                <font-awesome-icon icon="lock" class="input-icon" />
              </template>
            </el-input>
          </el-form-item>
          <el-form-item>
            <el-input v-model="form.confirmPassword" type="password" placeholder="确认密码" size="large" show-password>
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
              @click="handleRegister"
              style="width: 100%"
            >
              {{ loading ? '注册中...' : '注 册' }}
            </el-button>
          </el-form-item>
        </el-form>
      </template>

      <!-- Step 2: Email verification -->
      <template v-if="step === 2">
        <p style="text-align:center;color:var(--color-text-muted);margin-bottom:8px;font-size:13px;">验证码已发送至</p>
        <p style="text-align:center;color:var(--color-primary);margin-bottom:8px;font-size:14px;font-weight:500;">{{ form.email }}</p>
        <p v-if="fromLogin" style="text-align:center;color:#e6a23c;margin-bottom:20px;font-size:12px;">
          如未收到验证码或验证码已过期，请点击下方「重新发送」
        </p>
        <el-form @submit.prevent="handleVerify">
          <el-form-item>
            <div style="display:flex;justify-content:center;">
              <OtpInput v-model="verifyCode" :length="6" />
            </div>
          </el-form-item>
          <el-form-item>
            <el-button
              type="primary"
              size="large"
              :loading="verifying"
              @click="handleVerify"
              style="width: 100%"
            >
              {{ verifying ? '验证中...' : '验证邮箱' }}
            </el-button>
          </el-form-item>
        </el-form>

        <div style="text-align:center;margin-top:12px;">
          <span style="color:var(--color-text-muted);font-size:13px;">没收到验证码？</span>
          <el-button
            link
            type="primary"
            size="small"
            :disabled="resendCountdown > 0"
            @click="handleResend"
            style="margin-left:4px;"
          >
            <template v-if="resendCountdown > 0">{{ resendCountdown }}秒后重发</template>
            <template v-else>重新发送</template>
          </el-button>
        </div>
      </template>

      <p class="auth-link">
        已有账号？<router-link to="/login">立即登录</router-link>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'
import * as authApi from '@/api/auth'
import { ElMessage } from 'element-plus'
import OtpInput from '@/components/OtpInput.vue'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

const step = ref(1)
const form = reactive({
  email: '',
  username: '',
  password: '',
  confirmPassword: '',
})

// 如果从登录页带 email 参数跳过来，直接进入验证码步骤
onMounted(() => {
  const emailParam = (route.query.email as string) || ''
  if (emailParam) {
    form.email = emailParam
    step.value = 2
    fromLogin.value = true // 从登录页跳过来，验证码可能已过期
  }
})
const loading = ref(false)
const verifyCode = ref('')
const verifying = ref(false)
const resendCountdown = ref(0)
const fromLogin = ref(false)

async function handleRegister() {
  if (!form.email || !form.username || !form.password) {
    ElMessage.warning('请填写所有必填字段')
    return
  }
  if (form.username.length < 2 || form.username.length > 32) {
    ElMessage.warning('昵称需要 2-32 个字符')
    return
  }
  if (form.password.length < 6) {
    ElMessage.warning('密码至少需要6个字符')
    return
  }
  if (form.password !== form.confirmPassword) {
    ElMessage.warning('两次输入的密码不一致')
    return
  }

  loading.value = true
  try {
    await authApi.register(form.email, form.username, form.password)
    ElMessage.success('注册成功，请查看邮箱验证码')
    step.value = 2
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || '注册失败')
  } finally {
    loading.value = false
  }
}

async function handleVerify() {
  if (!verifyCode.value || verifyCode.value.length !== 6) {
    ElMessage.warning('请输入6位验证码')
    return
  }

  verifying.value = true
  try {
    const resp = await authApi.verifyEmail(form.email, verifyCode.value)
    userStore.setAuth(resp.token, resp.user)
    ElMessage.success('邮箱验证成功')
    router.push('/download')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || '验证失败')
  } finally {
    verifying.value = false
  }
}

async function handleResend() {
  if (resendCountdown.value > 0) return

  try {
    await authApi.resendCode(form.email)
    ElMessage.success('验证码已重新发送')
    resendCountdown.value = 60
    const timer = setInterval(() => {
      resendCountdown.value--
      if (resendCountdown.value <= 0) {
        clearInterval(timer)
      }
    }, 1000)
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || '发送失败')
  }
}
</script>

<style scoped>
.input-icon {
  color: #aaa;
  font-size: 14px;
}
</style>
