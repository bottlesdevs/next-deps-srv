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

  // Resolves once the initial /auth/me call has settled. The router awaits
  // this so role guards never run against a not-yet-loaded user.
  const ready = ref(null)

  async function login(username, password) {
    const { data } = await api.post('/auth/login', { username, password })
    token.value = data.token
    localStorage.setItem('token', data.token)
    ready.value = fetchMe()
    await ready.value
  }

  async function ensureReady() {
    if (ready.value) await ready.value
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

  if (token.value) ready.value = fetchMe()

  return { token, user, roles, isAdmin, isMod, isContributor, ready, ensureReady, login, logout, fetchMe }
})
