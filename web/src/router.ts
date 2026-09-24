import { createRouter, createWebHistory } from 'vue-router'
import { me } from './api'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: () => import('./views/LoginView.vue') },
    { path: '/setup', component: () => import('./views/SetupView.vue') },
    {
      path: '/',
      component: () => import('./layouts/AdminLayout.vue'),
      meta: { auth: true },
      children: [
        { path: 'overview', component: () => import('./views/OverviewView.vue') },
        { path: 'people', component: () => import('./views/PeopleView.vue') },
        { path: 'groups', component: () => import('./views/GroupsView.vue') },
        { path: 'import', component: () => import('./views/ImportView.vue') },
        { path: 'audit', component: () => import('./views/AuditView.vue') },
        { path: 'settings', component: () => import('./views/SystemSettingsView.vue') },
        { path: 'help', component: () => import('./views/HelpView.vue') },
      ],
    },
    { path: '/password', component: () => import('./views/PasswordView.vue') },
    { path: '/:pathMatch(.*)*', redirect: '/overview' },
  ],
})

router.beforeEach(async (to) => {
  if (!to.meta.auth && !to.matched.some((m) => m.meta.auth)) return true
  try {
    await me()
    return true
  } catch {
    return '/'
  }
})
