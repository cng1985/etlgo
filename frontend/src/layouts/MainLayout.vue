<template>
  <div class="shell">
    <aside class="sidebar">
      <div class="brand">
        <span class="mark">E</span>
        <div>
          <strong>ETL Go</strong>
          <small>抽取 · 转换 · 加载</small>
        </div>
      </div>
      <el-menu :key="active" :default-active="active" router>
        <el-menu-item index="/">概览</el-menu-item>
        <el-menu-item index="/sources">数据源</el-menu-item>
        <el-menu-item index="/pipelines">管道</el-menu-item>
        <el-menu-item index="/runs">运行记录</el-menu-item>
        <el-menu-item index="/datasets">数据集</el-menu-item>
      </el-menu>
    </aside>
    <section class="workspace">
      <header class="topbar">
        <h1>{{ route.meta.title || 'ETL Go' }}</h1>
        <el-tag :type="online ? 'success' : 'info'" effect="plain">
          {{ online ? '服务已连接' : '正在连接' }}
        </el-tag>
      </header>
      <main class="page">
        <router-view />
      </main>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api'

const route = useRoute()
const online = ref(false)
let timer

const active = computed(() => {
  if (route.path.startsWith('/pipelines')) return '/pipelines'
  return route.path
})

async function ping() {
  try {
    await api.health()
    online.value = true
  } catch {
    online.value = false
  }
}

onMounted(() => {
  ping()
  timer = setInterval(ping, 15000)
})

onUnmounted(() => clearInterval(timer))
</script>
