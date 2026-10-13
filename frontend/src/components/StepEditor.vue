<template>
  <div class="step-card">
    <div class="step-head">
      <el-tag effect="plain">{{ index + 1 }}</el-tag>
      <span class="name">{{ stepTypeLabel[step.type] || step.type }}</span>
      <span class="spacer" />
      <el-button text :disabled="index === 0" @click="$emit('move', -1)">上移</el-button>
      <el-button text :disabled="index === total - 1" @click="$emit('move', 1)">下移</el-button>
      <el-button text type="danger" @click="$emit('remove')">删除</el-button>
    </div>

    <template v-if="step.type === 'filter'">
      <div class="grid-2">
        <el-select v-model="step.column" filterable allow-create default-first-option placeholder="列">
          <el-option v-for="col in columns" :key="col" :label="col" :value="col" />
        </el-select>
        <el-select v-model="step.op" placeholder="条件">
          <el-option v-for="item in filterOps" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
      </div>
      <el-input
        v-if="step.op !== 'empty' && step.op !== 'not_empty'"
        v-model="step.value"
        placeholder="比较值"
        style="margin-top: 10px"
      />
    </template>

    <template v-else-if="step.type === 'select' || step.type === 'drop'">
      <el-select v-model="step.columns" multiple filterable allow-create default-first-option placeholder="选择列" style="width: 100%">
        <el-option v-for="col in columns" :key="col" :label="col" :value="col" />
      </el-select>
    </template>

    <template v-else-if="step.type === 'rename'">
      <div v-for="(row, rowIndex) in renameRows" :key="rowIndex" class="grid-2" style="margin-bottom: 8px">
        <el-select v-model="row.from" filterable allow-create default-first-option placeholder="原列名" @change="syncRename">
          <el-option v-for="col in columns" :key="col" :label="col" :value="col" />
        </el-select>
        <div style="display: flex; gap: 8px">
          <el-input v-model="row.to" placeholder="新列名" @input="syncRename" />
          <el-button @click="removeRename(rowIndex)">移除</el-button>
        </div>
      </div>
      <el-button @click="addRename">添加映射</el-button>
    </template>

    <template v-else-if="step.type === 'cast'">
      <div class="grid-2">
        <el-select v-model="step.column" filterable allow-create default-first-option placeholder="列">
          <el-option v-for="col in columns" :key="col" :label="col" :value="col" />
        </el-select>
        <el-select v-model="step.to" placeholder="目标类型">
          <el-option label="文本" value="string" />
          <el-option label="整数" value="int" />
          <el-option label="小数" value="float" />
          <el-option label="布尔" value="bool" />
        </el-select>
      </div>
    </template>

    <template v-else-if="step.type === 'derive'">
      <div class="grid-2">
        <el-input v-model="step.column" placeholder="新列名" />
        <el-select v-model="step.op" placeholder="运算">
          <el-option v-for="item in deriveOps" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
      </div>
      <div v-if="step.op === 'concat' || step.op === 'coalesce'" style="margin-top: 10px">
        <el-select v-model="step.columns" multiple filterable allow-create default-first-option placeholder="参与列" style="width: 100%">
          <el-option v-for="col in columns" :key="col" :label="col" :value="col" />
        </el-select>
        <el-input v-if="step.op === 'concat'" v-model="step.sep" placeholder="分隔符，可留空" style="margin-top: 10px" />
        <el-input v-else v-model="step.value" placeholder="全部为空时的默认值" style="margin-top: 10px" />
      </div>
      <div v-else-if="step.op === 'const'" style="margin-top: 10px">
        <el-input v-model="step.value" placeholder="常量值" />
      </div>
      <div v-else-if="step.op === 'replace'" class="grid-2" style="margin-top: 10px">
        <el-select v-model="step.left" filterable allow-create default-first-option placeholder="来源列">
          <el-option v-for="col in columns" :key="col" :label="col" :value="col" />
        </el-select>
        <el-input v-model="step.old" placeholder="查找文本" />
        <el-input v-model="step.new" placeholder="替换为" />
      </div>
      <div v-else class="grid-2" style="margin-top: 10px">
        <el-select v-model="step.left" filterable allow-create default-first-option placeholder="左列">
          <el-option v-for="col in columns" :key="col" :label="col" :value="col" />
        </el-select>
        <el-select
          v-if="['add', 'sub', 'mul', 'div'].includes(step.op)"
          v-model="step.right"
          filterable
          allow-create
          default-first-option
          placeholder="右列或数字"
        >
          <el-option v-for="col in columns" :key="col" :label="col" :value="col" />
        </el-select>
      </div>
    </template>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { stepTypeLabel } from '../api'

const props = defineProps({
  step: { type: Object, required: true },
  columns: { type: Array, default: () => [] },
  index: { type: Number, required: true },
  total: { type: Number, required: true },
})

defineEmits(['remove', 'move'])

const filterOps = [
  { value: 'eq', label: '等于' },
  { value: 'ne', label: '不等于' },
  { value: 'contains', label: '包含' },
  { value: 'not_contains', label: '不包含' },
  { value: 'gt', label: '大于' },
  { value: 'gte', label: '大于等于' },
  { value: 'lt', label: '小于' },
  { value: 'lte', label: '小于等于' },
  { value: 'empty', label: '为空' },
  { value: 'not_empty', label: '不为空' },
]

const deriveOps = [
  { value: 'concat', label: '拼接' },
  { value: 'lower', label: '转小写' },
  { value: 'upper', label: '转大写' },
  { value: 'trim', label: '去空格' },
  { value: 'replace', label: '替换文本' },
  { value: 'add', label: '相加' },
  { value: 'sub', label: '相减' },
  { value: 'mul', label: '相乘' },
  { value: 'div', label: '相除' },
  { value: 'const', label: '常量' },
  { value: 'coalesce', label: '取第一个非空' },
]

const renameRows = ref([{ from: '', to: '' }])

onMounted(() => {
  const rows = Object.entries(props.step.mapping || {}).map(([from, to]) => ({ from, to }))
  renameRows.value = rows.length ? rows : [{ from: '', to: '' }]
})

function syncRename() {
  const mapping = {}
  for (const row of renameRows.value) {
    if (row.from) mapping[row.from] = row.to
  }
  props.step.mapping = mapping
}

function addRename() {
  renameRows.value.push({ from: '', to: '' })
}

function removeRename(index) {
  renameRows.value.splice(index, 1)
  if (!renameRows.value.length) renameRows.value.push({ from: '', to: '' })
  syncRename()
}
</script>
