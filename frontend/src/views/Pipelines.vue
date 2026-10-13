<template>
  <div>
    <div class="toolbar">
      <el-input v-model="keyword" placeholder="搜索管道" clearable style="width: 240px" />
      <span class="grow" />
      <el-button type="primary" @click="router.push('/pipelines/new')">新建管道</el-button>
    </div>
    <div class="panel">
      <el-table :data="filtered">
        <el-table-column prop="name" label="名称" min-width="180" />
        <el-table-column prop="description" label="说明" min-width="240" show-overflow-tooltip />
        <el-table-column label="目标" width="120">
          <template #default="{ row }">{{ targetTypeLabel[row.targetType] || row.targetType }}</template>
        </el-table-column>
        <el-table-column label="调度" width="120">
          <template #default="{ row }">{{ scheduleText(row.scheduleMinutes) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'" effect="plain">{{ row.enabled ? '启用' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <div class="ops">
              <el-button text type="primary" @click="run(row)">运行</el-button>
              <el-button text @click="router.push(`/pipelines/${row.id}`)">编辑</el-button>
              <el-button text type="danger" @click="remove(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
        <template #empty><div class="empty">还没有管道</div></template>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api, targetTypeLabel } from '../api'

const router = useRouter()
const items = ref([])
const keyword = ref('')
const filtered = computed(() => {
  const q = keyword.value.trim()
  if (!q) return items.value
  return items.value.filter((item) => item.name.includes(q) || (item.description || '').includes(q))
})

function scheduleText(minutes) {
  if (!minutes) return '手动'
  if (minutes === 1) return '每 1 分钟'
  if (minutes < 60) return `每 ${minutes} 分钟`
  if (minutes === 60) return '每小时'
  if (minutes === 1440) return '每天'
  return `每 ${minutes} 分钟`
}

async function load() {
  items.value = await api.pipelines()
}

async function run(row) {
  try {
    await api.runPipeline(row.id)
    ElMessage.success(`已开始运行「${row.name}」`)
    router.push('/runs')
  } catch (error) {
    ElMessage.error(error.message)
  }
}

async function remove(row) {
  try {
    await ElMessageBox.confirm(`删除管道「${row.name}」？已生成的数据集会保留。`, '确认删除', { type: 'warning' })
    await api.deletePipeline(row.id)
    ElMessage.success('已删除')
    await load()
  } catch (error) {
    if (error !== 'cancel' && error?.message) ElMessage.error(error.message)
  }
}

onMounted(async () => {
  try {
    await load()
  } catch (error) {
    ElMessage.error(error.message)
  }
})
</script>
