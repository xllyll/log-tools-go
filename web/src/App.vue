<template>
  <el-container class="layout">
    <el-header class="header">
      <h1>车机日志分析</h1>
      <span class="device">设备: {{ deviceId.slice(0, 8) }}…</span>
    </el-header>
    <el-container>
      <el-aside width="360px" class="aside">
        <el-tabs v-model="leftTab">
          <el-tab-pane label="上传" name="upload">
            <el-upload
              drag
              multiple
              :auto-upload="false"
              :show-file-list="true"
              accept=".log,.txt,.zip,.rar,.7z"
              :on-change="onFileChange"
            >
              <el-icon class="el-icon--upload"><upload-filled /></el-icon>
              <div class="el-upload__text">拖拽或点击上传 .log .txt .zip .rar .7z</div>
            </el-upload>
            <el-button type="primary" :loading="uploading" style="width:100%;margin-top:12px" @click="doUpload">
              开始上传
            </el-button>
            <el-divider />
            <div class="jira-box">
              <div class="label">Jira 同步</div>
              <el-input v-model="jira.base_url" placeholder="Jira Base URL" size="small" />
              <el-input v-model="jira.email" placeholder="邮箱" size="small" style="margin-top:6px" />
              <el-input v-model="jira.api_token" type="password" placeholder="API Token" size="small" show-password style="margin-top:6px" />
              <el-input v-model="jiraIssueKey" placeholder="Issue Key (如 PROJ-1)" size="small" style="margin-top:6px" />
              <el-button size="small" style="margin-top:6px" :loading="jiraLoading" @click="fetchJira">拉取附件</el-button>
              <el-checkbox-group v-if="jiraFiles.length" v-model="jiraSelected" style="margin-top:8px;display:block">
                <el-checkbox v-for="f in jiraFiles" :key="f.id" :label="f.id">{{ f.filename }}</el-checkbox>
              </el-checkbox-group>
              <el-button v-if="jiraFiles.length" size="small" type="success" style="margin-top:6px" @click="importJira">导入选中</el-button>
            </div>
          </el-tab-pane>
          <el-tab-pane label="搜索" name="search">
            <el-input v-model="searchKeywords" type="textarea" :rows="3" placeholder="关键词，每行一个（AND）" />
            <el-checkbox v-model="useRegex" style="margin:8px 0">正则匹配</el-checkbox>
            <el-select v-model="selectedScenes" multiple placeholder="场景（OR）" style="width:100%">
              <el-option v-for="s in allSceneNames" :key="s" :label="s" :value="s" />
            </el-select>
            <el-button type="primary" style="width:100%;margin-top:12px" :loading="loadingLogs" @click="searchLogs">查询</el-button>
          </el-tab-pane>
          <el-tab-pane label="场景" name="scene">
            <el-button size="small" @click="resetScene">恢复示例</el-button>
            <el-button size="small" type="primary" style="margin-left:8px" @click="saveSceneLocal">保存本地</el-button>
            <el-button size="small" @click="syncSceneServer">上传服务器</el-button>
            <el-input v-model="sceneJson" type="textarea" :rows="18" style="margin-top:8px;font-family:monospace;font-size:12px" />
          </el-tab-pane>
        </el-tabs>
        <el-divider />
        <div class="file-list">
          <div class="label">我的文件</div>
          <el-scrollbar height="220px">
            <div
              v-for="f in files"
              :key="f.id"
              :class="['file-item', { active: currentFileId === f.id }]"
              @click="selectFile(f)"
            >
              <span>{{ f.name }}</span>
              <el-tag size="small" :type="statusType(f.status)">{{ f.status }}</el-tag>
              <el-button link type="danger" size="small" @click.stop="removeFile(f.id)">删</el-button>
            </div>
          </el-scrollbar>
        </div>
      </el-aside>
      <el-main class="main">
        <div v-if="!currentFileId" class="empty">请选择或上传日志文件</div>
        <template v-else>
          <div class="toolbar">
            <span>{{ currentFileName }}</span>
            <span class="muted">共 {{ logs.length }} 条</span>
          </div>
          <el-scrollbar height="calc(100vh - 140px)">
            <div
              v-for="row in logs"
              :key="row.id"
              class="log-line"
              :style="{ color: row.color || '#c9d1d9' }"
              @click="expandContext(row)"
            >
              <span class="ln">{{ row.line }}</span>
              <span v-html="highlightLine(row)"></span>
            </div>
          </el-scrollbar>
          <el-drawer v-model="ctxOpen" title="上下文 (前后10条)" size="50%">
            <div v-for="row in ctxLines" :key="row.id" class="log-line ctx">
              <span class="ln">{{ row.line }}</span>
              <span :style="{ color: row.color }">{{ row.display || row.content }}</span>
            </div>
          </el-drawer>
        </template>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { UploadFilled } from '@element-plus/icons-vue'
import { api } from './api'
import { getDeviceId } from './utils/device'
import {
  collectSceneKeywords,
  decorateEntries,
  defaultSceneConfig,
  loadLocalScene,
  saveLocalScene,
} from './utils/scene'

const deviceId = ref(getDeviceId())
const leftTab = ref('upload')
const files = ref([])
const currentFileId = ref('')
const currentFileName = ref('')
const logs = ref([])
const loadingLogs = ref(false)
const uploading = ref(false)
const pendingFiles = ref([])

const sceneConfig = ref(loadLocalScene())
const sceneJson = ref(JSON.stringify(sceneConfig.value, null, 2))
const selectedScenes = ref([])
const searchKeywords = ref('')
const useRegex = ref(false)

const jira = ref({ base_url: '', email: '', api_token: '' })
const jiraIssueKey = ref('')
const jiraFiles = ref([])
const jiraSelected = ref([])
const jiraLoading = ref(false)

const ctxOpen = ref(false)
const ctxLines = ref([])
let sceneMeta = []

const allSceneNames = computed(() => {
  const names = []
  for (const m of sceneConfig.value.modules || []) {
    for (const s of m.scenes || []) names.push(s.name)
  }
  return names
})

watch(sceneJson, () => {
  try {
    sceneConfig.value = JSON.parse(sceneJson.value)
  } catch (_) {}
})

function statusType(s) {
  if (s === 'ready') return 'success'
  if (s === 'failed') return 'danger'
  return 'warning'
}

function onFileChange(_file, list) {
  pendingFiles.value = list.map((x) => x.raw).filter(Boolean)
}

async function loadFiles() {
  const { data } = await api.listFiles()
  if (data.success) files.value = data.data || []
}

async function doUpload() {
  if (!pendingFiles.value.length) {
    ElMessage.warning('请先选择文件')
    return
  }
  uploading.value = true
  try {
    for (const f of pendingFiles.value) {
      await api.upload(f)
    }
    ElMessage.success('上传成功，后台解析中')
    pendingFiles.value = []
    await loadFiles()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    uploading.value = false
  }
}

async function selectFile(f) {
  if (f.status !== 'ready') {
    ElMessage.info('文件解析中，请稍后刷新')
    await pollFile(f.id)
    return
  }
  currentFileId.value = f.id
  currentFileName.value = f.name
  await searchLogs()
}

async function pollFile(id) {
  const timer = setInterval(async () => {
    const { data } = await api.fileStatus(id)
    if (data.success && data.data?.status === 'ready') {
      clearInterval(timer)
      await loadFiles()
      const nf = files.value.find((x) => x.id === id)
      if (nf) await selectFile(nf)
    }
    if (data.data?.status === 'failed') {
      clearInterval(timer)
      ElMessage.error(data.data.status_msg || '解析失败')
    }
  }, 2000)
}

async function removeFile(id) {
  await api.deleteFile(id)
  if (currentFileId.value === id) {
    currentFileId.value = ''
    logs.value = []
  }
  await loadFiles()
}

async function searchLogs() {
  if (!currentFileId.value) return
  loadingLogs.value = true
  try {
    const kws = searchKeywords.value
      .split('\n')
      .map((s) => s.trim())
      .filter(Boolean)
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
    const entries = data.data?.entries || []
    logs.value = decorateEntries(entries, meta)
  } catch (e) {
    ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    loadingLogs.value = false
  }
}

function highlightLine(row) {
  const text = (row.display || row.content || '').replace(/</g, '&lt;')
  return text
}

async function expandContext(row) {
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

function resetScene() {
  sceneConfig.value = defaultSceneConfig()
  sceneJson.value = JSON.stringify(sceneConfig.value, null, 2)
}

function saveSceneLocal() {
  try {
    const cfg = JSON.parse(sceneJson.value)
    saveLocalScene(cfg)
    sceneConfig.value = cfg
    ElMessage.success('已保存到本地')
  } catch {
    ElMessage.error('JSON 格式错误')
  }
}

async function syncSceneServer() {
  try {
    const cfg = JSON.parse(sceneJson.value)
    await api.saveScene({ name: 'default', config: cfg })
    ElMessage.success('已同步到服务器')
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '同步失败')
  }
}

async function fetchJira() {
  if (!jiraIssueKey.value) return ElMessage.warning('请输入 Issue Key')
  jiraLoading.value = true
  try {
    const { data } = await api.jiraAttachments(jiraIssueKey.value, jira.value)
    if (!data.success) throw new Error(data.error)
    jiraFiles.value = data.data || []
    jiraSelected.value = jiraFiles.value.map((f) => f.id)
  } catch (e) {
    ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    jiraLoading.value = false
  }
}

async function importJira() {
  const picked = jiraFiles.value.filter((f) => jiraSelected.value.includes(f.id))
  const { data } = await api.jiraImport({
    config: jira.value,
    issue_key: jiraIssueKey.value,
    attachments: picked.map((f) => ({
      id: f.id,
      filename: f.filename,
      content_url: f.content_url,
    })),
  })
  if (data.success) {
    ElMessage.success('Jira 附件已排队入库')
    await loadFiles()
  } else {
    ElMessage.error(data.error)
  }
}

onMounted(loadFiles)
</script>

<style>
html, body, #app {
  margin: 0;
  height: 100%;
  background: #0d1117;
  color: #c9d1d9;
}
.layout {
  height: 100vh;
}
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid #30363d;
  background: #161b22;
}
.header h1 {
  margin: 0;
  font-size: 18px;
}
.device {
  font-size: 12px;
  color: #8b949e;
}
.aside {
  background: #161b22;
  border-right: 1px solid #30363d;
  padding: 12px;
}
.main {
  background: #0d1117;
  padding: 12px 16px;
}
.file-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  cursor: pointer;
  border-radius: 4px;
  font-size: 13px;
}
.file-item:hover,
.file-item.active {
  background: #21262d;
}
.log-line {
  font-family: Consolas, monospace;
  font-size: 12px;
  line-height: 1.5;
  padding: 2px 4px;
  cursor: pointer;
  white-space: pre-wrap;
  word-break: break-all;
}
.log-line:hover {
  background: #21262d;
}
.ln {
  color: #6e7681;
  margin-right: 8px;
  user-select: none;
}
.empty {
  color: #8b949e;
  margin-top: 40px;
  text-align: center;
}
.label {
  font-size: 13px;
  margin-bottom: 6px;
  color: #8b949e;
}
.muted {
  color: #8b949e;
  margin-left: 12px;
  font-size: 12px;
}
.toolbar {
  margin-bottom: 8px;
}
</style>
