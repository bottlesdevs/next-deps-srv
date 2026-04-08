<template>
  <div>
    <div class="page-header">
      <div>
        <h2>Community</h2>
        <p class="text-muted">Discuss and share with the team</p>
      </div>
      <Button icon="pi pi-plus" label="New Post" @click="showNew = true" />
    </div>

    <div class="posts-list">
      <div v-if="loading" class="center"><ProgressSpinner /></div>
      <div v-else-if="posts.length === 0" class="empty">No posts yet. Be the first!</div>
      <PostCard
        v-for="post in posts"
        :key="post.id"
        :post="post"
        @deleted="load"
        @click="openPost(post)"
        class="clickable"
      />
      <Paginator
        v-if="total > limit"
        :rows="limit"
        :totalRecords="total"
        @page="onPage"
      />
    </div>

    <!-- new post -->
    <Dialog v-model:visible="showNew" header="New Post" modal style="width: 500px">
      <div class="field">
        <label>Message</label>
        <Textarea v-model="newBody" rows="5" autoResize class="w-full" />
      </div>
      <template #footer>
        <Button label="Cancel" text @click="showNew = false" />
        <Button label="Post" :disabled="!newBody.trim()" @click="createPost" />
      </template>
    </Dialog>

    <!-- thread view -->
    <Dialog v-model:visible="showThread" :header="'Thread'" modal style="width: 600px">
      <PostCard v-if="activePost" :post="activePost" :no-click="true" />
      <Divider />
      <h4 class="replies-title">Replies</h4>
      <div class="replies-list">
        <PostCard v-for="r in replies" :key="r.id" :post="r" :no-click="true" @deleted="loadReplies" />
      </div>
      <div class="reply-form">
        <Textarea v-model="replyBody" rows="3" autoResize class="w-full" placeholder="Write a reply…" />
        <Button label="Reply" :disabled="!replyBody.trim()" @click="createReply" class="mt-2" />
      </div>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '../api/client.js'
import { useToast } from 'primevue/usetoast'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import Textarea from 'primevue/textarea'
import Divider from 'primevue/divider'
import Paginator from 'primevue/paginator'
import ProgressSpinner from 'primevue/progressspinner'
import PostCard from '../components/PostCard.vue'

const toast = useToast()
const posts = ref([])
const total = ref(0)
const limit = 20
const loading = ref(false)
const showNew = ref(false)
const newBody = ref('')
const showThread = ref(false)
const activePost = ref(null)
const replies = ref([])
const replyBody = ref('')
let currentPage = 0

async function load() {
  loading.value = true
  try {
    const { data } = await api.get(`/community?page=${currentPage}&limit=${limit}`)
    posts.value = data.items || []
    total.value = data.total || 0
  } finally {
    loading.value = false
  }
}

async function createPost() {
  await api.post('/community', { body: newBody.value })
  showNew.value = false
  newBody.value = ''
  load()
}

async function openPost(post) {
  activePost.value = post
  showThread.value = true
  await loadReplies()
}

async function loadReplies() {
  const { data } = await api.get(`/community/${activePost.value.id}/replies`)
  replies.value = data.items || []
}

async function createReply() {
  await api.post(`/community/${activePost.value.id}/replies`, { body: replyBody.value })
  replyBody.value = ''
  loadReplies()
}

function onPage(e) {
  currentPage = e.page
  load()
}

onMounted(load)
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 1.5rem; }
.text-muted { color: #64748b; font-size: .875rem; }
.posts-list { display: flex; flex-direction: column; gap: .75rem; }
.empty { text-align: center; padding: 3rem; color: #94a3b8; }
.center { text-align: center; padding: 2rem; }
.clickable { cursor: pointer; }
.replies-title { margin-bottom: .75rem; font-weight: 600; }
.replies-list { display: flex; flex-direction: column; gap: .5rem; margin-bottom: 1rem; max-height: 300px; overflow-y: auto; }
.reply-form { border-top: 1px solid #f1f5f9; padding-top: 1rem; }
.w-full { width: 100%; }
.mt-2 { margin-top: .5rem; }
</style>
