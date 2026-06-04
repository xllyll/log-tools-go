<template>
  <div class="selected-items-list">
    <div v-if="!entries.length" class="selected-empty">未选择文件或文件夹</div>
    <el-scrollbar v-else class="selected-scroll">
      <div
        v-for="row in entries"
        :key="`${row.type}-${row.id}`"
        class="selected-row"
      >
        <el-icon class="selected-icon">
          <Folder v-if="row.type === 'folder'" />
          <Document v-else />
        </el-icon>
        <div class="selected-meta">
          <span class="selected-name" :title="row.label">{{ row.label }}</span>
          <span class="selected-type">{{ row.type === 'folder' ? '文件夹' : row.sub }}</span>
        </div>
        <button
          type="button"
          class="selected-remove"
          aria-label="移出已选"
          @click="emit('remove', row.id)"
        >
          <el-icon><Close /></el-icon>
        </button>
      </div>
    </el-scrollbar>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Close, Document, Folder } from '@element-plus/icons-vue'
import { displayFileName } from '../utils/fileDisplay'
import { statusLabel } from '../utils/fileStatus'

const props = defineProps({
  items: { type: Array, default: () => [] },
  selectedIds: { type: Array, default: () => [] },
})

const emit = defineEmits(['remove'])

const itemById = computed(() => new Map((props.items || []).map((i) => [i.id, i])))

const entries = computed(() => {
  const out = []
  for (const id of props.selectedIds) {
    const item = itemById.value.get(id)
    if (!item) continue
    if (item.entry_type === 'folder') {
      out.push({
        id,
        type: 'folder',
        label: item.name || item.original_name || id,
        sub: '文件夹',
      })
    } else {
      out.push({
        id,
        type: 'file',
        label: displayFileName(item),
        sub: statusLabel(item.status),
      })
    }
  }
  return out
})
</script>

<style scoped>
.selected-items-list {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.selected-scroll {
  flex: 1;
  min-height: 0;
}

.selected-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  font-size: 12px;
  color: var(--app-text-muted);
  text-align: center;
}

.selected-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--app-border-light);
}

.selected-row:last-child {
  border-bottom: none;
}

.selected-icon {
  flex-shrink: 0;
  font-size: 14px;
  color: var(--app-accent);
}

.selected-meta {
  flex: 1;
  min-width: 0;
}

.selected-name {
  display: block;
  font-size: 12px;
  font-weight: 500;
  color: var(--app-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.selected-type {
  display: block;
  font-size: 10px;
  color: var(--app-text-muted);
  margin-top: 2px;
}

.selected-remove {
  flex-shrink: 0;
  width: 22px;
  height: 22px;
  padding: 0;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--app-text-muted);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.selected-remove:hover {
  background: var(--el-color-danger-light-9);
  color: var(--el-color-danger);
}
</style>
