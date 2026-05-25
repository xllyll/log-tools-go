<template>
  <div class="app-shell">
    <header class="topbar">
      <div class="brand">
        <div class="brand-icon">
          <el-icon :size="20"><Document /></el-icon>
        </div>
        <div>
          <h1>车机日志分析</h1>
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
                解析任务
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
                  <el-select v-model="selectedScenes" multiple placeholder="选择场景" class="full-width">
                    <el-option v-for="s in allSceneNames" :key="s" :label="s" :value="s" />
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

        <div class="panel-card file-panel">
          <div class="card-title">
            <span>我的文件</span>
            <el-badge :value="files.length" :max="99" type="primary" />
          </div>
          <el-scrollbar max-height="240px">
            <div v-if="!files.length" class="file-empty">暂无文件，请先上传</div>
            <div
              v-for="f in files"
              :key="f.id"
              :class="['file-item', { active: currentFileId === f.id }]"
              @click="selectFile(f)"
            >
              <div class="file-item-main">
                <el-icon class="file-icon"><Document /></el-icon>
                <div class="file-meta">
                  <span class="file-name" :title="f.name">{{ f.name }}</span>
                  <span class="file-sub">{{ formatSize(f.size) }} · {{ formatTime(f.upload_at) }}</span>
                </div>
              </div>
              <div class="file-item-foot">
                <el-progress
                  v-if="isProcessing(f.status)"
                  :percentage="f.progress || 0"
                  :stroke-width="4"
                  :show-text="false"
                  class="file-progress"
                />
                <el-tag size="small" :type="statusType(f.status)" effect="plain">{{ statusLabel(f.status) }}</el-tag>
                <el-button
                  v-if="f.status === 'failed'"
                  link
                  type="primary"
                  size="small"
                  :loading="retryingId === f.id"
                  @click.stop="retryIngest(f)"
                >
                  重新入库
                </el-button>
                <el-button link type="danger" size="small" @click.stop="removeFile(f.id)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </div>
            </div>
          </el-scrollbar>
        </div>
      </aside>

      <main class="content">
        <div v-if="!currentFileId" class="empty-state">
          <div class="empty-icon">
            <el-icon :size="48"><DocumentCopy /></el-icon>
          </div>
          <h2>选择或上传日志文件</h2>
          <p>从左侧文件列表选择已解析完成的日志，或上传新的 logcat 文件</p>
        </div>

        <template v-else>
          <div class="log-toolbar panel-card">
            <div class="log-toolbar-left">
              <el-icon><Document /></el-icon>
              <span class="log-filename">{{ currentFileName }}</span>
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
                  class="log-line"
                  :style="logLineStyle(row)"
                  :title="row.display || row.content"
                  @click="expandContext(row)"
                >
                  <span class="level-mark" :style="{ background: levelColor(row.level) }" />
                  <span class="level-badge" :style="{ color: levelColor(row.level) }">{{ levelShort(row.level) }}</span>
                  <span class="ln">{{ row.line }}</span>
                  <span class="log-text" v-html="highlightLine(row)"></span>
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
                <span class="level-mark" :style="{ background: levelColor(row.level) }" />
                <span class="level-badge" :style="{ color: levelColor(row.level) }">{{ levelShort(row.level) }}</span>
                <span class="ln">{{ row.line }}</span>
                <span class="log-text">{{ row.display || row.content }}</span>
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
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Collection,
  Delete,
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
import { api } from './api'
import { getDeviceId } from './utils/device'
import { applyTheme, getPreferredTheme } from './utils/theme'
import {
  collectSceneKeywords,
  decorateEntries,
  loadLocalScene,
} from './utils/scene'
import { levelColor, levelShort } from './utils/logLevel'

const deviceId = ref(getDeviceId())
const isDark = ref(getPreferredTheme() === 'dark')
const leftTab = ref('upload')
const files = ref([])
const currentFileId = ref('')
const currentFileName = ref('')
const logs = ref([])
const loadingLogs = ref(false)
const uploading = ref(false)
const uploadProgress = ref(0)
const pendingFiles = ref([])
const parseTasks = ref([])
const parseLogs = ref([])
const retryingId = ref('')
let pollTimer = null

const sceneConfig = ref(loadLocalScene())
const sceneDialogVisible = ref(false)
const jiraDialogVisible = ref(false)
const selectedScenes = ref([])
const searchKeywords = ref('')
const useRegex = ref(false)

const ctxOpen = ref(false)
const ctxLines = ref([])
const ctxCenterLine = ref(0)
let sceneMeta = []

const allSceneNames = computed(() => {
  const names = []
  for (const m of sceneConfig.value.modules || []) {
    for (const s of m.scenes || []) names.push(s.name)
  }
  return names
})

function toggleTheme() {
  isDark.value = !isDark.value
  applyTheme(isDark.value ? 'dark' : 'light')
}

function formatSize(bytes) {
  if (!bytes) return '-'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function formatTime(t) {
  if (!t) return ''
  return new Date(t).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function statusType(s) {
  if (s === 'ready') return 'success'
  if (s === 'failed') return 'danger'
  return 'warning'
}

function statusLabel(s) {
  const map = { parsing: '解析中', inserting: '入库中', ready: '完成', failed: '失败' }
  return map[s] || s
}

function isProcessing(status) {
  return status === 'parsing' || status === 'inserting'
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
        data.file_ids.forEach((id) => appendParseLog(`已提交解析任务: ${id.slice(0, 8)}…`))
      }
    }
    ElMessage.success('上传成功，后台解析中')
    pendingFiles.value = []
    uploadProgress.value = 100
    await loadFiles()
    startPolling()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    uploading.value = false
  }
}

async function selectFile(f) {
  if (f.status !== 'ready') {
    ElMessage.info(f.status_msg || '文件解析中，请稍候')
    startPolling()
    return
  }
  currentFileId.value = f.id
  currentFileName.value = f.name
  await searchLogs()
}

async function removeFile(id) {
  await api.deleteFile(id)
  if (currentFileId.value === id) {
    currentFileId.value = ''
    logs.value = []
  }
  await loadFiles()
}

async function retryIngest(f) {
  retryingId.value = f.id
  try {
    const { data } = await api.retryIngest(f.id)
    if (!data.success) throw new Error(data.error)
    ElMessage.success('已重新开始入库')
    await loadFiles()
    startPolling()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    retryingId.value = ''
  }
}

async function searchLogs() {
  if (!currentFileId.value) return
  loadingLogs.value = true
  try {
    const kws = searchKeywords.value.split('\n').map((s) => s.trim()).filter(Boolean)
    const { keywords: sceneKw, meta } = collectSceneKeywords(sceneConfig.value, selectedScenes.value)
    sceneMeta = meta
    const { data } = await api.queryLogs({
      file_id: currentFileId.value,
      keywords: kws,
      scene_keywords: sceneKw,
      use_regex: useRegex.value,
      limit: 5000,
    })
    if (!data.success) throw new Error(data.error)
    logs.value = decorateEntries(data.data?.entries || [], meta)
  } catch (e) {
    ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    loadingLogs.value = false
  }
}

function highlightLine(row) {
  return (row.display || row.content || '').replace(/</g, '&lt;')
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
  ctxCenterLine.value = row.line
  const { data } = await api.logContext({
    file_id: currentFileId.value,
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
  startPolling()
}

onMounted(async () => {
  await loadFiles()
  if (files.value.some((f) => isProcessing(f.status))) startPolling()
})

onUnmounted(stopPolling)
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
  flex: 1;
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

.file-panel {
  margin-bottom: 0;
  flex-shrink: 0;
}

.file-empty {
  text-align: center;
  padding: 24px 12px;
  font-size: 13px;
  color: var(--app-text-muted);
}

.file-item {
  padding: 10px 12px;
  border-radius: var(--app-radius-sm);
  cursor: pointer;
  border: 1px solid transparent;
  margin-bottom: 6px;
  transition: background 0.15s, border-color 0.15s;
}

.file-item:hover {
  background: var(--app-accent-soft);
}

.file-item.active {
  background: var(--app-accent-soft);
  border-color: var(--app-accent);
}

.file-item-main {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.file-icon {
  color: var(--app-accent);
  margin-top: 2px;
  flex-shrink: 0;
}

.file-meta {
  min-width: 0;
  flex: 1;
}

.file-name {
  display: block;
  font-size: 13px;
  font-weight: 500;
  color: var(--app-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-sub {
  font-size: 11px;
  color: var(--app-text-muted);
  margin-top: 2px;
}

.file-item-foot {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
  padding-left: 26px;
}

.file-progress {
  flex: 1;
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
  padding: 8px 0;
}

.log-line {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 16px 4px 8px;
  font-family: 'Cascadia Code', Consolas, 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.5;
  cursor: pointer;
  border-left: 3px solid var(--level-color, #3fb950);
  transition: background 0.1s;
  overflow: hidden;
  max-width: 100%;
}

.level-mark {
  flex-shrink: 0;
  width: 4px;
  height: 14px;
  border-radius: 2px;
}

.level-badge {
  flex-shrink: 0;
  width: 14px;
  font-size: 11px;
  font-weight: 700;
  text-align: center;
  user-select: none;
}

.log-line:hover {
  background: var(--app-log-hover);
}

.log-line .ln {
  flex-shrink: 0;
  width: 44px;
  text-align: right;
  color: var(--app-log-gutter);
  user-select: none;
  font-size: 11px;
}

.log-text {
  flex: 1;
  min-width: 0;
  color: var(--line-color, var(--app-text-secondary));
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ctx-list .log-line {
  cursor: default;
  align-items: flex-start;
  overflow: visible;
  white-space: normal;
}

.ctx-list .log-text {
  overflow: visible;
  text-overflow: unset;
  white-space: pre-wrap;
  word-break: break-all;
}

.ctx-list .level-mark {
  margin-top: 3px;
}

.ctx-list .level-badge {
  margin-top: 1px;
}

.ctx-list .ln {
  margin-top: 1px;
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
  font-size: 10px;
  font-weight: 600;
  color: #fff;
  background: var(--app-accent);
  padding: 1px 6px;
  border-radius: 4px;
  line-height: 1.4;
  user-select: none;
}
</style>
