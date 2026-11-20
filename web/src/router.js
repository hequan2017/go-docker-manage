import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from './store/auth'

const routes = [
  { path: '/login', name: 'login', component: () => import('./views/Login.vue'), meta: { public: true } },
  {
    path: '/',
    component: () => import('./layout/MainLayout.vue'),
    children: [
      { path: '', redirect: '/dashboard' },
      { path: 'dashboard', name: 'dashboard', component: () => import('./views/Dashboard.vue'), meta: { title: '仪表盘' } },
      { path: 'nodes', name: 'nodes', component: () => import('./views/Nodes.vue'), meta: { title: '节点管理' } },
      { path: 'containers', name: 'containers', component: () => import('./views/Containers.vue'), meta: { title: '容器管理' } },
      { path: 'images', name: 'images', component: () => import('./views/Images.vue'), meta: { title: '镜像管理' } },
    ],
  },
]

const router = createRouter({ history: createWebHistory(), routes })

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (!to.meta.public && !auth.loggedIn) return { name: 'login' }
  if (to.name === 'login' && auth.loggedIn) return { path: '/dashboard' }
})

export default router
