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
    </el-alert>

    <el-empty v-if="!files.length && !loading" description="输入 Issue Key 后拉取日志附件" />

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
      <el-table-column prop="filename" label="文件名" min-width="220" show-overflow-tooltip />
      <el-table-column label="大小" width="100">
        <template #default="{ row }">{{ formatSize(row.size) }}</template>
      </el-table-column>
      <el-table-column prop="mime_type" label="类型" width="120" show-overflow-tooltip />
    </el-table>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="importing" :disabled="!selected.length" @click="doImport">
        导入选中 ({{ selected.length }})
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'

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
}

async function fetchList() {
  const key = issueKey.value.trim()
  if (!key) return ElMessage.warning('请输入 Issue Key')
  loading.value = true
  files.value = []
  selected.value = []
  try {
    const { data } = await api.jiraAttachments(key)
    if (!data.success) throw new Error(data.error)
    files.value = data.data || []
    if (!files.value.length) {
      ElMessage.info('该 Issue 下没有可导入的日志附件')
    }
  } catch (e) {
    ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    loading.value = false
  }
}

async function doImport() {
  if (!selected.value.length) return
  importing.value = true
  try {
    const { data } = await api.jiraImport({
      issue_key: issueKey.value.trim(),
      attachments: selected.value.map((f) => ({
        id: f.id,
        filename: f.filename,
        content_url: f.content_url,
      })),
    })
    if (!data.success) throw new Error(data.error)
    ElMessage.success(data.message || '已开始导入')
    visible.value = false
    emit('imported')
  } catch (e) {
    ElMessage.error(e.response?.data?.error || e.message)
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
</style>
