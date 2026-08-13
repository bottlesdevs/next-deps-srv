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
          <Button
            v-if="auth.isAdmin"
            label="Delete reference"
            icon="pi pi-trash"
            severity="danger"
            size="small"
            text
            :loading="deleting"
            @click="deleteReference"
          />
        </div>
        <p class="page-subtitle" v-if="dep.description">{{ dep.description }}</p>
        <div class="dep-meta-row">
          <span class="dep-meta-chip" v-if="dep.license"><i class="pi pi-file"/>{{ dep.license }}</span>
          <span class="dep-meta-chip" v-if="dep.entry?.version"><i class="pi pi-tag"/>v{{ dep.entry.version }}</span>
          <span class="dep-meta-chip" v-if="dep.kind"><i class="pi pi-box"/>{{ dep.kind }}</span>
          <span class="dep-meta-chip" v-if="dep.entry?.slot"><i class="pi pi-sitemap"/>{{ dep.entry.slot }}</span>
          <span class="dep-meta-chip" v-for="p in platforms" :key="p"><i class="pi pi-desktop"/>{{ p }}</span>
        </div>
      </div>
    </div>

    <div class="detail-grid">
      <!-- Left: catalog item + artifacts -->
      <div class="card">
        <div class="card-title">Artifacts</div>
        <div class="info-rows">
          <div class="info-row">
            <span class="info-key">Entry ID</span>
            <code class="info-val mono">{{ dep.entry?.id }}</code>
          </div>
          <div class="info-row" v-if="dep.entry?.requirements?.length">
            <span class="info-key">Requirements</span>
            <span class="info-val">
              <span v-for="(r, i) in dep.entry.requirements" :key="i" class="req-chip">
                {{ requirementLabel(r) }}
              </span>
            </span>
          </div>
        </div>
        <div v-if="!dep.entry?.artifacts?.length" class="empty">
          <i class="pi pi-box"/>No artifacts declared
        </div>
        <div v-else class="artifact-list">
          <div v-for="(a, i) in dep.entry.artifacts" :key="i" class="artifact-block">
            <div class="artifact-head">
              <span class="artifact-name">{{ a.file_name }}</span>
              <span v-if="a.platform" class="artifact-plat">{{ a.platform.os }}/{{ a.platform.arch }}</span>
            </div>
            <div class="info-rows">
              <div class="info-row">
                <span class="info-key">Download URL</span>
                <a :href="a.url" target="_blank" class="info-val link">{{ a.url }}</a>
              </div>
              <div class="info-row" v-if="a.checksum">
                <span class="info-key">{{ a.checksum.algorithm }}</span>
                <code class="info-val mono">{{ a.checksum.value }}</code>
              </div>
              <div class="info-row" v-if="a.component_root">
                <span class="info-key">Component root</span>
                <code class="info-val mono">{{ a.component_root }}</code>
              </div>
              <div class="info-row" v-if="a.steps?.length">
                <span class="info-key">Steps</span>
                <code class="info-val mono">{{ JSON.stringify(a.steps) }}</code>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Right: indexed files -->
      <div class="card">
        <div class="card-title">Indexed Files</div>
        <div v-if="loadingFiles" class="empty"><i class="pi pi-spin pi-spinner"/>Loading...</div>
        <div v-else-if="filesError" class="empty">
          <i class="pi pi-exclamation-triangle"/>{{ filesError }}
          <div><button class="retry-btn" @click="loadFiles($route.params.id)">Retry</button></div>
        </div>
        <div v-else-if="!files.length" class="empty">
          <i class="pi pi-folder-open"/>No indexed files yet
        </div>
        <div v-else>
          <div class="files-search-wrap">
            <i class="pi pi-search" style="position:absolute;left:.75rem;top:50%;transform:translateY(-50%);font-size:.8rem;color:var(--text-faint)"/>
            <input v-model="fileSearch" placeholder="Filter files..." class="files-search" />
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
            <code class="rev-hash">{{ rev.hash?.slice(0,16) }}...</code>
            <span class="rev-size">{{ formatSize(rev.size_bytes) }}</span>
            <span v-if="rev.platform" class="rev-size">{{ rev.platform }}</span>
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
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useAuthStore } from '../stores/auth.js'
import api from '../api/client.js'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'

const route = useRoute()
const router = useRouter()
const toast = useToast()
const auth = useAuthStore()
const dep          = ref(null)
const loading      = ref(true)
const deleting     = ref(false)
const loadingFiles = ref(false)
const files        = ref([])
const fileSearch   = ref('')
const filesError   = ref('')
const fileDialog   = ref({ visible: false, name: '', revisions: [] })

function requirementLabel(r) {
  if (r.name) return `name: ${r.name}`
  if (r.slot) return `slot: ${r.slot}`
  if (r.id) return `id: ${r.id}`
  return '?'
}

const platforms = computed(() => {
  const seen = (dep.value?.entry?.artifacts || [])
    .filter(a => a.platform)
    .map(a => `${a.platform.os}/${a.platform.arch}`)
  return [...new Set(seen)]
})

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
  filesError.value = ''
  try {
    const { data } = await api.get(`/deps/${depId}/files`)
    // Only ever hand the list an array: anything else (an HTML error page,
    // an object) would be iterated as characters or keys by v-for.
    files.value = Array.isArray(data?.items) ? data.items
      : Array.isArray(data) ? data
      : []
  } catch (e) {
    // Surface the failure: silently showing an empty list made a rate-limit
    // rejection look identical to a dependency with no indexed files.
    files.value = []
    filesError.value = e.response?.status === 429
      ? 'Rate limited - wait a moment and retry'
      : e.response?.data?.error || 'Could not load indexed files'
  } finally {
    loadingFiles.value = false
  }
}

async function deleteReference() {
  const name = dep.value?.name || 'this entry'
  if (!confirm(`Delete the catalog reference for ${name}? Indexed bucket files will be kept.`)) return

  deleting.value = true
  try {
    await api.delete(`/admin/deps/${route.params.id}`)
    toast.add({ severity: 'success', summary: 'Reference deleted', life: 2000 })
    await router.push('/deps')
  } finally {
    deleting.value = false
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
.dep-hero { display: flex; align-items: flex-start; gap: 1.25rem; margin-bottom: 2rem; flex-wrap: wrap; }
.dep-hero-info { min-width: 0; flex: 1; }
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
.detail-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(340px, 100%), 1fr)); gap: 1.25rem; }

/* Info rows */
.info-rows { display: flex; flex-direction: column; gap: .75rem; }
.info-row { display: flex; flex-direction: column; gap: .2rem; }
.info-key { font-size: .72rem; font-weight: 600; text-transform: uppercase; letter-spacing: .06em; color: var(--text-faint); }
.info-val { font-size: .875rem; color: var(--text); }
.info-val.link { color: var(--primary); word-break: break-all; }
.info-val.mono { font-family: 'JetBrains Mono', monospace; font-size: .8rem; background: var(--surface2); padding: .2rem .5rem; border-radius: 4px; word-break: break-all; }

/* Artifacts */
.artifact-list { display: flex; flex-direction: column; gap: .75rem; margin-top: .875rem; }
.artifact-block { border: 1px solid var(--border); border-radius: var(--radius-sm); padding: .75rem; }
.artifact-head { display: flex; align-items: center; justify-content: space-between; gap: .5rem; margin-bottom: .5rem; }
.artifact-name { font-size: .875rem; font-weight: 600; word-break: break-all; }
.req-chip {
  display: inline-block; background: var(--surface2); border-radius: 999px;
  padding: .1rem .55rem; font-size: .72rem; margin-right: .35rem;
}
.artifact-plat {
  background: var(--surface2); border-radius: 999px; padding: .15rem .6rem;
  font-size: .7rem; color: var(--text-muted); flex-shrink: 0;
}

/* Files */
.retry-btn {
  margin-top: .5rem; background: var(--surface2); border: 1px solid var(--border);
  color: var(--text); border-radius: var(--radius-sm); padding: .3rem .75rem;
  font-size: .8125rem; cursor: pointer;
}
.retry-btn:hover { border-color: var(--primary); color: var(--primary); }
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
