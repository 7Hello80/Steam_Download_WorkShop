import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User } from '@/types'
import * as authApi from '@/api/auth'

export const useUserStore = defineStore('user', () => {
  const user = ref<User | null>(null)
  const token = ref<string>(localStorage.getItem('token') || '')

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  function setAuth(authToken: string, authUser: User) {
    token.value = authToken
    user.value = authUser
    localStorage.setItem('token', authToken)
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
  }

  async function fetchProfile() {
    if (!token.value) return
    try {
      user.value = await authApi.getProfile()
    } catch {
      logout()
    }
  }

  return {
    user,
    token,
    isLoggedIn,
    isAdmin,
    setAuth,
    logout,
    fetchProfile,
  }
})
