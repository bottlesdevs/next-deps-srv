<template>
  <div>
    <div class="page-header">
      <h2>Users</h2>
      <Button icon="pi pi-plus" label="New user" @click="showNew = true" />
    </div>

    <DataTable :value="users" :loading="loading" responsiveLayout="scroll">
      <Column field="username" header="Username" sortable />
      <Column field="email" header="Email" />
      <Column header="Roles">
        <template #body="{ data }">
          <Tag v-for="r in data.roles" :key="r" :value="r" class="mr-1" />
        </template>
      </Column>
      <Column header="Status">
        <template #body="{ data }">
          <Tag :value="data.enabled ? 'active' : 'disabled'" :severity="data.enabled ? 'success' : 'danger'" />
        </template>
      </Column>
      <Column header="">
        <template #body="{ data }">
          <Button icon="pi pi-pencil" text size="small" @click="editUser(data)" />
          <Button
            :icon="data.enabled ? 'pi pi-ban' : 'pi pi-check'"
            text size="small"
            :severity="data.enabled ? 'warning' : 'success'"
            @click="toggleUser(data)"
          />
          <Button icon="pi pi-trash" text size="small" severity="danger" @click="deleteUser(data)" />
        </template>
      </Column>
    </DataTable>

    <Dialog v-model:visible="showEdit" header="Edit user" modal style="width: 460px">
      <div v-if="editing" class="form">
        <div class="field">
          <label>Roles</label>
          <MultiSelect v-model="editing.roles" :options="allRoles" class="w-full" placeholder="Select roles" />
        </div>
        <div class="field">
          <label>Reset password</label>
          <Password v-model="editing.new_password" toggleMask :feedback="false" class="w-full" />
        </div>
      </div>
      <template #footer>
        <Button label="Cancel" text @click="showEdit = false" />
        <Button label="Save" @click="saveEdit" />
      </template>
    </Dialog>

    <Dialog v-model:visible="showNew" header="Create user" modal style="width: 460px">
      <div class="form">
        <div class="field"><label>Username</label><InputText v-model="newUser.username" class="w-full" /></div>
        <div class="field"><label>Email</label><InputText v-model="newUser.email" type="email" class="w-full" /></div>
        <div class="field"><label>Password</label><Password v-model="newUser.password" toggleMask class="w-full" /></div>
        <div class="field">
          <label>Roles</label>
          <MultiSelect v-model="newUser.roles" :options="allRoles" class="w-full" />
        </div>
      </div>
      <template #footer>
        <Button label="Cancel" text @click="showNew = false" />
        <Button label="Create" @click="createUser" />
      </template>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import api from '../api/client.js'
import { useToast } from 'primevue/usetoast'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Tag from 'primevue/tag'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import MultiSelect from 'primevue/multiselect'

const toast = useToast()
const users = ref([])
const loading = ref(false)
const showEdit = ref(false)
const showNew = ref(false)
const editing = ref(null)
const allRoles = ['admin', 'mod', 'contributor', 'viewer']
const newUser = reactive({ username: '', email: '', password: '', roles: ['viewer'] })

async function load() {
  loading.value = true
  try {
    const { data } = await api.get('/admin/users')
    users.value = data.items || []
  } finally {
    loading.value = false
  }
}

function editUser(u) {
  editing.value = { ...u, new_password: '' }
  showEdit.value = true
}

async function saveEdit() {
  const payload = { roles: editing.value.roles }
  if (editing.value.new_password) payload.password = editing.value.new_password
  await api.put(`/admin/users/${editing.value.id}`, payload)
  showEdit.value = false
  toast.add({ severity: 'success', summary: 'Saved', life: 2000 })
  load()
}

async function toggleUser(u) {
  await api.put(`/admin/users/${u.id}`, { enabled: !u.enabled })
  load()
}

async function deleteUser(u) {
  if (!confirm(`Delete user ${u.username}?`)) return
  await api.delete(`/admin/users/${u.id}`)
  load()
}

async function createUser() {
  await api.post('/admin/users', newUser)
  showNew.value = false
  Object.assign(newUser, { username: '', email: '', password: '', roles: ['viewer'] })
  load()
}

onMounted(load)
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem; }
.form .field { display: flex; flex-direction: column; gap: .25rem; margin-bottom: 1rem; }
.form .field label { font-size: .875rem; color: #64748b; }
.w-full { width: 100%; }
.mr-1 { margin-right: .25rem; }
</style>
