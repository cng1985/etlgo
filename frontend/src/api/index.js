import axios from 'axios'

const token = import.meta.env.VITE_API_TOKEN || 'etl-demo-token'

const http = axios.create({
  baseURL: '/api/v1',
  timeout: 60000,
  headers: {
    Authorization: `Bearer ${token}`,
  },
})

http.interceptors.response.use(
  (response) => response,
  (error) => {
    const message = error.response?.data?.msg || error.message || '请求失败'
    return Promise.reject(new Error(message))
  },
)

async function call(path, body) {
  const response = await http.post(path, body ?? {})
  return response.data
}

export const api = {
  health: () => call('/health/get').then((body) => body.data),
  stats: () => call('/stats/get').then((body) => body.data),
  sources: () => call('/source/list', { page: 1, size: 200 }).then((body) => body.data),
  createSource: (body) => call('/source/create', body).then((body) => body.data),
  updateSource: (id, body) => call('/source/update', { ...body, id }).then((body) => body.data),
  deleteSource: (id) => call('/source/delete', { id }),
  previewSource: (id) => call('/source/preview', { id, limit: 20 }).then((body) => body.data),
  uploadSource: (form) => {
    const data = new FormData()
    data.append('file', form.file)
    data.append('businessType', form.type)
    data.append('businessId', form.name || '')
    return http.post('/file/upload', data).then((response) => response.data.data)
  },
  pipelines: () => call('/pipeline/list', { page: 1, size: 200 }).then((body) => body.data),
  pipeline: (id) => call('/pipeline/get', { id }).then((body) => body.data),
  createPipeline: (body) => call('/pipeline/create', body).then((body) => body.data),
  updatePipeline: (id, body) => call('/pipeline/update', { ...body, id }).then((body) => body.data),
  deletePipeline: (id) => call('/pipeline/delete', { id }),
  previewPipeline: (body) => call('/pipeline/preview', body).then((body) => body.data),
  runPipeline: (id) => call('/pipeline/execute', { id }).then((body) => body.data),
  runs: (body) => call('/run/list', { page: 1, size: 100, ...body }).then((body) => body.data),
  run: (id) => call('/run/get', { id }).then((body) => body.data),
  datasets: () => call('/dataset/list', { page: 1, size: 200 }).then((body) => body.data),
  previewDataset: (name) => call('/dataset/preview', { name, limit: 50 }).then((body) => body.data),
  exportDataset: (name, format) => call('/dataset/export', { name, format }).then((body) => body.data),
  deleteDataset: (name) => call('/dataset/delete', { name }),
}

export function readJSON(value, fallback) {
  if (value == null || value === '') return fallback
  if (typeof value === 'string') return JSON.parse(value)
  return value
}

export function formatTime(value) {
  if (!value) return '—'
  if (typeof value === 'string' && /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(value)) return value
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { hour12: false })
}

export const sourceTypeLabel = {
  csv: 'CSV',
  json: 'JSON',
  sqlite: 'SQLite',
  http: 'HTTP',
}

export const targetTypeLabel = {
  sqlite: 'SQLite 表',
  csv: 'CSV 文件',
  json: 'JSON 文件',
}

export const statusLabel = {
  running: '运行中',
  success: '成功',
  failed: '失败',
}

export const stepTypeLabel = {
  filter: '过滤',
  select: '选择列',
  drop: '删除列',
  rename: '重命名',
  cast: '类型转换',
  derive: '派生列',
}
