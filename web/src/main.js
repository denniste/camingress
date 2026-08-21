import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import Channels from './views/Channels.vue'
import Discovery from './views/Discovery.vue'
import Player from './views/Player.vue'
import Meeting from './views/Meeting.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/channels' },
    { path: '/channels', component: Channels },
    { path: '/discovery', component: Discovery },
    { path: '/meeting', component: Meeting },
    { path: '/player/:room', component: Player, props: true },
  ],
})

createApp(App).use(router).mount('#app')
