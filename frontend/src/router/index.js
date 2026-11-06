import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/visits' },
    {
      path: '/visits',
      name: 'visit-list',
      component: () => import('../views/VisitListView.vue'),
    },
    {
      path: '/visits/new',
      name: 'visit-create',
      component: () => import('../views/VisitCreateView.vue'),
    },
    {
      path: '/visits/:id(\\d+)',
      name: 'visit-detail',
      component: () => import('../views/VisitDetailView.vue'),
    },
    {
      path: '/dashboard',
      name: 'dashboard',
      component: () => import('../views/DashboardView.vue'),
    },
    { path: '/:pathMatch(.*)*', redirect: '/visits' },
  ],
  scrollBehavior() {
    return { top: 0 }
  },
})

export default router
