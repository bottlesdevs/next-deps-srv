<template>
  <div>
    <h2>Review Queue</h2>
    <p class="subtitle">Pending dependency submissions awaiting review</p>

    <div v-if="loading" class="center"><ProgressSpinner /></div>
    <div v-else-if="pending.length === 0" class="empty">
      <i class="pi pi-check-circle" style="font-size:2rem;color:#16a34a"></i>
      <p>All caught up! No pending reviews.</p>
    </div>
    <div v-else>
      <div v-for="dep in pending" :key="dep.id" class="dep-card">
        <div class="dep-header">
          <h3>{{ dep.entry?.name }} <span class="ver">{{ dep.entry?.version }}</span></h3>
          <Tag :value="dep.status" severity="warning" />
        </div>
        <div class="dep-info">
          <p><b>Entry ID:</b> <code>{{ dep.entry?.id }}</code></p>
          <p><b>Kind:</b> {{ dep.kind }}<span v-if="dep.entry?.slot"> · slot: {{ dep.entry.slot }}</span></p>
          <p v-if="dep.entry?.requirements?.length"><b>Requires:</b>
            <span v-for="(r, i) in dep.entry.requirements" :key="i">
              {{ r.name || r.slot || r.id }}<span v-if="i < dep.entry.requirements.length - 1">, </span>
            </span>
          </p>
          <p><b>License:</b> {{ dep.license || '-' }}</p>
          <p><b>Submitted by:</b> {{ dep.submitted_by }}</p>
          <div class="artifacts">
            <b>Artifacts ({{ dep.entry?.artifacts?.length || 0 }}):</b>
            <div v-for="(a, i) in dep.entry?.artifacts || []" :key="i" class="artifact">
              <a :href="a.url" target="_blank">{{ a.file_name }}</a>
              <span v-if="a.platform" class="tagline">{{ a.platform.os }}/{{ a.platform.arch }}</span>
              <span v-else class="tagline">any platform</span>
              <code class="sum">{{ a.checksum.algorithm }}:{{ a.checksum.value }}</code>
              <span v-if="a.steps?.length" class="tagline">{{ a.steps.length }} step(s)</span>
            </div>
          </div>
        </div>
        <div class="dep-actions">
          <InputText v-model="notes[dep.id]" placeholder="Review note (optional)" />
          <Button label="Approve" icon="pi pi-check" severity="success" :loading="loading" @click="approve(dep)" />
          <Button label="Reject" icon="pi pi-times" severity="danger" :loading="loading" @click="reject(dep)" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useToast } from 'primevue/usetoast'
import { useDepsStore } from '../stores/deps.js'
import Tag from 'primevue/tag'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import ProgressSpinner from 'primevue/progressspinner'

const store = useDepsStore()
const toast = useToast()
const pending = ref([])
const loading = ref(false)
const notes = ref({})

async function load() {
  loading.value = true
  try {
    pending.value = await store.fetchPending()
  } finally {
    loading.value = false
  }
}

async function approve(dep) {
  try {
    await store.approveDep(dep.id)
    toast.add({ severity: 'success', summary: 'Approved', detail: dep.entry.name + ' approved', life: 3000 })
    await load()
  } catch (e) {
    toast.add({ severity: 'error', summary: 'Error', detail: e.response?.data?.error || 'Failed', life: 3000 })
  }
}

async function reject(dep) {
  try {
    await store.rejectDep(dep.id, notes.value[dep.id] || '')
    toast.add({ severity: 'info', summary: 'Rejected', detail: dep.entry.name + ' rejected', life: 3000 })
    await load()
  } catch (e) {
    toast.add({ severity: 'error', summary: 'Error', detail: e.response?.data?.error || 'Failed', life: 3000 })
  }
}

onMounted(load)
</script>

<style scoped>
h2 { margin-bottom: .25rem; }
.subtitle { color: #64748b; font-size: .875rem; margin-bottom: 1.5rem; }
.center { display: flex; justify-content: center; padding: 2rem; }
.empty { text-align: center; padding: 3rem; color: #64748b; }
.empty i { display: block; margin-bottom: .75rem; }
.dep-card { background: #fff; border-radius: 12px; padding: 1.5rem; margin-bottom: 1.25rem; box-shadow: 0 1px 4px rgba(0,0,0,.08); }
.dep-header { display: flex; align-items: center; gap: .75rem; margin-bottom: 1rem; }
.dep-info p { font-size: .875rem; margin-bottom: .35rem; }
code { font-size: .75rem; background: #f1f5f9; padding: 2px 6px; border-radius: 4px; }
.dep-actions { display: flex; gap: .75rem; align-items: center; margin-top: 1rem; padding-top: 1rem; border-top: 1px solid #f1f5f9; }
.ver { font-weight: 400; color: var(--text-muted, #6b7280); font-size: .875rem; }
.artifacts { margin-top: .5rem; }
.artifact { display: flex; flex-wrap: wrap; align-items: center; gap: .5rem; padding: .35rem 0; font-size: .8125rem; }
.artifact .sum { font-family: monospace; font-size: .72rem; word-break: break-all; }
.artifact .tagline { color: var(--text-muted, #6b7280); font-size: .72rem; }
.artifact .warn { color: #d97706; font-size: .72rem; }
</style>
