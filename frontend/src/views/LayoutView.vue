<template>
  <div class="shell">
    <!-- Sidebar -->
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-icon"><i class="pi pi-box" /></div>
        <div>
          <div class="brand-name">Deps</div>
          <div class="brand-sub">by Bottles</div>
        </div>
      </div>

      <nav class="nav">
        <RouterLink to="/" exact-active-class="nav-active" :class="['nav-item', $route.path === '/' ? 'nav-active' : '']">
          <i class="pi pi-home" /><span>Dashboard</span>
        </RouterLink>
        <RouterLink to="/deps" :class="['nav-item', $route.path.startsWith('/deps') ? 'nav-active' : '']">
          <i class="pi pi-box" /><span>Dependencies</span>
        </RouterLink>
        <RouterLink to="/community" :class="['nav-item', $route.path.startsWith('/community') ? 'nav-active' : '']">
          <i class="pi pi-comments" /><span>Community</span>
        </RouterLink>
        <template v-if="auth.isMod">
          <div class="nav-sep">Moderation</div>
          <RouterLink to="/review" :class="['nav-item', $route.path.startsWith('/review') ? 'nav-active' : '']">
            <i class="pi pi-check-circle" /><span>Review Queue</span>
            <span v-if="pendingCount > 0" class="nav-badge">{{ pendingCount }}</span>
          </RouterLink>
        </template>
        <template v-if="auth.isAdmin">
          <div class="nav-sep">Administration</div>
          <RouterLink to="/admin" :class="['nav-item', $route.path === '/admin' ? 'nav-active' : '']">
            <i class="pi pi-gauge" /><span>Overview</span>
          </RouterLink>
          <RouterLink to="/admin/users" :class="['nav-item', $route.path.startsWith('/admin/users') ? 'nav-active' : '']">
            <i class="pi pi-users" /><span>Users</span>
          </RouterLink>
          <RouterLink to="/admin/jobs" :class="['nav-item', $route.path.startsWith('/admin/jobs') ? 'nav-active' : '']">
            <i class="pi pi-server" /><span>Build Jobs</span>
          </RouterLink>
        </template>
      </nav>

      <div class="sidebar-footer">
        <RouterLink to="/profile" class="user-chip">
          <div class="avatar">
            <img v-if="auth.user?.avatar_url" :src="auth.user.avatar_url" alt="" />
            <span v-else>{{ initials }}</span>
          </div>
          <div class="user-info">
            <div class="user-name">{{ auth.user?.username }}</div>
            <div class="user-role">{{ primaryRole }}</div>
          </div>
        </RouterLink>
        <button class="logout-btn" title="Sign out" @click="auth.logout(); $router.push('/login')">
          <i class="pi pi-sign-out" />
        </button>
      </div>
    </aside>

    <!-- Main -->
    <main class="main">
      <div class="main-inner">
        <RouterView />
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth.js'
import api from '../api/client.js'

const auth = useAuthStore()
const pendingCount = ref(0)

const initials = computed(() => {
  const u = auth.user?.username || ''
  return u.slice(0, 2).toUpperCase()
})

const primaryRole = computed(() => {
  const roles = auth.user?.roles || []
  if (roles.includes('admin')) return 'Administrator'
  if (roles.includes('mod')) return 'Moderator'
  if (roles.includes('contributor')) return 'Contributor'
  return 'Member'
})

onMounted(async () => {
  if (auth.isMod) {
    try {
      const { data } = await api.get('/deps/pending?limit=100')
      pendingCount.value = (data.items || []).length
    } catch {}
  }
})
</script>

<style scoped>
.shell { display: flex; min-height: 100vh; background: var(--bg); }

/* Sidebar */
.sidebar {
  width: 240px; flex-shrink: 0;
  background: var(--surface);
  border-right: 1px solid var(--border);
  display: flex; flex-direction: column;
  padding: 1.25rem .75rem;
  position: sticky; top: 0; height: 100vh;
}

.brand {
  display: flex; align-items: center; gap: .75rem;
  padding: .25rem .5rem 1.5rem;
  border-bottom: 1px solid var(--border);
  margin-bottom: .75rem;
}
.brand-icon {
  width: 36px; height: 36px; border-radius: 8px;
  background: linear-gradient(135deg, #1f6feb, #58a6ff);
  display: flex; align-items: center; justify-content: center;
  font-size: 1rem; flex-shrink: 0;
}
.brand-name { font-weight: 700; font-size: .9375rem; line-height: 1.2; }
.brand-sub  { font-size: .7rem; color: var(--text-muted); }

/* Nav */
.nav { flex: 1; display: flex; flex-direction: column; gap: 2px; }
.nav-sep {
  font-size: .6875rem; font-weight: 600; letter-spacing: .08em;
  text-transform: uppercase; color: var(--text-faint);
  padding: .75rem .5rem .25rem;
}
.nav-item {
  display: flex; align-items: center; gap: .625rem;
  padding: .5rem .75rem; border-radius: var(--radius-sm);
  color: var(--text-muted); font-size: .875rem;
  text-decoration: none; cursor: pointer;
  transition: background var(--transition), color var(--transition);
  position: relative;
}
.nav-item i { font-size: .9375rem; width: 16px; text-align: center; flex-shrink: 0; }
.nav-item:hover { background: var(--surface2); color: var(--text); text-decoration: none; }
.nav-active { background: var(--primary-bg) !important; color: var(--primary) !important; font-weight: 500; }
.nav-badge {
  margin-left: auto;
  background: var(--danger); color: #fff;
  font-size: .65rem; font-weight: 700;
  min-width: 18px; height: 18px; border-radius: 999px;
  display: flex; align-items: center; justify-content: center;
  padding: 0 .35rem;
}

/* Footer */
.sidebar-footer {
  display: flex; align-items: center; gap: .5rem;
  padding: .75rem .25rem 0;
  border-top: 1px solid var(--border);
  margin-top: .5rem;
}
.user-chip {
  display: flex; align-items: center; gap: .625rem;
  flex: 1; text-decoration: none; min-width: 0;
  padding: .375rem .5rem; border-radius: var(--radius-sm);
  transition: background var(--transition);
}
.user-chip:hover { background: var(--surface2); text-decoration: none; }
.avatar {
  width: 30px; height: 30px; border-radius: 50%; flex-shrink: 0;
  background: linear-gradient(135deg, #1f6feb, #bc8cff);
  display: flex; align-items: center; justify-content: center;
  font-size: .7rem; font-weight: 700; color: #fff; overflow: hidden;
}
.avatar img { width: 100%; height: 100%; object-fit: cover; }
.user-info { min-width: 0; }
.user-name { font-size: .8125rem; font-weight: 600; color: var(--text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.user-role { font-size: .7rem; color: var(--text-muted); }
.logout-btn {
  width: 30px; height: 30px; border: none; background: none;
  cursor: pointer; border-radius: var(--radius-sm);
  color: var(--text-muted); display: flex; align-items: center; justify-content: center;
  font-size: .875rem; transition: background var(--transition), color var(--transition);
  flex-shrink: 0;
}
.logout-btn:hover { background: rgba(248,81,73,.15); color: var(--danger); }

/* Main */
.main { flex: 1; overflow: auto; display: flex; flex-direction: column; }
.main-inner { flex: 1; padding: 2rem 2.5rem; max-width: 1200px; width: 100%; }
</style>
