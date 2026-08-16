<template>
  <div class="profile-page">
    <!-- Horizontal profile card -->
    <div class="profile-card">
      <div class="profile-card-inner">
        <div class="avatar-wrapper" @click="triggerAvatarUpload">
          <el-avatar :src="avatarSrc" :size="72" class="profile-avatar">
            <span class="avatar-placeholder">{{ avatarLetter }}</span>
          </el-avatar>
          <div class="avatar-badge">
            <font-awesome-icon icon="camera" />
          </div>
        </div>
        <input
          ref="avatarInputRef"
          type="file"
          accept="image/png,image/jpeg,image/gif,image/webp"
          style="display:none"
          @change="handleAvatarChange"
        />
        <div class="profile-info">
          <h2 class="profile-username">{{ userStore.user?.username || '用户' }}</h2>
          <p class="profile-email">{{ userStore.user?.email || '' }}</p>
          <div class="profile-tags">
            <span class="profile-tag" :class="{ admin: userStore.user?.role === 'admin' }">
              {{ userStore.user?.role === 'admin' ? '管理员' : '普通用户' }}
            </span>
            <span v-if="userStore.user?.github_id" class="profile-tag github">
              <font-awesome-icon :icon="['fab', 'github']" />
              GitHub
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Settings-style list -->
    <div class="menu-group">
      <div class="menu-group-title">账号设置</div>
      <div class="menu-card">
        <div class="menu-item" @click="openEditName">
          <div class="menu-item-left">
            <font-awesome-icon icon="pen-to-square" />
            <span class="menu-item-label">修改昵称</span>
          </div>
          <div class="menu-item-right">
            <span class="menu-item-value">{{ userStore.user?.username }}</span>
            <font-awesome-icon icon="chevron-right" />
          </div>
        </div>
        <div class="menu-item" @click="triggerAvatarUpload">
          <div class="menu-item-left">
            <font-awesome-icon icon="camera" />
            <span class="menu-item-label">更换头像</span>
          </div>
          <div class="menu-item-right">
            <span class="menu-item-value">点击上传</span>
            <font-awesome-icon icon="chevron-right" />
          </div>
        </div>
      </div>
    </div>

    <!-- Steam Accounts -->
    <div class="menu-group">
      <div class="menu-group-title">Steam 账号</div>
      <div class="menu-card">
        <template v-if="steamAccounts.length > 0">
          <div v-for="acc in steamAccounts" :key="acc.id" class="menu-item">
            <div class="menu-item-left">
              <span class="steam-avatar-letter">{{ acc.steam_username.charAt(0).toUpperCase() }}</span>
              <div>
                <div class="menu-item-label">{{ acc.steam_username }}</div>
                <div class="menu-item-hint">{{ formatDateShort(acc.created_at) }}</div>
              </div>
            </div>
            <el-button size="small" type="danger" text @click.stop="handleDeleteAccount(acc.id)">
              移除
            </el-button>
          </div>
        </template>
        <div v-else class="menu-empty">
          <font-awesome-icon icon="gamepad" />
          <p>暂无保存的 Steam 账号</p>
          <p class="menu-empty-hint">在下载中心创建任务时可选择保存</p>
        </div>
      </div>
    </div>

    <!-- About -->
    <div class="menu-group">
      <div class="menu-group-title">关于</div>
      <div class="menu-card">
        <div class="menu-item" @click="showSponsor = true">
          <div class="menu-item-left">
            <font-awesome-icon icon="heart" class="fa-heart-red" />
            <span class="menu-item-label">赞赏支持</span>
          </div>
          <div class="menu-item-right">
            <span class="menu-item-value">请开发者喝杯咖啡</span>
            <font-awesome-icon icon="chevron-right" />
          </div>
        </div>
      </div>
    </div>

    <!-- Logout -->
    <div class="logout-section">
      <button class="logout-btn" @click="handleLogout">
        <font-awesome-icon icon="right-from-bracket" />
        <span>退出登录</span>
      </button>
    </div>

    <SponsorModal v-model:visible="showSponsor" />

    <!-- Edit nickname dialog -->
    <el-dialog v-model="showEditName" title="修改昵称" width="300px" :close-on-click-modal="false" center>
      <el-input
        v-model="editNameValue"
        placeholder="请输入新昵称"
        maxlength="32"
        minlength="2"
        show-word-limit
        size="large"
      />
      <template #footer>
        <el-button @click="showEditName = false" size="large" style="border-radius:20px;">取消</el-button>
        <el-button type="primary" @click="handleUpdateName" :loading="saving" size="large" style="border-radius:20px;">
          保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { updateProfile, uploadAvatar } from '@/api/auth'
import { getSteamCredentials, deleteSteamCredentials } from '@/api/download'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { SteamAccount } from '@/types'
import { formatDate } from '@/utils/format'
import SponsorModal from '@/components/SponsorModal.vue'

const router = useRouter()
const userStore = useUserStore()

const steamAccounts = ref<SteamAccount[]>([])
const showSponsor = ref(false)
const showEditName = ref(false)
const editNameValue = ref('')
const saving = ref(false)
const avatarInputRef = ref<HTMLInputElement>()

const avatarSrc = computed(() => {
  const url = userStore.user?.avatar_url
  return (url && url.trim()) ? url : ''
})

const avatarLetter = computed(() => {
  const name = userStore.user?.username || '用'
  return name.charAt(0).toUpperCase()
})

function triggerAvatarUpload() {
  avatarInputRef.value?.click()
}

async function handleAvatarChange(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  const validTypes = ['image/png', 'image/jpeg', 'image/gif', 'image/webp']
  if (!validTypes.includes(file.type)) {
    ElMessage.error('仅支持 PNG / JPG / GIF / WebP 格式')
    return
  }
  if (file.size > 2 * 1024 * 1024) {
    ElMessage.error('图片大小不能超过 2MB')
    return
  }
  try {
    const user = await uploadAvatar(file)
    userStore.user = user
    ElMessage.success('头像已更新')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || '上传失败')
  } finally {
    input.value = ''
  }
}

function openEditName() {
  editNameValue.value = userStore.user?.username || ''
  showEditName.value = true
}

async function handleUpdateName() {
  const name = editNameValue.value.trim()
  if (name.length < 2) { ElMessage.warning('昵称至少需要 2 个字符'); return }
  saving.value = true
  try {
    const user = await updateProfile(name)
    userStore.user = user
    ElMessage.success('昵称已更新')
    showEditName.value = false
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || '更新失败')
  } finally {
    saving.value = false
  }
}

async function fetchAccounts() {
  try { steamAccounts.value = await getSteamCredentials() } catch { /* ignore */ }
}

async function handleDeleteAccount(id: string) {
  try {
    await ElMessageBox.confirm('确定要移除此 Steam 账号吗？', '确认', { type: 'warning' })
    await deleteSteamCredentials(id)
    ElMessage.success('账号已移除')
    await fetchAccounts()
  } catch { /* cancelled */ }
}

function handleLogout() {
  ElMessageBox.confirm('确定要退出登录吗？', '提示', {
    confirmButtonText: '退出', cancelButtonText: '取消', type: 'warning',
  }).then(() => {
    userStore.logout()
    router.push('/login')
    ElMessage.success('已退出登录')
  }).catch(() => {})
}

function formatDateShort(dateStr: string): string {
  const d = new Date(dateStr)
  const now = new Date()
  const diffDays = Math.floor((now.getTime() - d.getTime()) / (1000 * 60 * 60 * 24))
  if (diffDays === 0) return '今天'
  if (diffDays === 1) return '昨天'
  if (diffDays < 7) return `${diffDays} 天前`
  return formatDate(dateStr)
}

onMounted(() => { fetchAccounts() })
</script>

<style scoped>
.fa-heart-red { color: #d32f2f; }
</style>
