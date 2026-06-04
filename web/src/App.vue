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
                <div class="upload-hint">支持 .log .txt .zip .rar .7z；分卷包（.part01.rar 等）请一次选全部分卷</div>
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
            <LogSearchPanel
              v-model:keywords="searchKeywords"
              v-model:use-regex="useRegex"
              v-model:keyword-case-sensitive="keywordCaseSensitive"
              v-model:scene-keys="selectedSceneKeys"
              :scene-config="sceneConfig"
              :loading="loadingLogs"
              @search="searchLogs"
            />
          </div>
        </div>

        <button
          type="button"
          class="sidebar-files-entry"
          @click="myFilesDialogVisible = true"
        >
          <el-icon><Folder /></el-icon>
          <span>我的文件</span>
          <el-badge
            v-if="logFiles.length"
            :value="logFiles.length"
            :max="9999"
            type="primary"
            class="sidebar-files-badge"
          />
        </button>
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
              :loaded-count="loadedLogCount"
              :has-more="hasMoreLogs"
              :loading="loadingLogs"
              v-model:sort-mode="logFileSortMode"
              v-model:line-fold="logLineFold"
              @refresh="searchLogs"
            />
          </div>

          <div class="log-viewer panel-card">
            <VirtualLogList
              :rows="displayLogs"
              :line-fold="logLineFold"
              :show-file-collapse="showFileCollapse"
              :collapsed-file-ids="collapsedFileIds"
              :search-keywords="activeSearchKeywords"
              :use-regex="useRegex"
              :keyword-case-sensitive="keywordCaseSensitive"
              :downloading-file-ids="downloadingLogFileIds"
              @load-more="loadMoreForFile"
              @toggle-collapse="toggleFileCollapse"
              @expand-context="expandContext"
              @download-file="onDownloadLogFile"
            />
          </div>

        </template>
      </main>
    </div>
    <SceneConfigDialog v-model="sceneDialogVisible" v-model:config="sceneConfig" />
    <JiraSyncDialog v-model="jiraDialogVisible" @imported="onJiraImported" />
    <MyFilesDialog
      v-model="myFilesDialogVisible"
      :selection-items="fileItems"
      :list-version="fileListVersion"
      v-model:selected-ids="selectedFileIds"
      @select-change="onFileSelectChange"
      @removed="afterFilesRemoved"
      @ingested="onFileIngested"
      @need-poll="startPolling"
      @folders-loaded="onFoldersLoaded"
      @files-loaded="onFilesLoaded"
    />
    <LogContextDrawer ref="contextDrawerRef" />
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  ArrowDown,
  ArrowRight,
  ArrowUp,
  Collection,
  Document,
  Expand,
  Fold,
  Folder,
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
import MyFilesDialog from './components/MyFilesDialog.vue'
import LogToolbar from './components/LogToolbar.vue'
import LogContextDrawer from './components/LogContextDrawer.vue'
import VirtualLogList from './components/VirtualLogList.vue'
import LogSearchPanel from './components/LogSearchPanel.vue'
import { api } from './api'
import { getDeviceId } from './utils/device'
import { applyTheme, getPreferredTheme } from './utils/theme'
import {
  cloneSceneConfig,
  collectSceneKeywords,
  decorateEntries,
  defaultSceneConfig,
  saveLocalScene,
} from './utils/scene'
import { isProcessing, statusLabel, statusType } from './utils/fileStatus'
import { displayFileName } from './utils/fileDisplay'
import { expandRemovedItemIds, filterLogFileIds } from './utils/fileTree'
import { orderLogFileIds } from './utils/logSort'
import { groupUploadFiles } from './utils/archiveVolume'
import { filteredLogFilename, triggerBlobDownload } from './utils/download'

const SIDEBAR_VISIBLE_KEY = 'log_tools_sidebar_visible'
const TOOLS_PANEL_VISIBLE_KEY = 'log_tools_tools_panel_visible'
const LOG_LINE_FOLD_KEY = 'log_tools_log_line_fold'

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

function readLogLineFold() {
  try {
    return localStorage.getItem(LOG_LINE_FOLD_KEY) === '1'
  } catch {
    return false
  }
}

const deviceId = ref(getDeviceId())
const isDark = ref(getPreferredTheme() === 'dark')
const sidebarVisible = ref(readSidebarVisible())
const logLineFold = ref(readLogLineFold())
const toolsPanelVisible = ref(readToolsPanelVisible())
const leftTab = ref('upload')
const myFilesDialogVisible = ref(false)
const folderItems = ref([])
const fileCache = ref({})
const fileItems = computed(() => [
  ...folderItems.value,
  ...Object.values(fileCache.value),
])
const fileListVersion = ref(0)
const logFiles = computed(() => Object.values(fileCache.value))
const selectedFileIds = ref([])
/** @type {import('vue').Ref<Record<string, { entries: unknown[], offset: number, hasMore: boolean, loadingMore: boolean }>>} */
const fileLogData = ref({})
const collapsedFileIds = ref(new Set())
const LOG_PAGE_SIZE = 1000
const LOG_PAGE_SIZE_WHEN_MANY = 100
/** 选中超过 5 个文件时，单次查询 limit 降为 100 */
const LOG_PAGE_SIZE_MANY_THRESHOLD = 5

function logPageSize() {
  return selectedLogFileIds.value.length > LOG_PAGE_SIZE_MANY_THRESHOLD
    ? LOG_PAGE_SIZE_WHEN_MANY
    : LOG_PAGE_SIZE
}
/** @type {Record<string, unknown> | null} */
let logQueryBase = null
let searchSeq = 0
let searchDebounceTimer = null
/** 已拉取过日志的文件 id，用于选择变更时增量查询 */
const fetchedLogFileIds = ref([])
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
const selectedSceneKeys = ref([])
const searchKeywords = ref('')
const downloadingLogFileIds = ref(new Set())
const useRegex = ref(false)
const keywordCaseSensitive = ref(false)

const contextDrawerRef = ref(null)
let sceneMeta = []

const logFileSortMode = ref('default')

const selectedLogFileIds = computed(() => {
  const ids = filterLogFileIds(selectedFileIds.value, fileItems.value)
  return orderLogFileIds(ids, logFiles.value, logFileSortMode.value)
})

const showFileCollapse = computed(() => selectedLogFileIds.value.length > 1)

function makeFileHeader(id, fileMap) {
  const f = fileMap.get(id)
  return {
    _fileHeader: true,
    id: `header-${id}`,
    file_id: id,
    file_name: f ? displayFileName(f) : id,
  }
}

const displayLogs = computed(() => {
  const order = selectedLogFileIds.value
  const fileMap = new Map(logFiles.value.map((f) => [f.id, f]))
  const out = []
  for (const fileId of order) {
    const bucket = fileLogData.value[fileId]
    if (!bucket) continue
    out.push(makeFileHeader(fileId, fileMap))
    if (collapsedFileIds.value.has(fileId)) continue
    out.push(...bucket.entries)
    if (bucket.hasMore) {
      out.push({
        _fileLoadMore: true,
        id: `load-more-${fileId}`,
        file_id: fileId,
        loading: bucket.loadingMore,
      })
    }
  }
  return out
})

const hasMoreLogs = computed(() =>
  Object.values(fileLogData.value).some((b) => b?.hasMore),
)

const loadedLogCount = computed(() =>
  Object.values(fileLogData.value).reduce((n, b) => n + (b?.entries?.length ?? 0), 0),
)

async function fetchFileLogPage(fileId, offset) {
  const limit = logPageSize()
  const { data } = await api.queryLogs({
    ...logQueryBase,
    file_id: fileId,
    limit,
    offset,
  })
  if (!data.success) throw new Error(data.error)
  return decorateEntries(data.data?.entries || [], sceneMeta)
}

async function afterFilesRemoved(ids) {
  const removed = expandRemovedItemIds(fileItems.value, ids)
  folderItems.value = folderItems.value.filter((item) => !removed.has(item.id))
  const nextCache = { ...fileCache.value }
  for (const id of Object.keys(nextCache)) {
    if (removed.has(id)) delete nextCache[id]
  }
  fileCache.value = nextCache
  fileListVersion.value += 1
  selectedFileIds.value = selectedFileIds.value.filter((x) => !removed.has(x))
  pruneFileLogDataForSelection()
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

watch(logLineFold, (fold) => {
  try {
    localStorage.setItem(LOG_LINE_FOLD_KEY, fold ? '1' : '0')
  } catch {
    /* ignore */
  }
})

function onFileSelectChange(ids) {
  const logIds = filterLogFileIds(ids, fileItems.value)
  const prev = fetchedLogFileIds.value

  if (!logIds.length) {
    fileLogData.value = {}
    fetchedLogFileIds.value = []
    return
  }

  const added = logIds.filter((id) => !prev.includes(id))
  const removed = prev.filter((id) => !logIds.includes(id))

  if (removed.length) {
    const next = { ...fileLogData.value }
    for (const id of removed) delete next[id]
    fileLogData.value = next
  }

  fetchedLogFileIds.value = logIds

  if (added.length) {
    scheduleIncrementalLogLoad()
  }
}

function pruneFileLogDataForSelection() {
  const logIds = new Set(selectedLogFileIds.value)
  const next = { ...fileLogData.value }
  for (const id of Object.keys(next)) {
    if (!logIds.has(id)) delete next[id]
  }
  fileLogData.value = next
  fetchedLogFileIds.value = selectedLogFileIds.value
  if (!fetchedLogFileIds.value.length) fileLogData.value = {}
}

function ensureLogQueryBase() {
  const kws = searchKeywords.value
    .split('\n')
    .map((s) => s.trim())
    .filter(Boolean)
  const { specs: sceneSpecs, meta } = collectSceneKeywords(sceneConfig.value, selectedSceneKeys.value)
  sceneMeta = meta
  logQueryBase = {
    keywords: kws,
    scene_keywords: sceneSpecs,
    use_regex: useRegex.value,
    keyword_case_sensitive: keywordCaseSensitive.value,
  }
}

function hasActiveLogFilter() {
  if (!logQueryBase) return false
  const scenes = logQueryBase.scene_keywords
  const hasScenes = Array.isArray(scenes) ? scenes.length > 0 : !!scenes
  return (logQueryBase.keywords?.length > 0) || hasScenes
}

function setFileDownloading(fileId, on) {
  const next = new Set(downloadingLogFileIds.value)
  if (on) next.add(fileId)
  else next.delete(fileId)
  downloadingLogFileIds.value = next
}

async function fetchAllFilteredLogLines(fileId) {
  ensureLogQueryBase()
  const { data } = await api.queryLogs({
    ...logQueryBase,
    file_id: fileId,
    limit: -1,
    offset: 0,
  })
  if (!data.success) throw new Error(data.error)
  const entries = data.data?.entries || []
  return entries.map((e) => e.content || e.message || '').join('\n')
}

async function onDownloadLogFile(fileId) {
  const f = logFiles.value.find((x) => x.id === fileId)
  const displayName = displayFileName(f) || `${fileId}.log`
  setFileDownloading(fileId, true)
  try {
    if (!hasActiveLogFilter()) {
      const res = await api.downloadLogFile(fileId)
      const blob = res.data instanceof Blob ? res.data : new Blob([res.data])
      triggerBlobDownload(blob, displayName)
      ElMessage.success('已开始下载源文件')
      return
    }
    const text = await fetchAllFilteredLogLines(fileId)
    const blob = new Blob([text], { type: 'text/plain;charset=utf-8' })
    triggerBlobDownload(blob, filteredLogFilename(displayName))
    ElMessage.success(`已导出 ${(text.match(/\n/g)?.length ?? 0) + (text ? 1 : 0)} 行匹配日志`)
  } catch (e) {
    const err = e.response?.data
    let msg = e.message
    if (err instanceof Blob) {
      try {
        const j = JSON.parse(await err.text())
        msg = j.error || msg
      } catch {
        /* ignore */
      }
    } else if (err?.error) {
      msg = err.error
    }
    ElMessage.error(msg || '下载失败')
  } finally {
    setFileDownloading(fileId, false)
  }
}

async function onFileIngested() {
  await refreshFileState()
  fileListVersion.value += 1
  startPolling()
}

function appendParseLog(msg) {
  const line = `[${new Date().toLocaleTimeString()}] ${msg}`
  parseLogs.value.unshift(line)
  if (parseLogs.value.length > 50) parseLogs.value.length = 50
}

function mergeFilesIntoCache(files) {
  if (!files?.length) return
  const next = { ...fileCache.value }
  for (const f of files) {
    next[f.id] = { ...f, entry_type: 'file' }
  }
  fileCache.value = next
}

function onFoldersLoaded(folders) {
  const next = (folders || []).map((f) => ({ ...f, entry_type: 'folder' }))
  const prevIds = folderItems.value.map((f) => f.id).join(',')
  const nextIds = next.map((f) => f.id).join(',')
  folderItems.value = next
  if (prevIds !== nextIds) fileListVersion.value += 1
}

function onFilesLoaded(files) {
  mergeFilesIntoCache(files)
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

async function pollProcessingFiles() {
  const { data } = await api.listProcessingFiles()
  if (!data?.success) return
  const files = normalizeFileListPayload(data.data)
  mergeFilesIntoCache(files)
  syncParseTasks()
}

function startPolling() {
  if (pollTimer) return
  pollTimer = setInterval(async () => {
    try {
      await pollProcessingFiles()
    } catch {
      /* ignore */
    }
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

function normalizeFileListPayload(payload) {
  if (Array.isArray(payload?.items)) {
    return payload.items.map((f) => ({
      ...f,
      entry_type: f.entry_type || 'file',
      parent_id: f.parent_id ?? f.parent_folder_id,
    }))
  }
  if (Array.isArray(payload?.files)) {
    const folders = Array.isArray(payload.folders) ? payload.folders : []
    return [
      ...folders.map((f) => ({ ...f, entry_type: 'folder', parent_id: f.parent_folder_id })),
      ...payload.files.map((f) => ({
        ...f,
        entry_type: f.entry_type || 'file',
        parent_id: f.parent_id ?? f.parent_folder_id,
      })),
    ]
  }
  if (Array.isArray(payload)) {
    return payload.map((f) => ({
      ...f,
      entry_type: f.entry_type || 'file',
      parent_id: f.parent_id ?? f.parent_folder_id,
    }))
  }
  return []
}

async function loadFolders() {
  const { data } = await api.listFolders()
  if (!data?.success) throw new Error(data?.error || '加载文件夹失败')
  onFoldersLoaded(normalizeFileListPayload(data.data))
}

async function refreshFileState() {
  await loadFolders()
  try {
    await pollProcessingFiles()
  } catch {
    /* ignore */
  }
}

async function doUpload() {
  if (!pendingFiles.value.length) {
    ElMessage.warning('请先选择文件')
    return
  }
  const jobs = groupUploadFiles(pendingFiles.value)
  const incomplete = jobs.find((j) => j.incomplete)
  if (incomplete) {
    ElMessage.warning(`分卷压缩包「${incomplete.label}」需同时选择全部分卷后再上传`)
    return
  }
  uploading.value = true
  uploadProgress.value = 0
  parseLogs.value = []
  try {
    const total = jobs.length
    for (let i = 0; i < total; i++) {
      const job = jobs[i]
      if (job.isVolumeGroup) {
        appendParseLog(`上传分卷: ${job.label}（${job.files.length} 个文件）`)
        const { data } = await api.uploadVolume(job.files, (e) => {
          const single = e.total ? Math.round((e.loaded / e.total) * 100) : 0
          uploadProgress.value = Math.round(((i + single / 100) / total) * 100)
        })
        if (data.file_ids?.length) {
          data.file_ids.forEach((id) => appendParseLog(`已上传: ${id.slice(0, 8)}…`))
        }
      } else {
        const f = job.files[0]
        appendParseLog(`上传文件: ${f.name}`)
        const { data } = await api.upload(f, (e) => {
          const single = e.total ? Math.round((e.loaded / e.total) * 100) : 0
          uploadProgress.value = Math.round(((i + single / 100) / total) * 100)
        })
        if (data.file_ids?.length) {
          data.file_ids.forEach((id) => appendParseLog(`已上传: ${id.slice(0, 8)}…`))
        }
      }
    }
    ElMessage.success('上传成功，可选择文件预览或点击入库')
    pendingFiles.value = []
    uploadRef.value?.clearFiles()
    uploadProgress.value = 100
    await refreshFileState()
    fileListVersion.value += 1
  } catch (e) {
    ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    uploading.value = false
  }
}

function scheduleIncrementalLogLoad() {
  if (searchDebounceTimer) clearTimeout(searchDebounceTimer)
  searchDebounceTimer = setTimeout(() => {
    searchDebounceTimer = null
    const need = selectedLogFileIds.value.filter((id) => !fileLogData.value[id])
    if (need.length) loadLogsForFiles(need, false)
  }, 350)
}

/** @param {string[]} fileIds @param {boolean} replace 是否替换现有日志缓存（手动「查询」时为 true） */
async function loadLogsForFiles(fileIds, replace) {
  if (!fileIds.length) return
  const seq = ++searchSeq
  loadingLogs.value = true
  ensureLogQueryBase()
  try {
    const pageLimit = logPageSize()
    const results = await Promise.all(
      fileIds.map(async (id) => {
        const batch = await fetchFileLogPage(id, 0)
        if (seq !== searchSeq) return null
        return { id, batch }
      }),
    )
    if (seq !== searchSeq) return
    const next = replace ? {} : { ...fileLogData.value }
    for (const item of results) {
      if (!item) return
      next[item.id] = {
        entries: item.batch,
        offset: item.batch.length,
        hasMore: item.batch.length >= pageLimit,
        loadingMore: false,
      }
    }
    fileLogData.value = next
  } catch (e) {
    if (seq === searchSeq) ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    if (seq === searchSeq) loadingLogs.value = false
  }
}

/** 按当前关键字/场景重新查询所有已选文件 */
async function searchLogs() {
  const logIds = selectedLogFileIds.value
  if (!logIds.length) return
  fetchedLogFileIds.value = logIds
  await loadLogsForFiles(logIds, true)
}

async function loadMoreForFile(fileId) {
  const bucket = fileLogData.value[fileId]
  if (!bucket?.hasMore || bucket.loadingMore || loadingLogs.value || !logQueryBase) return
  fileLogData.value = {
    ...fileLogData.value,
    [fileId]: { ...bucket, loadingMore: true },
  }
  try {
    const pageLimit = logPageSize()
    const batch = await fetchFileLogPage(fileId, bucket.offset)
    const prev = fileLogData.value[fileId]
    fileLogData.value = {
      ...fileLogData.value,
      [fileId]: {
        entries: [...prev.entries, ...batch],
        offset: prev.offset + batch.length,
        hasMore: batch.length >= pageLimit,
        loadingMore: false,
      },
    }
  } catch (e) {
    fileLogData.value = {
      ...fileLogData.value,
      [fileId]: { ...fileLogData.value[fileId], loadingMore: false },
    }
    ElMessage.error(e.response?.data?.error || e.message)
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

function expandContext(row) {
  contextDrawerRef.value?.openFromRow(row, sceneMeta)
}

async function onJiraImported() {
  await refreshFileState()
  fileListVersion.value += 1
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
  await refreshFileState()
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
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.sidebar-files-entry {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  padding: 11px 12px;
  border: 1px solid var(--app-border-light);
  border-radius: var(--app-radius);
  background: var(--app-surface-2);
  color: var(--app-text);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, color 0.15s;
}

.sidebar-files-entry:hover {
  border-color: var(--app-accent);
  color: var(--app-accent);
  background: var(--app-accent-soft);
}

.sidebar-files-entry .el-icon {
  font-size: 16px;
  color: var(--app-accent);
}

.sidebar-files-badge {
  margin-left: auto;
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
  min-height: 0;
  height: calc(100vh - var(--app-header-h) - 120px);
  padding: 0;
  overflow: hidden;
  background: var(--app-log-bg);
}

</style>
