import { createRouter, createWebHistory } from 'vue-router'
import HomeView from './views/HomeView.vue'
import NotesList from './views/NotesList.vue'
import NoteDetail from './views/NoteDetail.vue'
import StatsView from './views/StatsView.vue'
import ShelfView from './views/ShelfView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: HomeView, meta: { title: '首页' } },
    { path: '/notes', component: NotesList, meta: { title: '笔记' } },
    { path: '/notes/:bookId', component: NoteDetail, meta: { title: '笔记详情' } },
    { path: '/stats', component: StatsView, meta: { title: '阅读统计' } },
    { path: '/shelf', component: ShelfView, meta: { title: '书架' } },
  ],
})
