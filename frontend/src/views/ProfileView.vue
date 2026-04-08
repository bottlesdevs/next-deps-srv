<template>
  <div class="profile-page">
    <h2>Profile</h2>
    <div class="profile-grid">
      <!-- Avatar card -->
      <div class="card avatar-card">
        <div class="avatar-wrap">
          <img v-if="auth.user?.avatar_url" :src="auth.user.avatar_url+'?t='+ts" class="avatar-lg" />
          <div v-else class="avatar-placeholder-lg">{{ auth.user?.username?.[0]?.toUpperCase() }}</div>
        </div>
        <input ref="fileInput" type="file" accept="image/*" hidden @change="uploadAvatar" />
        <Button icon="pi pi-camera" label="Change avatar" text size="small" @click="fileInput.click()" />
      </div>

      <!-- Edit form -->
      <div class="card form-card">
        <h4>Edit profile</h4>
        <div class="field">
          <label>Username</label>
          <InputText v-model="form.username" class="w-full" />
        </div>
        <div class="field">
          <label>Email</label>
          <InputText v-model="form.email" type="email" class="w-full" />
        </div>
        <div class="field">
          <label>Website</label>
          <InputText v-model="form.website" class="w-full" />
        </div>
        <Button label="Save" @click="saveProfile" :loading="saving" />
      </div>

      <!-- Password -->
      <div class="card form-card">
        <h4>Change password</h4>
        <div class="field">
          <label>New password</label>
          <Password v-model="pw.new" toggleMask class="w-full" />
        </div>
        <div class="field">
          <label>Confirm</label>
          <Password v-model="pw.confirm" toggleMask :feedback="false" class="w-full" />
        </div>
        <Button label="Update password" @click="changePassword" :disabled="!pw.new || pw.new !== pw.confirm" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth.js'
import { useToast } from 'primevue/usetoast'
import api from '../api/client.js'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import Button from 'primevue/button'

const auth = useAuthStore()
const toast = useToast()
const fileInput = ref()
const saving = ref(false)
const ts = ref(Date.now())
const form = reactive({ username: '', email: '', website: '' })
const pw = reactive({ new: '', confirm: '' })

onMounted(() => {
  if (auth.user) {
    form.username = auth.user.username || ''
    form.email = auth.user.email || ''
    form.website = auth.user.website || ''
  }
})

async function saveProfile() {
  saving.value = true
  try {
    await api.put('/auth/me', form)
    await auth.fetchMe()
    toast.add({ severity: 'success', summary: 'Saved', life: 2000 })
  } finally {
    saving.value = false
  }
}

async function changePassword() {
  await api.put('/auth/me', { password: pw.new })
  pw.new = ''
  pw.confirm = ''
  toast.add({ severity: 'success', summary: 'Password updated', life: 2000 })
}

async function uploadAvatar(e) {
  const file = e.target.files[0]
  if (!file) return
  const fd = new FormData()
  fd.append('avatar', file)
  await api.post('/auth/me/avatar', fd)
  await auth.fetchMe()
  ts.value = Date.now()
  toast.add({ severity: 'success', summary: 'Avatar updated', life: 2000 })
}
</script>

<style scoped>
.profile-page { max-width: 900px; }
.profile-grid { display: grid; grid-template-columns: auto 1fr 1fr; gap: 1.5rem; margin-top: 1.5rem; align-items: start; }
.card { background: #fff; border-radius: 12px; padding: 1.5rem; box-shadow: 0 1px 4px rgba(0,0,0,.08); }
.avatar-card { display: flex; flex-direction: column; align-items: center; gap: 1rem; min-width: 160px; }
.avatar-lg { width: 96px; height: 96px; border-radius: 50%; object-fit: cover; }
.avatar-placeholder-lg { width: 96px; height: 96px; border-radius: 50%; background: #3b82f6; display: flex; align-items: center; justify-content: center; font-size: 2rem; font-weight: 700; color: #fff; }
.form-card h4 { margin-bottom: 1rem; font-weight: 600; }
.field { margin-bottom: 1rem; display: flex; flex-direction: column; gap: .25rem; }
.field label { font-size: .875rem; color: #64748b; }
.w-full { width: 100%; }
</style>
