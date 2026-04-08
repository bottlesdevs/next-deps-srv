import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '../api/client.js'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref(null)

  const roles = computed(() => user.value?.roles || [])
  const isAdmin = computed(() => roles.value.includes('admin'))
  const isMod = computed(() => roles.value.includes('mod') || isAdmin.value)
  const isContributor = computed(() => roles.value.includes('contributor') || isMod.value)

  async function login(username, password) {
    const { data } = await api.post('/auth/login', { username, password })
    token.value = data.token
    localStorage.setItem('token', data.token)
    await fetchMe()
  }

  async function fetchMe() {
    if (!token.value) return
    try {
      const { data } = await api.get('/auth/me')
      user.value = data
    } catch {
      logout()
    }
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
  }

  if (token.value) fetchMe()

  return { token, user, roles, isAdmin, isMod, isContributor, login, logout, fetchMe }
})
