<template>
  <div class="log-toolbar panel-card">
    <div class="log-toolbar-left">
      <el-icon><Document /></el-icon>
      <span class="log-filename" :title="filesLabel">{{ filesLabel }}</span>
    </div>
    <div class="log-toolbar-right">
      <div v-if="showSort" class="log-sort">
        <span class="log-sort-label">排序</span>
        <el-select
          :model-value="sortMode"
          size="small"
          class="log-sort-select"
          @update:model-value="onSortChange"
        >
          <el-option
            v-for="opt in LOG_FILE_SORT_OPTIONS"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </el-select>
      </div>
      <el-tag v-if="loading" effect="plain" round>查询中…</el-tag>
      <el-tag v-else effect="plain" round :title="countTitle">{{ countLabel }}</el-tag>
      <div class="log-line-fold">
        <span class="log-line-fold-label">折叠长行</span>
        <el-switch
          :model-value="lineFold"
          size="small"
          @update:model-value="emit('update:lineFold', $event)"
        />
      </div>
      <el-button size="small" :loading="loading" @click="emit('refresh')">
        <el-icon><Refresh /></el-icon>
        刷新
      </el-button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Document, Refresh } from '@element-plus/icons-vue'
import { displayFileName } from '../utils/fileDisplay'
import { LOG_FILE_SORT_OPTIONS } from '../utils/logSort'

const props = defineProps({
  fileIds: { type: Array, default: () => [] },
  files: { type: Array, default: () => [] },
  /** 与日志列表 v-for 相同的数据源 */
  rows: { type: Array, default: () => [] },
  hasMore: { type: Boolean, default: false },
  loading: { type: Boolean, default: false },
  sortMode: { type: String, default: 'default' },
  /** 过长日志单行省略 */
  lineFold: { type: Boolean, default: false },
})

const emit = defineEmits(['refresh', 'update:sortMode', 'update:lineFold'])

const showSort = computed(() => props.fileIds.length > 1)

const lineCount = computed(() => props.rows.filter((r) => !r._fileHeader).length)

const countLabel = computed(() => {
  const n = lineCount.value
  const files = props.fileIds.length
  if (n === 0 && files > 0) {
    return files > 1 ? `已选 ${files} 个文件，无匹配日志` : '无匹配日志'
  }
  if (props.hasMore) {
    return files > 1 ? `${files} 个文件 · 已加载 ${n} 条` : `已加载 ${n} 条`
  }
  if (files > 1) {
    return `${files} 个文件 · ${n} 条日志`
  }
  return `${n} 条日志`
})

const countTitle = computed(() => {
  if (lineCount.value === 0) return ''
  if (props.hasMore) {
    return '列表下方可加载更多日志'
  }
  return `当前列表共 ${lineCount.value} 行日志（不含文件标题行）`
})

const filesLabel = computed(() => {
  const names = props.fileIds
    .map((id) => {
      const f = props.files.find((x) => x.id === id)
      return f ? displayFileName(f) : ''
    })
    .filter(Boolean)
  if (!names.length) return ''
  if (names.length === 1) return names[0]
  return names.join(' → ')
})

function onSortChange(mode) {
  emit('update:sortMode', mode)
}
</script>

<style scoped>
.log-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0;
  padding: 12px 16px;
}

.log-toolbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex: 1;
}

.log-filename {
  font-size: 14px;
  font-weight: 600;
  color: var(--app-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.log-toolbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.log-sort {
  display: flex;
  align-items: center;
  gap: 6px;
}

.log-sort-label {
  font-size: 12px;
  color: var(--app-text-muted);
  white-space: nowrap;
}

.log-sort-select {
  width: 128px;
}

.log-line-fold {
  display: flex;
  align-items: center;
  gap: 6px;
}

.log-line-fold-label {
  font-size: 12px;
  color: var(--app-text-muted);
  white-space: nowrap;
}
</style>
