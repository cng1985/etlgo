<template>
  <div>
    <el-form label-position="top">
      <div class="panel" style="margin-bottom: 16px">
        <div class="grid-2">
          <el-form-item label="名称">
            <el-input v-model="form.name" />
          </el-form-item>
          <el-form-item label="数据源">
            <el-select v-model="form.sourceId" filterable style="width: 100%" @change="loadColumns">
              <el-option v-for="item in sources" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
        </div>
        <el-form-item label="说明">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
        <div class="grid-2">
          <el-form-item label="调度">
            <el-select v-model="form.scheduleMinutes" style="width: 100%">
              <el-option label="仅手动" :value="0" />
              <el-option label="每 1 分钟" :value="1" />
              <el-option label="每 5 分钟" :value="5" />
              <el-option label="每小时" :value="60" />
              <el-option label="每天" :value="1440" />
            </el-select>
          </el-form-item>
          <el-form-item label="启用">
            <el-switch v-model="form.enabled" />
          </el-form-item>
        </div>
      </div>

      <div class="section-title">
        <h2>转换步骤</h2>
        <div>
          <el-button @click="loadColumns">读取字段</el-button>
          <el-dropdown @command="addStep">
            <el-button type="primary">添加步骤</el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item v-for="item in stepTypes" :key="item" :command="item">
                  {{ stepTypeLabel[item] }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
      <p class="hint">步骤按从上到下的顺序执行。字段列表来自数据源预览，也可以直接输入列名。</p>
      <StepEditor
        v-for="(step, index) in form.steps"
        :key="step._id"
        :step="step"
        :columns="columns"
        :index="index"
        :total="form.steps.length"
        @remove="form.steps.splice(index, 1)"
        @move="moveStep(index, $event)"
      />
      <div v-if="!form.steps.length" class="panel empty">这条管道会原样加载数据源。</div>

      <div class="section-title"><h2>加载目标</h2></div>
      <div class="panel">
        <div class="grid-2">
          <el-form-item label="目标类型">
            <el-select v-model="form.targetType" style="width: 100%">
              <el-option label="SQLite 表" value="sqlite" />
              <el-option label="CSV 文件" value="csv" />
              <el-option label="JSON 文件" value="json" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="form.targetType === 'sqlite'" label="表名">
            <el-input v-model="form.table" placeholder="例如 orders_paid" />
          </el-form-item>
          <el-form-item v-else label="相对路径">
            <el-input v-model="form.path" :placeholder="form.targetType === 'csv' ? 'outputs/result.csv' : 'outputs/result.json'" />
          </el-form-item>
        </div>
        <el-form-item v-if="form.targetType === 'sqlite'" label="写入模式">
          <el-radio-group v-model="form.mode">
            <el-radio value="replace">覆盖</el-radio>
            <el-radio value="append">追加</el-radio>
          </el-radio-group>
        </el-form-item>
      </div>
    </el-form>

    <div class="toolbar" style="margin-top: 16px">
      <el-button @click="preview">预览结果</el-button>
      <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      <el-button v-if="pipelineId" type="success" @click="run">运行</el-button>
      <el-button @click="router.push('/pipelines')">返回</el-button>
    </div>

    <el-drawer v-model="previewer.visible" title="转换预览" size="62%">
      <pre v-if="previewer.logs.length" class="log">{{ previewer.logs.join('\n') }}</pre>
      <div style="height: 12px" />
      <DataTable :columns="previewer.columns" :rows="previewer.rows" :total="previewer.total" :truncated="previewer.truncated" />
    </el-drawer>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import DataTable from '../components/DataTable.vue'
import StepEditor from '../components/StepEditor.vue'
import { api, stepTypeLabel } from '../api'

const route = useRoute()
const router = useRouter()
const sources = ref([])
const columns = ref([])
const saving = ref(false)
const stepTypes = ['filter', 'select', 'drop', 'rename', 'cast', 'derive']
const previewer = reactive({ visible: false, columns: [], rows: [], total: 0, truncated: false, logs: [] })

const pipelineId = ref(route.params.id || '')
const form = reactive({
  name: '',
  description: '',
  enabled: true,
  sourceId: undefined,
  scheduleMinutes: 0,
  steps: [],
  targetType: 'sqlite',
  table: '',
  mode: 'replace',
  path: '',
})

function newId() {
  return Math.random().toString(36).slice(2)
}

function addStep(type) {
  const base = { _id: newId(), type }
  if (type === 'filter') Object.assign(base, { column: '', op: 'eq', value: '' })
  if (type === 'select' || type === 'drop') Object.assign(base, { columns: [] })
  if (type === 'rename') Object.assign(base, { mapping: {} })
  if (type === 'cast') Object.assign(base, { column: '', to: 'string' })
  if (type === 'derive') Object.assign(base, { column: '', op: 'trim', left: '', right: '', columns: [], sep: ' ', value: '', old: '', new: '' })
  form.steps.push(base)
}

function moveStep(index, delta) {
  const next = index + delta
  if (next < 0 || next >= form.steps.length) return
  const [item] = form.steps.splice(index, 1)
  form.steps.splice(next, 0, item)
}

function payload() {
  const steps = form.steps.map((step) => {
    const { _id, ...rest } = step
    return rest
  })
  const targetConfig = form.targetType === 'sqlite'
    ? { table: form.table.trim(), mode: form.mode }
    : { path: form.path.trim() }
  return {
    name: form.name.trim(),
    description: form.description.trim(),
    enabled: form.enabled,
    sourceId: form.sourceId,
    scheduleMinutes: form.scheduleMinutes,
    steps,
    targetType: form.targetType,
    targetConfig,
  }
}

async function loadColumns() {
  if (!form.sourceId) return
  try {
    const data = await api.previewSource(form.sourceId)
    columns.value = data.columns || []
  } catch (error) {
    ElMessage.error(error.message)
  }
}

async function preview() {
  try {
    const body = payload()
    const data = await api.previewPipeline({ sourceId: body.sourceId, steps: body.steps, limit: 20 })
    Object.assign(previewer, {
      visible: true,
      columns: data.columns || [],
      rows: data.rows || [],
      total: data.total || 0,
      truncated: data.truncated,
      logs: data.logs || [],
    })
  } catch (error) {
    ElMessage.error(error.message)
  }
}

async function save() {
  saving.value = true
  try {
    const body = payload()
    if (pipelineId.value) {
      await api.updatePipeline(pipelineId.value, body)
      ElMessage.success('已保存')
    } else {
      const created = await api.createPipeline(body)
      ElMessage.success('已创建')
      pipelineId.value = String(created.id)
      router.replace(`/pipelines/${created.id}`)
    }
    return true
  } catch (error) {
    ElMessage.error(error.message)
    return false
  } finally {
    saving.value = false
  }
}

async function run() {
  const saved = await save()
  if (!saved) return
  try {
    await api.runPipeline(pipelineId.value)
    ElMessage.success('已开始运行')
    router.push('/runs')
  } catch (error) {
    ElMessage.error(error.message)
  }
}

onMounted(async () => {
  try {
    sources.value = await api.sources()
    if (!pipelineId.value) return
    const item = await api.pipeline(pipelineId.value)
    const target = JSON.parse(item.targetConfig || '{}')
    const steps = JSON.parse(item.steps || '[]').map((step) => ({ ...step, _id: newId() }))
    Object.assign(form, {
      name: item.name,
      description: item.description,
      enabled: item.enabled,
      sourceId: item.sourceId,
      scheduleMinutes: item.scheduleMinutes,
      steps,
      targetType: item.targetType,
      table: target.table || '',
      mode: target.mode || 'replace',
      path: target.path || '',
    })
    await loadColumns()
  } catch (error) {
    ElMessage.error(error.message)
  }
})
</script>
