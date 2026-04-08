<template>
  <div>
    <!-- Header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">Overview</h1>
        <p class="page-subtitle">Welcome back, {{ auth.user?.username }}</p>
      </div>
    </div>

    <!-- Stat cards -->
    <div class="grid-4 section">
      <div class="stat-card">
        <div class="stat-icon" style="background:rgba(88,166,255,.15);color:#58a6ff"><i class="pi pi-box"/></div>
        <div>
          <div class="stat-value">{{ stats.deps }}</div>
          <div class="stat-label">Dependencies</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background:rgba(63,185,80,.15);color:#3fb950"><i class="pi pi-check-circle"/></div>
        <div>
          <div class="stat-value">{{ stats.approved }}</div>
          <div class="stat-label">Approved</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background:rgba(188,140,255,.15);color:#bc8cff"><i class="pi pi-server"/></div>
        <div>
          <div class="stat-value">{{ stats.jobs }}</div>
          <div class="stat-label">Build Jobs</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background:rgba(210,153,34,.15);color:#d29922"><i class="pi pi-users"/></div>
        <div>
          <div class="stat-value">{{ stats.users }}</div>
          <div class="stat-label">Users</div>
        </div>
      </div>
    </div>

    <!-- Two columns -->
    <div class="grid-2">
      <!-- Recent deps -->
      <div class="card">
        <div class="card-title-row">
          <span class="card-title" style="margin-bottom:0">Recent Dependencies</span>
          <RouterLink to="/deps" class="card-link">View all →</RouterLink>
        </div>
        <div class="divider" />
        <div v-if="!recentDeps.length" class="empty">
          <i class="pi pi-box"/>No dependencies yet
        </div>
        <div v-else class="dep-list">
          <RouterLink
            v-for="dep in recentDeps"
            :key="dep.id"
            :to="`/deps/${dep.id}`"
            class="dep-row"
          >
            <div class="dep-info">
              <span class="dep-name">{{ dep.name }}</span>
              <span class="dep-ver">{{ dep.version }}</span>
            </div>
            <span :class="badgeClass(dep.status)" class="badge">{{ dep.status }}</span>
          </RouterLink>
        </div>
      </div>

      <!-- Recent activity -->
      <div class="card">
        <div class="card-title-row">
          <span class="card-title" style="margin-bottom:0">Recent Activity</span>
        </div>
        <div class="divider" />
        <div v-if="!recentAudit.length" class="empty">
          <i class="pi pi-clock"/>No activity yet
        </div>
        <div v-else class="timeline">
          <div v-for="(entry, i) in recentAudit" :key="entry.id" class="tl-item">
            <div class="tl-dot" :class="actionColor(entry.action)"/>
            <div class="tl-line" v-if="i < recentAudit.length - 1"/>
            <div class="tl-content">
              <div class="tl-action">{{ entry.action }}</div>
              <div class="tl-meta">{{ entry.username }} · {{ formatDate(entry.created_at) }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { useAuthStore } from '../stores/auth.js'
import api from '../api/client.js'

const auth = useAuthStore()

const stats     = ref({ deps: 0, approved: 0, jobs: 0, users: 0 })
const recentDeps  = ref([])
const recentAudit = ref([])

onMounted(async () => {
  try {
    const [depsRes, statsRes] = await Promise.all([
      api.get('/deps?limit=6'),
      api.get('/admin/stats').catch(() => null),
    ])
    recentDeps.value = depsRes.data?.items || depsRes.data || []
    if (statsRes?.data) {
      const s = statsRes.data
      stats.value = {
        deps:     s.deps     ?? recentDeps.value.length,
        approved: s.approved ?? recentDeps.value.filter(d => d.status === 'approved').length,
        jobs:     s.jobs     ?? 0,
        users:    s.users    ?? 0,
      }
    } else {
      stats.value.deps = recentDeps.value.length
      stats.value.approved = recentDeps.value.filter(d => d.status === 'approved').length
    }
  } catch {}
  try {
    const r = await api.get('/admin/audit?limit=8')
    recentAudit.value = r.data?.items || r.data || []
  } catch {}
})

function badgeClass(status) {
  const m = {
    approved: 'badge-success',
    built: 'badge-info',
    pending_review: 'badge-warning',
    rejected: 'badge-danger',
    building: 'badge-accent',
  }
  return m[status] || 'badge-default'
}

function actionColor(action) {
  if (!action) return 'dot-default'
  const a = action.toLowerCase()
  if (a.includes('creat') || a.includes('add')) return 'dot-success'
  if (a.includes('delet') || a.includes('reject')) return 'dot-danger'
  if (a.includes('updat') || a.includes('approv')) return 'dot-info'
  return 'dot-default'
}

function formatDate(d) {
  if (!d) return ''
  const date = new Date(d)
  const diff = Date.now() - date
  if (diff < 60000)  return 'just now'
  if (diff < 3600000) return `${Math.floor(diff/60000)}m ago`
  if (diff < 86400000) return `${Math.floor(diff/3600000)}h ago`
  return date.toLocaleDateString()
}
</script>

<style scoped>
.card-title-row { display: flex; align-items: center; justify-content: space-between; }
.card-link { font-size: .8125rem; color: var(--primary); text-decoration: none; }
.card-link:hover { text-decoration: underline; }

/* Dep list */
.dep-list { display: flex; flex-direction: column; gap: 2px; }
.dep-row {
  display: flex; align-items: center; justify-content: space-between;
  padding: .625rem .5rem; border-radius: var(--radius-sm);
  text-decoration: none; transition: background var(--transition);
}
.dep-row:hover { background: var(--surface2); text-decoration: none; }
.dep-info { display: flex; align-items: baseline; gap: .5rem; min-width: 0; }
.dep-name { font-size: .875rem; font-weight: 500; color: var(--text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.dep-ver  { font-size: .75rem; color: var(--text-muted); }

/* Timeline */
.timeline { display: flex; flex-direction: column; }
.tl-item  { display: flex; align-items: flex-start; gap: .75rem; padding: .5rem 0; position: relative; }
.tl-dot   {
  width: 8px; height: 8px; border-radius: 50%; margin-top: .4rem; flex-shrink: 0;
  position: relative; z-index: 1;
}
.dot-success { background: var(--success); }
.dot-danger  { background: var(--danger); }
.dot-info    { background: var(--info); }
.dot-default { background: var(--border2); }
.tl-line {
  position: absolute; left: 3.5px; top: 22px;
  width: 1px; height: calc(100% - 10px);
  background: var(--border); z-index: 0;
}
.tl-content { flex: 1; min-width: 0; }
.tl-action  { font-size: .8125rem; font-weight: 500; color: var(--text); }
.tl-meta    { font-size: .75rem; color: var(--text-muted); margin-top: .1rem; }
</style>

