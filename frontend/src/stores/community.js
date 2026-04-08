import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '../api/client.js'

export const useCommunityStore = defineStore('community', () => {
  const posts = ref([])
  const total = ref(0)
  const loading = ref(false)

  async function fetchPosts(page = 0, limit = 20) {
    loading.value = true
    try {
      const { data } = await api.get('/community', { params: { page, limit } })
      posts.value = data.items || []
      total.value = data.total || 0
    } finally {
      loading.value = false
    }
  }

  async function createPost(body) {
    const { data } = await api.post('/community', { body })
    return data
  }

  async function fetchReplies(postId, page = 0, limit = 50) {
    const { data } = await api.get(`/community/${postId}/replies`, { params: { page, limit } })
    return data.items || []
  }

  async function createReply(postId, body) {
    const { data } = await api.post(`/community/${postId}/replies`, { body })
    return data
  }

  async function deletePost(postId) {
    await api.delete(`/community/${postId}`)
  }

  return { posts, total, loading, fetchPosts, createPost, fetchReplies, createReply, deletePost }
})
