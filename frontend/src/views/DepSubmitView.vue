<template>
  <div class="max-w-2xl mx-auto">
    <h1 class="text-3xl font-bold mb-6">Submit Dependency</h1>

    <div v-if="!isContributor" class="bg-yellow-50 border border-yellow-200 rounded p-4 text-yellow-800">
      You need at least contributor role to submit dependencies.
    </div>

    <div v-else class="bg-white rounded-xl shadow p-6">
      <div v-if="success" class="text-green-600 text-sm mb-3">Dependency submitted for review!</div>
      <SubmitDepForm @submitted="onSubmitted" @cancel="router.push('/deps')" />
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth.js'
import SubmitDepForm from '../components/SubmitDepForm.vue'

const auth = useAuthStore()
const router = useRouter()
const success = ref(false)

// The user model carries a roles array; use the store's own role helper.
const isContributor = computed(() => auth.isContributor)

function onSubmitted() {
  success.value = true
  setTimeout(() => router.push('/deps'), 1500)
}
</script>
