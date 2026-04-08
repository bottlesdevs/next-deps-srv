<template>
  <div v-if="dep">
    <!-- Back + header -->
    <button class="back-btn" @click="$router.back()">
      <i class="pi pi-arrow-left"/> Back
    </button>

    <div class="dep-hero">
      <div class="dep-hero-icon">{{ (dep.name || '?')[0].toUpperCase() }}</div>
      <div class="dep-hero-info">
        <div class="dep-hero-top">
          <h1 class="page-title">{{ dep.name }}</h1>
          <span :class="['badge', badgeClass(dep.status)]">{{ dep.status }}</span>
        </div>
        <p class="page-subtitle" v-if="dep.manifest?.description">{{ dep.manifest.description }}</p>
        <div class="dep-meta-row">
          <span class="dep-meta-chip" v-if="dep.manifest?.license"><i class="pi pi-file"/>{{ dep.manifest.license }}</span>
          <span class="dep-meta-chip" v-if="dep.manifest?.arch"><i class="pi pi-desktop"/>{{ dep.manifest.arch }}</span>
          <span class="dep-meta-chip" v-if="dep.manifest?.version"><i class="pi pi-tag"/>v{{ dep.manifest.version }}</span>
        </div>
      </div>
    </div>

    <div class="detail-grid">
      <!-- Left: manifest info -->
      <div class="card">
        <div class="card-title">Manifest</div>
        <div class="info-rows">
          <div class="info-row">
            <span class="info-key">Download URL</span>
            <a :href="dep.manifest?.url" target="_blank" class="info-val link">{{ dep.manifest?.url }}</a>
          </div>
          <div class="info-row" v-if="dep.manifest?.expected_hash">
            <span class="info-key">Expected hash</span>
            <code class="info-val mono">{{ dep.manifest.expected_hash }}</code>
          </div>
          <div class="info-row" v-if="dep.manifest?.license">
            <span class="info-key">License</span>
            <span class="info-val">{{ dep.manifest.license }}</span>
          </div>
        </div>
      </div>

      <!-- Right: indexed files -->
      <div class="card">
        <div class="card-title">Indexed Files</div>
        <div v-if="loadingFiles" class="empty"><i class="pi pi-spin pi-spinner"/>Loading…</div>
        <div v-else-if="!files.length" class="empty">
          <i class="pi pi-folder-open"/>No indexed files yet
        </div>
        <div v-else>
          <div class="files-search-wrap">
            <i class="pi pi-search" style="position:absolute;left:.75rem;top:50%;transform:translateY(-50%);font-size:.8rem;color:var(--text-faint)"/>
            <input v-model="fileSearch" placeholder="Filter files…" class="files-search" />
          </div>
          <div class="file-list">
            <button
              v-for="f in filteredFiles"
              :key="f.id"
              class="file-row"
              @click="viewFile(f.name)"
            >
              <i class="pi pi-file file-icon"/>
              <span class="file-name">{{ f.name }}</span>
              <span class="file-revs">{{ f.revision_count }} rev</span>
              <i class="pi pi-download file-dl"/>
            </button>
          </div>
          <div v-if="files.length > filteredFiles.length" class="files-count">
            Showing {{ filteredFiles.length }} of {{ files.length }}
          </div>
        </div>
      </div>
    </div>

    <!-- File revision dialog -->
    <Dialog v-model:visible="fileDialog.visible" :header="fileDialog.name" modal style="width:600px">
      <div v-if="fileDialog.revisions.length" class="rev-list">
        <div v-for="rev in fileDialog.revisions" :key="rev.id" class="rev-row">
          <div class="rev-info">
            <span class="rev-num">Rev {{ rev.revision_num }}</span>
            <code class="rev-hash">{{ rev.hash?.slice(0,16) }}…</code>
            <span class="rev-size">{{ formatSize(rev.size_bytes) }}</span>
          </div>
          <a :href="rev.download_url" download class="rev-dl">
            <i class="pi pi-download"/> Download
          </a>
        </div>
      </div>
      <div v-else class="empty">No revisions found</div>
    </Dialog>
  </div>

  <!-- Loading -->
  <div v-else-if="loading" class="loading-center">
    <i class="pi pi-spin pi-spinner" style="font-size:2rem;color:var(--text-muted)"/>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import api from '../api/client.js'
import Dialog from 'primevue/dialog'

const route = useRoute()
const dep          = ref(null)
const loading      = ref(true)
const loadingFiles = ref(false)
const files        = ref([])
const fileSearch   = ref('')
const fileDialog   = ref({ visible: false, name: '', revisions: [] })

const filteredFiles = computed(() =>
  fileSearch.value
    ? files.value.filter(f => f.name.toLowerCase().includes(fileSearch.value.toLowerCase()))
    : files.value
)

function badgeClass(status) {
  return {
    built: 'badge-success', approved: 'badge-success',
    pending_review: 'badge-warning', building: 'badge-accent',
    rejected: 'badge-danger',
  }[status] || 'badge-default'
}

function formatSize(bytes) {
  if (!bytes) return '?'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1048576) return `${(bytes/1024).toFixed(1)} KB`
  return `${(bytes/1048576).toFixed(1)} MB`
}

async function viewFile(name) {
  const { data } = await api.get(`/files/${encodeURIComponent(name)}`)
  fileDialog.value = { visible: true, name, revisions: data.revisions || [] }
}

async function loadFiles(depId) {
  loadingFiles.value = true
  try {
    const { data } = await api.get(`/deps/${depId}/files`)
    files.value = data.items || data || []
  } catch {
    files.value = []
  } finally {
    loadingFiles.value = false
  }
}

onMounted(async () => {
  try {
    const { data } = await api.get(`/deps/${route.params.id}`)
    dep.value = data
    await loadFiles(route.params.id)
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.back-btn {
  display: inline-flex; align-items: center; gap: .375rem;
  background: none; border: none; cursor: pointer;
  color: var(--text-muted); font-size: .875rem;
  padding: .375rem .5rem; border-radius: var(--radius-sm);
  margin-bottom: 1.5rem; transition: color var(--transition), background var(--transition);
}
.back-btn:hover { color: var(--text); background: var(--surface2); }

/* Hero */
.dep-hero { display: flex; align-items: flex-start; gap: 1.25rem; margin-bottom: 2rem; }
.dep-hero-icon {
  width: 56px; height: 56px; border-radius: var(--radius);
  background: linear-gradient(135deg, var(--primary-bg), rgba(88,166,255,.3));
  display: flex; align-items: center; justify-content: center;
  font-size: 1.5rem; font-weight: 700; color: var(--primary); flex-shrink: 0;
}
.dep-hero-top { display: flex; align-items: center; gap: .75rem; flex-wrap: wrap; margin-bottom: .25rem; }
.dep-meta-row { display: flex; gap: .625rem; margin-top: .625rem; flex-wrap: wrap; }
.dep-meta-chip {
  display: inline-flex; align-items: center; gap: .3rem;
  background: var(--surface2); border: 1px solid var(--border);
  border-radius: 999px; padding: .2rem .75rem;
  font-size: .75rem; color: var(--text-muted);
}
.dep-meta-chip i { font-size: .7rem; }

/* Detail grid */
.detail-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 1.25rem; }

/* Info rows */
.info-rows { display: flex; flex-direction: column; gap: .75rem; }
.info-row { display: flex; flex-direction: column; gap: .2rem; }
.info-key { font-size: .72rem; font-weight: 600; text-transform: uppercase; letter-spacing: .06em; color: var(--text-faint); }
.info-val { font-size: .875rem; color: var(--text); }
.info-val.link { color: var(--primary); word-break: break-all; }
.info-val.mono { font-family: 'JetBrains Mono', monospace; font-size: .8rem; background: var(--surface2); padding: .2rem .5rem; border-radius: 4px; word-break: break-all; }

/* Files */
.files-search-wrap { position: relative; margin-bottom: .75rem; }
.files-search {
  width: 100%; padding: .4rem .75rem .4rem 2.25rem;
  background: var(--surface2); border: 1px solid var(--border2);
  border-radius: var(--radius-sm); color: var(--text); font-size: .875rem;
  outline: none;
}
.files-search:focus { border-color: var(--primary); }
.file-list { display: flex; flex-direction: column; gap: 1px; max-height: 320px; overflow-y: auto; }
.file-row {
  display: flex; align-items: center; gap: .625rem;
  padding: .5rem .5rem; border-radius: var(--radius-sm);
  background: none; border: none; cursor: pointer;
  color: var(--text); font-size: .8125rem; text-align: left; width: 100%;
  transition: background var(--transition);
}
.file-row:hover { background: var(--surface2); }
.file-icon { color: var(--text-faint); font-size: .8rem; flex-shrink: 0; }
.file-name { flex: 1; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.file-revs { font-size: .72rem; color: var(--text-faint); flex-shrink: 0; }
.file-dl   { color: var(--text-faint); font-size: .8rem; flex-shrink: 0; }
.file-row:hover .file-dl { color: var(--primary); }
.files-count { font-size: .75rem; color: var(--text-faint); text-align: right; margin-top: .5rem; }

/* Rev dialog */
.rev-list { display: flex; flex-direction: column; gap: .5rem; }
.rev-row {
  display: flex; align-items: center; justify-content: space-between;
  padding: .75rem 1rem; background: var(--surface2);
  border-radius: var(--radius-sm); gap: 1rem;
}
.rev-info { display: flex; align-items: center; gap: 1rem; flex-wrap: wrap; }
.rev-num  { font-weight: 600; font-size: .875rem; }
.rev-hash { font-size: .75rem; font-family: monospace; color: var(--text-muted); }
.rev-size { font-size: .75rem; color: var(--text-muted); }
.rev-dl {
  display: flex; align-items: center; gap: .375rem;
  background: var(--primary-bg); color: var(--primary);
  padding: .375rem .875rem; border-radius: var(--radius-sm);
  font-size: .8125rem; text-decoration: none; flex-shrink: 0;
  transition: background var(--transition);
}
.rev-dl:hover { background: rgba(88,166,255,.25); text-decoration: none; }

.loading-center { display: flex; align-items: center; justify-content: center; padding: 5rem; }
</style>

