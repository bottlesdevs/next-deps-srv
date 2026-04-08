<template>
  <div class="post-card" :class="{ 'no-click': noClick }">
    <div class="post-header">
      <div class="author">
        <img v-if="post.author?.avatar_url" :src="post.author.avatar_url" class="avatar" />
        <div v-else class="avatar-ph">{{ post.author?.username?.[0]?.toUpperCase() }}</div>
        <span class="username">{{ post.author?.username || 'Unknown' }}</span>
      </div>
      <span class="date">{{ fmtDate(post.created_at) }}</span>
    </div>
    <p class="body">{{ post.body }}</p>
    <div class="post-footer" v-if="!noClick">
      <span class="replies-count"><i class="pi pi-comments" /> {{ post.reply_count || 0 }}</span>
      <Button
        v-if="canDelete"
        icon="pi pi-trash"
        text
        size="small"
        severity="danger"
        @click.stop="deletePost"
      />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useAuthStore } from '../stores/auth.js'
import api from '../api/client.js'
import Button from 'primevue/button'

const props = defineProps({
  post: { type: Object, required: true },
  noClick: { type: Boolean, default: false },
})
const emit = defineEmits(['deleted'])

const auth = useAuthStore()
const canDelete = computed(() =>
  auth.isAdmin || auth.isMod || auth.user?.id === props.post.author_id
)

function fmtDate(d) { return d ? new Date(d).toLocaleString() : '' }

async function deletePost() {
  if (!confirm('Delete this post?')) return
  await api.delete(`/community/${props.post.id}`)
  emit('deleted')
}
</script>

<style scoped>
.post-card { background: #fff; border-radius: 10px; padding: 1rem 1.25rem; box-shadow: 0 1px 3px rgba(0,0,0,.07); }
.post-card:not(.no-click):hover { box-shadow: 0 3px 8px rgba(0,0,0,.12); }
.post-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: .5rem; }
.author { display: flex; align-items: center; gap: .5rem; }
.avatar { width: 28px; height: 28px; border-radius: 50%; object-fit: cover; }
.avatar-ph { width: 28px; height: 28px; border-radius: 50%; background: #3b82f6; display: flex; align-items: center; justify-content: center; font-size: .75rem; font-weight: 700; color: #fff; }
.username { font-weight: 600; font-size: .875rem; }
.date { font-size: .75rem; color: #94a3b8; }
.body { font-size: .9rem; color: #374151; line-height: 1.5; margin-bottom: .5rem; white-space: pre-wrap; }
.post-footer { display: flex; align-items: center; gap: 1rem; }
.replies-count { font-size: .8rem; color: #94a3b8; display: flex; align-items: center; gap: .25rem; }
</style>
