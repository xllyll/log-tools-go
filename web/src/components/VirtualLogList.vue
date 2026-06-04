<template>
  <div ref="containerRef" class="virtual-log-list" @scroll="onScroll">
    <div class="virtual-log-list-phantom" :style="{ height: `${layout.totalHeight}px` }">
      <div
        v-for="item in visibleItems"
        :key="item.row.id"
        class="virtual-log-list-row"
        :style="{ transform: `translateY(${item.offset}px)` }"
      >
        <div :ref="(el) => bindRowMeasure(el, item.row)">
          <div
            :class="rowClass(item.row)"
            :style="item.row._fileHeader || item.row._fileLoadMore ? undefined : logLineStyle(item.row)"
            :title="rowTitle(item.row)"
            @dblclick="onRowDblClick(item.row)"
          >
            <template v-if="item.row._fileHeader">
              <el-icon class="log-file-header-icon"><Document /></el-icon>
              <span class="log-file-header-name">
                <span v-if="filePathPrefix(item.row.file_name)" class="log-file-path">{{
                  filePathPrefix(item.row.file_name)
                }}</span><span class="log-file-basename">{{ fileBaseName(item.row.file_name) }}</span>
              </span>
              <div class="log-file-header-actions">
                <el-button
                  link
                  size="small"
                  class="log-file-download-btn"
                  :loading="isFileDownloading(item.row.file_id)"
                  title="下载日志文件"
                  @click.stop="emit('download-file', item.row.file_id)"
                >
                  <el-icon v-if="!isFileDownloading(item.row.file_id)"><Download /></el-icon>
                </el-button>
                <el-button
                  v-if="showFileCollapse"
                  link
                  size="small"
                  class="log-file-collapse-btn"
                  :title="isFileCollapsed(item.row.file_id) ? '展开' : '收起'"
                  @click.stop="emit('toggle-collapse', item.row.file_id)"
                >
                  <el-icon>
                    <ArrowDown v-if="!isFileCollapsed(item.row.file_id)" />
                    <ArrowRight v-else />
                  </el-icon>
                </el-button>
              </div>
            </template>
            <template v-else-if="item.row._fileLoadMore">
              <el-button
                size="small"
                :loading="item.row.loading"
                @click="emit('load-more', item.row.file_id)"
              >
                加载更多
              </el-button>
            </template>
            <template v-else>
              <span class="ln">{{ item.row.line }}</span>
              <span
                class="log-body"
                :class="{ 'has-scene-desc': sceneDescList(item.row).length, 'is-content-wrap': !lineFold }"
              >
                <span class="log-flow" :class="{ 'is-wrap': !lineFold }">
                  <span
                    class="log-text"
                    :class="{ 'is-wrap': !lineFold }"
                    v-html="highlightRow(item.row)"
                  />
                  <span
                    v-for="(tag, di) in sceneDescList(item.row)"
                    :key="`${item.row.id}-desc-${di}`"
                    class="scene-desc"
                    :style="sceneDescStyle(tag.color)"
                  >{{ tag.desc }}</span>
                </span>
              </span>
            </template>
          </div>
        </div>
      </div>
    </div>
    <div v-if="!rows.length" class="virtual-log-empty">暂无日志，请选择文件并查询</div>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { ArrowDown, ArrowRight, Document, Download } from '@element-plus/icons-vue'
import { highlightLogLine } from '../utils/highlight'
import { levelColor } from '../utils/logLevel'
import { sceneDescListForEntry, sceneDescStyle } from '../utils/scene'

const OVERSCAN = 10
const LOG_LINE_PX = Math.ceil(12 * 1.45)
const MIN_LOG_H = 22
const MIN_HEADER_H = 32
const MIN_LOAD_MORE_H = 36

const props = defineProps({
  rows: { type: Array, default: () => [] },
  lineFold: { type: Boolean, default: true },
  showFileCollapse: { type: Boolean, default: false },
  collapsedFileIds: { type: Object, default: () => new Set() },
  searchKeywords: { type: Array, default: () => [] },
  useRegex: { type: Boolean, default: false },
  keywordCaseSensitive: { type: Boolean, default: false },
  /** Set<string> 正在下载的 file_id */
  downloadingFileIds: { type: Object, default: () => new Set() },
})

const emit = defineEmits(['load-more', 'toggle-collapse', 'expand-context', 'download-file'])

const containerRef = ref(null)
const scrollTop = ref(0)
const viewHeight = ref(500)
const listWidth = ref(800)
/** row.id -> 实测高度（px） */
const measuredHeights = ref(new Map())
const layoutEpoch = ref(0)

const rowObservers = new Map()
let layoutFlush = 0

function estimateLogLineHeight(row) {
  if (props.lineFold) return MIN_LOG_H
  const text = row.content || row.message || row.display || ''
  const bodyWidth = Math.max(160, listWidth.value - 52)
  const charsPerLine = Math.max(12, Math.floor(bodyWidth / 7.2))
  const lines = Math.max(1, Math.ceil(text.length / charsPerLine))
  const descCount = sceneDescListForEntry(row).length
  const sceneExtra = descCount ? Math.min(88, 20 + descCount * 18) : 0
  return Math.max(MIN_LOG_H, 4 + lines * LOG_LINE_PX + sceneExtra)
}

function estimateRowHeight(row) {
  if (row._fileHeader) return MIN_HEADER_H
  if (row._fileLoadMore) return MIN_LOAD_MORE_H
  return estimateLogLineHeight(row)
}

function getRowHeight(row) {
  const measured = measuredHeights.value.get(row.id)
  if (measured != null && measured > 0) return measured
  return estimateRowHeight(row)
}

const layout = computed(() => {
  layoutEpoch.value
  const list = props.rows || []
  const offsets = new Array(list.length + 1)
  offsets[0] = 0
  for (let i = 0; i < list.length; i++) {
    offsets[i + 1] = offsets[i] + getRowHeight(list[i])
  }
  return { offsets, totalHeight: offsets[list.length] || 0, count: list.length }
})

function findStartIndex(scroll) {
  const offsets = layout.value.offsets
  const n = layout.value.count
  if (n === 0) return 0
  let lo = 0
  let hi = n - 1
  while (lo < hi) {
    const mid = (lo + hi + 1) >> 1
    if (offsets[mid] <= scroll) lo = mid
    else hi = mid - 1
  }
  return lo
}

function findEndIndex(scrollBottom) {
  const offsets = layout.value.offsets
  const n = layout.value.count
  let lo = 0
  let hi = n
  while (lo < hi) {
    const mid = (lo + hi) >> 1
    if (offsets[mid] < scrollBottom) lo = mid + 1
    else hi = mid
  }
  return lo
}

const visibleItems = computed(() => {
  layoutEpoch.value
  const list = props.rows || []
  const n = list.length
  if (!n) return []
  const { offsets } = layout.value
  const start = Math.max(0, findStartIndex(scrollTop.value) - OVERSCAN)
  const end = Math.min(n, findEndIndex(scrollTop.value + viewHeight.value) + OVERSCAN)
  const out = []
  for (let i = start; i < end; i++) {
    out.push({ row: list[i], offset: offsets[i] })
  }
  return out
})

function scheduleLayoutFlush() {
  if (layoutFlush) return
  layoutFlush = requestAnimationFrame(() => {
    layoutFlush = 0
    layoutEpoch.value += 1
  })
}

function bindRowMeasure(el, row) {
  const id = row?.id
  if (!id) return
  if (!el) {
    const obs = rowObservers.get(id)
    if (obs) {
      obs.disconnect()
      rowObservers.delete(id)
    }
    return
  }
  const apply = () => {
    const h = Math.ceil(el.getBoundingClientRect().height)
    if (h <= 0) return
    if (measuredHeights.value.get(id) !== h) {
      const next = new Map(measuredHeights.value)
      next.set(id, h)
      measuredHeights.value = next
      scheduleLayoutFlush()
    }
  }
  apply()
  let obs = rowObservers.get(id)
  if (!obs) {
    obs = new ResizeObserver(apply)
    rowObservers.set(id, obs)
  }
  obs.observe(el)
}

function clearMeasuredHeights() {
  for (const obs of rowObservers.values()) obs.disconnect()
  rowObservers.clear()
  measuredHeights.value = new Map()
  layoutEpoch.value += 1
}

function isFileCollapsed(fileId) {
  return fileId ? props.collapsedFileIds.has(fileId) : false
}

function isFileDownloading(fileId) {
  return fileId ? props.downloadingFileIds.has(fileId) : false
}

function filePathPrefix(fullName) {
  const n = fullName || ''
  const i = n.lastIndexOf('/')
  if (i < 0) return ''
  return n.slice(0, i + 1)
}

function fileBaseName(fullName) {
  const n = fullName || ''
  const i = n.lastIndexOf('/')
  return i < 0 ? n : n.slice(i + 1)
}

function rowClass(row) {
  if (row._fileHeader) return 'log-file-header'
  if (row._fileLoadMore) return 'log-load-more-row'
  return ['log-line', { 'is-content-wrap': !props.lineFold }]
}

function logLineStyle(row) {
  const lc = levelColor(row.level)
  return {
    '--level-color': lc,
    borderLeftColor: lc,
  }
}

function rowTitle(row) {
  if (row._fileHeader) return row.file_name
  if (row._fileLoadMore) return undefined
  return `${row.display || row.content || ''}（双击查看上下文）`
}

function sceneDescList(row) {
  return sceneDescListForEntry(row)
}

function highlightRow(row) {
  const text = row.content || row.message || row.display || ''
  const sceneKw = sceneDescList(row).length ? row.scene_match_keywords || [] : []
  return highlightLogLine(text, props.searchKeywords, props.useRegex, sceneKw, props.keywordCaseSensitive)
}

function onRowDblClick(row) {
  if (!row._fileHeader && !row._fileLoadMore) emit('expand-context', row)
}

function onScroll() {
  if (containerRef.value) scrollTop.value = containerRef.value.scrollTop
}

function measureViewport() {
  if (containerRef.value) {
    viewHeight.value = containerRef.value.clientHeight || 500
    listWidth.value = containerRef.value.clientWidth || 800
  }
}

let containerRo
onMounted(() => {
  measureViewport()
  containerRo = new ResizeObserver(measureViewport)
  if (containerRef.value) containerRo.observe(containerRef.value)
})

onUnmounted(() => {
  containerRo?.disconnect()
  clearMeasuredHeights()
  if (layoutFlush) cancelAnimationFrame(layoutFlush)
})

watch(
  () => props.lineFold,
  () => {
    clearMeasuredHeights()
    scrollTop.value = 0
    if (containerRef.value) containerRef.value.scrollTop = 0
    nextTick(measureViewport)
  },
)

watch(
  () => `${props.useRegex}:${props.keywordCaseSensitive}:${props.searchKeywords.join('\n')}`,
  () => {
    clearMeasuredHeights()
    nextTick(measureViewport)
  },
)

watch(
  () => props.rows.length,
  (len, prev) => {
    if (len === 0) {
      clearMeasuredHeights()
      scrollTop.value = 0
      if (containerRef.value) containerRef.value.scrollTop = 0
      return
    }
    if (prev != null && len < prev) {
      const valid = new Set((props.rows || []).map((r) => r.id))
      const next = new Map()
      for (const [id, h] of measuredHeights.value) {
        if (valid.has(id)) next.set(id, h)
      }
      measuredHeights.value = next
      layoutEpoch.value += 1
    }
    nextTick(measureViewport)
  },
)
</script>

<style scoped>
.virtual-log-list {
  height: 100%;
  overflow-y: auto;
  overflow-x: hidden;
  position: relative;
  background: var(--app-log-bg);
}

.virtual-log-list-phantom {
  position: relative;
  width: 100%;
}

.virtual-log-list-row {
  position: absolute;
  left: 0;
  right: 0;
  width: 100%;
}

.virtual-log-empty {
  padding: 48px 16px;
  text-align: center;
  font-size: 13px;
  color: var(--app-text-muted);
}

.log-file-header {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 6px 12px 4px 4px;
  margin-top: 6px;
  font-size: 12px;
  font-weight: 500;
  color: var(--app-text-secondary);
  background: var(--app-accent-soft);
  border-top: 1px solid var(--app-border-light);
  border-bottom: 1px solid var(--app-border-light);
  cursor: default;
  user-select: none;
}

.virtual-log-list-row:first-child .log-file-header {
  margin-top: 0;
}

.log-file-header-actions {
  flex-shrink: 0;
  margin-left: auto;
  margin-top: 2px;
  display: flex;
  align-items: center;
  gap: 2px;
}

.log-file-download-btn,
.log-file-collapse-btn {
  flex-shrink: 0;
  padding: 2px 4px;
  color: var(--app-accent);
}

.log-file-header-icon {
  flex-shrink: 0;
  font-size: 14px;
  margin-top: 2px;
}

.log-file-header-name {
  flex: 1;
  min-width: 0;
  white-space: normal;
  word-break: break-all;
  line-height: 1.4;
}

.log-file-path {
  color: var(--app-text-muted);
  font-weight: 500;
}

.log-file-basename {
  color: var(--app-accent);
  font-weight: 600;
}

.log-load-more-row {
  display: flex;
  justify-content: center;
  padding: 8px 12px 12px;
}

.log-line {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 3px 12px 3px 2px;
  font-family: 'Cascadia Code', Consolas, 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.45;
  cursor: pointer;
  border-left: 2px solid var(--level-color, var(--app-log-level-info));
  transition: background 0.1s;
  max-width: 100%;
  box-sizing: border-box;
}

.log-line:hover {
  background: var(--app-log-hover);
}

.log-line.is-content-wrap {
  align-items: flex-start;
  overflow: visible;
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
  flex-wrap: nowrap;
  align-items: center;
  justify-content: flex-start;
  overflow: hidden;
}

.log-line.is-content-wrap .log-body:not(.has-scene-desc) {
  flex-wrap: wrap;
  align-items: flex-start;
  overflow: visible;
}

.log-body:not(.has-scene-desc) .log-text {
  flex: 1 1 auto;
}

/* 换行时 log 与 desc 同一行内流排列，desc 跟在最后一行文字后 */
.log-line.is-content-wrap .log-body.has-scene-desc {
  display: block;
  overflow: visible;
}

.log-line.is-content-wrap .log-body.has-scene-desc .log-flow {
  display: inline;
  line-height: 1.45;
  word-break: break-all;
}

.log-line.is-content-wrap .log-body.has-scene-desc .log-text {
  display: inline;
  max-width: none;
}

.log-line.is-content-wrap .log-body.has-scene-desc .log-text.is-wrap {
  white-space: normal;
  word-break: break-all;
}

.log-line.is-content-wrap .log-body.has-scene-desc .scene-desc {
  display: inline;
  margin-left: 6px;
  vertical-align: baseline;
}

.log-line.is-content-wrap .log-body.has-scene-desc .scene-desc + .scene-desc {
  margin-left: 4px;
}

.log-text {
  min-width: 0;
  color: var(--level-color, var(--app-log-level-info));
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.log-text.is-wrap {
  overflow: visible;
  text-overflow: unset;
  white-space: normal;
  word-break: break-all;
}

.log-text :deep(strong.scene-kw-bold) {
  font-weight: bold;
  font-size: 12px;
}

.log-text :deep(mark.kw-highlight) {
  background: var(--app-kw-highlight-bg);
  color: var(--app-kw-highlight-color);
  padding: 0 1px;
  border-radius: 2px;
  font-weight: 600;
}

/* 折叠单行：flex 横排，desc 在右侧 */
.log-body.has-scene-desc:not(.is-content-wrap) {
  gap: 6px;
}

.log-body.has-scene-desc:not(.is-content-wrap) .log-flow {
  display: flex;
  flex: 1 1 auto;
  min-width: 0;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px 6px;
}

.log-body.has-scene-desc:not(.is-content-wrap) .log-text {
  flex: 1 1 auto;
  min-width: 0;
  max-width: 100%;
  font-size: 11px;
  line-height: 1.4;
}

.log-body.has-scene-desc:not(.is-content-wrap) .scene-desc {
  flex: 0 0 auto;
}

.log-body.has-scene-desc.is-content-wrap .log-text {
  font-size: 11px;
  line-height: 1.4;
}
</style>
