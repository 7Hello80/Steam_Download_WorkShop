<template>
  <div class="layout">
    <AnnouncementBanner v-if="userStore.isLoggedIn" />
    <AppNav v-if="userStore.isLoggedIn" />
    <main class="layout-content" :class="{ 'no-nav': !userStore.isLoggedIn }">
      <!-- Material App Bar -->
      <header v-if="userStore.isLoggedIn" class="appbar">
        <span class="appbar-title">{{ pageTitle }}</span>
      </header>
      <div class="appbar-body">
        <router-view />
      </div>
    </main>
    <SponsorModal v-model:visible="showSponsor" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'
import AppNav from '@/components/AppNav.vue'
import AnnouncementBanner from '@/components/AnnouncementBanner.vue'
import SponsorModal, { shouldAutoShowSponsor } from '@/components/SponsorModal.vue'

const route = useRoute()
const userStore = useUserStore()
const showSponsor = ref(false)

const pageTitle = computed(() => {
  const titles: Record<string, string> = {
    Download: '下载中心',
    MyFiles: '我的文件',
    Apps: '应用中心',
    Convert: '壁纸转换',
    WallpaperMaker: '壁纸制作',
    Profile: '个人中心',
    Admin: '管理后台',
  }
  return titles[route.name as string] || 'Steam 下载工具'
})

onMounted(async () => {
  if (userStore.token) {
    await userStore.fetchProfile()
  }
  if (shouldAutoShowSponsor()) {
    setTimeout(() => { showSponsor.value = true }, 1000)
  }
})
</script>
