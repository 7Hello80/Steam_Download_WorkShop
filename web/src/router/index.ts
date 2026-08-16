import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      redirect: '/download',
    },
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/LoginView.vue'),
      meta: { guest: true },
    },
    {
      path: '/register',
      name: 'Register',
      component: () => import('@/views/RegisterView.vue'),
      meta: { guest: true },
    },
    {
      path: '/auth/github/callback',
      name: 'GitHubCallback',
      component: () => import('@/views/GitHubCallbackView.vue'),
      meta: { guest: true },
    },
    {
      path: '/download',
      name: 'Download',
      component: () => import('@/views/DownloadView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/files',
      name: 'MyFiles',
      component: () => import('@/views/MyFilesView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/apps',
      name: 'Apps',
      component: () => import('@/views/ApplicationsView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/apps/convert',
      name: 'Convert',
      component: () => import('@/views/ConvertView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/apps/wallpaper-maker',
      name: 'WallpaperMaker',
      component: () => import('@/views/WallpaperMakerView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/sponsors',
      name: 'Sponsors',
      component: () => import('@/views/SponsorsView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/profile',
      name: 'Profile',
      component: () => import('@/views/ProfileView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/admin',
      name: 'Admin',
      component: () => import('@/views/AdminView.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'NotFound',
      component: () => import('@/views/NotFoundView.vue'),
    },
  ],
})

router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('token')

  if (to.meta.requiresAuth && !token) {
    next({ name: 'Login', query: { redirect: to.fullPath } })
  } else if (to.meta.guest && token) {
    next({ name: 'Download' })
  } else if (to.meta.requiresAdmin) {
    // Need to check admin role — pinia store may not be initialized yet,
    // so parse role from stored user info or do a quick check.
    const userStore = useUserStore()
    if (!userStore.isAdmin) {
      next({ name: 'Download' })
    } else {
      next()
    }
  } else {
    next()
  }
})

export default router
