<template>
  <div :class="['file-panel', { 'file-panel--dialog': dialogMode }]">
    <div class="file-panel-toolbar">
      <div class="file-panel-toolbar-left">
        <span v-if="!dialogMode" class="file-panel-title-text">我的文件</span>
        <el-badge :value="fileCount" :max="9999" type="primary" :hidden="!fileCount" />
      </div>
      <div class="file-panel-toolbar-actions">
        <el-button
          v-if="itemCount"
          type="danger"
          size="small"
          plain
          class="file-panel-action-btn"
          :disabled="!selectedIds.length"
          :loading="batchDeleting"
          @click="batchRemove"
        >
          删除选中{{ selectedIds.length ? `(${selectedIds.length})` : '' }}
        </el-button>
      </div>
    </div>

    <div v-if="!itemCount" class="file-empty">暂无文件，请先上传</div>

    <div v-else class="file-panel-cols">
      <section class="file-col file-col--folder">
        <header class="col-header">文件夹</header>
        <el-scrollbar class="col-body">
          <el-tree
            :key="`file-tree-${listVersion}-${treeEpoch}`"
            ref="treeRef"
            lazy
            :data="[]"
            node-key="id"
            :load="loadTreeNode"
            :expand-on-click-node="false"
            :props="{ label: 'label', children: 'children', isLeaf: 'isLeaf' }"
            highlight-current
            class="file-tree"
            @node-click="onFolderNodeClick"
          >
            <template #default="{ data }">
              <div
                :class="[
                  'tree-folder-node',
                  { 'is-nav-active': activeFolderId === data.folderId },
                  { 'is-folder-selected': selectedSet.has(data.folderId) && data.folderId },
                ]"
                @click="onFolderBrowse(data)"
              >
                <el-checkbox
                  v-if="data.folderId"
                  :model-value="selectedSet.has(data.folderId)"
                  class="tree-folder-check"
                  @click.stop
                  @change="(v) => setFolderSelected(data.folderId, v)"
                />
                <el-icon class="tree-folder-icon"><Folder /></el-icon>
                <span class="tree-folder-label" :title="data.label">{{ data.label }}</span>
                <span v-if="folderFileCount(data.folderId)" class="tree-folder-count">
                  {{ folderFileCount(data.folderId) }}
                </span>
              </div>
            </template>
          </el-tree>
        </el-scrollbar>
      </section>

      <section class="file-col file-col--files">
        <header class="col-header">
          <span class="col-header-title" :title="activeFolderLabel">{{ activeFolderLabel }}</span>
          <div class="col-header-right">
            <span class="col-header-meta">{{ currentFiles.length }} 个</span>
            <el-button
              v-if="currentFolderSelectableIds.length"
              type="primary"
              plain
              size="small"
              class="col-header-btn"
              @click="toggleSelectCurrentFolder"
            >
              {{ allCurrentFolderSelected ? '取消全选' : '全选' }}
            </el-button>
          </div>
        </header>
        <VirtualFileList
          v-loading="filesLoading"
          class="col-body"
          :files="currentFiles"
          :selected-ids="selectedIds"
          :file-select-order="fileSelectOrder"
          empty-text="此目录下暂无文件"
          @toggle="onFileToggle"
          @command="onFileCommand"
        />
      </section>

      <section class="file-col file-col--selected">
        <header class="col-header">
          <span>已选择</span>
          <div class="col-header-right">
            <span class="col-header-meta">{{ selectedIds.length }} 项</span>
            <el-button
              v-if="selectedIds.length"
              plain
              size="small"
              class="col-header-btn"
              @click="clearSelection"
            >
              清空选择
            </el-button>
          </div>
        </header>
        <SelectedItemsList
          class="col-body"
          :items="selectionItems"
          :selected-ids="selectedIds"
          @remove="removeFromSelection"
        />
      </section>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, ref, shallowRef, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Folder } from '@element-plus/icons-vue'
import { api } from '../api'
import {
  buildFolderIndex,
  findFirstFolderWithFiles,
  getRootTreeNodes,
  getLazyFolderNodes,
} from '../utils/fileIndex'
import VirtualFileList from './VirtualFileList.vue'
import SelectedItemsList from './SelectedItemsList.vue'

const props = defineProps({
  selectionItems: { type: Array, default: () => [] },
  selectedIds: { type: Array, default: () => [] },
  listVersion: { type: Number, default: 0 },
  dialogMode: { type: Boolean, default: false },
})

const emit = defineEmits([
  'update:selectedIds',
  'select-change',
  'removed',
  'ingested',
  'need-poll',
  'folders-loaded',
  'files-loaded',
])

const folders = ref([])
const folderIndex = shallowRef(buildFolderIndex([]))
const currentFiles = ref([])
const rootFileCount = ref(0)
const filesLoading = ref(false)
const activeFolderId = ref('')
const treeEpoch = ref(0)
const treeRef = ref(null)
const batchDeleting = ref(false)
const ingestingLocalId = ref('')
/** 当前目录列表里上次点击的文件下标，用于 Shift 连选 */
const lastClickedFileIndex = ref(-1)

function normalizeListItems(payload) {
  const raw = Array.isArray(payload)
    ? payload
    : Array.isArray(payload?.items)
      ? payload.items
      : []
  return raw.map((f) => ({
    ...f,
    entry_type: f.entry_type || 'file',
    parent_id: f.parent_id ?? f.parent_folder_id ?? '',
  }))
}

async function loadFolders() {
  const { data } = await api.listFolders()
  if (!data?.success) throw new Error(data?.error || '加载文件夹失败')
  folders.value = normalizeListItems(data.data).map((f) => ({ ...f, entry_type: 'folder' }))
  folderIndex.value = buildFolderIndex(folders.value)
  treeEpoch.value += 1
  emit('folders-loaded', folders.value)
}

watch(activeFolderId, () => {
  lastClickedFileIndex.value = -1
})

async function loadFilesForFolder(parentId) {
  filesLoading.value = true
  try {
    const { data } = await api.listFilesByParent(parentId || '')
    if (!data?.success) throw new Error(data?.error || '加载文件失败')
    const files = normalizeListItems(data.data)
    currentFiles.value = files
    if (!parentId) rootFileCount.value = files.length
    emit('files-loaded', files)
  } finally {
    filesLoading.value = false
  }
}

async function refreshAll() {
  await loadFolders()
  const pick = findFirstFolderWithFiles(folderIndex.value)
  if (pick !== activeFolderId.value) activeFolderId.value = pick
  await loadFilesForFolder(activeFolderId.value)
  nextTick(() => {
    const key = activeFolderId.value ? `folder:${activeFolderId.value}` : 'folder:__root__'
    treeRef.value?.setCurrentKey?.(key)
    const rootNode = treeRef.value?.getNode?.('folder:__root__')
    if (rootNode && !rootNode.expanded) rootNode.expand()
  })
}

watch(
  () => props.listVersion,
  () => {
    refreshAll().catch((e) => ElMessage.error(e.message || '刷新失败'))
  },
)

onMounted(() => {
  refreshAll().catch((e) => ElMessage.error(e.message || '加载失败'))
})

defineExpose({ refresh: refreshAll })

const selectedSet = computed(() => new Set(props.selectedIds))
const fileIdSet = computed(() => new Set(currentFiles.value.map((f) => f.id)))

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

const itemCount = computed(() => folders.value.length > 0 || rootFileCount.value > 0)
const fileCount = computed(() => {
  let n = rootFileCount.value
  for (const f of folders.value) n += f.child_file_count || 0
  return n
})

const activeFolderLabel = computed(() => {
  if (!activeFolderId.value) return '根目录'
  const f = folderIndex.value.folderById.get(activeFolderId.value)
  return f ? f.name || f.original_name : '文件夹'
})

function folderFileCount(folderId) {
  if (!folderId) return rootFileCount.value
  const f = folderIndex.value.folderById.get(folderId)
  return f?.child_file_count ?? 0
}

function loadTreeNode(node, resolve) {
  if (node.level === 0) {
    resolve(getRootTreeNodes(folderIndex.value, rootFileCount.value > 0))
    return
  }
  resolve(getLazyFolderNodes(folderIndex.value, node.data.folderId))
}

async function onFolderBrowse(data) {
  if (data.type !== 'folder') return
  if (data.folderId === activeFolderId.value && currentFiles.value.length) return
  activeFolderId.value = data.folderId
  treeRef.value?.setCurrentKey?.(data.id)
  try {
    await loadFilesForFolder(data.folderId)
  } catch (e) {
    ElMessage.error(e.message || '加载文件失败')
  }
}

function onFolderNodeClick(data) {
  onFolderBrowse(data)
}

function setFolderSelected(folderId, checked) {
  if (!folderId) return
  const ids = [...props.selectedIds]
  const idx = ids.indexOf(folderId)
  if (checked && idx < 0) ids.push(folderId)
  if (!checked && idx >= 0) ids.splice(idx, 1)
  setSelection(ids)
}

const currentFolderSelectableIds = computed(() =>
  currentFiles.value.filter(isViewable).map((f) => f.id),
)

const allCurrentFolderSelected = computed(() => {
  const ids = currentFolderSelectableIds.value
  return ids.length > 0 && ids.every((id) => selectedSet.value.has(id))
})

function isViewable(f) {
  return f.status === 'uploaded' || f.status === 'ready' || f.status === 'failed'
}

function setSelection(ids) {
  emit('update:selectedIds', ids)
  emit('select-change', ids)
}

function removeFromSelection(id) {
  setSelection(props.selectedIds.filter((x) => x !== id))
}

function selectableFilesInList() {
  return currentFiles.value.filter(isViewable)
}

function onFileToggle(f, event) {
  if (!isViewable(f)) {
    ElMessage.info(f.status_msg || '文件正在入库，请稍候')
    emit('need-poll')
    return
  }
  const list = selectableFilesInList()
  const index = list.findIndex((x) => x.id === f.id)
  if (index < 0) return

  if (event?.shiftKey && lastClickedFileIndex.value >= 0) {
    const from = Math.min(lastClickedFileIndex.value, index)
    const to = Math.max(lastClickedFileIndex.value, index)
    const rangeIds = list.slice(from, to + 1).map((x) => x.id)
    setSelection([...new Set([...props.selectedIds, ...rangeIds])])
    lastClickedFileIndex.value = index
    return
  }

  const ids = [...props.selectedIds]
  const idx = ids.indexOf(f.id)
  if (idx >= 0) ids.splice(idx, 1)
  else ids.push(f.id)
  setSelection(ids)
  lastClickedFileIndex.value = index
}

function toggleSelectCurrentFolder() {
  const ids = currentFolderSelectableIds.value
  if (!ids.length) {
    ElMessage.info('当前目录没有可选择的文件')
    return
  }
  if (allCurrentFolderSelected.value) {
    const set = new Set(ids)
    setSelection(props.selectedIds.filter((id) => !set.has(id)))
    return
  }
  const skipped = currentFiles.value.filter((f) => !isViewable(f)).length
  if (skipped > 0) ElMessage.info(`已跳过 ${skipped} 个入库中的文件`)
  setSelection([...new Set([...props.selectedIds, ...ids])])
}

function clearSelection() {
  if (!props.selectedIds.length) return
  lastClickedFileIndex.value = -1
  setSelection([])
}

function onFileCommand(cmd, f) {
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
    const { data } = await api.deleteFile(id)
    if (!data?.success) throw new Error(data.error || '删除失败')
    setSelection(props.selectedIds.filter((x) => x !== id))
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
    if (!data?.success) throw new Error(data.error || '删除失败')
    setSelection([])
    emit('removed', ids)
    ElMessage.success(data.message || `已删除 ${ids.length} 项`)
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
.file-panel {
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
  overflow: hidden;
}

.file-panel:not(.file-panel--dialog) {
  background: var(--app-surface-2);
  border: 1px solid var(--app-border-light);
  border-radius: var(--app-radius);
  padding: 14px;
  margin-bottom: 12px;
}

.file-panel--dialog {
  flex: 1;
  min-height: 0;
}

.file-panel-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  flex-shrink: 0;
  margin-bottom: 10px;
}

.file-panel-toolbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
}

.file-panel-toolbar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.file-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: var(--app-text-muted);
}

.file-panel-cols {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(160px, 26%) minmax(200px, 1fr) minmax(180px, 28%);
  gap: 10px;
  overflow: hidden;
}

.file-col {
  min-height: 0;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--app-border-light);
  border-radius: var(--app-radius-sm);
  overflow: hidden;
  background: var(--app-surface);
}

.col-header {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  font-size: 12px;
  font-weight: 600;
  color: var(--app-text);
  background: var(--app-surface-2);
  border-bottom: 1px solid var(--app-border-light);
}

.col-header-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.col-header-right {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 6px;
}

.col-header-meta {
  flex-shrink: 0;
  font-size: 11px;
  font-weight: 500;
  color: var(--app-text-muted);
}

.file-panel-action-btn {
  border: 1px solid var(--el-button-border-color, var(--app-border-light));
}

.col-header-btn {
  padding: 2px 8px;
  height: 24px;
  font-size: 11px;
  font-weight: 500;
  border: 1px solid var(--el-button-border-color, var(--app-border-light));
}

.col-body {
  flex: 1;
  min-height: 0;
}

.col-body:deep(.virtual-file-list) {
  height: 100%;
}

.file-tree {
  background: transparent;
  --el-tree-node-hover-bg-color: var(--app-accent-soft);
  padding: 4px 0;
}

.file-tree :deep(.el-tree-node__content) {
  height: auto;
  min-height: 28px;
  padding: 2px 0;
}

.tree-folder-node {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 6px 4px 2px;
  border-radius: var(--app-radius-sm);
  font-size: 12px;
  font-weight: 600;
  color: var(--app-text);
  width: 100%;
  box-sizing: border-box;
  cursor: pointer;
}

.tree-folder-node:hover {
  background: var(--app-accent-soft);
}

.tree-folder-node.is-nav-active {
  color: var(--app-accent);
  background: var(--app-accent-soft);
}

.tree-folder-node.is-folder-selected {
  border: 1px solid var(--app-accent);
}

.tree-folder-check {
  flex-shrink: 0;
  height: auto;
}

.tree-folder-check :deep(.el-checkbox__inner) {
  width: 14px;
  height: 14px;
}

.tree-folder-icon {
  flex-shrink: 0;
  color: var(--app-accent);
  font-size: 14px;
}

.tree-folder-label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tree-folder-count {
  flex-shrink: 0;
  font-size: 10px;
  font-weight: 500;
  color: var(--app-text-muted);
  padding: 0 5px;
  border-radius: 8px;
  background: var(--app-border-light);
}
</style>
