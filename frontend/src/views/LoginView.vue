<template>
  <div class="login-page">
    <!-- Background grid pattern -->
    <div class="bg-pattern"/>

    <div class="login-card">
      <div class="login-brand">
        <div class="login-icon"><i class="pi pi-box"/></div>
        <div>
          <div class="login-title">Bottles Deps</div>
          <div class="login-sub">Dependency Registry</div>
        </div>
      </div>

      <div class="divider"/>

      <form @submit.prevent="submit" class="login-form">
        <div class="field">
          <label>Username</label>
          <div class="input-wrap">
            <i class="pi pi-user input-icon"/>
            <InputText v-model="form.username" autofocus required placeholder="admin" />
          </div>
        </div>
        <div class="field">
          <label>Password</label>
          <div class="input-wrap">
            <i class="pi pi-lock input-icon"/>
            <Password v-model="form.password" :feedback="false" required placeholder="••••••••" input-class="pw-input" />
          </div>
        </div>
        <Button type="submit" label="Sign in" icon="pi pi-arrow-right" icon-pos="right" :loading="loading" class="submit-btn" />
        <div v-if="error" class="error-msg">
          <i class="pi pi-exclamation-circle"/> {{ error }}
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth.js'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import Button from 'primevue/button'

const auth = useAuthStore()
const router = useRouter()

const form    = ref({ username: '', password: '' })
const loading = ref(false)
const error   = ref('')

async function submit() {
  loading.value = true
  error.value = ''
  try {
    await auth.login(form.value.username, form.value.password)
    router.push('/')
  } catch (e) {
    error.value = e.response?.data?.error || 'Invalid username or password'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg);
  position: relative;
  overflow: hidden;
}

/* subtle dot grid */
.bg-pattern {
  position: absolute; inset: 0; pointer-events: none;
  background-image: radial-gradient(circle, rgba(88,166,255,.08) 1px, transparent 1px);
  background-size: 32px 32px;
}

.login-card {
  position: relative; z-index: 1;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 2.5rem;
  width: 100%; max-width: 400px;
  box-shadow: var(--shadow-lg);
}

.login-brand { display: flex; align-items: center; gap: 1rem; margin-bottom: 1.5rem; }
.login-icon {
  width: 44px; height: 44px; border-radius: var(--radius);
  background: linear-gradient(135deg, #1f6feb, #58a6ff);
  display: flex; align-items: center; justify-content: center;
  font-size: 1.25rem; color: #fff; flex-shrink: 0;
}
.login-title { font-size: 1.125rem; font-weight: 700; color: var(--text); }
.login-sub   { font-size: .8rem; color: var(--text-muted); }

.login-form { display: flex; flex-direction: column; gap: 1.125rem; }

.field { display: flex; flex-direction: column; gap: .4rem; }
.field label { font-size: .8125rem; font-weight: 500; color: var(--text-muted); }

.input-wrap { position: relative; }
.input-icon {
  position: absolute; left: .875rem; top: 50%; transform: translateY(-50%);
  color: var(--text-faint); font-size: .875rem; pointer-events: none; z-index: 1;
}
.input-wrap :deep(input),
.input-wrap :deep(.p-inputtext) {
  padding-left: 2.5rem !important; width: 100% !important;
}
.input-wrap :deep(.p-password) { width: 100%; }
.input-wrap :deep(.p-password-input) { width: 100%; padding-left: 2.5rem !important; }

.submit-btn { width: 100%; margin-top: .25rem; }

.error-msg {
  display: flex; align-items: center; gap: .5rem;
  background: rgba(248,81,73,.1); border: 1px solid rgba(248,81,73,.3);
  color: var(--danger); font-size: .875rem;
  padding: .625rem .875rem; border-radius: var(--radius-sm);
}
</style>

