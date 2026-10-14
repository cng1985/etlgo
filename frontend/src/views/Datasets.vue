<template>
  <div>
    <div class="toolbar">
      <el-input v-model="keyword" placeholder="搜索数据集" clearable style="width: 240px" />
      <span class="grow" />
      <el-button @click="load">刷新</el-button>
    </div>
    <div class="panel">
      <el-table :data="filtered">
        <el-table-column prop="name" label="名称" min-width="180" />
        <el-table-column label="类型" width="120">
          <template #default="{ row }">{{ targetTypeLabel[row.kind] || row.kind }}</template>
        </el-table-column>
        <el-table-column prop="path" label="位置" min-width="180" show-overflow-tooltip />
        <el-table-column prop="rowCount" label="行数" width="90" />
        <el-table-column label="更新时间" min-width="180">
          <template #default="{ row }">{{ formatTime(row.updatedAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <div class="ops">
              <el-button text @click="preview(row)">预览</el-button>
              <el-button text @click="download(row, 'csv')">CSV</el-button>
              <el-button text @click="download(row, 'json')">JSON</el-button>
              <el-button text type="danger" @click="remove(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
        <template #empty><div class="empty">运行管道后，结果会出现在这里。</div></template>
      </el-table>
    </div>
    <el-drawer v-model="previewer.visible" :title="previewer.title" size="64%">
      <DataTable :columns="previewer.columns" :rows="previewer.rows" :total="previewer.total" :truncated="previewer.truncated" />
    </el-drawer>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import DataTable from '../components/DataTable.vue'
import { api, formatTime, targetTypeLabel } from '../api'

const items = ref([])
const keyword = ref('')
const previewer = reactive({ visible: false, title: '', columns: [], rows: [], total: 0, truncated: false })
const filtered = computed(() => {
  const q = keyword.value.trim()
  if (!q) return items.value
  return items.value.filter((item) => item.name.includes(q))
})

async function load() {
  items.value = await api.datasets()
}

async function preview(row) {
  try {
    const data = await api.previewDataset(row.name)
    Object.assign(previewer, {
      visible: true,
      title: row.name,
      columns: data.columns || [],
      rows: data.rows || [],
      total: data.total || 0,
      truncated: data.truncated,
    })
  } catch (error) {
    ElMessage.error(error.message)
  }
}

async function download(row, format) {
  try {
    const file = await api.exportDataset(row.name, format)
    const blob = new Blob([file.content], { type: file.contentType || 'text/plain' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = file.fileName || `${row.name}.${format}`
    link.click()
    URL.revokeObjectURL(url)
  } catch (error) {
    ElMessage.error(error.message)
  }
}

async function remove(row) {
  try {
    await ElMessageBox.confirm(`删除数据集「${row.name}」及其内容？`, '确认删除', { type: 'warning' })
    await api.deleteDataset(row.name)
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
