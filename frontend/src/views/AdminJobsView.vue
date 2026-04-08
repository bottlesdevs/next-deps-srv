<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">Build Jobs</h1>
        <p class="page-subtitle">Monitor the dependency build queue</p>
      </div>
      <Button icon="pi pi-refresh" label="Refresh" severity="secondary" size="small" @click="load" :loading="loading" />
    </div>

    <!-- Jobs table -->
    <div class="card" style="padding:0;overflow:hidden">
      <div v-if="loading && !jobs.length" class="empty">
        <i class="pi pi-spin pi-spinner"/>Loading jobs…
      </div>
      <div v-else-if="!jobs.length" class="empty">
        <i class="pi pi-server"/>No build jobs yet
      </div>
      <table v-else class="data-table">
        <thead>
          <tr>
            <th>Job ID</th>
            <th>Dep ID</th>
            <th>Status</th>
            <th>Created</th>
            <th>Error</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="job in jobs" :key="job.id" class="job-row" @click="openLog(job)">
            <td><code class="id-code">{{ job.id.slice(0,8) }}</code></td>
            <td><code class="id-code muted">{{ job.dep_id?.slice(0,8) }}</code></td>
            <td>
              <span :class="['badge', jobBadge(job.status)]">
                <span v-if="job.status === 'running'" class="dot-pulse"/>
                {{ job.status }}
              </span>
            </td>
            <td class="td-muted">{{ fmtDate(job.created_at) }}</td>
            <td class="td-error">{{ job.error || '' }}</td>
            <td class="td-action">
              <button class="log-btn" title="View logs">
                <i class="pi pi-align-left"/> Logs
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Log dialog -->
    <Dialog v-model:visible="logVisible" :header="`Job: ${selectedJob?.id?.slice(0,8)}`"
            :style="{ width: '72vw', maxWidth: '1000px' }" modal @hide="closeLog">
      <div class="log-meta">
        <span :class="['badge', jobBadge(selectedJob?.status)]">{{ selectedJob?.status }}</span>
        <span class="log-dep" v-if="selectedJob?.dep_id">dep: {{ selectedJob.dep_id.slice(0,8) }}</span>
        <span class="log-time" v-if="selectedJob?.created_at">{{ fmtDate(selectedJob.created_at) }}</span>
        <span v-if="selectedJob?.error" class="log-error"><i class="pi pi-exclamation-circle"/>{{ selectedJob.error }}</span>
      </div>
      <div ref="logContainer" class="log-terminal">
        <div v-if="!logLines.length" class="log-empty">Waiting for output…</div>
        <div v-for="(line, i) in logLines" :key="i" class="log-line" :class="lineClass(line)">{{ line }}</div>
      </div>
      <template #footer>
        <Button label="Close" icon="pi pi-times" severity="secondary" size="small" @click="closeLog" />
      </template>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick } from 'vue'
import api from '../api/client.js'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'

const jobs        = ref([])
const loading     = ref(false)
const logVisible  = ref(false)
const selectedJob = ref(null)
const logLines    = ref([])
const logContainer = ref(null)
let es = null

function jobBadge(s) {
  return { done: 'badge-success', failed: 'badge-danger', running: 'badge-accent', queued: 'badge-warning' }[s] || 'badge-default'
}

function fmtDate(d) {
  if (!d) return ''
  const date = new Date(d)
  const diff  = Date.now() - date
  if (diff < 60000)    return 'just now'
  if (diff < 3600000)  return `${Math.floor(diff/60000)}m ago`
  if (diff < 86400000) return `${Math.floor(diff/3600000)}h ago`
  return date.toLocaleString()
}

function lineClass(line) {
  if (!line) return ''
  if (line.startsWith('✅') || line.includes('OK') || line.includes('complete')) return 'line-ok'
  if (line.startsWith('❌') || line.includes('error') || line.includes('Error') || line.includes('fail')) return 'line-err'
  if (line.startsWith('⬇️') || line.startsWith('🔍') || line.startsWith('📦') || line.startsWith('🗂️')) return 'line-info'
  if (line.startsWith('  📄')) return 'line-file'
  if (line.startsWith('    [')) return 'line-sub'
  return ''
}

async function load() {
  loading.value = true
  try {
    const { data } = await api.get('/admin/jobs')
    jobs.value = data.items || []
  } finally {
    loading.value = false
  }
}

function openLog(job) {
  selectedJob.value = job
  logLines.value    = job.logs || []
  logVisible.value  = true

  if (job.status === 'running' || job.status === 'queued') {
    const token = localStorage.getItem('token')
    es = new EventSource(`/api/v1/admin/jobs/${job.id}/log?token=${token}`)
    es.onmessage = (e) => {
      logLines.value.push(e.data)
      nextTick(() => {
        if (logContainer.value) logContainer.value.scrollTop = logContainer.value.scrollHeight
      })
    }
    es.addEventListener('done', () => { es.close(); es = null })
    es.onerror = () => { es?.close(); es = null }
  }
}

function closeLog() {
  es?.close(); es = null
  logVisible.value = false
  logLines.value   = []
}

onMounted(load)
</script>

<style scoped>
/* Table */
.job-row { cursor: pointer; }
.id-code { font-family: 'JetBrains Mono', monospace; font-size: .8rem; color: var(--text); background: var(--surface2); padding: .15rem .45rem; border-radius: 4px; }
.id-code.muted { color: var(--text-muted); }
.td-muted  { color: var(--text-muted); font-size: .8125rem; white-space: nowrap; }
.td-error  { color: var(--danger); font-size: .8rem; max-width: 240px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.td-action { text-align: right; }
.log-btn {
  display: inline-flex; align-items: center; gap: .3rem;
  background: var(--surface2); border: 1px solid var(--border2);
  color: var(--text-muted); border-radius: var(--radius-sm);
  padding: .3rem .75rem; font-size: .8rem; cursor: pointer;
  transition: all var(--transition);
}
.log-btn:hover { color: var(--primary); border-color: var(--primary); }

/* Running pulse dot */
.dot-pulse {
  display: inline-block; width: 6px; height: 6px; border-radius: 50%;
  background: currentColor; animation: pulse 1.2s infinite;
}
@keyframes pulse { 0%,100% { opacity: 1 } 50% { opacity: .3 } }

/* Log dialog */
.log-meta {
  display: flex; align-items: center; gap: .75rem; flex-wrap: wrap;
  padding-bottom: .875rem;
}
.log-dep  { font-size: .75rem; color: var(--text-muted); font-family: monospace; }
.log-time { font-size: .75rem; color: var(--text-faint); }
.log-error { display: flex; align-items: center; gap: .375rem; font-size: .8rem; color: var(--danger); }

.log-terminal {
  background: #0d1117; border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  height: 60vh; overflow-y: auto;
  padding: 1rem 1.25rem;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: .8125rem; line-height: 1.6;
}
.log-empty { color: var(--text-faint); font-style: italic; }
.log-line  { white-space: pre-wrap; word-break: break-all; color: #c9d1d9; }
.line-ok   { color: #3fb950; }
.line-err  { color: #f85149; }
.line-info { color: #58a6ff; }
.line-file { color: #a5d6ff; padding-left: .5rem; }
.line-sub  { color: #6e7681; font-size: .75rem; }
</style>

