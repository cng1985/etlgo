<template>
  <div>
    <div class="stats">
      <div class="stat"><span>数据源</span><strong>{{ stats.sources || 0 }}</strong></div>
      <div class="stat"><span>管道</span><strong>{{ stats.pipelines || 0 }}</strong></div>
      <div class="stat"><span>成功 / 失败</span><strong>{{ stats.successRuns || 0 }} / {{ stats.failedRuns || 0 }}</strong></div>
      <div class="stat"><span>已加载行数</span><strong>{{ stats.rowsLoaded || 0 }}</strong></div>
    </div>
    <div class="section-title">
      <h2>最近运行</h2>
      <el-button @click="load">刷新</el-button>
    </div>
    <div class="panel">
      <el-table :data="stats.recentRuns || []" size="default">
        <el-table-column prop="pipelineName" label="管道" min-width="180" />
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="tagType(row.status)" effect="plain">{{ statusLabel[row.status] || row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="rowsLoaded" label="写入行数" width="110" />
        <el-table-column label="开始时间" min-width="180">
          <template #default="{ row }">{{ formatTime(row.startedAt) }}</template>
        </el-table-column>
        <el-table-column prop="error" label="错误" min-width="220" show-overflow-tooltip />
        <template #empty><div class="empty">还没有运行记录。打开一条管道并执行即可。</div></template>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { api, formatTime, statusLabel } from '../api'

const stats = reactive({})
let timer

function tagType(status) {
  if (status === 'success') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'running') return 'warning'
  return 'info'
}

async function load() {
  try {
    Object.assign(stats, await api.stats())
  } catch (error) {
    ElMessage.error(error.message)
  }
}

onMounted(() => {
  load()
  timer = setInterval(load, 4000)
})
onUnmounted(() => clearInterval(timer))
</script>
