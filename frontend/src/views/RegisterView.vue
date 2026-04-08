<template>
  <div class="register-page">
    <div class="register-card">
      <h1>Create Account</h1>
      <p class="subtitle">Join the Bottles Deps community</p>
      <form @submit.prevent="submit">
        <div class="field">
          <label>Username</label>
          <InputText v-model="form.username" autofocus required />
        </div>
        <div class="field">
          <label>Email</label>
          <InputText v-model="form.email" type="email" required />
        </div>
        <div class="field">
          <label>Password</label>
          <Password v-model="form.password" required />
        </div>
        <div class="field">
          <label>Confirm Password</label>
          <Password v-model="form.confirm" :feedback="false" required />
        </div>
        <p v-if="error" class="error">{{ error }}</p>
        <Button type="submit" label="Register" :loading="loading" class="w-full" />
        <p class="login-link">Already have an account? <RouterLink to="/login">Sign in</RouterLink></p>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api/client.js'
import { useAuthStore } from '../stores/auth.js'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import Button from 'primevue/button'

const router = useRouter()
const auth = useAuthStore()
const form = ref({ username: '', email: '', password: '', confirm: '' })
const loading = ref(false)
const error = ref('')

async function submit() {
  if (form.value.password !== form.value.confirm) {
    error.value = 'Passwords do not match'
    return
  }
  loading.value = true
  error.value = ''
  try {
    const { data } = await api.post('/auth/register', {
      username: form.value.username,
      email: form.value.email,
      password: form.value.password,
    })
    localStorage.setItem('token', data.token)
    await auth.fetchMe()
    router.push('/')
  } catch (e) {
    error.value = e.response?.data?.error || 'Registration failed'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.register-page {
  min-height: 100vh; display: flex; align-items: center; justify-content: center;
  background: linear-gradient(135deg, #1e40af 0%, #7c3aed 100%);
}
.register-card {
  background: #fff; border-radius: 16px; padding: 2.5rem;
  width: 100%; max-width: 420px; box-shadow: 0 20px 60px rgba(0,0,0,.2);
}
h1 { font-size: 1.75rem; font-weight: 700; margin-bottom: .25rem; }
.subtitle { color: #64748b; margin-bottom: 1.5rem; }
.field { margin-bottom: 1.1rem; display: flex; flex-direction: column; gap: .4rem; }
.w-full { width: 100%; margin-top: .5rem; }
.error { color: #dc2626; margin-bottom: .75rem; font-size: .875rem; }
.login-link { margin-top: 1rem; text-align: center; font-size: .875rem; color: #64748b; }
.login-link a { color: #6366f1; }
</style>
