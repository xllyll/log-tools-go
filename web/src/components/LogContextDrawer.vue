<template>
  <el-drawer
    v-model="visible"
    :title="drawerTitle"
    size="55%"
    class="log-context-drawer"
  >
    <div class="ctx-toolbar">
      <el-button
        type="primary"
        size="small"
        :disabled="!centerLine || !lines.length"
        @click="goToCenterLine"
      >
        <el-icon><Aim /></el-icon>
        定位当前行
      </el-button>
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

    <div ref="scrollRef" class="ctx-scroll" @scroll.passive="onScroll">
      <div v-if="loadingBefore" class="ctx-scroll-hint">正在加载更早日志…</div>
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
          <span class="log-body" :class="{ 'has-scene-desc': sceneDescList(row).length }">
            <span class="log-flow">
              <span class="log-text">{{ row.display || row.content }}</span>
              <span
                v-for="(tag, di) in sceneDescList(row)"
                :key="`${row.id}-desc-${di}`"
                class="scene-desc"
                :style="sceneDescStyle(tag.color)"
              >{{ tag.desc }}</span>
            </span>
          </span>
        </div>
      </div>
      <div v-if="loadingAfter" class="ctx-scroll-hint">正在加载更晚日志…</div>
      <div v-if="!hasMoreBefore && lines.length" class="ctx-scroll-edge">已到文件开头</div>
      <div v-if="!hasMoreAfter && lines.length" class="ctx-scroll-edge">已到文件末尾</div>
    </div>
  </el-drawer>
</template>

<script setup>
import { computed, nextTick, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Aim } from '@element-plus/icons-vue'
import { api } from '../api'
import { decorateEntries, sceneDescListForEntry, sceneDescStyle } from '../utils/scene'
import { levelColor } from '../utils/logLevel'

const SCROLL_EDGE_PX = 56
const SCROLL_LOAD_CHUNK = 20

function sceneDescList(row) {
  return sceneDescListForEntry(row)
}

const visible = defineModel({ type: Boolean, default: false })

const props = defineProps({
  /** 每次滚动/按钮加载条数 */
  chunk: { type: Number, default: SCROLL_LOAD_CHUNK },
  initialBefore: { type: Number, default: 10 },
  initialAfter: { type: Number, default: 10 },
})

const lines = ref([])
const centerLine = ref(0)
const fileId = ref('')
const scrollRef = ref(null)
let sceneMetaCache = []

const hasMoreBefore = ref(true)
const hasMoreAfter = ref(true)
const loadingBefore = ref(false)
const loadingAfter = ref(false)
let scrollLoadPaused = false

const drawerTitle = computed(() => {
  if (!centerLine.value) return '上下文'
  return `上下文 · 第 ${centerLine.value} 行 · 共 ${lines.value.length} 条`
})

function logLineStyle(row) {
  const lc = levelColor(row.level)
  return {
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

function restoreScrollAfterPrepend(wrap, prevScrollHeight, prevScrollTop) {
  if (!wrap) return
  scrollLoadPaused = true
  wrap.scrollTop = prevScrollTop + (wrap.scrollHeight - prevScrollHeight)
  requestAnimationFrame(() => {
    scrollLoadPaused = false
  })
}

function scrollToCenterLine() {
  nextTick(() => {
    const wrap = scrollRef.value
    const origin = wrap?.querySelector('.ctx-origin')
    if (wrap && origin) {
      scrollLoadPaused = true
      origin.scrollIntoView({ block: 'center', behavior: 'smooth' })
      requestAnimationFrame(() => {
        scrollLoadPaused = false
      })
    }
  })
}

function goToCenterLine() {
  if (!centerLine.value) return
  if (!lines.value.some((r) => r.line === centerLine.value)) {
    ElMessage.warning('当前行尚未加载，请先向上或向下滚动加载')
    return
  }
  scrollToCenterLine()
}

function onScroll(e) {
  if (scrollLoadPaused || !visible.value) return
  const el = e.target
  const { scrollTop, scrollHeight, clientHeight } = el
  const nearTop = scrollTop <= SCROLL_EDGE_PX
  const nearBottom = scrollHeight - scrollTop - clientHeight <= SCROLL_EDGE_PX
  if (nearTop && hasMoreBefore.value && !loadingBefore.value && !loadingAfter.value) {
    loadMoreBefore()
  } else if (nearBottom && hasMoreAfter.value && !loadingBefore.value && !loadingAfter.value) {
    loadMoreAfter()
  }
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
      scrollToCenterLine()
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
  const wrap = scrollRef.value
  const prevScrollHeight = wrap?.scrollHeight ?? 0
  const prevScrollTop = wrap?.scrollTop ?? 0
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
    await nextTick()
    restoreScrollAfterPrepend(wrap, prevScrollHeight, prevScrollTop)
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
  overflow-y: auto;
  overflow-x: hidden;
}

.ctx-scroll-hint,
.ctx-scroll-edge {
  padding: 8px 12px;
  text-align: center;
  font-size: 12px;
  color: var(--app-text-muted);
}

.ctx-scroll-edge {
  padding: 6px 12px;
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
  border-left: 2px solid var(--level-color, var(--app-log-level-info));
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
  overflow: visible;
}

.log-flow {
  display: inline;
  line-height: 1.45;
  word-break: break-all;
}

.log-text {
  display: inline;
  color: var(--level-color, var(--app-log-level-info));
  white-space: pre-wrap;
  word-break: break-all;
}

.log-body.has-scene-desc .log-text {
  font-size: 11px;
  line-height: 1.4;
}

.log-body.has-scene-desc .scene-desc {
  display: inline;
  margin-left: 6px;
  vertical-align: baseline;
}

.ctx-origin {
  background: var(--app-accent-soft) !important;
  box-shadow: inset 3px 0 0 var(--app-accent);
}

.ctx-origin .log-text {
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
