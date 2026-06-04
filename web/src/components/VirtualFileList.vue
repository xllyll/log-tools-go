<template>
  <div ref="containerRef" class="virtual-file-list" @scroll="onScroll">
    <div class="virtual-file-list-phantom" :style="{ height: `${totalHeight}px` }">
      <div
        v-for="row in visibleRows"
        :key="row.file.id"
        class="virtual-file-list-row"
        :style="{ transform: `translateY(${row.offset}px)` }"
      >
        <div
          :class="['file-item', { active: selectedSet.has(row.file.id) }]"
          @click="onItemClick(row.file, $event)"
        >
          <span v-if="orderMap.get(row.file.id)" class="file-order">{{ orderMap.get(row.file.id) }}</span>
          <div class="file-item-body">
            <div class="file-row">
              <el-icon class="file-icon"><Document /></el-icon>
              <div class="file-meta">
                <span class="file-name" :title="row.label">{{ row.label }}</span>
                <span class="file-sub">{{ row.sub }}</span>
              </div>
            </div>
          </div>
          <div class="file-item-side" @click.stop>
            <el-dropdown trigger="click" placement="bottom-end" @command="(cmd) => emit('command', cmd, row.file)">
              <button type="button" class="file-more-btn" aria-label="更多操作">
                <el-icon><MoreFilled /></el-icon>
              </button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item v-if="canIngest(row.file)" command="ingest">
                    {{ row.file.status === 'failed' ? '重新入库' : '入库' }}
                  </el-dropdown-item>
                  <el-dropdown-item command="delete" :divided="canIngest(row.file)">
                    <span class="menu-danger">删除</span>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </div>
    </div>
    <div v-if="!files.length" class="file-empty">{{ emptyText }}</div>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { Document, MoreFilled } from '@element-plus/icons-vue'
import { displayFileName } from '../utils/fileDisplay'
import { canIngest } from '../utils/fileStatus'

const ROW_HEIGHT = 50
const OVERSCAN = 8

const props = defineProps({
  files: { type: Array, default: () => [] },
  selectedIds: { type: Array, default: () => [] },
  fileSelectOrder: { type: Object, default: () => new Map() },
  emptyText: { type: String, default: '此目录下暂无文件' },
})

const emit = defineEmits(['toggle', 'command'])

function onItemClick(file, event) {
  emit('toggle', file, event)
}

const containerRef = ref(null)
const scrollTop = ref(0)
const viewHeight = ref(400)

const selectedSet = computed(() => new Set(props.selectedIds))
const orderMap = computed(() => props.fileSelectOrder)

const rows = computed(() =>
  props.files.map((file) => ({
    file,
    label: displayFileName(file),
    sub: formatSub(file),
  })),
)

const totalHeight = computed(() => rows.value.length * ROW_HEIGHT)

const visibleRows = computed(() => {
  const list = rows.value
  if (!list.length) return []
  const start = Math.max(0, Math.floor(scrollTop.value / ROW_HEIGHT) - OVERSCAN)
  const visibleCount = Math.ceil(viewHeight.value / ROW_HEIGHT) + OVERSCAN * 2
  const end = Math.min(list.length, start + visibleCount)
  return list.slice(start, end).map((row, i) => ({
    ...row,
    offset: (start + i) * ROW_HEIGHT,
  }))
})

function formatSub(file) {
  const size = formatSize(file.size)
  const time = file.upload_at
    ? new Date(file.upload_at).toLocaleString('zh-CN', {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
      })
    : ''
  return `${size} · ${time}`
}

function formatSize(bytes) {
  if (!bytes) return '-'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function onScroll() {
  if (containerRef.value) scrollTop.value = containerRef.value.scrollTop
}

function measure() {
  if (containerRef.value) viewHeight.value = containerRef.value.clientHeight || 400
}

let ro
onMounted(() => {
  measure()
  ro = new ResizeObserver(measure)
  if (containerRef.value) ro.observe(containerRef.value)
})

onUnmounted(() => {
  ro?.disconnect()
})

watch(
  () => props.files.length,
  () => {
    scrollTop.value = 0
    if (containerRef.value) containerRef.value.scrollTop = 0
    nextTick(measure)
  },
)
</script>

<style scoped>
.virtual-file-list {
  flex: 1;
  min-height: 0;
  height: 100%;
  overflow-y: auto;
  overflow-x: hidden;
  position: relative;
}

.virtual-file-list-phantom {
  position: relative;
  width: 100%;
}

.virtual-file-list-row {
  position: absolute;
  left: 0;
  right: 0;
  height: 50px;
}

.file-empty {
  text-align: center;
  padding: 24px 12px;
  font-size: 12px;
  color: var(--app-text-muted);
}

.file-item {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: 6px;
  height: 50px;
  box-sizing: border-box;
  padding: 5px 6px 5px 8px;
  border-radius: var(--app-radius-sm);
  cursor: pointer;
  border: 1px solid transparent;
  transition: background 0.15s, border-color 0.15s;
}

.file-item:hover {
  background: var(--app-accent-soft);
}

.file-item.active {
  background: var(--app-accent-soft);
  border-color: var(--app-accent);
}

.file-order {
  position: absolute;
  top: 4px;
  left: 4px;
  z-index: 2;
  pointer-events: none;
  width: 13px;
  height: 13px;
  border-radius: 3px;
  background: var(--app-accent);
  color: #fff;
  font-size: 9px;
  font-weight: 700;
  line-height: 13px;
  text-align: center;
}

.file-item-body {
  flex: 1;
  min-width: 0;
}

.file-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.file-icon {
  flex-shrink: 0;
  font-size: 14px;
  color: var(--app-accent);
}

.file-meta {
  min-width: 0;
  flex: 1;
}

.file-name {
  display: block;
  font-size: 12px;
  font-weight: 500;
  line-height: 1.3;
  color: var(--app-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-sub {
  display: block;
  font-size: 10px;
  line-height: 1.3;
  color: var(--app-text-muted);
  margin-top: 1px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-item-side {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  padding-top: 1px;
}

.file-more-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--app-text-muted);
  cursor: pointer;
}

.file-more-btn:hover {
  background: var(--app-border-light);
  color: var(--app-text);
}

.menu-danger {
  color: var(--el-color-danger);
}
</style>
