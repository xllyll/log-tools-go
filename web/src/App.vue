<template>
  <div class="app-shell">
    <header class="topbar">
      <div class="brand">
        <div class="brand-icon">
          <el-icon :size="20"><Document /></el-icon>
        </div>
        <div>
          <h1>BWIC日志分析</h1>
          <p>Log Tools · 上传 · 检索 · 场景匹配</p>
        </div>
      </div>
      <div class="topbar-actions">
        <el-tag effect="plain" round size="small" class="device-tag">
          <el-icon><Monitor /></el-icon>
          {{ deviceId.slice(0, 8) }}…
        </el-tag>
        <el-tooltip :content="isDark ? '切换浅色' : '切换深色'" placement="bottom">
          <el-button circle class="theme-btn" @click="toggleTheme">
            <el-icon><Sunny v-if="isDark" /><Moon v-else /></el-icon>
          </el-button>
        </el-tooltip>
        <el-tooltip content="场景配置" placement="bottom">
          <el-button circle class="theme-btn" @click="sceneDialogVisible = true">
            <el-icon><Collection /></el-icon>
          </el-button>
        </el-tooltip>
        <el-tooltip content="Jira 同步" placement="bottom">
          <el-button circle class="theme-btn" @click="jiraDialogVisible = true">
            <el-icon><Link /></el-icon>
          </el-button>
        </el-tooltip>
      </div>
    </header>

    <div class="workspace">
      <aside class="sidebar">
        <el-tabs v-model="leftTab" class="sidebar-tabs" stretch>
          <el-tab-pane label="上传" name="upload">
            <div class="panel-card upload-card">
              <el-upload
                ref="uploadRef"
                drag
                multiple
                :auto-upload="false"
                :show-file-list="true"
                accept=".log,.txt,.zip,.rar,.7z"
                :on-change="onFileChange"
                class="upload-zone"
              >
                <el-icon class="upload-icon"><UploadFilled /></el-icon>
                <div class="upload-title">拖拽日志到此处</div>
                <div class="upload-hint">支持 .log .txt .zip .rar .7z</div>
              </el-upload>
              <el-button type="primary" size="large" :loading="uploading" class="full-btn" @click="doUpload">
                <el-icon><Upload /></el-icon>
                开始上传
              </el-button>
              <el-progress v-if="uploading" :percentage="uploadProgress" :stroke-width="8" striped striped-flow />
            </div>

            <div v-if="parseTasks.length" class="panel-card parse-card">
              <div class="card-title">
                <el-icon><Loading /></el-icon>
                入库任务
              </div>
              <div v-for="task in parseTasks" :key="task.id" class="parse-task">
                <div class="parse-task-head">
                  <span class="parse-name">{{ task.name }}</span>
                  <el-tag size="small" :type="statusType(task.status)" effect="light">{{ statusLabel(task.status) }}</el-tag>
                </div>
                <el-progress
                  :percentage="task.progress || 0"
                  :status="task.status === 'failed' ? 'exception' : task.status === 'ready' ? 'success' : undefined"
                  :stroke-width="6"
                />
                <div class="parse-msg">{{ task.status_msg || '等待中...' }}</div>
              </div>
              <el-scrollbar max-height="100px" class="parse-log-scroll">
                <div v-for="(line, idx) in parseLogs" :key="idx" class="parse-log-line">{{ line }}</div>
              </el-scrollbar>
            </div>
          </el-tab-pane>

          <el-tab-pane label="搜索" name="search">
            <div class="panel-card">
              <el-form label-position="top" size="default">
                <el-form-item label="关键词（每行一个，AND）">
                  <el-input v-model="searchKeywords" type="textarea" :rows="4" placeholder="输入关键词或正则..." />
                </el-form-item>
                <el-form-item>
                  <el-checkbox v-model="useRegex">启用正则匹配</el-checkbox>
                </el-form-item>
                <el-form-item label="场景（OR 组合）">
                  <el-select
                    v-model="selectedSceneKeys"
                    multiple
                    filterable
                    collapse-tags
                    collapse-tags-tooltip
                    placeholder="选择场景（按模块分组）"
                    class="full-width"
                  >
                    <el-option-group
                      v-for="group in sceneSelectGroups"
                      :key="group.moduleName"
                      :label="group.moduleName"
                    >
                      <el-option
                        v-for="opt in group.options"
                        :key="opt.key"
                        :label="opt.label"
                        :value="opt.key"
                      />
                    </el-option-group>
                  </el-select>
                </el-form-item>
                <el-button type="primary" :loading="loadingLogs" class="full-btn" @click="searchLogs">
                  <el-icon><Search /></el-icon>
                  查询日志
                </el-button>
              </el-form>
            </div>
          </el-tab-pane>
        </el-tabs>

        <FileListPanel
          :files="files"
          v-model:selected-ids="selectedFileIds"
          @select-change="onFileSelectChange"
          @removed="afterFilesRemoved"
          @ingested="onFileIngested"
          @need-poll="startPolling"
        />
      </aside>

      <main class="content">
        <div v-if="!selectedFileIds.length" class="empty-state">
          <div class="empty-icon">
            <el-icon :size="48"><DocumentCopy /></el-icon>
          </div>
          <h2>选择或上传日志文件</h2>
          <p>从左侧选择日志文件（未入库也可预览，可多选按顺序展示），或上传新文件</p>
        </div>

        <template v-else>
          <div class="log-toolbar panel-card">
            <div class="log-toolbar-left">
              <el-icon><Document /></el-icon>
              <span class="log-filename" :title="selectedFilesLabel">{{ selectedFilesLabel }}</span>
            </div>
            <div class="log-toolbar-right">
              <el-tag effect="plain" round>{{ logs.length }} 条记录</el-tag>
              <el-button size="small" :loading="loadingLogs" @click="searchLogs">
                <el-icon><Refresh /></el-icon>
                刷新
              </el-button>
            </div>
          </div>

          <div class="log-viewer panel-card">
            <el-scrollbar height="calc(100vh - var(--app-header-h) - 120px)">
              <div class="log-list">
                <div
                  v-for="row in logs"
                  :key="row.id"
                  v-memo="[row.id, row.scene_desc, row.content, row._fileHeader]"
                  :class="row._fileHeader ? 'log-file-header' : 'log-line'"
                  :style="row._fileHeader ? undefined : logLineStyle(row)"
                  :title="row._fileHeader ? row.file_name : `${row.display || row.content || ''}（双击查看上下文）`"
                  @dblclick="!row._fileHeader && expandContext(row)"
                >
                  <template v-if="row._fileHeader">
                    <el-icon class="log-file-header-icon"><Document /></el-icon>
                    <span class="log-file-header-name">{{ row.file_name }}</span>
                  </template>
                  <template v-else>
                    <span class="ln">{{ row.line }}</span>
                    <span class="log-body" :class="{ 'has-scene-desc': !!row.scene_desc }">
                      <span class="log-text" v-html="highlightLine(row)"></span>
                      <span
                        v-if="row.scene_desc"
                        class="scene-desc"
                        :style="sceneDescStyle(row.color)"
                      >{{ row.scene_desc }}</span>
                    </span>
                  </template>
                </div>
              </div>
            </el-scrollbar>
          </div>

          <el-drawer v-model="ctxOpen" title="上下文 · 前后 10 条" size="55%" class="ctx-drawer">
            <div class="log-list ctx-list">
              <div
                v-for="row in ctxLines"
                :key="row.id"
                :class="['log-line', 'ctx', { 'ctx-origin': row.line === ctxCenterLine }]"
                :style="logLineStyle(row)"
                :title="row.display || row.content"
              >
                <span v-if="row.line === ctxCenterLine" class="ctx-origin-tag">当前</span>
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
          </el-drawer>
        </template>
      </main>
    </div>
    <SceneConfigDialog v-model="sceneDialogVisible" v-model:config="sceneConfig" />
    <JiraSyncDialog v-model="jiraDialogVisible" @imported="onJiraImported" />
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref, shallowRef } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Collection,
  Document,
  DocumentCopy,
  Link,
  Loading,
  Monitor,
  Moon,
  Refresh,
  Search,
  Sunny,
  Upload,
  UploadFilled,
} from '@element-plus/icons-vue'
import SceneConfigDialog from './components/SceneConfigDialog.vue'
import JiraSyncDialog from './components/JiraSyncDialog.vue'
import FileListPanel from './components/FileListPanel.vue'
import { api } from './api'
import { getDeviceId } from './utils/device'
import { applyTheme, getPreferredTheme } from './utils/theme'
import {
  buildSceneSelectGroups,
  collectSceneKeywords,
  decorateEntries,
  loadLocalScene,
  sceneDescStyle,
} from './utils/scene'
import { levelColor } from './utils/logLevel'
import { isProcessing, statusLabel, statusType } from './utils/fileStatus'

const deviceId = ref(getDeviceId())
const isDark = ref(getPreferredTheme() === 'dark')
const leftTab = ref('upload')
const files = ref([])
const selectedFileIds = ref([])
const logs = shallowRef([])
const MAX_LOG_ROWS = 10000
let searchSeq = 0
let searchDebounceTimer = null
const loadingLogs = ref(false)
const uploading = ref(false)
const uploadProgress = ref(0)
const uploadRef = ref(null)
const pendingFiles = ref([])
const parseTasks = ref([])
const parseLogs = ref([])
let pollTimer = null

const sceneConfig = ref(loadLocalScene())
const sceneDialogVisible = ref(false)
const jiraDialogVisible = ref(false)
const selectedSceneKeys = ref([])
const searchKeywords = ref('')
const useRegex = ref(false)

const ctxOpen = ref(false)
const ctxLines = ref([])
const ctxCenterLine = ref(0)
let sceneMeta = []

const sceneSelectGroups = computed(() => buildSceneSelectGroups(sceneConfig.value))

const selectedFilesLabel = computed(() => {
  const names = selectedFileIds.value
    .map((id) => files.value.find((f) => f.id === id)?.name)
    .filter(Boolean)
  if (!names.length) return ''
  if (names.length === 1) return names[0]
  return names.join(' → ')
})

async function afterFilesRemoved(ids) {
  const removed = new Set(ids)
  selectedFileIds.value = selectedFileIds.value.filter((x) => !removed.has(x))
  if (!selectedFileIds.value.length) {
    logs.value = []
  } else {
    scheduleSearchLogs()
  }
  await loadFiles()
}

function toggleTheme() {
  isDark.value = !isDark.value
  applyTheme(isDark.value ? 'dark' : 'light')
}

function onFileSelectChange(ids) {
  if (ids.length) {
    scheduleSearchLogs()
  } else {
    logs.value = []
  }
}

async function onFileIngested() {
  await loadFiles()
  startPolling()
}

function appendParseLog(msg) {
  const line = `[${new Date().toLocaleTimeString()}] ${msg}`
  parseLogs.value.unshift(line)
  if (parseLogs.value.length > 50) parseLogs.value.length = 50
}

function syncParseTasks() {
  for (const f of files.value) {
    const prev = parseTasks.value.find((t) => t.id === f.id)
    if (prev && (prev.status_msg !== f.status_msg || prev.progress !== f.progress || prev.status !== f.status)) {
      appendParseLog(`${f.name}: ${f.status_msg || statusLabel(f.status)} (${f.progress || 0}%)`)
    }
  }
  parseTasks.value = files.value.filter((f) => isProcessing(f.status))
}

function startPolling() {
  if (pollTimer) return
  pollTimer = setInterval(async () => {
    await loadFiles()
    syncParseTasks()
    if (!files.value.some((f) => isProcessing(f.status))) stopPolling()
  }, 1500)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function onFileChange(_file, list) {
  pendingFiles.value = list.map((x) => x.raw).filter(Boolean)
}

async function loadFiles() {
  const { data } = await api.listFiles()
  if (data.success) {
    files.value = data.data || []
    syncParseTasks()
  }
}

async function doUpload() {
  if (!pendingFiles.value.length) {
    ElMessage.warning('请先选择文件')
    return
  }
  uploading.value = true
  uploadProgress.value = 0
  parseLogs.value = []
  try {
    const total = pendingFiles.value.length
    for (let i = 0; i < total; i++) {
      const f = pendingFiles.value[i]
      appendParseLog(`上传文件: ${f.name}`)
      const { data } = await api.upload(f, (e) => {
        const single = e.total ? Math.round((e.loaded / e.total) * 100) : 0
        uploadProgress.value = Math.round(((i + single / 100) / total) * 100)
      })
      if (data.file_ids?.length) {
        data.file_ids.forEach((id) => appendParseLog(`已上传: ${id.slice(0, 8)}…`))
      }
    }
    ElMessage.success('上传成功，可选择文件预览或点击入库')
    pendingFiles.value = []
    uploadRef.value?.clearFiles()
    uploadProgress.value = 100
    await loadFiles()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    uploading.value = false
  }
}

function scheduleSearchLogs() {
  if (searchDebounceTimer) clearTimeout(searchDebounceTimer)
  searchDebounceTimer = setTimeout(() => {
    searchDebounceTimer = null
    searchLogs()
  }, 350)
}

function perFileQueryLimit(fileCount) {
  if (fileCount <= 1) return 5000
  return Math.min(3000, Math.max(400, Math.floor(6000 / fileCount)))
}

async function searchLogs() {
  if (!selectedFileIds.value.length) return
  const seq = ++searchSeq
  loadingLogs.value = true
  logs.value = []
  try {
    const kws = searchKeywords.value.split('\n').map((s) => s.trim()).filter(Boolean)
    const { keywords: sceneKw, meta } = collectSceneKeywords(sceneConfig.value, selectedSceneKeys.value)
    sceneMeta = meta
    const order = [...selectedFileIds.value]
    const fileMap = new Map(files.value.map((f) => [f.id, f]))
    const limit = perFileQueryLimit(order.length)
    const baseQuery = {
      keywords: kws,
      scene_keywords: sceneKw,
      use_regex: useRegex.value,
      limit,
    }

    const responses = await Promise.all(
      order.map((id) => api.queryLogs({ ...baseQuery, file_id: id }))
    )
    if (seq !== searchSeq) return

    const merged = []
    let truncated = false
    for (let i = 0; i < order.length; i++) {
      const id = order[i]
      const { data } = responses[i]
      if (!data.success) throw new Error(data.error)
      const f = fileMap.get(id)
      merged.push({
        _fileHeader: true,
        id: `header-${id}`,
        file_name: f?.name || id,
      })
      const batch = decorateEntries(data.data?.entries || [], meta)
      const room = MAX_LOG_ROWS - merged.length
      if (batch.length >= room) {
        merged.push(...batch.slice(0, room))
        truncated = true
        break
      }
      merged.push(...batch)
    }

    if (seq !== searchSeq) return
    await nextTick()
    if (seq !== searchSeq) return
    logs.value = merged
    if (truncated) {
      ElMessage.warning(`已选 ${order.length} 个文件，仅展示前 ${MAX_LOG_ROWS} 行，请加关键词缩小范围`)
    }
  } catch (e) {
    if (seq === searchSeq) ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    if (seq === searchSeq) loadingLogs.value = false
  }
}

function highlightLine(row) {
  return (row.content || row.message || row.display || '').replace(/</g, '&lt;')
}

function logLineStyle(row) {
  const lc = levelColor(row.level)
  return {
    '--line-color': row.color || 'inherit',
    '--level-color': lc,
    borderLeftColor: lc,
  }
}

async function expandContext(row) {
  if (!row.file_id) return
  ctxCenterLine.value = row.line
  const { data } = await api.logContext({
    file_id: row.file_id,
    line: row.line,
    before: 10,
    after: 10,
  })
  if (data.success) {
    ctxLines.value = decorateEntries(data.data || [], sceneMeta)
    ctxOpen.value = true
  }
}

async function onJiraImported() {
  await loadFiles()
}

onMounted(async () => {
  await loadFiles()
  if (files.value.some((f) => isProcessing(f.status))) startPolling()
})

onUnmounted(() => {
  stopPolling()
  if (searchDebounceTimer) clearTimeout(searchDebounceTimer)
  searchSeq++
})
</script>

<style scoped>
.app-shell {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--app-bg);
}

.topbar {
  height: var(--app-header-h);
  padding: 0 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--app-surface);
  border-bottom: 1px solid var(--app-border);
  box-shadow: var(--app-shadow);
  position: sticky;
  top: 0;
  z-index: 100;
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
}

.brand-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: var(--app-accent-soft);
  color: var(--app-accent);
  display: flex;
  align-items: center;
  justify-content: center;
}

.brand h1 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--app-text);
  line-height: 1.3;
}

.brand p {
  margin: 0;
  font-size: 11px;
  color: var(--app-text-muted);
}

.topbar-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.device-tag {
  display: flex;
  align-items: center;
  gap: 4px;
}

.theme-btn {
  border-color: var(--app-border);
}

.workspace {
  flex: 1;
  display: flex;
  min-height: 0;
}

.sidebar {
  width: var(--app-sidebar-w);
  flex-shrink: 0;
  background: var(--app-surface);
  border-right: 1px solid var(--app-border);
  display: flex;
  flex-direction: column;
  padding: 12px;
  gap: 12px;
  overflow: hidden;
}

.sidebar-tabs {
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.sidebar-tabs :deep(.el-tabs__content) {
  flex: 1;
  overflow-y: auto;
  padding-right: 4px;
}

.sidebar-tabs :deep(.el-tabs__header) {
  margin-bottom: 12px;
}

.panel-card {
  background: var(--app-surface-2);
  border: 1px solid var(--app-border-light);
  border-radius: var(--app-radius);
  padding: 14px;
  margin-bottom: 12px;
}
.file-panel{
  flex: 1;
}
.card-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--app-text);
  margin-bottom: 12px;
}

.card-title .el-icon {
  margin-right: 4px;
  vertical-align: -2px;
}

.full-btn {
  width: 100%;
  margin-top: 10px;
}

.full-width {
  width: 100%;
}

.upload-zone :deep(.el-upload-dragger) {
  border-radius: var(--app-radius-sm);
  border-color: var(--app-border);
  background: var(--app-surface);
  padding: 24px 16px;
  transition: border-color 0.2s, background 0.2s;
}

.upload-zone :deep(.el-upload-dragger:hover) {
  border-color: var(--app-accent);
  background: var(--app-accent-soft);
}

.upload-icon {
  font-size: 40px;
  color: var(--app-accent);
  margin-bottom: 8px;
}

.upload-title {
  font-size: 14px;
  color: var(--app-text);
  font-weight: 500;
}

.upload-hint {
  font-size: 12px;
  color: var(--app-text-muted);
  margin-top: 4px;
}

.parse-task {
  margin-bottom: 12px;
}

.parse-task-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
  font-size: 12px;
}

.parse-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 220px;
  color: var(--app-text-secondary);
}

.parse-msg {
  font-size: 11px;
  color: var(--app-text-muted);
  margin-top: 4px;
}

.parse-log-scroll {
  margin-top: 8px;
  border-top: 1px solid var(--app-border-light);
  padding-top: 8px;
}

.parse-log-line {
  font-family: 'Cascadia Code', Consolas, monospace;
  font-size: 11px;
  color: var(--app-text-muted);
  line-height: 1.5;
  padding: 2px 0;
}

.log-file-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px 6px 4px;
  margin-top: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--app-accent);
  background: var(--app-accent-soft);
  border-top: 1px solid var(--app-border-light);
  border-bottom: 1px solid var(--app-border-light);
  cursor: default;
  user-select: none;
}

.log-list > .log-file-header:first-child {
  margin-top: 0;
}

.log-file-header-icon {
  flex-shrink: 0;
  font-size: 14px;
}

.log-file-header-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.content {
  flex: 1;
  padding: 16px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 40px;
}

.empty-icon {
  width: 88px;
  height: 88px;
  border-radius: 50%;
  background: var(--app-accent-soft);
  color: var(--app-accent);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 20px;
}

.empty-state h2 {
  margin: 0 0 8px;
  font-size: 18px;
  color: var(--app-text);
  font-weight: 600;
}

.empty-state p {
  margin: 0;
  font-size: 14px;
  color: var(--app-text-muted);
  max-width: 360px;
}

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

.log-viewer {
  flex: 1;
  padding: 0;
  overflow: hidden;
  background: var(--app-log-bg);
}

.log-list {
  padding: 4px 0;
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
  border-left: 2px solid var(--level-color, #3fb950);
  transition: background 0.1s;
  overflow: hidden;
  max-width: 100%;
}

.log-line:hover {
  background: var(--app-log-hover);
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

/* 无 desc：正文占满一行并可省略 */
.log-body:not(.has-scene-desc) .log-text {
  flex: 1 1 auto;
}

.log-text {
  min-width: 0;
  color: var(--line-color, var(--app-text-secondary));
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 有 desc：正文紧跟内容后、过长则压缩省略；desc 紧贴正文右侧，不顶到行尾 */
.log-body.has-scene-desc {
  gap: 6px;
}

.log-body.has-scene-desc .log-text {
  flex: 0 1 auto;
  max-width: 100%;
  font-size: 11px;
  line-height: 1.4;
}

.log-body.has-scene-desc .scene-desc {
  flex: 0 0 auto;
}

.ctx-list .log-line {
  cursor: default;
  align-items: center;
  overflow: hidden;
  white-space: nowrap;
}

.ctx-list .log-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ctx-list .log-body {
  flex-wrap: nowrap;
  justify-content: flex-start;
  overflow: hidden;
}

.ctx-list .log-body:not(.has-scene-desc) .log-text {
  flex: 1 1 auto;
}

.ctx-list .log-body.has-scene-desc .log-text {
  flex: 0 1 auto;
  max-width: 100%;
  font-size: 11px;
}

.ctx-list .ln {
  line-height: 1.45;
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
