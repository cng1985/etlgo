<template>
  <div>
    <p v-if="summary" class="hint">{{ summary }}</p>
    <el-table v-if="columns.length" :data="rows" border size="small" max-height="460">
      <el-table-column
        v-for="col in columns"
        :key="col"
        :prop="col"
        :label="col"
        min-width="130"
        show-overflow-tooltip
      >
        <template #default="{ row }">{{ show(row[col]) }}</template>
      </el-table-column>
    </el-table>
    <div v-else class="empty">没有可展示的数据</div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  columns: { type: Array, default: () => [] },
  rows: { type: Array, default: () => [] },
  total: { type: Number, default: 0 },
  truncated: { type: Boolean, default: false },
})

const summary = computed(() => {
  if (!props.total && !props.rows.length) return ''
  const shown = props.rows.length
  return props.truncated || shown < props.total
    ? `共 ${props.total} 行，当前显示 ${shown} 行`
    : `共 ${props.total || shown} 行`
})

function show(value) {
  if (value === null || value === undefined || value === '') return '—'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}
</script>
