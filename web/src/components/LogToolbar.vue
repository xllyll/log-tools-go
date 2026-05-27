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
      <el-tag effect="plain" round>{{ logCount }} 条记录</el-tag>
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
  logCount: { type: Number, default: 0 },
  loading: { type: Boolean, default: false },
  sortMode: { type: String, default: 'default' },
})

const emit = defineEmits(['refresh', 'update:sortMode'])

const showSort = computed(() => props.fileIds.length > 1)

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
</style>
