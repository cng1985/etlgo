<template>
  <div>
    <div class="toolbar">
      <el-input v-model="keyword" placeholder="搜索数据源" clearable style="width: 240px" />
      <span class="grow" />
      <el-button @click="openUpload">上传文件</el-button>
      <el-button type="primary" @click="openCreate">新建数据源</el-button>
    </div>
    <div class="panel">
      <el-table :data="filtered">
        <el-table-column prop="name" label="名称" min-width="180" />
        <el-table-column label="类型" width="120">
          <template #default="{ row }">
            <el-tag effect="plain">{{ sourceTypeLabel[row.type] || row.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" min-width="180">
          <template #default="{ row }">{{ formatTime(row.updatedAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <div class="ops">
              <el-button text @click="preview(row)">预览</el-button>
              <el-button text @click="openEdit(row)">编辑</el-button>
              <el-button text type="danger" @click="remove(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
        <template #empty><div class="empty">还没有数据源</div></template>
      </el-table>
    </div>

    <el-dialog v-model="editor.visible" :title="editor.id ? '编辑数据源' : '新建数据源'" width="560px">
      <el-form label-position="top">
        <el-form-item label="名称">
          <el-input v-model="editor.name" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="editor.type" style="width: 100%">
            <el-option label="CSV 文件" value="csv" />
            <el-option label="JSON 文件" value="json" />
            <el-option label="SQLite 查询" value="sqlite" />
            <el-option label="HTTP JSON" value="http" />
          </el-select>
        </el-form-item>
        <template v-if="editor.type === 'csv'">
          <el-form-item label="相对路径">
            <el-input v-model="editor.path" placeholder="samples/orders.csv" />
          </el-form-item>
          <div class="grid-2">
            <el-form-item label="分隔符">
              <el-input v-model="editor.delimiter" maxlength="1" />
            </el-form-item>
            <el-form-item label="包含表头">
              <el-switch v-model="editor.hasHeader" />
            </el-form-item>
          </div>
        </template>
        <el-form-item v-else-if="editor.type === 'json'" label="相对路径">
          <el-input v-model="editor.path" placeholder="samples/customers.json" />
        </el-form-item>
        <template v-else-if="editor.type === 'sqlite'">
          <el-form-item label="外部库相对路径">
            <el-input v-model="editor.path" placeholder="留空表示当前应用库" />
          </el-form-item>
          <el-form-item label="SELECT 语句">
            <el-input v-model="editor.query" type="textarea" :rows="4" />
          </el-form-item>
        </template>
        <template v-else>
          <el-form-item label="URL">
            <el-input v-model="editor.url" placeholder="https://example.com/items.json" />
          </el-form-item>
          <el-form-item label="数组路径">
            <el-input v-model="editor.dataPath" placeholder="例如 data.items，根节点是数组时可留空" />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="editor.visible = false">取消</el-button>
        <el-button type="primary" :loading="editor.saving" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="upload.visible" title="上传 CSV / JSON" width="480px">
      <el-form label-position="top">
        <el-form-item label="名称">
          <el-input v-model="upload.name" placeholder="可留空，默认使用文件名" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="upload.type" style="width: 100%">
            <el-option label="CSV" value="csv" />
            <el-option label="JSON" value="json" />
          </el-select>
        </el-form-item>
        <el-form-item label="文件">
          <input type="file" :accept="upload.type === 'csv' ? '.csv,text/csv' : '.json,application/json'" @change="onFile" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="upload.visible = false">取消</el-button>
        <el-button type="primary" :loading="upload.saving" @click="submitUpload">上传</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="previewer.visible" :title="previewer.title" size="60%">
      <DataTable :columns="previewer.columns" :rows="previewer.rows" :total="previewer.total" :truncated="previewer.truncated" />
    </el-drawer>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import DataTable from '../components/DataTable.vue'
import { api, formatTime, readJSON, sourceTypeLabel } from '../api'

const items = ref([])
const keyword = ref('')
const filtered = computed(() => {
  const q = keyword.value.trim()
  if (!q) return items.value
  return items.value.filter((item) => item.name.includes(q) || item.type.includes(q))
})

const editor = reactive(blankEditor())
const upload = reactive({ visible: false, name: '', type: 'csv', file: null, saving: false })
const previewer = reactive({ visible: false, title: '', columns: [], rows: [], total: 0, truncated: false })

function blankEditor() {
  return {
    visible: false,
    saving: false,
    id: 0,
    name: '',
    type: 'csv',
    path: '',
    delimiter: ',',
    hasHeader: true,
    query: '',
    url: '',
    dataPath: '',
  }
}

function configOf(form) {
  if (form.type === 'csv') {
    return { path: form.path.trim(), delimiter: form.delimiter || ',', hasHeader: form.hasHeader }
  }
  if (form.type === 'json') return { path: form.path.trim() }
  if (form.type === 'sqlite') return { path: form.path.trim(), query: form.query.trim() }
  return { url: form.url.trim(), dataPath: form.dataPath.trim() }
}

async function load() {
  items.value = await api.sources()
}

function openCreate() {
  Object.assign(editor, blankEditor(), { visible: true })
}

function openEdit(row) {
  const cfg = readJSON(row.config, {})
  Object.assign(editor, blankEditor(), {
    visible: true,
    id: row.id,
    name: row.name,
    type: row.type,
    path: cfg.path || '',
    delimiter: cfg.delimiter || ',',
    hasHeader: cfg.hasHeader !== false,
    query: cfg.query || '',
    url: cfg.url || '',
    dataPath: cfg.dataPath || '',
  })
}

async function save() {
  editor.saving = true
  const body = { name: editor.name.trim(), type: editor.type, config: configOf(editor) }
  try {
    if (editor.id) await api.updateSource(editor.id, body)
    else await api.createSource(body)
    editor.visible = false
    ElMessage.success('已保存')
    await load()
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    editor.saving = false
  }
}

async function remove(row) {
  try {
    await ElMessageBox.confirm(`删除数据源「${row.name}」？`, '确认删除', { type: 'warning' })
    await api.deleteSource(row.id)
    ElMessage.success('已删除')
    await load()
  } catch (error) {
    if (error !== 'cancel' && error?.message) ElMessage.error(error.message)
  }
}

async function preview(row) {
  try {
    const data = await api.previewSource(row.id)
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

function openUpload() {
  Object.assign(upload, { visible: true, name: '', type: 'csv', file: null, saving: false })
}

function onFile(event) {
  upload.file = event.target.files?.[0] || null
}

async function submitUpload() {
  if (!upload.file) {
    ElMessage.warning('请选择文件')
    return
  }
  upload.saving = true
  try {
    await api.uploadSource(upload)
    upload.visible = false
    ElMessage.success('已上传')
    await load()
  } catch (error) {
    ElMessage.error(error.message)
  } finally {
    upload.saving = false
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
