<template>
  <div class="panel-card file-panel">
    <div class="card-title file-panel-title">
      <div class="file-panel-title-left">
        <span>我的文件</span>
        <el-badge :value="fileCount" :max="99" type="primary" />
      </div>
      <div v-if="fileCount" class="file-panel-title-actions">
        <el-button
          size="small"
          plain
          :disabled="!selectableIds.length"
          @click="toggleSelectAll"
        >
          {{ allSelectableSelected ? '取消全选' : '全选' }}
        </el-button>
        <el-button
          type="danger"
          size="small"
          plain
          :disabled="!selectedIds.length"
          :loading="batchDeleting"
          @click="batchRemove"
        >
          删除选中{{ selectedIds.length ? `(${selectedIds.length})` : '' }}
        </el-button>
      </div>
    </div>
    <el-scrollbar class="file-list-scroll">
      <div v-if="!fileCount" class="file-empty">暂无文件，请先上传</div>
      <el-tree
        v-else
        :data="treeData"
        node-key="id"
        default-expand-all
        :expand-on-click-node="false"
        :props="{ label: 'label', children: 'children' }"
        class="file-tree"
      >
        <template #default="{ data }">
          <div
            v-if="data.type === 'folder'"
            :class="['tree-folder-node', 'tree-selectable', { active: isSelected(data.folderId) }]"
            @click="toggleSelectFolder(data)"
          >
            <el-icon class="tree-folder-icon"><Folder /></el-icon>
            <span class="tree-folder-label">{{ data.label }}</span>
          </div>
          <div
            v-else
            :class="['file-item', 'tree-file-node', { active: isSelected(data.id), 'has-order': !!fileSelectOrderNum(data.id) }]"
            @click="toggleSelect(data.file)"
          >
            <span v-if="fileSelectOrderNum(data.id)" class="file-order">{{ fileSelectOrderNum(data.id) }}</span>
            <div class="file-item-body">
              <div class="file-row">
                <el-icon class="file-icon"><Document /></el-icon>
                <div class="file-meta">
                  <span class="file-name" :title="data.label">{{ data.label }}</span>
                  <span class="file-sub">
                    {{ formatSize(data.file.size) }} · {{ formatTime(data.file.upload_at) }}
                  </span>
                </div>
              </div>
              <el-progress
                v-if="isProcessing(data.file.status)"
                :percentage="data.file.progress || 0"
                :stroke-width="3"
                :show-text="false"
                class="file-progress"
              />
            </div>
            <div class="file-item-side" @click.stop>
              <el-tag size="small" :type="statusType(data.file.status)" effect="plain" class="file-status">
                {{ statusLabel(data.file.status) }}
              </el-tag>
              <el-dropdown trigger="click" placement="bottom-end" @command="(cmd) => onMenuCommand(cmd, data.file)">
                <button type="button" class="file-more-btn" aria-label="更多操作">
                  <el-icon><MoreFilled /></el-icon>
                </button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item
                      v-if="canIngest(data.file)"
                      command="ingest"
                      :disabled="ingestingLocalId === data.file.id"
                    >
                      {{ data.file.status === 'failed' ? '重新入库' : '入库' }}
                    </el-dropdown-item>
                    <el-dropdown-item command="delete" :divided="canIngest(data.file)">
                      <span class="menu-danger">删除</span>
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </div>
        </template>
      </el-tree>
    </el-scrollbar>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Document, Folder, MoreFilled } from '@element-plus/icons-vue'
import { api } from '../api'
import { canIngest, isProcessing, statusLabel, statusType } from '../utils/fileStatus'
import { buildFileTree, collectTreeSelectableIds } from '../utils/fileTree'

const props = defineProps({
  items: { type: Array, default: () => [] },
  selectedIds: { type: Array, default: () => [] },
})

function isFileItem(item) {
  return item?.entry_type !== 'folder'
}

const fileEntries = computed(() => props.items.filter(isFileItem))
const fileCount = computed(() => fileEntries.value.length)

const emit = defineEmits([
  'update:selectedIds',
  'select-change',
  'removed',
  'ingested',
  'need-poll',
])

const batchDeleting = ref(false)
const ingestingLocalId = ref('')

const treeData = computed(() => buildFileTree(props.items))

const selectableIds = computed(() => collectTreeSelectableIds(treeData.value, isViewable))

const allSelectableSelected = computed(() => {
  const ids = selectableIds.value
  return ids.length > 0 && ids.every((id) => props.selectedIds.includes(id))
})

function isSelected(id) {
  return props.selectedIds.includes(id)
}

const fileIdSet = computed(() => new Set(fileEntries.value.map((f) => f.id)))

const fileSelectOrder = computed(() => {
  const order = new Map()
  let n = 0
  for (const id of props.selectedIds) {
    if (!fileIdSet.value.has(id)) continue
    n += 1
    order.set(id, n)
  }
  return order
})

function fileSelectOrderNum(id) {
  return fileSelectOrder.value.get(id) || 0
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

function isViewable(f) {
  return f.status === 'uploaded' || f.status === 'ready' || f.status === 'failed'
}

function setSelection(ids) {
  emit('update:selectedIds', ids)
  emit('select-change', ids)
}

function toggleSelect(f) {
  if (!isViewable(f)) {
    ElMessage.info(f.status_msg || '文件正在入库，请稍候')
    emit('need-poll')
    return
  }
  const ids = [...props.selectedIds]
  const idx = ids.indexOf(f.id)
  if (idx >= 0) {
    ids.splice(idx, 1)
  } else {
    ids.push(f.id)
  }
  setSelection(ids)
}

function toggleSelectFolder(node) {
  const id = node.folderId
  const ids = [...props.selectedIds]
  const idx = ids.indexOf(id)
  if (idx >= 0) {
    ids.splice(idx, 1)
  } else {
    ids.push(id)
  }
  setSelection(ids)
}

function toggleSelectAll() {
  const ids = selectableIds.value
  if (!ids.length) {
    ElMessage.info('没有可选择的项目')
    return
  }
  if (allSelectableSelected.value) {
    const set = new Set(ids)
    setSelection(props.selectedIds.filter((id) => !set.has(id)))
    return
  }
  const skipped = fileEntries.value.filter((f) => !isViewable(f)).length
  if (skipped > 0) {
    ElMessage.info(`已跳过 ${skipped} 个入库中的文件`)
  }
  setSelection([...ids])
}

function onMenuCommand(cmd, f) {
  if (cmd === 'ingest') doIngest(f)
  else if (cmd === 'delete') removeOne(f.id)
}

async function removeOne(id) {
  try {
    await ElMessageBox.confirm('确定删除该文件？关联日志将一并清除。', '删除文件', { type: 'warning' })
  } catch {
    return
  }
  try {
    await api.deleteFile(id)
    emit('removed', [id])
    ElMessage.success('已删除')
  } catch (e) {
    ElMessage.error(e.response?.data?.error || e.message)
  }
}

async function batchRemove() {
  const ids = [...props.selectedIds]
  if (!ids.length) return
  try {
    await ElMessageBox.confirm(
      `确定删除选中的 ${ids.length} 项？文件夹及其中的文件将一并删除，关联日志将清除。`,
      '批量删除',
      { type: 'warning' },
    )
  } catch {
    return
  }
  batchDeleting.value = true
  try {
    const { data } = await api.batchDelete(ids)
    if (!data.success) throw new Error(data.error)
    emit('removed', ids)
    ElMessage.success(`已删除 ${ids.length} 项`)
  } catch (e) {
    ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    batchDeleting.value = false
  }
}

async function doIngest(f) {
  ingestingLocalId.value = f.id
  try {
    const { data } = await api.startIngest(f.id)
    if (!data.success) throw new Error(data.error)
    ElMessage.success('已开始入库')
    emit('ingested', f)
  } catch (e) {
    ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    ingestingLocalId.value = ''
  }
}
</script>

<style scoped>
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

.file-panel {
  flex: 1;
  min-height: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  margin-bottom: 0;
}

.file-list-scroll {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.file-list-scroll :deep(.el-scrollbar) {
  height: 100%;
}

.file-list-scroll :deep(.el-scrollbar__wrap) {
  overflow-x: hidden;
}

.file-panel-title {
  flex-wrap: wrap;
  gap: 8px;
  flex-shrink: 0;
}

.file-panel-title-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.file-panel-title-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

.file-empty {
  text-align: center;
  padding: 20px 12px;
  font-size: 12px;
  color: var(--app-text-muted);
}

.file-tree {
  background: transparent;
  --el-tree-node-hover-bg-color: var(--app-accent-soft);
}

.file-tree :deep(.el-tree-node__content) {
  height: auto;
  min-height: 28px;
  padding: 2px 0;
  align-items: flex-start;
}

.tree-folder-node {
  position: relative;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 6px 5px 8px;
  border-radius: var(--app-radius-sm);
  font-size: 12px;
  font-weight: 600;
  color: var(--app-text);
  border: 1px solid transparent;
  transition: background 0.15s, border-color 0.15s;
}

.tree-selectable {
  cursor: pointer;
  width: 100%;
}

.tree-selectable:hover {
  background: var(--app-accent-soft);
}

.tree-folder-node.active {
  background: var(--app-accent-soft);
  border-color: var(--app-accent);
}

.tree-folder-icon {
  color: var(--app-accent);
  font-size: 14px;
}

.tree-folder-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tree-file-node {
  width: 100%;
  margin: 0;
}

.file-item {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: 6px;
  padding: 5px 6px 5px 8px;
  border-radius: var(--app-radius-sm);
  cursor: pointer;
  border: 1px solid transparent;
  transition: background 0.15s, border-color 0.15s;
}

.file-item:hover {
  background: var(--app-accent-soft);
}

.file-item.active {
  background: var(--app-accent-soft);
  border-color: var(--app-accent);
}

.file-item.has-order .file-item-body {
  padding-top: 10px;
}

.file-order {
  position: absolute;
  top: 3px;
  left: 3px;
  z-index: 1;
  width: 13px;
  height: 13px;
  border-radius: 3px;
  background: var(--app-accent);
  color: #fff;
  font-size: 9px;
  font-weight: 700;
  line-height: 13px;
  text-align: center;
}

.file-item-body {
  flex: 1;
  min-width: 0;
}

.file-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.file-icon {
  flex-shrink: 0;
  font-size: 14px;
  color: var(--app-accent);
}

.file-meta {
  min-width: 0;
  flex: 1;
}

.file-name {
  display: block;
  font-size: 12px;
  font-weight: 500;
  line-height: 1.3;
  color: var(--app-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-sub {
  display: block;
  font-size: 10px;
  line-height: 1.3;
  color: var(--app-text-muted);
  margin-top: 1px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-progress {
  margin-top: 4px;
}

.file-item-side {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding-top: 1px;
}

.file-status {
  transform: scale(0.92);
  transform-origin: top right;
}

.file-more-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--app-text-muted);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.file-more-btn:hover {
  background: var(--app-border-light);
  color: var(--app-text);
}

.file-more-btn .el-icon {
  font-size: 14px;
}

.menu-danger {
  color: var(--el-color-danger);
}
</style>
