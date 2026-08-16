<template>
  <!-- Desktop sidebar -->
  <aside class="sidebar">
    <div class="sidebar-header">
      <h1 class="logo">Steam 下载工具</h1>
    </div>
    <nav class="sidebar-nav">
      <router-link to="/download" class="nav-item" active-class="active">
        <font-awesome-icon icon="download" />
        <span>下载中心</span>
      </router-link>
      <router-link to="/files" class="nav-item" active-class="active">
        <font-awesome-icon icon="folder-open" />
        <span>我的文件</span>
      </router-link>
      <router-link to="/apps" class="nav-item" :class="{ active: isAppsActive }">
        <font-awesome-icon icon="th" />
        <span>应用</span>
      </router-link>
      <router-link to="/sponsors" class="nav-item" active-class="active">
        <font-awesome-icon icon="heart" />
        <span>赞助名单</span>
      </router-link>
      <router-link to="/profile" class="nav-item" active-class="active">
        <font-awesome-icon icon="user" />
        <span>个人中心</span>
      </router-link>
      <router-link v-if="userStore.isAdmin" to="/admin" class="nav-item" active-class="active">
        <font-awesome-icon icon="shield-halved" />
        <span>管理后台</span>
      </router-link>
    </nav>
    <div class="sidebar-footer">
      <span class="user-name">{{ userStore.user?.username || '未登录' }}</span>
    </div>
  </aside>

  <!-- Mobile bottom nav -->
  <nav class="bottom-nav">
    <router-link to="/download" class="bottom-nav-item" active-class="active">
      <font-awesome-icon icon="download" />
      <span>下载</span>
    </router-link>
    <router-link to="/files" class="bottom-nav-item" active-class="active">
      <font-awesome-icon icon="folder-open" />
      <span>文件</span>
    </router-link>
    <router-link to="/apps" class="bottom-nav-item" :class="{ active: isAppsActive }">
      <font-awesome-icon icon="th" />
      <span>应用</span>
    </router-link>
    <router-link to="/sponsors" class="bottom-nav-item" active-class="active">
      <font-awesome-icon icon="heart" />
      <span>赞助</span>
    </router-link>
    <router-link to="/profile" class="bottom-nav-item" active-class="active">
      <font-awesome-icon icon="user" />
      <span>我的</span>
    </router-link>
    <router-link v-if="userStore.isAdmin" to="/admin" class="bottom-nav-item" active-class="active">
      <font-awesome-icon icon="shield-halved" />
      <span>管理</span>
    </router-link>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'

const route = useRoute()
const userStore = useUserStore()

const isAppsActive = computed(() => route.path.startsWith('/apps'))
</script>
