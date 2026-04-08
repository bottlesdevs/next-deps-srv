<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">Administration</h1>
        <p class="page-subtitle">Platform overview and settings</p>
      </div>
      <Button icon="pi pi-download" label="Backup" severity="secondary" size="small" @click="downloadBackup" />
    </div>

    <!-- Stat cards -->
    <div class="grid-4 section">
      <div class="stat-card">
        <div class="stat-icon" style="background:rgba(88,166,255,.15);color:#58a6ff"><i class="pi pi-box"/></div>
        <div>
          <div class="stat-value">{{ stats.deps ?? '-' }}</div>
          <div class="stat-label">Dependencies</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background:rgba(63,185,80,.15);color:#3fb950"><i class="pi pi-file"/></div>
        <div>
          <div class="stat-value">{{ stats.files ?? '-' }}</div>
          <div class="stat-label">Indexed Files</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background:rgba(188,140,255,.15);color:#bc8cff"><i class="pi pi-server"/></div>
        <div>
          <div class="stat-value">{{ stats.jobs ?? '-' }}</div>
          <div class="stat-label">Build Jobs</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background:rgba(210,153,34,.15);color:#d29922"><i class="pi pi-users"/></div>
        <div>
          <div class="stat-value">{{ stats.users ?? '-' }}</div>
          <div class="stat-label">Users</div>
        </div>
      </div>
    </div>

    <div class="grid-2">
      <!-- Audit log -->
      <div class="card">
        <div class="card-title">Recent Activity</div>
        <div v-if="!audit.length" class="empty"><i class="pi pi-list"/>No activity yet</div>
        <table v-else class="data-table">
          <thead>
            <tr>
              <th>Time</th>
              <th>Actor</th>
              <th>Action</th>
              <th>Target</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in audit" :key="row.id">
              <td class="td-muted">{{ fmtDate(row.timestamp) }}</td>
              <td><span class="actor-chip">{{ row.actor }}</span></td>
              <td>{{ row.action }}</td>
              <td class="td-muted">{{ row.target }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Config panel -->
      <div class="card">
        <div class="card-title">Platform Config</div>
        <div v-if="cfg" class="config-fields">
          <div class="field">
            <label>Rate limit (req/min)</label>
            <InputNumber v-model="cfg.rate_limit" :min="1" />
          </div>
          <div class="field">
            <label>Max build workers</label>
            <InputNumber v-model="cfg.max_workers" :min="1" />
          </div>
          <Button label="Save" icon="pi pi-check" @click="saveConfig" />
        </div>
        <div v-else class="empty"><i class="pi pi-cog"/>Loading config…</div>

        <div class="divider"/>

        <!-- Quick nav -->
        <div class="quick-nav">
          <RouterLink to="/admin/users" class="quick-link">
            <div class="quick-icon" style="background:rgba(88,166,255,.15);color:#58a6ff"><i class="pi pi-users"/></div>
            <div>
              <div class="quick-label">Users</div>
              <div class="quick-sub">Manage accounts & roles</div>
            </div>
            <i class="pi pi-chevron-right quick-arrow"/>
          </RouterLink>
          <RouterLink to="/admin/jobs" class="quick-link">
            <div class="quick-icon" style="background:rgba(188,140,255,.15);color:#bc8cff"><i class="pi pi-server"/></div>
            <div>
              <div class="quick-label">Build Jobs</div>
              <div class="quick-sub">Monitor queue & logs</div>
            </div>
            <i class="pi pi-chevron-right quick-arrow"/>
          </RouterLink>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '../api/client.js'
import { useToast } from 'primevue/usetoast'
import InputNumber from 'primevue/inputnumber'
import Button from 'primevue/button'

const toast = useToast()
const stats = ref({})
const audit = ref([])
const cfg   = ref(null)

function fmtDate(d) {
  if (!d) return ''
  const date = new Date(d)
  const diff  = Date.now() - date
  if (diff < 60000)    return 'just now'
  if (diff < 3600000)  return `${Math.floor(diff/60000)}m ago`
  if (diff < 86400000) return `${Math.floor(diff/3600000)}h ago`
  return date.toLocaleDateString()
}

async function load() {
  const [sRes, aRes, cRes] = await Promise.allSettled([
    api.get('/admin/stats'),
    api.get('/admin/audit?limit=10'),
    api.get('/admin/config'),
  ])
  if (sRes.status === 'fulfilled') stats.value = sRes.value.data
  if (aRes.status === 'fulfilled') audit.value = aRes.value.data?.items || []
  if (cRes.status === 'fulfilled') cfg.value = cRes.value.data
}

async function saveConfig() {
  await api.put('/admin/config', cfg.value)
  toast.add({ severity: 'success', summary: 'Config saved', life: 2000 })
}

async function downloadBackup() {
  try {
    const { data } = await api.post('/admin/backup', {}, { responseType: 'blob' })
    const url = URL.createObjectURL(data)
    const a = document.createElement('a')
    a.href = url; a.download = `backup-${Date.now()}.zip`; a.click()
    URL.revokeObjectURL(url)
  } catch {
    toast.add({ severity: 'error', summary: 'Backup failed', life: 3000 })
  }
}

onMounted(load)
</script>

<style scoped>
.field { display: flex; flex-direction: column; gap: .4rem; margin-bottom: 1rem; }
.field label { font-size: .8125rem; font-weight: 500; color: var(--text-muted); }
.config-fields { display: flex; flex-direction: column; max-width: 280px; }
.td-muted { color: var(--text-muted); font-size: .8125rem; }
.actor-chip {
  background: var(--surface2); border: 1px solid var(--border);
  border-radius: 999px; padding: .1rem .625rem; font-size: .75rem;
  color: var(--text-muted);
}

/* Quick nav */
.quick-nav { display: flex; flex-direction: column; gap: .5rem; }
.quick-link {
  display: flex; align-items: center; gap: .875rem;
  padding: .75rem; border-radius: var(--radius-sm);
  background: var(--surface2); border: 1px solid var(--border);
  text-decoration: none; color: var(--text);
  transition: border-color var(--transition), background var(--transition);
}
.quick-link:hover { border-color: var(--primary); background: var(--primary-bg); text-decoration: none; }
.quick-icon {
  width: 36px; height: 36px; border-radius: var(--radius-sm);
  display: flex; align-items: center; justify-content: center;
  font-size: .9375rem; flex-shrink: 0;
}
.quick-label { font-size: .875rem; font-weight: 500; }
.quick-sub   { font-size: .75rem; color: var(--text-muted); }
.quick-arrow { margin-left: auto; color: var(--text-faint); font-size: .8rem; }
</style>

