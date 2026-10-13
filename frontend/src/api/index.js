import axios from 'axios'

const http = axios.create({ baseURL: '/api', timeout: 60000 })

http.interceptors.response.use(
  (response) => response,
  (error) => {
    const message = error.response?.data?.error || error.message || '请求失败'
    return Promise.reject(new Error(message))
  },
)

export const api = {
  health: () => http.get('/health').then((r) => r.data),
  stats: () => http.get('/stats').then((r) => r.data),
  sources: () => http.get('/sources').then((r) => r.data),
  createSource: (body) => http.post('/sources', body).then((r) => r.data),
  updateSource: (id, body) => http.put(`/sources/${id}`, body).then((r) => r.data),
  deleteSource: (id) => http.delete(`/sources/${id}`),
  previewSource: (id) => http.post(`/sources/${id}/preview?limit=20`).then((r) => r.data),
  uploadSource: (form) => {
    const data = new FormData()
    data.append('file', form.file)
    data.append('name', form.name || '')
    data.append('type', form.type)
    return http.post('/uploads', data).then((r) => r.data)
  },
  pipelines: () => http.get('/pipelines').then((r) => r.data),
  pipeline: (id) => http.get(`/pipelines/${id}`).then((r) => r.data),
  createPipeline: (body) => http.post('/pipelines', body).then((r) => r.data),
  updatePipeline: (id, body) => http.put(`/pipelines/${id}`, body).then((r) => r.data),
  deletePipeline: (id) => http.delete(`/pipelines/${id}`),
  previewPipeline: (body) => http.post('/pipelines/preview', body).then((r) => r.data),
  runPipeline: (id) => http.post(`/pipelines/${id}/run`).then((r) => r.data),
  runs: (params) => http.get('/runs', { params }).then((r) => r.data),
  run: (id) => http.get(`/runs/${id}`).then((r) => r.data),
  datasets: () => http.get('/datasets').then((r) => r.data),
  previewDataset: (name) => http.get(`/datasets/${name}/preview?limit=50`).then((r) => r.data),
  deleteDataset: (name) => http.delete(`/datasets/${name}`),
}

export function formatTime(value) {
  if (!value) return '—'
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
