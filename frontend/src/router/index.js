import { createRouter, createWebHistory } from 'vue-router'
import MainLayout from '../layouts/MainLayout.vue'
import Dashboard from '../views/Dashboard.vue'
import Sources from '../views/Sources.vue'
import Pipelines from '../views/Pipelines.vue'
import PipelineEdit from '../views/PipelineEdit.vue'
import Runs from '../views/Runs.vue'
import Datasets from '../views/Datasets.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: MainLayout,
      children: [
        { path: '', name: 'dashboard', component: Dashboard, meta: { title: '概览' } },
        { path: 'sources', name: 'sources', component: Sources, meta: { title: '数据源' } },
        { path: 'pipelines', name: 'pipelines', component: Pipelines, meta: { title: '管道' } },
        { path: 'pipelines/new', name: 'pipeline-new', component: PipelineEdit, meta: { title: '新建管道' } },
        { path: 'pipelines/:id', name: 'pipeline-edit', component: PipelineEdit, meta: { title: '编辑管道' } },
        { path: 'runs', name: 'runs', component: Runs, meta: { title: '运行记录' } },
        { path: 'datasets', name: 'datasets', component: Datasets, meta: { title: '数据集' } },
      ],
    },
  ],
})

export default router
