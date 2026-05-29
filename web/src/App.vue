<template>
  <div class="app-shell">
    <header class="topbar">
      <div class="brand">
        <div class="brand-icon">
          <el-icon :size="20"><Document /></el-icon>
        </div>
        <div>
          <h1>FunnyLog</h1>
          <p>BWlC Log analysis tool</p>
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

    <div class="workspace" :class="{ 'is-sidebar-hidden': !sidebarVisible }">
      <aside class="sidebar" :class="{ 'is-collapsed': !sidebarVisible }">
        <div v-if="sidebarVisible" class="sidebar-head">
          <span class="sidebar-head-title">文件与搜索</span>
          <el-tooltip content="收起侧栏" placement="bottom">
            <button
              type="button"
              class="sidebar-icon-btn"
              aria-label="收起侧栏"
              @click="toggleSidebar"
            >
              <el-icon><Fold /></el-icon>
            </button>
          </el-tooltip>
        </div>

        <div v-show="sidebarVisible" class="sidebar-body">
        <div class="sidebar-nav-row">
          <nav class="sidebar-nav" role="tablist" aria-label="侧栏功能">
            <button
              type="button"
              role="tab"
              :aria-selected="leftTab === 'upload'"
              :class="['sidebar-nav-btn', { 'is-active': leftTab === 'upload' }]"
              @click="leftTab = 'upload'"
            >
              <el-icon><Upload /></el-icon>
              <span>上传</span>
            </button>
            <button
              type="button"
              role="tab"
              :aria-selected="leftTab === 'search'"
              :class="['sidebar-nav-btn', { 'is-active': leftTab === 'search' }]"
              @click="leftTab = 'search'"
            >
              <el-icon><Search /></el-icon>
              <span>搜索</span>
            </button>
          </nav>
          <el-tooltip
            :content="toolsPanelVisible ? '收起上传/搜索区域' : '展开上传/搜索区域'"
            placement="bottom"
          >
            <button
              type="button"
              class="sidebar-icon-btn sidebar-tools-toggle"
              :aria-expanded="toolsPanelVisible"
              :aria-label="toolsPanelVisible ? '收起上传/搜索区域' : '展开上传/搜索区域'"
              @click="toggleToolsPanel"
            >
              <el-icon>
                <ArrowUp v-if="toolsPanelVisible" />
                <ArrowDown v-else />
              </el-icon>
            </button>
          </el-tooltip>
        </div>

        <div v-show="toolsPanelVisible" class="sidebar-tools">
          <div v-show="leftTab === 'upload'" class="sidebar-pane">
            <div class="panel-card upload-card">
              <el-upload
                ref="uploadRef"
                drag
                multiple
                :auto-upload="false"
                :show-file-list="true"
                accept=".log,.txt,.json,.zip,.rar,.7z"
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
                  <span class="parse-name">{{ displayFileName(task) }}</span>
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
          </div>

          <div v-show="leftTab === 'search'" class="sidebar-pane">
            <div class="panel-card search-card">
              <el-form label-position="top" size="default">
                <el-form-item>
                  <template #label>
                    <div class="search-kw-label-row">
                      <span>关键词（每行一个，AND）</span>
                      <el-checkbox v-model="useRegex" class="search-regex-check">启用正则匹配</el-checkbox>
                    </div>
                  </template>
                  <el-input v-model="searchKeywords" type="textarea" :rows="2" placeholder="输入关键词或正则..." />
                </el-form-item>
                <el-form-item label="模块 / 场景" class="scene-picker-form-item">
                  <div class="scene-picker-row">
                    <el-select
                      v-model="activeModuleIndex"
                      filterable
                      clearable
                      placeholder="模块"
                      class="scene-picker-module"
                    >
                      <el-option
                        v-for="opt in moduleSelectOptions"
                        :key="opt.value"
                        :label="opt.label"
                        :value="opt.value"
                      />
                    </el-select>
                    <el-select
                      v-model="currentModuleSceneKeys"
                      multiple
                      filterable
                      collapse-tags
                      collapse-tags-tooltip
                      :disabled="activeModuleIndex == null"
                      :placeholder="activeModuleIndex != null ? '场景（可多选，可跨模块）' : '先选模块'"
                      class="scene-picker-scenes"
                    >
                      <el-option
                        v-for="opt in currentModuleSceneOptions"
                        :key="opt.value"
                        :label="opt.label"
                        :value="opt.value"
                      />
                    </el-select>
                  </div>
                </el-form-item>
                <el-button type="primary" :loading="loadingLogs" class="full-btn" @click="searchLogs">
                  <el-icon><Search /></el-icon>
                  查询日志
                </el-button>
              </el-form>
            </div>
          </div>
        </div>

        <div class="file-panel-slot">
          <FileListPanel
            :items="fileItems"
            :list-version="fileListVersion"
            v-model:selected-ids="selectedFileIds"
            @select-change="onFileSelectChange"
            @removed="afterFilesRemoved"
            @ingested="onFileIngested"
            @need-poll="startPolling"
          />
        </div>
        </div>
      </aside>

      <main class="content">
        <el-tooltip
          v-if="!sidebarVisible && !selectedLogFileIds.length"
          content="展开侧栏"
          placement="right"
        >
          <button
            type="button"
            class="sidebar-icon-btn sidebar-restore-btn sidebar-restore-btn--solo"
            aria-label="展开侧栏"
            @click="toggleSidebar"
          >
            <el-icon><Expand /></el-icon>
          </button>
        </el-tooltip>
        <div v-if="!selectedLogFileIds.length" class="empty-state">
          <div class="empty-icon">
            <el-icon :size="48"><DocumentCopy /></el-icon>
          </div>
          <h2>选择或上传日志文件</h2>
          <p>从左侧选择日志文件（未入库也可预览，可多选按顺序展示），或上传新文件</p>
        </div>

        <template v-else>
          <div class="log-toolbar-row">
            <el-tooltip v-if="!sidebarVisible" content="展开侧栏" placement="right">
              <button
                type="button"
                class="sidebar-icon-btn sidebar-restore-btn"
                aria-label="展开侧栏"
                @click="toggleSidebar"
              >
                <el-icon><Expand /></el-icon>
              </button>
            </el-tooltip>
            <LogToolbar
              class="log-toolbar-in-row"
              :file-ids="selectedLogFileIds"
              :files="logFiles"
              :rows="displayLogs"
              :matched-total="logMatchedTotal"
              :truncated="logResultTruncated"
              :loading="loadingLogs"
              v-model:sort-mode="logFileSortMode"
              @refresh="searchLogs"
            />
          </div>

          <div class="log-viewer panel-card">
            <el-scrollbar height="calc(100vh - var(--app-header-h) - 120px)">
              <div class="log-list">
                <div
                  v-for="row in displayLogs"
                  :key="row.id"
                  v-memo="[row.id, row.level, row.scene_desc, row.content, row._fileHeader, isFileCollapsed(row.file_id), searchKeywords, useRegex]"
                  :class="row._fileHeader ? 'log-file-header' : 'log-line'"
                  :style="row._fileHeader ? undefined : logLineStyle(row)"
                  :title="row._fileHeader ? row.file_name : `${row.display || row.content || ''}（双击查看上下文）`"
                  @dblclick="!row._fileHeader && expandContext(row)"
                >
                  <template v-if="row._fileHeader">
                    <el-icon class="log-file-header-icon"><Document /></el-icon>
                    <span class="log-file-header-name">{{ row.file_name }}</span>
                    <el-button
                      v-if="showFileCollapse"
                      link
                      size="small"
                      class="log-file-collapse-btn"
                      :title="isFileCollapsed(row.file_id) ? '展开' : '收起'"
                      @click.stop="toggleFileCollapse(row.file_id)"
                    >
                      <el-icon>
                        <ArrowDown v-if="!isFileCollapsed(row.file_id)" />
                        <ArrowRight v-else />
                      </el-icon>
                    </el-button>
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

        </template>
      </main>
    </div>
    <SceneConfigDialog v-model="sceneDialogVisible" v-model:config="sceneConfig" />
    <JiraSyncDialog v-model="jiraDialogVisible" @imported="onJiraImported" />
    <LogContextDrawer ref="contextDrawerRef" />
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref, shallowRef, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  ArrowDown,
  ArrowRight,
  ArrowUp,
  Collection,
  Document,
  Expand,
  Fold,
  DocumentCopy,
  Link,
  Loading,
  Monitor,
  Moon,
  Search,
  Sunny,
  Upload,
  UploadFilled,
} from '@element-plus/icons-vue'
import SceneConfigDialog from './components/SceneConfigDialog.vue'
import JiraSyncDialog from './components/JiraSyncDialog.vue'
import FileListPanel from './components/FileListPanel.vue'
import LogToolbar from './components/LogToolbar.vue'
import LogContextDrawer from './components/LogContextDrawer.vue'
import { api } from './api'
import { getDeviceId } from './utils/device'
import { applyTheme, getPreferredTheme } from './utils/theme'
import {
  buildModuleSelectOptions,
  buildSceneSelectOptionsForModule,
  cloneSceneConfig,
  collectSceneKeywords,
  decorateEntries,
  defaultSceneConfig,
  pruneSceneKeys,
  saveLocalScene,
  sceneDescStyle,
} from './utils/scene'
import { levelColor } from './utils/logLevel'
import { isProcessing, statusLabel, statusType } from './utils/fileStatus'
import { displayFileName } from './utils/fileDisplay'
import { expandRemovedItemIds, filterLogFileIds } from './utils/fileTree'
import { highlightLogLine } from './utils/highlight'
import { orderLogFileIds } from './utils/logSort'

const SIDEBAR_VISIBLE_KEY = 'log_tools_sidebar_visible'
const TOOLS_PANEL_VISIBLE_KEY = 'log_tools_tools_panel_visible'

function readSidebarVisible() {
  try {
    return localStorage.getItem(SIDEBAR_VISIBLE_KEY) !== '0'
  } catch {
    return true
  }
}

function readToolsPanelVisible() {
  try {
    return localStorage.getItem(TOOLS_PANEL_VISIBLE_KEY) !== '0'
  } catch {
    return true
  }
}

const deviceId = ref(getDeviceId())
const isDark = ref(getPreferredTheme() === 'dark')
const sidebarVisible = ref(readSidebarVisible())
const toolsPanelVisible = ref(readToolsPanelVisible())
const leftTab = ref('upload')
const fileItems = ref([])
const fileListVersion = ref(0)
const logFiles = computed(() => fileItems.value.filter((i) => i.entry_type !== 'folder'))
const selectedFileIds = ref([])
const logs = shallowRef([])
const collapsedFileIds = ref(new Set())
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

const sceneConfig = ref(defaultSceneConfig())
const sceneDialogVisible = ref(false)
const jiraDialogVisible = ref(false)
const activeModuleIndex = ref(null)
const selectedSceneKeys = ref([])
const searchKeywords = ref('')
const useRegex = ref(false)

const contextDrawerRef = ref(null)
let sceneMeta = []

const moduleSelectOptions = computed(() => buildModuleSelectOptions(sceneConfig.value))

const currentModuleSceneOptions = computed(() =>
  buildSceneSelectOptionsForModule(sceneConfig.value, activeModuleIndex.value),
)

/** 第二个下拉只编辑当前模块；selectedSceneKeys 保留其它模块已选场景 */
const currentModuleSceneKeys = computed({
  get() {
    const mi = activeModuleIndex.value
    if (mi == null) return []
    const prefix = `${mi}:`
    return selectedSceneKeys.value.filter((k) => k.startsWith(prefix))
  },
  set(keysForModule) {
    const mi = activeModuleIndex.value
    if (mi == null) return
    const prefix = `${mi}:`
    const other = selectedSceneKeys.value.filter((k) => !k.startsWith(prefix))
    selectedSceneKeys.value = [...other, ...keysForModule]
  },
})

watch(
  sceneConfig,
  (cfg) => {
    selectedSceneKeys.value = pruneSceneKeys(cfg, selectedSceneKeys.value)
    const validMods = new Set(buildModuleSelectOptions(cfg).map((o) => o.value))
    if (activeModuleIndex.value != null && !validMods.has(activeModuleIndex.value)) {
      activeModuleIndex.value = null
    }
  },
  { deep: true },
)

const logFileSortMode = ref('default')

const selectedLogFileIds = computed(() => {
  const ids = filterLogFileIds(selectedFileIds.value, fileItems.value)
  return orderLogFileIds(ids, logFiles.value, logFileSortMode.value)
})

const showFileCollapse = computed(() => selectedLogFileIds.value.length > 1)

const displayLogs = computed(() => {
  const collapsed = collapsedFileIds.value
  const out = []
  let currentFileId = null
  for (const row of logs.value) {
    if (row._fileHeader) {
      currentFileId = row.file_id
      out.push(row)
      continue
    }
    if (currentFileId && collapsed.has(currentFileId)) {
      continue
    }
    out.push(row)
  }
  return out
})

const logMatchedTotal = ref(0)
const logResultTruncated = ref(false)

async function afterFilesRemoved(ids) {
  const removed = expandRemovedItemIds(fileItems.value, ids)
  fileItems.value = fileItems.value.filter((item) => !removed.has(item.id))
  fileListVersion.value += 1
  selectedFileIds.value = selectedFileIds.value.filter((x) => !removed.has(x))
  if (!selectedLogFileIds.value.length) {
    logs.value = []
    logMatchedTotal.value = 0
    logResultTruncated.value = false
  } else {
    scheduleSearchLogs()
  }
  try {
    await loadFiles()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || e.message || '刷新文件列表失败')
  }
}

function toggleTheme() {
  isDark.value = !isDark.value
  applyTheme(isDark.value ? 'dark' : 'light')
}

function toggleSidebar() {
  sidebarVisible.value = !sidebarVisible.value
  try {
    localStorage.setItem(SIDEBAR_VISIBLE_KEY, sidebarVisible.value ? '1' : '0')
  } catch {
    /* ignore */
  }
}

function toggleToolsPanel() {
  toolsPanelVisible.value = !toolsPanelVisible.value
  try {
    localStorage.setItem(TOOLS_PANEL_VISIBLE_KEY, toolsPanelVisible.value ? '1' : '0')
  } catch {
    /* ignore */
  }
}

function onFileSelectChange(ids) {
  const logIds = filterLogFileIds(ids, fileItems.value)
  if (logIds.length) {
    scheduleSearchLogs()
  } else {
    logs.value = []
    logMatchedTotal.value = 0
    logResultTruncated.value = false
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
  for (const f of logFiles.value) {
    const prev = parseTasks.value.find((t) => t.id === f.id)
    if (prev && (prev.status_msg !== f.status_msg || prev.progress !== f.progress || prev.status !== f.status)) {
      appendParseLog(`${displayFileName(f)}: ${f.status_msg || statusLabel(f.status)} (${f.progress || 0}%)`)
    }
  }
  parseTasks.value = logFiles.value.filter((f) => isProcessing(f.status))
}

function startPolling() {
  if (pollTimer) return
  pollTimer = setInterval(async () => {
    await loadFiles()
    syncParseTasks()
    if (!logFiles.value.some((f) => isProcessing(f.status))) stopPolling()
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

function applyFileListPayload(payload) {
  if (Array.isArray(payload?.items)) {
    fileItems.value = [...payload.items]
  } else if (Array.isArray(payload?.files)) {
    const folders = Array.isArray(payload.folders) ? payload.folders : []
    fileItems.value = [
      ...folders.map((f) => ({ ...f, entry_type: 'folder', parent_id: f.parent_folder_id })),
      ...payload.files.map((f) => ({ ...f, entry_type: f.entry_type || 'file', parent_id: f.parent_id ?? f.parent_folder_id })),
    ]
  } else if (Array.isArray(payload)) {
    fileItems.value = payload.map((f) => ({
      ...f,
      entry_type: f.entry_type || 'file',
      parent_id: f.parent_id ?? f.parent_folder_id,
    }))
  } else {
    fileItems.value = []
  }
  fileListVersion.value += 1
  syncParseTasks()
}

async function loadFiles() {
  const { data } = await api.listFiles()
  if (!data?.success) {
    throw new Error(data?.error || '加载文件列表失败')
  }
  applyFileListPayload(data.data)
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
  const logIds = selectedLogFileIds.value
  if (!logIds.length) return
  const seq = ++searchSeq
  loadingLogs.value = true
  logs.value = []
  logMatchedTotal.value = 0
  logResultTruncated.value = false
  try {
    const kws = searchKeywords.value.split('\n').map((s) => s.trim()).filter(Boolean)
    const { specs: sceneSpecs, meta } = collectSceneKeywords(sceneConfig.value, selectedSceneKeys.value)
    sceneMeta = meta
    const order = [...logIds]
    const fileMap = new Map(logFiles.value.map((f) => [f.id, f]))
    const limit = perFileQueryLimit(order.length)
    const baseQuery = {
      keywords: kws,
      scene_keywords: sceneSpecs,
      use_regex: useRegex.value,
      limit,
    }

    const responses = await Promise.all(
      order.map((id) => api.queryLogs({ ...baseQuery, file_id: id }))
    )
    if (seq !== searchSeq) return

    const merged = []
    let truncated = false
    let matchedTotal = 0
    for (let i = 0; i < order.length; i++) {
      const id = order[i]
      const { data } = responses[i]
      if (!data.success) throw new Error(data.error)
      const batch = decorateEntries(data.data?.entries || [], meta)
      matchedTotal += batch.length
      const f = fileMap.get(id)
      merged.push({
        _fileHeader: true,
        id: `header-${id}`,
        file_id: id,
        file_name: f ? displayFileName(f) : id,
      })
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
    logMatchedTotal.value = matchedTotal
    logResultTruncated.value = truncated
    if (truncated) {
      ElMessage.warning(`已选 ${order.length} 个文件，仅展示前 ${MAX_LOG_ROWS} 行，请加关键词缩小范围`)
    }
  } catch (e) {
    if (seq === searchSeq) ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    if (seq === searchSeq) loadingLogs.value = false
  }
}

function isFileCollapsed(fileId) {
  return fileId ? collapsedFileIds.value.has(fileId) : false
}

function toggleFileCollapse(fileId) {
  if (!fileId) return
  const next = new Set(collapsedFileIds.value)
  if (next.has(fileId)) {
    next.delete(fileId)
  } else {
    next.add(fileId)
  }
  collapsedFileIds.value = next
}

watch(logFileSortMode, () => {
  if (selectedLogFileIds.value.length) scheduleSearchLogs()
})

watch(selectedFileIds, (ids) => {
  const allowed = new Set(ids)
  const next = new Set(collapsedFileIds.value)
  let changed = false
  for (const id of next) {
    if (!allowed.has(id)) {
      next.delete(id)
      changed = true
    }
  }
  if (changed) {
    collapsedFileIds.value = next
  }
})

const activeSearchKeywords = computed(() =>
  searchKeywords.value
    .split('\n')
    .map((s) => s.trim())
    .filter(Boolean),
)

function highlightLine(row) {
  const text = row.content || row.message || row.display || ''
  const sceneKw = row.scene_desc ? row.scene_match_keywords || [] : []
  return highlightLogLine(text, activeSearchKeywords.value, useRegex.value, sceneKw)
}

function logLineStyle(row) {
  const lc = levelColor(row.level)
  return {
    '--level-color': lc,
    borderLeftColor: lc,
  }
}

function expandContext(row) {
  contextDrawerRef.value?.openFromRow(row, sceneMeta)
}

async function onJiraImported() {
  await loadFiles()
}

async function initSceneConfig() {
  try {
    const { data } = await api.fetchSharedScene()
    if (data?.success && data.data?.config?.modules?.length) {
      sceneConfig.value = cloneSceneConfig(data.data.config)
      saveLocalScene(sceneConfig.value)
      return
    }
  } catch {
    /* 无服务器配置时使用默认 */
  }
  sceneConfig.value = defaultSceneConfig()
  saveLocalScene(sceneConfig.value)
}

onMounted(async () => {
  await initSceneConfig()
  await loadFiles()
  if (logFiles.value.some((f) => isProcessing(f.status))) startPolling()
})

onUnmounted(() => {
  stopPolling()
  if (searchDebounceTimer) clearTimeout(searchDebounceTimer)
  searchSeq++
})
</script>

<style scoped>
.app-shell {
  height: 100vh;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
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
  overflow: hidden;
}

.sidebar {
  width: var(--app-sidebar-w);
  flex-shrink: 0;
  min-height: 0;
  background: var(--app-surface);
  border-right: 1px solid var(--app-border);
  display: flex;
  flex-direction: column;
  padding: 12px;
  gap: 12px;
  overflow: hidden;
  transition: width 0.22s ease, padding 0.22s ease, border-color 0.22s ease;
}

.sidebar.is-collapsed {
  width: 0;
  min-width: 0;
  padding: 0;
  border-right-color: transparent;
  overflow: hidden;
}

.sidebar-head {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding-bottom: 10px;
  margin-bottom: 2px;
  border-bottom: 1px solid var(--app-border-light);
}

.sidebar-head-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--app-text);
  white-space: nowrap;
}

.sidebar-icon-btn {
  flex-shrink: 0;
  width: 38px;
  height: 38px;
  margin: 0;
  padding: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--app-border-light);
  border-radius: var(--app-radius);
  background: var(--app-bg);
  color: var(--app-text-muted);
  cursor: pointer;
  transition: background 0.15s, color 0.15s, border-color 0.15s;
}

.sidebar-icon-btn:hover {
  color: var(--app-accent);
  border-color: var(--app-accent);
  background: var(--app-accent-soft);
}

.sidebar-icon-btn .el-icon {
  font-size: 18px;
}

.sidebar-head .sidebar-icon-btn {
  margin-left: auto;
}

.sidebar-body {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow: hidden;
}

.sidebar-nav-row {
  flex-shrink: 0;
  display: flex;
  align-items: stretch;
  gap: 6px;
}

.sidebar-nav {
  flex: 1;
  min-width: 0;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px;
  padding: 4px;
  background: var(--app-bg);
  border: 1px solid var(--app-border-light);
  border-radius: var(--app-radius);
}

.sidebar-tools-toggle {
  align-self: stretch;
  width: 38px;
  height: auto;
}

.sidebar-tools-toggle .el-icon {
  font-size: 16px;
}

.sidebar-nav-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 9px 10px;
  border: none;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--app-text-muted);
  font-size: 13px;
  font-weight: 500;
  line-height: 1;
  cursor: pointer;
  transition: background 0.18s ease, color 0.18s ease, box-shadow 0.18s ease;
}

.sidebar-nav-btn .el-icon {
  font-size: 15px;
}

.sidebar-nav-btn:hover {
  color: var(--app-text);
  background: var(--app-surface);
}

.sidebar-nav-btn.is-active {
  background: var(--app-surface);
  color: var(--app-accent);
  font-weight: 600;
  box-shadow: var(--app-shadow);
}

.sidebar-tools {
  flex: 0 1 auto;
  min-height: 0;
  max-height: min(42vh, 360px);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.sidebar-pane {
  min-height: 0;
  overflow-y: auto;
  padding-right: 2px;
}

.sidebar-pane .panel-card:last-child {
  margin-bottom: 0;
}

.sidebar-restore-btn--solo {
  align-self: flex-start;
  margin-bottom: 4px;
}

.log-toolbar-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
  min-width: 0;
}

.log-toolbar-row .sidebar-restore-btn {
  flex-shrink: 0;
}

.log-toolbar-row .log-toolbar-in-row {
  flex: 1;
  min-width: 0;
  margin-bottom: 0;
}

.file-panel-slot {
  flex: 1 1 0;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.panel-card {
  background: var(--app-surface-2);
  border: 1px solid var(--app-border-light);
  border-radius: var(--app-radius);
  padding: 14px;
  margin-bottom: 12px;
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

.search-card :deep(.el-form-item__label) {
  display: flex;
  width: 100%;
  padding-right: 0;
}

.search-kw-label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  width: 100%;
}

.search-regex-check {
  height: auto;
  margin-right: 0;
}

.search-regex-check :deep(.el-checkbox__label) {
  font-size: 12px;
  font-weight: normal;
  color: var(--app-text-secondary);
}

.scene-picker-form-item :deep(.el-form-item__content) {
  flex: 1;
  min-width: 0;
}

.scene-picker-row {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.scene-picker-module {
  flex: 0 0 36%;
  min-width: 0;
}

.scene-picker-scenes {
  flex: 1;
  min-width: 0;
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

.log-file-collapse-btn {
  flex-shrink: 0;
  margin-left: auto;
  padding: 2px 4px;
  color: var(--app-accent);
}

.log-file-collapse-btn:hover {
  color: var(--app-text);
}

.log-list > .log-file-header:first-child {
  margin-top: 0;
}

.log-file-header-icon {
  flex-shrink: 0;
  font-size: 14px;
}

.log-file-header-name {
  flex: 1;
  min-width: 0;
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
  border-left: 2px solid var(--level-color, var(--app-log-level-info));
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

.log-body:not(.has-scene-desc) .log-text {
  flex: 1 1 auto;
}

.log-text {
  min-width: 0;
  color: var(--level-color, var(--app-log-level-info));
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.log-text :deep(strong.scene-kw-bold) {
  font-weight: 700;
}

.log-text :deep(mark.kw-highlight) {
  background: var(--app-kw-highlight-bg);
  color: var(--app-kw-highlight-color);
  padding: 0 1px;
  border-radius: 2px;
  font-weight: 600;
}

.log-text :deep(mark.kw-highlight strong.scene-kw-bold) {
  font-weight: 700;
}

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

</style>
