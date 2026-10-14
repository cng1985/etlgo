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
        <el-table-column label="类型" width="140">
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

    <el-dialog v-model="editor.visible" :title="editor.id ? '编辑数据源' : '新建数据源'" width="640px">
      <el-form label-position="top">
        <el-form-item label="名称">
          <el-input v-model="editor.name" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="editor.type" style="width: 100%" @change="onTypeChange">
            <el-option label="CSV 文件" value="csv" />
            <el-option label="Excel 文件" value="excel" />
            <el-option label="MySQL" value="mysql" />
            <el-option label="PostgreSQL" value="pg" />
            <el-option label="SQL Server" value="sqlserver" />
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
        <template v-else-if="editor.type === 'excel'">
          <el-form-item label="相对路径">
            <el-input v-model="editor.path" placeholder="uploads/orders.xlsx" />
          </el-form-item>
          <div class="grid-2">
            <el-form-item label="工作表">
              <el-input v-model="editor.sheet" placeholder="留空读取第一张" />
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
        <template v-else-if="isDatabase(editor.type)">
          <div class="grid-2">
            <el-form-item label="主机">
              <el-input v-model="editor.host" placeholder="127.0.0.1" />
            </el-form-item>
            <el-form-item label="端口">
              <el-input-number v-model="editor.port" :min="1" :max="65535" controls-position="right" style="width: 100%" />
            </el-form-item>
          </div>
          <div class="grid-2">
            <el-form-item label="用户名">
              <el-input v-model="editor.user" />
            </el-form-item>
            <el-form-item label="密码">
              <el-input v-model="editor.password" type="password" show-password />
            </el-form-item>
          </div>
          <el-form-item label="数据库">
            <el-input v-model="editor.database" />
          </el-form-item>
          <el-form-item v-if="editor.type === 'pg'" label="SSL">
            <el-select v-model="editor.sslMode" style="width: 100%">
              <el-option label="disable" value="disable" />
              <el-option label="require" value="require" />
              <el-option label="verify-ca" value="verify-ca" />
              <el-option label="verify-full" value="verify-full" />
            </el-select>
          </el-form-item>
          <el-form-item v-else-if="editor.type === 'sqlserver'" label="加密">
            <el-select v-model="editor.sslMode" style="width: 100%">
              <el-option label="不加密" value="disable" />
              <el-option label="加密，不校验证书" value="false" />
              <el-option label="加密并校验证书" value="true" />
            </el-select>
          </el-form-item>
          <el-form-item label="SELECT 语句">
            <el-input v-model="editor.query" type="textarea" :rows="4" placeholder="select id, name from orders" />
          </el-form-item>
          <p class="hint">只允许单条 SELECT 或 WITH 查询。</p>
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

    <el-dialog v-model="upload.visible" title="上传 CSV / Excel / JSON" width="480px">
      <el-form label-position="top">
        <el-form-item label="名称">
          <el-input v-model="upload.name" placeholder="可留空，默认使用文件名" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="upload.type" style="width: 100%">
            <el-option label="CSV" value="csv" />
            <el-option label="Excel" value="excel" />
            <el-option label="JSON" value="json" />
          </el-select>
        </el-form-item>
        <el-form-item label="文件">
          <input type="file" :accept="uploadAccept" @change="onFile" />
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
const uploadAccept = computed(() => {
  if (upload.type === 'excel') return '.xlsx,.xlsm,.xltx'
  if (upload.type === 'json') return '.json,application/json'
  return '.csv,text/csv'
})
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
    sheet: '',
    query: '',
    url: '',
    dataPath: '',
    host: '127.0.0.1',
    port: 3306,
    user: '',
    password: '',
    database: '',
    sslMode: 'disable',
  }
}

function isDatabase(type) {
  return type === 'mysql' || type === 'pg' || type === 'sqlserver'
}

function defaultPort(type) {
  if (type === 'mysql') return 3306
  if (type === 'pg') return 5432
  if (type === 'sqlserver') return 1433
  return 0
}

function onTypeChange(type) {
  const port = defaultPort(type)
  if (port) editor.port = port
  if (type === 'pg' || type === 'sqlserver') editor.sslMode = 'disable'
}

function configOf(form) {
  if (form.type === 'csv') {
    return { path: form.path.trim(), delimiter: form.delimiter || ',', hasHeader: form.hasHeader }
  }
  if (form.type === 'excel') {
    return { path: form.path.trim(), sheet: form.sheet.trim(), hasHeader: form.hasHeader }
  }
  if (form.type === 'json') return { path: form.path.trim() }
  if (form.type === 'sqlite') return { path: form.path.trim(), query: form.query.trim() }
  if (isDatabase(form.type)) {
    const config = {
      host: form.host.trim(),
      port: Number(form.port) || 0,
      user: form.user.trim(),
      password: form.password,
      database: form.database.trim(),
      query: form.query.trim(),
    }
    if (form.type === 'pg' || form.type === 'sqlserver') config.sslMode = form.sslMode || 'disable'
    return config
  }
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
    sheet: cfg.sheet || '',
    query: cfg.query || '',
    url: cfg.url || '',
    dataPath: cfg.dataPath || '',
    host: cfg.host || '127.0.0.1',
    port: Number(cfg.port) || defaultPort(row.type) || 3306,
    user: cfg.user || '',
    password: cfg.password || '',
    database: cfg.database || '',
    sslMode: cfg.sslMode || 'disable',
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
