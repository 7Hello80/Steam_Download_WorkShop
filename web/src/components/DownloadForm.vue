<template>
  <div class="form-card">
    <h3 class="section-title">新建下载任务</h3>
    <el-form :model="form" label-position="top" @submit.prevent="handleSubmit">
      <el-row :gutter="16">
        <el-col :span="12">
          <el-form-item label="应用 App ID">
            <el-input
              v-model.number="form.app_id"
              placeholder="例如: 730（CS:GO）"
              type="number"
              size="large"
            >
              <template #prefix>
                <el-icon><Grid /></el-icon>
              </template>
            </el-input>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="创意工坊资源 ID">
            <el-input
              v-model.number="form.pubfile_id"
              placeholder="例如: 1885082371"
              type="number"
              size="large"
            >
              <template #prefix>
                <el-icon><Files /></el-icon>
              </template>
            </el-input>
          </el-form-item>
        </el-col>
      </el-row>

      <el-divider />

      <!-- Saved accounts -->
      <el-form-item v-if="savedAccounts.length > 0" label="已保存的 Steam 账号">
        <el-select
          v-model="selectedAccount"
          placeholder="选择已保存的账号"
          clearable
          @change="onAccountSelect"
          size="large"
          style="width: 100%"
        >
          <el-option
            v-for="acc in savedAccounts"
            :key="acc.id"
            :label="acc.steam_username"
            :value="acc.steam_username"
          />
        </el-select>
      </el-form-item>

      <el-row :gutter="16">
        <el-col :span="12">
          <el-form-item label="Steam 账号" required>
            <el-input
              v-model="form.steam_username"
              placeholder="Steam 账号名称"
              size="large"
            >
              <template #prefix>
                <el-icon><User /></el-icon>
              </template>
            </el-input>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="Steam 密码" required>
            <el-input
              v-model="form.steam_password"
              type="password"
              placeholder="Steam 密码"
              size="large"
              show-password
            >
              <template #prefix>
                <el-icon><Lock /></el-icon>
              </template>
            </el-input>
          </el-form-item>
        </el-col>
      </el-row>

      <el-form-item>
        <el-checkbox v-model="form.save_credentials" size="large">
          <span style="font-size:13px;">记住 Steam 账号密码（加密存储）</span>
        </el-checkbox>
      </el-form-item>

      <el-button
        type="primary"
        :loading="loading"
        :disabled="!isValid"
        @click="handleSubmit"
        size="large"
        style="width: 100%"
      >
        <el-icon v-if="!loading" style="margin-right:4px;"><VideoPlay /></el-icon>
        {{ loading ? '正在提交...' : '开始下载' }}
      </el-button>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import type { SteamAccount, StartDownloadRequest } from '@/types'
import { getSteamCredentials } from '@/api/download'
const emit = defineEmits<{
  submit: [req: StartDownloadRequest]
}>()

const form = ref<StartDownloadRequest>({
  app_id: 0,
  pubfile_id: 0,
  steam_username: '',
  steam_password: '',
  save_credentials: false,
})

const loading = ref(false)
const savedAccounts = ref<SteamAccount[]>([])
const selectedAccount = ref('')

const isValid = computed(() => {
  return form.value.app_id > 0 &&
    form.value.pubfile_id > 0 &&
    form.value.steam_username.trim() !== '' &&
    form.value.steam_password.trim() !== ''
})

function onAccountSelect(username: string) {
  if (username) {
    form.value.steam_username = username
    form.value.steam_password = ''
  }
}

async function handleSubmit() {
  if (!isValid.value) return
  loading.value = true
  try {
    emit('submit', { ...form.value })
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  try {
    savedAccounts.value = await getSteamCredentials()
  } catch {
    // Ignore
  }
})
</script>
