<template>
  <div class="max-w-2xl mx-auto">
    <h1 class="text-3xl font-bold mb-6">Submit Dependency</h1>

    <div v-if="!isContributor" class="bg-yellow-50 border border-yellow-200 rounded p-4 text-yellow-800">
      You need at least contributor role to submit dependencies.
    </div>

    <form v-else @submit.prevent="submit" class="space-y-6 bg-white rounded-xl shadow p-6">
      <div>
        <label class="block font-medium mb-1">Name <span class="text-red-500">*</span></label>
        <input v-model="form.name" type="text" required class="w-full border rounded px-3 py-2" placeholder="e.g. libpng" />
      </div>
      <div>
        <label class="block font-medium mb-1">Category</label>
        <input v-model="form.category" type="text" class="w-full border rounded px-3 py-2" placeholder="e.g. graphics" />
      </div>
      <div>
        <label class="block font-medium mb-1">Description</label>
        <textarea v-model="form.description" rows="3" class="w-full border rounded px-3 py-2" placeholder="Describe the dependency"></textarea>
      </div>

      <hr />
      <h2 class="text-lg font-semibold">Manifest</h2>

      <div>
        <label class="block font-medium mb-1">Download URL <span class="text-red-500">*</span></label>
        <input v-model="form.manifest.url" type="url" required class="w-full border rounded px-3 py-2" placeholder="https://..." />
      </div>
      <div>
        <label class="block font-medium mb-1">Expected SHA256 Hash <span class="text-red-500">*</span></label>
        <input v-model="form.manifest.expected_hash" type="text" required class="w-full border rounded px-3 py-2" placeholder="sha256 hex string" />
      </div>
      <div>
        <label class="block font-medium mb-1">License</label>
        <input v-model="form.manifest.license" type="text" class="w-full border rounded px-3 py-2" placeholder="e.g. MIT, GPL-2.0" />
      </div>

      <div v-if="error" class="text-red-600 text-sm">{{ error }}</div>
      <div v-if="success" class="text-green-600 text-sm">Dependency submitted for review!</div>

      <button type="submit" :disabled="loading" class="w-full bg-blue-600 hover:bg-blue-700 text-white font-semibold py-2 px-4 rounded disabled:opacity-50">
        {{ loading ? 'Submitting…' : 'Submit for Review' }}
      </button>
    </form>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth.js'
import api from '../api/client.js'

const auth = useAuthStore()
const router = useRouter()
const loading = ref(false)
const error = ref('')
const success = ref(false)

const isContributor = computed(() =>
  ['admin', 'mod', 'contributor'].includes(auth.user?.role)
)

const form = ref({
  name: '',
  category: '',
  description: '',
  manifest: {
    url: '',
    expected_hash: '',
    license: '',
    license_files: [],
  },
})

async function submit() {
  loading.value = true
  error.value = ''
  success.value = false
  try {
    await api.post('/deps', form.value)
    success.value = true
    setTimeout(() => router.push('/deps'), 1500)
  } catch (e) {
    error.value = e.response?.data?.error || 'Submission failed'
  } finally {
    loading.value = false
  }
}
</script>
