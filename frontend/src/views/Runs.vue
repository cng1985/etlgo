<template>
  <div>
    <div class="toolbar">
      <el-select v-model="status" clearable placeholder="全部状态" style="width: 160px" @change="load">
        <el-option label="运行中" value="running" />
        <el-option label="成功" value="success" />
        <el-option label="失败" value="failed" />
      </el-select>
      <span class="grow" />
      <el-button @click="load">刷新</el-button>
    </div>
    <div class="panel">
      <el-table :data="items" @row-click="open">
        <el-table-column prop="id" label="编号" width="80" />
        <el-table-column prop="pipelineName" label="管道" min-width="180" />
        <el-table-column label="触发" width="100">
          <template #default="{ row }">{{ row.trigger === 'schedule' ? '调度' : '手动' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="tagType(row.status)" effect="plain">{{ statusLabel[row.status] || row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="rowsExtracted" label="抽取" width="90" />
        <el-table-column prop="rowsFiltered" label="过滤" width="90" />
        <el-table-column prop="rowsLoaded" label="写入" width="90" />
        <el-table-column label="开始时间" min-width="180">
          <template #default="{ row }">{{ formatTime(row.startedAt) }}</template>
        </el-table-column>
        <template #empty><div class="empty">还没有运行记录</div></template>
      </el-table>
    </div>

    <el-drawer v-model="detail.visible" :title="detail.item?.pipelineName || '运行详情'" size="480px">
      <template v-if="detail.item">
        <p class="hint">
          {{ statusLabel[detail.item.status] }} · 抽取 {{ detail.item.rowsExtracted }} · 过滤 {{ detail.item.rowsFiltered }} · 写入 {{ detail.item.rowsLoaded }}
        </p>
        <p v-if="detail.item.error" class="hint" style="color: #b42318">{{ detail.item.error }}</p>
        <pre class="log">{{ detail.item.log || '没有日志' }}</pre>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api, formatTime, statusLabel } from '../api'

const items = ref([])
const status = ref('')
const detail = reactive({ visible: false, item: null })
let timer

function tagType(value) {
  if (value === 'success') return 'success'
  if (value === 'failed') return 'danger'
  if (value === 'running') return 'warning'
  return 'info'
}

async function load() {
  try {
    items.value = await api.runs({ status: status.value || undefined, limit: 100 })
    if (detail.visible && detail.item) await open(detail.item, true)
  } catch (error) {
    ElMessage.error(error.message)
  }
}

async function open(row, keepOpen) {
  try {
    detail.item = await api.run(row.id)
    if (!keepOpen) detail.visible = true
  } catch (error) {
    ElMessage.error(error.message)
  }
}

onMounted(() => {
  load()
  timer = setInterval(() => {
    if (items.value.some((item) => item.status === 'running') || detail.item?.status === 'running') load()
  }, 1500)
})
onUnmounted(() => clearInterval(timer))
</script>
