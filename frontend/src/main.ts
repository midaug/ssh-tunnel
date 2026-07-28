import {createApp} from 'vue'
import {createPinia} from 'pinia'
import {createRouter, createWebHashHistory} from 'vue-router'
import App from './App.vue'
import './style.css'

const routes = [
  {path: '/', component: () => import('./views/Dashboard.vue')},
  {path: '/edit/:id?', component: () => import('./views/TunnelEdit.vue')},
  {path: '/settings', component: () => import('./views/Settings.vue')},
  {path: '/import', component: () => import('./views/ImportView.vue')},
  {path: '/about', component: () => import('./views/About.vue')},
]

const router = createRouter({history: createWebHashHistory(), routes})

createApp(App).use(createPinia()).use(router).mount('#app')
