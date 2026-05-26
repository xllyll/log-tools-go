<template>
  <el-drawer
    v-model="visible"
    :title="drawerTitle"
    size="55%"
    class="log-context-drawer"
  >
    <div class="ctx-toolbar">
      <el-button
        size="small"
        :loading="loadingBefore"
        :disabled="!hasMoreBefore || loadingAfter"
        @click="loadMoreBefore"
      >
        加载更早 {{ chunk }} 条
      </el-button>
      <el-button
        size="small"
        :loading="loadingAfter"
        :disabled="!hasMoreAfter || loadingBefore"
        @click="loadMoreAfter"
      >
        加载更晚 {{ chunk }} 条
      </el-button>
    </div>

    <el-scrollbar class="ctx-scroll">
      <div class="log-list ctx-list">
        <div
          v-for="row in lines"
          :key="row.id"
          :class="['log-line', 'ctx', { 'ctx-origin': row.line === centerLine }]"
          :style="logLineStyle(row)"
          :title="row.display || row.content"
        >
          <span v-if="row.line === centerLine" class="ctx-origin-tag">当前</span>
          <span class="ln">{{ row.line }}</span>
          <span class="log-body" :class="{ 'has-scene-desc': !!row.scene_desc }">
            <span class="log-text">{{ row.display || row.content }}</span>
            <span
              v-if="row.scene_desc"
              class="scene-desc"
              :style="sceneDescStyle(row.color)"
            >{{ row.scene_desc }}</span>
          </span>
        </div>
      </div>
    </el-scrollbar>
  </el-drawer>
</template>

<script setup>
import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import { decorateEntries, sceneDescStyle } from '../utils/scene'
import { levelColor } from '../utils/logLevel'

const visible = defineModel({ type: Boolean, default: false })

const props = defineProps({
  chunk: { type: Number, default: 10 },
  initialBefore: { type: Number, default: 10 },
  initialAfter: { type: Number, default: 10 },
})

const lines = ref([])
const centerLine = ref(0)
const fileId = ref('')
let sceneMetaCache = []

const hasMoreBefore = ref(true)
const hasMoreAfter = ref(true)
const loadingBefore = ref(false)
const loadingAfter = ref(false)

const drawerTitle = computed(() => {
  if (!centerLine.value) return '上下文'
  return `上下文 · 第 ${centerLine.value} 行 · 共 ${lines.value.length} 条`
})

function logLineStyle(row) {
  const lc = levelColor(row.level)
  return {
    '--line-color': row.color || 'inherit',
    '--level-color': lc,
    borderLeftColor: lc,
  }
}

function mergeLines(existing, incoming, position) {
  const seen = new Set(existing.map((r) => r.id))
  const unique = incoming.filter((r) => !seen.has(r.id))
  if (!unique.length) return existing
  const merged = position === 'before' ? [...unique, ...existing] : [...existing, ...unique]
  merged.sort((a, b) => a.line - b.line)
  return merged
}

async function fetchContext(line, before, after) {
  const { data } = await api.logContext({
    file_id: fileId.value,
    line,
    before,
    after,
  })
  if (!data.success) {
    throw new Error(data.error || '加载上下文失败')
  }
  return decorateEntries(data.data || [], sceneMetaCache)
}

async function openFromRow(row, sceneMeta = []) {
  if (!row?.file_id) return
  fileId.value = row.file_id
  sceneMetaCache = sceneMeta
  centerLine.value = row.line
  hasMoreBefore.value = row.line > 1
  hasMoreAfter.value = true
  loadingBefore.value = false
  loadingAfter.value = false
  try {
    lines.value = await fetchContext(row.line, props.initialBefore, props.initialAfter)
    visible.value = true
    if (lines.value.length === 0) {
      hasMoreBefore.value = false
      hasMoreAfter.value = false
    } else {
      hasMoreBefore.value = lines.value[0].line > 1
      hasMoreAfter.value = true
    }
  } catch (e) {
    ElMessage.error(e.response?.data?.error || e.message)
  }
}

async function loadMoreBefore() {
  if (!fileId.value || !lines.value.length || loadingBefore.value) return
  const firstLine = lines.value[0].line
  if (firstLine <= 1) {
    hasMoreBefore.value = false
    return
  }
  const anchor = firstLine - 1
  loadingBefore.value = true
  try {
    const batch = await fetchContext(anchor, props.chunk - 1, 0)
    if (!batch.length) {
      hasMoreBefore.value = false
      return
    }
    const merged = mergeLines(lines.value, batch, 'before')
    if (merged.length === lines.value.length) {
      hasMoreBefore.value = false
      return
    }
    lines.value = merged
    if (lines.value[0].line <= 1) {
      hasMoreBefore.value = false
    }
  } catch (e) {
    ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    loadingBefore.value = false
  }
}

async function loadMoreAfter() {
  if (!fileId.value || !lines.value.length || loadingAfter.value) return
  const lastLine = lines.value[lines.value.length - 1].line
  const anchor = lastLine + 1
  loadingAfter.value = true
  try {
    const batch = await fetchContext(anchor, 0, props.chunk - 1)
    if (!batch.length) {
      hasMoreAfter.value = false
      return
    }
    const merged = mergeLines(lines.value, batch, 'after')
    if (merged.length === lines.value.length) {
      hasMoreAfter.value = false
      return
    }
    lines.value = merged
    if (batch.length < props.chunk) {
      hasMoreAfter.value = false
    }
  } catch (e) {
    ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    loadingAfter.value = false
  }
}

defineExpose({ openFromRow })
</script>

<style scoped>
.ctx-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}

.ctx-scroll {
  height: calc(100vh - var(--app-header-h) - 140px);
}

.log-list {
  padding: 4px 0;
}

.log-line {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  padding: 6px 12px 6px 2px;
  font-family: 'Cascadia Code', Consolas, 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.45;
  cursor: default;
  border-left: 2px solid var(--level-color, #3fb950);
  max-width: 100%;
}

.log-line .ln {
  flex-shrink: 0;
  min-width: 28px;
  padding-right: 2px;
  text-align: right;
  color: var(--app-log-gutter);
  user-select: none;
  font-size: 11px;
  line-height: 1.45;
}

.log-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
}

.log-text {
  width: 100%;
  color: var(--line-color, var(--app-text-secondary));
  white-space: pre-wrap;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.log-body.has-scene-desc .log-text {
  font-size: 11px;
  line-height: 1.4;
}

.log-body.has-scene-desc .scene-desc {
  align-self: flex-start;
}

.ctx-origin {
  background: var(--app-accent-soft) !important;
  box-shadow: inset 3px 0 0 var(--app-accent);
}

.ctx-origin .log-text {
  color: var(--app-accent) !important;
  font-weight: 600;
}

.ctx-origin-tag {
  flex-shrink: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Microsoft YaHei', sans-serif;
  font-size: 12px;
  font-weight: 500;
  color: #fff;
  background: var(--app-accent);
  padding: 2px 8px;
  border-radius: var(--app-radius-sm);
  line-height: 1.35;
  user-select: none;
  -webkit-font-smoothing: antialiased;
}
</style>
