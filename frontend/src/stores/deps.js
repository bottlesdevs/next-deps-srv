import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '../api/client.js'

export const useDepsStore = defineStore('deps', () => {
  const deps = ref([])
  const total = ref(0)
  const loading = ref(false)
  const page = ref(0)
  const limit = ref(20)

  async function fetchDeps() {
    loading.value = true
    try {
      const { data } = await api.get('/deps', { params: { page: page.value, limit: limit.value } })
      deps.value = data.items || []
      total.value = data.total || 0
    } finally {
      loading.value = false
    }
  }

  async function fetchPending() {
    const { data } = await api.get('/deps/pending')
    return data.items || []
  }

  async function submitDep(manifest) {
    const { data } = await api.post('/deps', manifest)
    return data
  }

  async function approveDep(id) {
    const { data } = await api.post(`/deps/${id}/approve`)
    return data
  }

  async function rejectDep(id, reason) {
    const { data } = await api.post(`/deps/${id}/reject`, { reason })
    return data
  }

  async function getDep(id) {
    const { data } = await api.get(`/deps/${id}`)
    return data
  }

  return { deps, total, loading, page, limit, fetchDeps, fetchPending, submitDep, approveDep, rejectDep, getDep }
})
