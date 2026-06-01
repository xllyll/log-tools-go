<template>
  <el-dialog v-model="visible" title="Jira 日志同步" width="640px" destroy-on-close @closed="onClosed">
    <el-form label-position="top" size="default">
      <el-form-item label="Issue Key">
        <div class="issue-row">
          <el-input v-model="issueKey" placeholder="如 PROJ-123" clearable @keyup.enter="fetchList" />
          <el-button type="primary" :loading="loading" @click="fetchList">拉取附件</el-button>
        </div>
      </el-form-item>
    </el-form>

    <el-alert type="info" :closable="false" show-icon class="tip">
      Jira 连接信息由服务端配置，此处只需填写 Issue Key。选中 .zip / .7z / .rar 压缩包时会自动解压并导入其中的 .log / .txt / .json。
      同一压缩包的分卷（如 .part01.rar、.part02.rar）会合并为一项，需一并导入。
    </el-alert>

    <div v-if="progressVisible" class="progress-block" :class="{ 'is-done': progressStatus === 'success' }">
      <div class="progress-label">
        <el-icon v-if="progressStatus === 'success'" class="progress-done-icon"><CircleCheck /></el-icon>
        <span>{{ progressLabel }}</span>
        <span v-if="progressStatus === 'success' && progressPercent === 100" class="progress-percent-text">100%</span>
      </div>
      <el-progress
        :percentage="progressPercent"
        :indeterminate="progressIndeterminate"
        :stroke-width="10"
        :status="progressStatus || undefined"
        striped
        :striped-flow="progressRunning"
      />
    </div>

    <el-empty v-if="!files.length && !loading && !importing" description="输入 Issue Key 后拉取日志附件" />

    <el-table
      v-if="files.length"
      ref="tableRef"
      :data="files"
      border
      size="small"
      max-height="320"
      @selection-change="onSelect"
    >
      <el-table-column type="selection" width="46" />
      <el-table-column prop="filename" label="文件名" min-width="220" show-overflow-tooltip>
        <template #default="{ row }">
          <span>{{ row.filename }}</span>
          <el-tag v-if="row.isVolumeGroup" size="small" type="warning" class="vol-tag">
            分卷 ×{{ row.partCount }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="大小" width="100">
        <template #default="{ row }">{{ formatSize(row.size) }}</template>
      </el-table-column>
      <el-table-column prop="mime_type" label="类型" width="120" show-overflow-tooltip />
    </el-table>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button
        type="primary"
        :loading="importing"
        :disabled="!selected.length || loading"
        @click="doImport"
      >
        导入选中 ({{ selected.length }})
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { CircleCheck } from '@element-plus/icons-vue'
import { api, jiraImportStream } from '../api'
import { groupJiraAttachments, flattenSelectedForImport } from '../utils/archiveVolume'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
})

const emit = defineEmits(['update:modelValue', 'imported'])

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const issueKey = ref('')
const files = ref([])
const selected = ref([])
const loading = ref(false)
const importing = ref(false)
const tableRef = ref(null)

const progressVisible = ref(false)
const progressPercent = ref(0)
const progressIndeterminate = ref(false)
const progressLabel = ref('')
const progressStatus = ref('')

/** 进行中才显示条纹动画 */
const progressRunning = computed(
  () =>
    (loading.value || importing.value) &&
    progressStatus.value !== 'success' &&
    progressStatus.value !== 'exception',
)

let fetchTimer = null
let resetTimer = null

function clearResetTimer() {
  if (resetTimer) {
    clearTimeout(resetTimer)
    resetTimer = null
  }
}

function scheduleResetProgress(delayMs = 2500) {
  clearResetTimer()
  resetTimer = setTimeout(() => {
    resetProgress()
    resetTimer = null
  }, delayMs)
}

function finishProgress(success, label) {
  stopFetchTimer()
  progressIndeterminate.value = false
  progressPercent.value = success ? 100 : progressPercent.value
  progressStatus.value = success ? 'success' : 'exception'
  if (label) progressLabel.value = label
}

function resetProgress() {
  clearResetTimer()
  progressVisible.value = false
  progressPercent.value = 0
  progressIndeterminate.value = false
  progressLabel.value = ''
  progressStatus.value = ''
  stopFetchTimer()
}

function stopFetchTimer() {
  if (fetchTimer) {
    clearInterval(fetchTimer)
    fetchTimer = null
  }
}

function startFetchTimer() {
  stopFetchTimer()
  progressVisible.value = true
  progressIndeterminate.value = false
  progressPercent.value = 8
  progressLabel.value = '正在从 Jira 拉取附件列表…'
  progressStatus.value = ''
  fetchTimer = setInterval(() => {
    if (progressPercent.value < 88) {
      progressPercent.value = Math.min(88, progressPercent.value + 4)
    }
  }, 280)
}

function applyProgress(ev) {
  if (ev.percent != null) {
    progressPercent.value = ev.percent
    progressIndeterminate.value = false
  }
  const parts = []
  if (ev.phase === 'download') {
    parts.push('下载中')
  } else if (ev.phase === 'extract') {
    parts.push('解压入库中')
  }
  if (ev.total > 0 && ev.current > 0) {
    parts.push(`${ev.current} / ${ev.total}`)
  }
  if (ev.filename) parts.push(ev.filename)
  if (ev.message && ev.phase !== 'done') parts.push(ev.message)
  progressLabel.value = parts.join(' · ') || progressLabel.value
}

function formatSize(bytes) {
  if (!bytes) return '-'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function onSelect(rows) {
  selected.value = rows
}

function onClosed() {
  issueKey.value = ''
  files.value = []
  selected.value = []
  resetProgress()
}

async function fetchList() {
  const key = issueKey.value.trim()
  if (!key) return ElMessage.warning('请输入 Issue Key')
  clearResetTimer()
  loading.value = true
  files.value = []
  selected.value = []
  startFetchTimer()
  let ok = false
  try {
    const { data } = await api.jiraAttachments(key)
    if (!data.success) throw new Error(data.error)
    files.value = groupJiraAttachments(data.data || [])
    ok = true
    if (!files.value.length) {
      ElMessage.info('该 Issue 下没有可导入的日志附件')
      finishProgress(true, '拉取完成 · 未找到可导入的日志附件')
    } else {
      const volCount = files.value.filter((r) => r.isVolumeGroup).length
      const hint = volCount ? `（含 ${volCount} 个分卷包）` : ''
      finishProgress(true, `拉取完成 · 共 ${files.value.length} 项${hint}`)
    }
  } catch (e) {
    finishProgress(false, '拉取失败')
    progressLabel.value = e.response?.data?.error || e.message
    ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    loading.value = false
    if (ok) {
      scheduleResetProgress(3000)
    } else {
      scheduleResetProgress(4000)
    }
  }
}

async function doImport() {
  if (!selected.value.length) return
  const toImport = flattenSelectedForImport(selected.value)
  const importItemCount = selected.value.length
  clearResetTimer()
  importing.value = true
  progressVisible.value = true
  progressIndeterminate.value = false
  progressPercent.value = 0
  progressStatus.value = ''
  progressLabel.value = '准备导入…'
  try {
    const result = await jiraImportStream(
      {
        issue_key: issueKey.value.trim(),
        attachments: toImport.map((f) => ({
          id: f.id,
          filename: f.filename,
          content_url: f.content_url,
        })),
      },
      (ev) => applyProgress(ev),
    )
    finishProgress(true, `导入完成 · 共 ${importItemCount} 项`)
    ElMessage.success(result.message || '已导入')
    scheduleResetProgress(1800)
    await new Promise((r) => setTimeout(r, 1500))
    visible.value = false
    emit('imported')
  } catch (e) {
    finishProgress(false, '导入失败')
    progressLabel.value = e.message
    ElMessage.error(e.message)
    scheduleResetProgress(4000)
  } finally {
    importing.value = false
  }
}
</script>

<style scoped>
.issue-row {
  display: flex;
  gap: 10px;
  width: 100%;
}

.issue-row .el-input {
  flex: 1;
}

.tip {
  margin-bottom: 14px;
}

.progress-block {
  margin-bottom: 14px;
  padding: 12px 14px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
}

.progress-label {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
  font-size: 13px;
  color: var(--el-text-color-regular);
}

.progress-block.is-done {
  background: var(--el-color-success-light-9);
}

.progress-done-icon {
  flex-shrink: 0;
  font-size: 16px;
  color: var(--el-color-success);
}

.progress-percent-text {
  margin-left: auto;
  font-size: 12px;
  font-weight: 600;
  color: var(--el-color-success);
}

.vol-tag {
  margin-left: 8px;
  vertical-align: middle;
}
</style>
