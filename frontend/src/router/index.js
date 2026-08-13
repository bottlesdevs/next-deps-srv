import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth.js'

const routes = [
  { path: '/login', component: () => import('../views/LoginView.vue'), meta: { public: true } },
  { path: '/register', component: () => import('../views/RegisterView.vue'), meta: { public: true } },
  {
    path: '/',
    component: () => import('../views/LayoutView.vue'),
    children: [
      { path: '', component: () => import('../views/DashboardView.vue') },
      { path: 'deps', component: () => import('../views/DepsView.vue') },
      { path: 'deps/submit', component: () => import('../views/DepSubmitView.vue'), meta: { roles: ['admin', 'mod', 'contributor'] } },
      { path: 'deps/:id', component: () => import('../views/DepDetailView.vue') },
      { path: 'community', component: () => import('../views/CommunityView.vue') },
      { path: 'profile', component: () => import('../views/ProfileView.vue') },
      { path: 'admin', component: () => import('../views/AdminView.vue'), meta: { roles: ['admin'] } },
      { path: 'admin/users', component: () => import('../views/AdminUsersView.vue'), meta: { roles: ['admin'] } },
      { path: 'admin/jobs', component: () => import('../views/AdminJobsView.vue'), meta: { roles: ['admin'] } },
      { path: 'review', component: () => import('../views/ReviewView.vue'), meta: { roles: ['admin', 'mod'] } },
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!to.meta.public && !auth.token) {
    return '/login'
  }
  // Wait for /auth/me so a hard refresh of a role-gated route is not
  // bounced to '/' just because roles have not loaded yet.
  await auth.ensureReady()
  if (to.meta.roles && !to.meta.roles.some(r => auth.roles.includes(r))) {
    return '/'
  }
})

export default router
