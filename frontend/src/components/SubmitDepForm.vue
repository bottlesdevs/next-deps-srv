<template>
  <form @submit.prevent="submit">
    <div class="row">
      <div class="field">
        <label>Kind</label>
        <Dropdown v-model="form.kind" :options="kinds" option-label="label" option-value="value" />
      </div>
      <div class="field" v-if="form.kind === 'component'">
        <label>Slot</label>
        <Dropdown v-model="form.slot" :options="slots" placeholder="Select a slot" />
      </div>
    </div>

    <div class="row">
      <div class="field">
        <label>Name</label>
        <InputText v-model="form.name" required placeholder="e.g. Wine Stable" />
      </div>
      <div class="field">
        <label>Version</label>
        <InputText v-model="form.version" required placeholder="e.g. 11.0" />
      </div>
    </div>

    <div class="repeatable">
      <div class="repeatable-head">
        <label>Artifacts</label>
        <Button type="button" label="Add artifact" size="small" text @click="addArtifact" />
      </div>

      <div v-for="(a, i) in form.artifacts" :key="i" class="block">
        <div class="block-head">
          <span class="block-num">#{{ i + 1 }}</span>
          <Button
            v-if="form.artifacts.length > 1"
            type="button" icon="pi pi-trash" size="small" text severity="danger"
            @click="form.artifacts.splice(i, 1)"
          />
        </div>
        <div class="field">
          <label>Download URL</label>
          <InputText v-model="a.url" required placeholder="https://example.com/file.tar.xz" />
        </div>
        <div class="field">
          <label>File name</label>
          <InputText v-model="a.file_name" required placeholder="file.tar.xz" />
        </div>
        <div class="row">
          <div class="field">
            <label>Checksum algorithm</label>
            <Dropdown v-model="a.checksum.algorithm" :options="algorithms" />
          </div>
          <div class="field">
            <label>Checksum value</label>
            <InputText v-model="a.checksum.value" required placeholder="lowercase hex digest" />
          </div>
        </div>
        <div class="row">
          <div class="field">
            <label>OS</label>
            <Dropdown v-model="a.platform.os" :options="operatingSystems" placeholder="any" show-clear />
          </div>
          <div class="field">
            <label>Arch</label>
            <Dropdown v-model="a.platform.arch" :options="architectures" placeholder="any" show-clear />
          </div>
        </div>
        <div class="field">
          <label>Steps <span class="hint">optional, JSON array</span></label>
          <Textarea v-model="a.steps" rows="2" placeholder="[]" :class="{ invalid: stepsError(a) }" />
          <small v-if="stepsError(a)" class="err">{{ stepsError(a) }}</small>
        </div>
      </div>
    </div>

    <div class="repeatable">
      <div class="repeatable-head">
        <label>Requirements <span class="hint">optional</span></label>
        <Button type="button" label="Add requirement" size="small" text @click="addRequirement" />
      </div>
      <div v-for="(r, i) in form.requirements" :key="i" class="req-row">
        <Dropdown v-model="r.type" :options="reqTypes" option-label="label" option-value="value" />
        <Dropdown v-if="r.type === 'slot'" v-model="r.value" :options="slots" placeholder="Select a slot" />
        <InputText v-else v-model="r.value" :placeholder="r.type === 'id' ? 'entry uuid' : 'addon name'" />
        <Button type="button" icon="pi pi-trash" size="small" text severity="danger"
                @click="form.requirements.splice(i, 1)" />
      </div>
    </div>

    <div class="row">
      <div class="field">
        <label>License</label>
        <InputText v-model="form.license" placeholder="e.g. MIT, LGPL" />
      </div>
      <div class="field">
        <label>Category</label>
        <InputText v-model="form.category" placeholder="e.g. graphics" />
      </div>
    </div>
    <div class="field">
      <label>Description</label>
      <Textarea v-model="form.description" rows="3" />
    </div>

    <p v-if="error" class="error">{{ error }}</p>
    <div class="actions">
      <Button type="button" label="Cancel" severity="secondary" @click="$emit('cancel')" />
      <Button type="submit" label="Submit" :loading="loading" />
    </div>
  </form>
</template>

<script setup>
import { ref } from 'vue'
import api from '../api/client.js'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import Dropdown from 'primevue/dropdown'
import Button from 'primevue/button'

const emit = defineEmits(['submitted', 'cancel'])

const kinds = [
  { label: 'Dependency', value: 'dependency' },
  { label: 'Component', value: 'component' },
]
const slots = ['winebridge', 'runner', 'umu', 'dxvk', 'vkd3d', 'nvapi', 'latency-flex']
const algorithms = ['sha256', 'sha512']
const operatingSystems = ['linux', 'mac-os', 'windows']
const architectures = ['x86', 'x86_64', 'aarch64']
const reqTypes = [
  { label: 'Name', value: 'name' },
  { label: 'Slot', value: 'slot' },
  { label: 'Entry ID', value: 'id' },
]

function blankArtifact() {
  return {
    url: '', file_name: '',
    checksum: { algorithm: 'sha256', value: '' },
    platform: { os: null, arch: null },
    steps: '',
  }
}

const form = ref({
  kind: 'dependency', slot: null,
  name: '', version: '',
  artifacts: [blankArtifact()],
  requirements: [],
  license: '', category: '', description: '',
})
const loading = ref(false)
const error = ref('')

function addArtifact() { form.value.artifacts.push(blankArtifact()) }
function addRequirement() { form.value.requirements.push({ type: 'name', value: '' }) }

// Steps are an opaque recipe passed through verbatim, so they are entered as
// raw JSON and only checked for being a well-formed array.
function stepsError(a) {
  const raw = (a.steps || '').trim()
  if (!raw) return ''
  try {
    return Array.isArray(JSON.parse(raw)) ? '' : 'must be a JSON array'
  } catch {
    return 'invalid JSON'
  }
}

function payload() {
  const f = form.value
  const body = {
    kind: f.kind,
    name: f.name.trim(),
    version: f.version.trim(),
    artifacts: f.artifacts.map(a => {
      const art = {
        url: a.url.trim(),
        file_name: a.file_name.trim(),
        checksum: { algorithm: a.checksum.algorithm, value: a.checksum.value.trim() },
      }
      // platform is nullable: send it only when fully specified.
      if (a.platform.os && a.platform.arch) {
        art.platform = { os: a.platform.os, arch: a.platform.arch }
      }
      const steps = (a.steps || '').trim()
      if (steps) art.steps = JSON.parse(steps)
      return art
    }),
    license: f.license,
    category: f.category,
    description: f.description,
  }
  if (f.kind === 'component') body.slot = f.slot
  const reqs = f.requirements
    .filter(r => r.value)
    .map(r => ({ [r.type]: r.value }))
  if (reqs.length) body.requirements = reqs
  return body
}

async function submit() {
  const bad = form.value.artifacts.find(a => stepsError(a))
  if (bad) {
    error.value = 'Fix the steps JSON before submitting'
    return
  }
  loading.value = true
  error.value = ''
  try {
    await api.post('/deps', payload())
    emit('submitted')
  } catch (e) {
    error.value = e.response?.data?.error || 'Submission failed'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.field { margin-bottom: 1rem; display: flex; flex-direction: column; gap: .35rem; flex: 1; min-width: 0; }
.row { display: flex; gap: .75rem; }
.error { color: #dc2626; margin-bottom: .75rem; font-size: .875rem; }
.err { color: #dc2626; font-size: .75rem; }
.hint { color: var(--text-faint, #9ca3af); font-weight: 400; }
.actions { display: flex; justify-content: flex-end; gap: .75rem; margin-top: 1rem; }
.repeatable { margin-bottom: 1rem; }
.repeatable-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: .5rem; }
.block { border: 1px solid var(--border, #e5e7eb); border-radius: 8px; padding: .875rem; margin-bottom: .75rem; }
.block-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: .5rem; }
.block-num { font-size: .75rem; font-weight: 600; color: var(--text-faint, #9ca3af); }
.req-row { display: flex; gap: .5rem; align-items: center; margin-bottom: .5rem; }
.req-row > :nth-child(2) { flex: 1; }
:deep(.invalid) { border-color: #dc2626 !important; }
</style>
