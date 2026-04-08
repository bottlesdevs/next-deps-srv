<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">Dependencies</h1>
        <p class="page-subtitle">Browse and install Windows runtime dependencies</p>
      </div>
      <Button v-if="auth.isContributor" icon="pi pi-plus" label="Submit" @click="showSubmit = true" />
    </div>

    <!-- Filters row -->
    <div class="filters">
      <div class="search-wrap">
        <i class="pi pi-search search-icon"/>
        <InputText v-model="search" placeholder="Search dependencies…" class="search-input" />
      </div>
      <div class="filter-pills">
        <button
          v-for="f in filters"
          :key="f.value"
          :class="['pill', activeFilter === f.value && 'pill-active']"
          @click="activeFilter = f.value"
        >{{ f.label }}</button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading-grid">
      <div v-for="i in 6" :key="i" class="card-skeleton"/>
    </div>

    <!-- Empty -->
    <div v-else-if="!filtered.length" class="empty">
      <i class="pi pi-box"/>
      <p>No dependencies found</p>
    </div>

    <!-- Card grid -->
    <div v-else class="deps-grid">
      <RouterLink
        v-for="dep in filtered"
        :key="dep.id"
        :to="`/deps/${dep.id}`"
        class="dep-card"
      >
        <div class="dep-card-header">
          <div class="dep-icon">{{ iconLetter(dep.name) }}</div>
          <span :class="['badge', badgeClass(dep.status)]">{{ dep.status }}</span>
        </div>
        <div class="dep-card-body">
          <div class="dep-card-name">{{ dep.name }}</div>
          <div class="dep-card-ver">{{ dep.manifest?.version || dep.version || '' }}</div>
          <div v-if="dep.manifest?.description" class="dep-card-desc">{{ dep.manifest.description }}</div>
        </div>
        <div class="dep-card-footer">
          <span v-if="dep.manifest?.license" class="dep-meta"><i class="pi pi-file"/>{{ dep.manifest.license }}</span>
          <span v-if="dep.manifest?.arch" class="dep-meta"><i class="pi pi-desktop"/>{{ dep.manifest.arch }}</span>
          <i class="pi pi-arrow-right dep-arrow"/>
        </div>
      </RouterLink>
    </div>

    <!-- Submit dialog -->
    <Dialog v-model:visible="showSubmit" header="Submit Dependency" modal style="width:560px">
      <SubmitDepForm @submitted="onSubmitted" @cancel="showSubmit = false" />
    </Dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth.js'
import { useToast } from 'primevue/usetoast'
import api from '../api/client.js'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Dialog from 'primevue/dialog'
import SubmitDepForm from '../components/SubmitDepForm.vue'

const auth = useAuthStore()
const toast = useToast()

const deps        = ref([])
const loading     = ref(false)
const search      = ref('')
const showSubmit  = ref(false)
const activeFilter = ref('all')

const filters = [
  { label: 'All',     value: 'all' },
  { label: 'Ready',   value: 'built' },
  { label: 'Review',  value: 'pending_review' },
  { label: 'Building',value: 'building' },
]

const filtered = computed(() => {
  let list = deps.value
  if (activeFilter.value !== 'all') list = list.filter(d => d.status === activeFilter.value)
  if (search.value) list = list.filter(d => d.name.toLowerCase().includes(search.value.toLowerCase()))
  return list
})

function badgeClass(status) {
  return {
    built: 'badge-success', approved: 'badge-success',
    pending_review: 'badge-warning', building: 'badge-accent',
    rejected: 'badge-danger',
  }[status] || 'badge-default'
}

function iconLetter(name) {
  return (name || '?')[0].toUpperCase()
}

async function load() {
  loading.value = true
  try {
    const { data } = await api.get('/deps?limit=200')
    deps.value = data.items || []
  } finally {
    loading.value = false
  }
}

function onSubmitted() {
  showSubmit.value = false
  toast.add({ severity: 'success', summary: 'Submitted', detail: 'Dependency submitted for review', life: 3000 })
  load()
}

onMounted(load)
</script>

<style scoped>
/* Filters */
.filters { display: flex; align-items: center; gap: 1rem; margin-bottom: 1.5rem; flex-wrap: wrap; }
.search-wrap { position: relative; }
.search-icon { position: absolute; left: .75rem; top: 50%; transform: translateY(-50%); color: var(--text-faint); font-size: .875rem; pointer-events: none; }
.search-input { padding-left: 2.25rem !important; width: 260px; }
.filter-pills { display: flex; gap: .375rem; }
.pill {
  padding: .3rem .875rem; border-radius: 999px;
  border: 1px solid var(--border2); background: none;
  color: var(--text-muted); font-size: .8125rem; cursor: pointer;
  transition: all var(--transition);
}
.pill:hover { border-color: var(--primary); color: var(--primary); }
.pill-active { background: var(--primary-bg); border-color: var(--primary); color: var(--primary); font-weight: 500; }

/* Grid */
.deps-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 1rem; }

/* Dep card */
.dep-card {
  display: flex; flex-direction: column;
  background: var(--surface); border: 1px solid var(--border);
  border-radius: var(--radius); padding: 1.25rem;
  text-decoration: none; color: var(--text);
  transition: border-color var(--transition), transform var(--transition), box-shadow var(--transition);
  cursor: pointer;
}
.dep-card:hover { border-color: var(--primary); transform: translateY(-2px); box-shadow: var(--shadow-lg); text-decoration: none; }

.dep-card-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: .875rem; }
.dep-icon {
  width: 36px; height: 36px; border-radius: 8px;
  background: linear-gradient(135deg, var(--primary-bg), rgba(88,166,255,.3));
  display: flex; align-items: center; justify-content: center;
  font-size: .9375rem; font-weight: 700; color: var(--primary);
}
.dep-card-body { flex: 1; }
.dep-card-name { font-size: .9375rem; font-weight: 600; color: var(--text); margin-bottom: .125rem; }
.dep-card-ver  { font-size: .75rem; color: var(--text-muted); margin-bottom: .5rem; }
.dep-card-desc { font-size: .8125rem; color: var(--text-muted); line-height: 1.5; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }

.dep-card-footer {
  display: flex; align-items: center; gap: .75rem;
  margin-top: 1rem; padding-top: .875rem;
  border-top: 1px solid var(--border);
}
.dep-meta { display: flex; align-items: center; gap: .3rem; font-size: .75rem; color: var(--text-faint); }
.dep-meta i { font-size: .7rem; }
.dep-arrow { margin-left: auto; color: var(--text-faint); font-size: .875rem; transition: transform var(--transition); }
.dep-card:hover .dep-arrow { transform: translateX(3px); color: var(--primary); }

/* Loading skeleton */
.loading-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 1rem; }
.card-skeleton {
  height: 160px; border-radius: var(--radius);
  background: linear-gradient(90deg, var(--surface) 25%, var(--surface2) 50%, var(--surface) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.4s infinite;
}
@keyframes shimmer { 0% { background-position: 200% 0 } 100% { background-position: -200% 0 } }
</style>

