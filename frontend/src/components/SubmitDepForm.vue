<template>
  <form @submit.prevent="submit">
    <div class="field">
      <label>Name</label>
      <InputText v-model="form.name" required placeholder="e.g. openssl" />
    </div>
    <div class="field">
      <label>Download URL</label>
      <InputText v-model="form.url" required placeholder="https://example.com/file.zip" />
    </div>
    <div class="field">
      <label>Expected SHA256</label>
      <InputText v-model="form.expected_hash" required placeholder="sha256 hex" />
    </div>
    <div class="field">
      <label>License</label>
      <InputText v-model="form.license" placeholder="e.g. MIT, LGPL" />
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
import Button from 'primevue/button'

const emit = defineEmits(['submitted', 'cancel'])
const form = ref({ name: '', url: '', expected_hash: '', license: '', description: '' })
const loading = ref(false)
const error = ref('')

async function submit() {
  loading.value = true
  error.value = ''
  try {
    await api.post('/deps', form.value)
    emit('submitted')
  } catch (e) {
    error.value = e.response?.data?.error || 'Submission failed'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.field { margin-bottom: 1rem; display: flex; flex-direction: column; gap: .35rem; }
.error { color: #dc2626; margin-bottom: .75rem; font-size: .875rem; }
.actions { display: flex; justify-content: flex-end; gap: .75rem; margin-top: 1rem; }
</style>
