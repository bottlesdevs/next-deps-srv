<template>
  <form @submit.prevent="submit">
    <div class="row">
      <div class="field">
        <label>ID</label>
        <InputText v-model="form.id" required placeholder="e.g. openssl" />
      </div>
      <div class="field">
        <label>Name</label>
        <InputText v-model="form.name" required placeholder="e.g. OpenSSL" />
      </div>
      <div class="field">
        <label>Version</label>
        <InputText v-model="form.version" required placeholder="e.g. 3.0.0" />
      </div>
    </div>

    <div class="row">
      <div class="field">
        <label>Kind type</label>
        <InputText v-model="form.kind.type" placeholder="e.g. library" />
      </div>
      <div class="field">
        <label>Kind flavour</label>
        <InputText v-model="form.kind.flavour" placeholder="e.g. shared" />
      </div>
    </div>

    <div class="artifacts">
      <div class="artifacts-head">
        <label>Artifacts</label>
        <Button type="button" label="Add artifact" size="small" text @click="addArtifact" />
      </div>

      <div v-for="(a, i) in form.artifacts" :key="i" class="artifact">
        <div class="artifact-head">
          <span class="artifact-num">#{{ i + 1 }}</span>
          <Button
            v-if="form.artifacts.length > 1"
            type="button" icon="pi pi-trash" size="small" text severity="danger"
            @click="form.artifacts.splice(i, 1)"
          />
        </div>
        <div class="field">
          <label>Download URL</label>
          <InputText v-model="a.url" required placeholder="https://example.com/file.zip" />
        </div>
        <div class="row">
          <div class="field">
            <label>File name</label>
            <InputText v-model="a.file_name" required placeholder="file.zip" />
          </div>
          <div class="field">
            <label>Size (bytes)</label>
            <InputText v-model="a.size" placeholder="0" />
          </div>
        </div>
        <div class="row">
          <div class="field">
            <label>Checksum algorithm</label>
            <Dropdown
              v-model="a.checksum.algorithm"
              :options="algorithms"
              placeholder="none"
              show-clear
            />
          </div>
          <div class="field">
            <label>Checksum value</label>
            <InputText v-model="a.checksum.value" placeholder="hex digest" />
          </div>
        </div>
        <div class="row">
          <div class="field">
            <label>OS</label>
            <InputText v-model="a.platform.os" placeholder="e.g. windows" />
          </div>
          <div class="field">
            <label>Arch</label>
            <InputText v-model="a.platform.arch" placeholder="e.g. x86_64" />
          </div>
          <div class="field">
            <label>Component root</label>
            <InputText v-model="a.component_root" required placeholder="e.g. openssl" />
          </div>
        </div>
      </div>
    </div>

    <div class="row">
      <div class="field">
        <label>License</label>
        <InputText v-model="form.license" placeholder="e.g. MIT, LGPL" />
      </div>
      <div class="field">
        <label>Category</label>
        <InputText v-model="form.category" placeholder="e.g. crypto" />
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
const algorithms = ['md5', 'sha1', 'sha256', 'sha512']

function blankArtifact() {
  return {
    url: '', file_name: '', size: '',
    checksum: { algorithm: null, value: '' },
    platform: { os: '', arch: '' },
    component_root: '',
  }
}

const form = ref({
  id: '', name: '', version: '',
  kind: { type: '', flavour: '' },
  artifacts: [blankArtifact()],
  license: '', category: '', description: '',
})
const loading = ref(false)
const error = ref('')

function addArtifact() {
  form.value.artifacts.push(blankArtifact())
}

// Build the catalog item the server expects: optional objects are omitted
// entirely rather than sent as empty strings.
function payload() {
  const f = form.value
  const body = {
    id: f.id.trim(),
    name: f.name.trim(),
    version: f.version.trim(),
    artifacts: f.artifacts.map(a => {
      const art = {
        url: a.url.trim(),
        file_name: a.file_name.trim(),
        size: Number(a.size) || 0,
        component_root: a.component_root.trim(),
      }
      if (a.checksum.algorithm && a.checksum.value.trim()) {
        art.checksum = { algorithm: a.checksum.algorithm, value: a.checksum.value.trim() }
      }
      if (a.platform.os.trim() && a.platform.arch.trim()) {
        art.platform = { os: a.platform.os.trim(), arch: a.platform.arch.trim() }
      }
      return art
    }),
    license: f.license,
    category: f.category,
    description: f.description,
  }
  if (f.kind.type.trim() && f.kind.flavour.trim()) {
    body.kind = { type: f.kind.type.trim(), flavour: f.kind.flavour.trim() }
  }
  return body
}

async function submit() {
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
.actions { display: flex; justify-content: flex-end; gap: .75rem; margin-top: 1rem; }
.artifacts { margin-bottom: 1rem; }
.artifacts-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: .5rem; }
.artifact { border: 1px solid var(--border, #e5e7eb); border-radius: 8px; padding: .875rem; margin-bottom: .75rem; }
.artifact-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: .5rem; }
.artifact-num { font-size: .75rem; font-weight: 600; color: var(--text-faint, #9ca3af); }
</style>
