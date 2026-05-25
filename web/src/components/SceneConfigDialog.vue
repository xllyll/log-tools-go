<template>
  <el-dialog
    v-model="visible"
    title="场景配置"
    width="80%"
    class="scene-dialog"
    destroy-on-close
    @open="onOpen"
  >
    <div class="scene-toolbar">
      <el-button type="primary" plain size="small" @click="addModule">
        <el-icon><Plus /></el-icon>
        添加模块
      </el-button>
      <el-button size="small" :disabled="!canAddScene" @click="addSceneToSelected">
        <el-icon><Plus /></el-icon>
        添加场景
      </el-button>
      <el-button size="small" :disabled="!selectedNode" type="danger" plain @click="removeSelected">
        <el-icon><Delete /></el-icon>
        删除选中
      </el-button>
      <el-divider direction="vertical" />
      <el-button size="small" @click="resetDefault">恢复示例</el-button>
      <el-button size="small" @click="importJson">导入 JSON</el-button>
      <el-button size="small" @click="exportJson">导出 JSON</el-button>
      <el-divider direction="vertical" />
      <el-button size="small" type="primary" plain :loading="libraryPublishing" @click="publishToLibrary">
        <el-icon><Upload /></el-icon>
        上传到场景库
      </el-button>
      <el-button size="small" type="success" plain @click="libraryVisible = true">
        场景库
      </el-button>
    </div>

    <div v-if="!draft.modules?.length" class="scene-empty">
      <el-empty description="暂无模块，点击「添加模块」开始配置" />
    </div>

    <div v-else class="scene-body">
      <div class="tree-pane">
        <div class="pane-title">模块 / 场景</div>
        <el-tree
          ref="treeRef"
          :data="treeData"
          node-key="id"
          highlight-current
          default-expand-all
          :expand-on-click-node="false"
          @node-click="onNodeClick"
        >
          <template #default="{ node, data }">
            <div class="tree-node">
              <el-icon class="tree-icon">
                <Folder v-if="data.type === 'module'" />
                <Document v-else />
              </el-icon>
              <span class="tree-label" :title="node.label">{{ node.label }}</span>
              <el-tag v-if="data.type === 'scene'" size="small" effect="plain" round>
                {{ data.kwCount }}
              </el-tag>
            </div>
          </template>
        </el-tree>
      </div>

      <div class="detail-pane">
        <el-empty v-if="!selectedNode" description="请在左侧选择模块或场景" />

        <!-- 模块编辑 -->
        <template v-else-if="selectedNode.type === 'module'">
          <div class="detail-head">
            <h4>模块设置</h4>
            <el-tag effect="plain">{{ currentModule?.scenes?.length || 0 }} 个场景</el-tag>
          </div>
          <el-form label-position="top" size="default">
            <el-form-item label="模块名称">
              <el-input
                v-model="currentModule.name"
                placeholder="如 DeviceService"
                @input="syncTreeLabels"
              />
            </el-form-item>
          </el-form>
          <el-alert type="info" :closable="false" show-icon>
            在此模块下添加场景，或点击左侧树中的场景节点编辑关键词规则。
          </el-alert>
        </template>

        <!-- 场景编辑 -->
        <template v-else-if="selectedNode.type === 'scene' && currentScene">
          <div class="detail-head">
            <h4>场景设置</h4>
            <span class="detail-path">{{ currentModule?.name }} / {{ currentScene.name || '未命名' }}</span>
          </div>
          <el-form label-position="top" size="default">
            <el-form-item label="场景名称">
              <el-input
                v-model="currentScene.name"
                placeholder="如 System问题分析"
                @input="syncTreeLabels"
              />
            </el-form-item>
          </el-form>

          <div class="kw-head">
            <span class="pane-title">关键词规则</span>
            <el-button type="primary" plain size="small" @click="addKeyword(currentScene)">
              <el-icon><Plus /></el-icon>
              添加关键词
            </el-button>
          </div>

          <el-table :data="currentScene.keywords" border size="small" class="kw-table">
            <el-table-column label="关键词" min-width="180">
              <template #default="{ row }">
                <el-input v-model="row.keyword" placeholder="匹配内容" />
              </template>
            </el-table-column>
            <el-table-column label="描述" min-width="120">
              <template #default="{ row }">
                <el-input v-model="row.desc" placeholder="展示说明" />
              </template>
            </el-table-column>
            <el-table-column label="模式" width="110">
              <template #default="{ row }">
                <el-select v-model="row.mode">
                  <el-option label="关键词" value="word" />
                  <el-option label="正则" value="regex" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column label="颜色" width="100">
              <template #default="{ row }">
                <el-color-picker v-model="row.color" size="small" />
              </template>
            </el-table-column>
            <el-table-column label="预览" min-width="160">
              <template #default="{ row }">
                <span
                  class="scene-desc"
                  :style="sceneDescStyle(row.color)"
                  :title="row.desc || row.keyword"
                >{{ row.desc || row.keyword || '预览' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="70" align="center">
              <template #default="{ $index }">
                <el-button link type="danger" @click="removeKeyword(currentScene, $index)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </template>
      </div>
    </div>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button @click="handleSaveLocal">保存到本地</el-button>
      <!-- <el-button :loading="syncing" @click="handleSyncServer">同步到服务器</el-button> -->
      <el-button type="primary" @click="handleConfirm">确定</el-button>
    </template>

    <input ref="fileInput" type="file" accept=".json,application/json" hidden @change="onFileImport" />

    <SceneLibraryDialog ref="libraryRef" v-model="libraryVisible" :config="draft" @apply="onLibraryApply" />
  </el-dialog>
</template>

<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Document, Folder, Plus, Upload } from '@element-plus/icons-vue'
import { api } from '../api'
import SceneLibraryDialog from './SceneLibraryDialog.vue'
import {
  cloneSceneConfig,
  defaultSceneConfig,
  emptyKeyword,
  saveLocalScene,
  sceneDescStyle,
} from '../utils/scene'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  config: { type: Object, required: true },
})

const emit = defineEmits(['update:modelValue', 'update:config'])

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const draft = ref(cloneSceneConfig(props.config))
const selectedNode = ref(null)
const syncing = ref(false)
const fileInput = ref(null)
const treeRef = ref(null)
const libraryVisible = ref(false)
const libraryRef = ref(null)
const libraryPublishing = ref(false)

const treeData = computed(() =>
  (draft.value.modules || []).map((mod, mi) => ({
    id: `m-${mi}`,
    label: mod.name || `模块 ${mi + 1}`,
    type: 'module',
    moduleIndex: mi,
    children: (mod.scenes || []).map((scene, si) => ({
      id: `m-${mi}-s-${si}`,
      label: scene.name || `场景 ${si + 1}`,
      type: 'scene',
      moduleIndex: mi,
      sceneIndex: si,
      kwCount: scene.keywords?.length || 0,
    })),
  })),
)

const currentModule = computed(() => {
  if (!selectedNode.value) return null
  return draft.value.modules?.[selectedNode.value.moduleIndex] || null
})

const currentScene = computed(() => {
  if (selectedNode.value?.type !== 'scene') return null
  return currentModule.value?.scenes?.[selectedNode.value.sceneIndex] || null
})

const canAddScene = computed(() => {
  if (!selectedNode.value) return draft.value.modules?.length > 0
  return true
})

watch(
  () => props.config,
  (c) => {
    if (!visible.value) draft.value = cloneSceneConfig(c)
  },
  { deep: true },
)

function onOpen() {
  draft.value = cloneSceneConfig(props.config)
  selectedNode.value = null
  nextTick(() => selectFirstNode())
}

function onLibraryApply(cfg) {
  draft.value = cloneSceneConfig(cfg)
  selectedNode.value = null
  nextTick(() => selectFirstNode())
}

async function publishToLibrary() {
  if (!validate()) return
  const cfg = cloneSceneConfig(draft.value)
  try {
    const { value: title } = await ElMessageBox.prompt('为该场景包取一个名称', '上传到场景库', {
      confirmButtonText: '上传',
      cancelButtonText: '取消',
      inputPlaceholder: '如：车机 System 分析包',
      inputValidator: (v) => (v?.trim() ? true : '请输入名称'),
    })
    const { value: desc } = await ElMessageBox.prompt('可选：补充说明', '场景说明', {
      confirmButtonText: '继续',
      cancelButtonText: '跳过',
      inputPlaceholder: '适用版本、模块说明等',
    }).catch(() => ({ value: '' }))
    libraryPublishing.value = true
    const { data } = await api.publishSceneLibrary({
      title: title.trim(),
      description: (desc || '').trim(),
      config: cfg,
    })
    if (!data.success) throw new Error(data.error)
    ElMessage.success('已分享到场景库')
    libraryRef.value?.loadList?.()
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error(e.response?.data?.error || e.message || '上传失败')
    }
  } finally {
    libraryPublishing.value = false
  }
}

function selectFirstNode() {
  const first = treeData.value[0]
  if (!first) return
  selectedNode.value = { type: 'module', moduleIndex: 0 }
  nextTick(() => treeRef.value?.setCurrentKey(first.id))
}

function onNodeClick(data) {
  selectedNode.value = {
    type: data.type,
    moduleIndex: data.moduleIndex,
    sceneIndex: data.sceneIndex,
  }
}

function syncTreeLabels() {
  // treeData 为 computed，模块/场景名称变更会自动反映
}

function addModule() {
  if (!draft.value.modules) draft.value.modules = []
  draft.value.modules.push({ name: '新模块', scenes: [] })
  const mi = draft.value.modules.length - 1
  selectedNode.value = { type: 'module', moduleIndex: mi }
  nextTick(() => treeRef.value?.setCurrentKey(`m-${mi}`))
}

function addSceneToSelected() {
  let mi = selectedNode.value?.moduleIndex
  if (mi === undefined) {
    if (!draft.value.modules?.length) {
      ElMessage.warning('请先添加模块')
      return
    }
    mi = 0
  }
  const mod = draft.value.modules[mi]
  if (!mod.scenes) mod.scenes = []
  mod.scenes.push({ name: '新场景', keywords: [emptyKeyword()] })
  const si = mod.scenes.length - 1
  selectedNode.value = { type: 'scene', moduleIndex: mi, sceneIndex: si }
  nextTick(() => treeRef.value?.setCurrentKey(`m-${mi}-s-${si}`))
}

function removeSelected() {
  if (!selectedNode.value) return
  const { type, moduleIndex, sceneIndex } = selectedNode.value
  if (type === 'module') {
    ElMessageBox.confirm('确定删除该模块及其所有场景？', '提示', { type: 'warning' })
      .then(() => {
        draft.value.modules.splice(moduleIndex, 1)
        selectedNode.value = null
        nextTick(() => selectFirstNode())
      })
      .catch(() => {})
  } else {
    ElMessageBox.confirm('确定删除该场景？', '提示', { type: 'warning' })
      .then(() => {
        draft.value.modules[moduleIndex].scenes.splice(sceneIndex, 1)
        selectedNode.value = { type: 'module', moduleIndex }
        nextTick(() => treeRef.value?.setCurrentKey(`m-${moduleIndex}`))
      })
      .catch(() => {})
  }
}

function addKeyword(scene) {
  if (!scene.keywords) scene.keywords = []
  scene.keywords.push(emptyKeyword())
}

function removeKeyword(scene, idx) {
  scene.keywords.splice(idx, 1)
}

function resetDefault() {
  ElMessageBox.confirm('将覆盖当前编辑内容，是否继续？', '恢复示例', { type: 'warning' })
    .then(() => {
      draft.value = defaultSceneConfig()
      selectedNode.value = null
      nextTick(() => selectFirstNode())
    })
    .catch(() => {})
}

function validate() {
  for (const mod of draft.value.modules || []) {
    if (!mod.name?.trim()) {
      ElMessage.warning('请填写模块名称')
      return false
    }
    for (const scene of mod.scenes || []) {
      if (!scene.name?.trim()) {
        ElMessage.warning(`模块「${mod.name}」存在未命名场景`)
        return false
      }
      for (const kw of scene.keywords || []) {
        if (!kw.keyword?.trim()) {
          ElMessage.warning(`场景「${scene.name}」存在空关键词`)
          return false
        }
      }
    }
  }
  return true
}

function applyConfig() {
  if (!validate()) return false
  const cfg = cloneSceneConfig(draft.value)
  emit('update:config', cfg)
  return cfg
}

function handleConfirm() {
  if (!applyConfig()) return
  visible.value = false
  ElMessage.success('场景配置已应用')
}

function handleSaveLocal() {
  const cfg = applyConfig()
  if (!cfg) return
  saveLocalScene(cfg)
  ElMessage.success('已保存到本地')
}

async function handleSyncServer() {
  const cfg = applyConfig()
  if (!cfg) return
  syncing.value = true
  try {
    await api.saveScene({ name: 'default', config: cfg })
    saveLocalScene(cfg)
    ElMessage.success('已同步到服务器')
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '同步失败')
  } finally {
    syncing.value = false
  }
}

function importJson() {
  fileInput.value?.click()
}

function onFileImport(e) {
  const file = e.target.files?.[0]
  if (!file) return
  const reader = new FileReader()
  reader.onload = () => {
    try {
      draft.value = JSON.parse(reader.result)
      if (!draft.value.modules) throw new Error('invalid')
      selectedNode.value = null
      nextTick(() => selectFirstNode())
      ElMessage.success('导入成功')
    } catch {
      ElMessage.error('JSON 格式无效')
    }
    e.target.value = ''
  }
  reader.readAsText(file)
}

function exportJson() {
  const blob = new Blob([JSON.stringify(draft.value, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'scene-config.json'
  a.click()
  URL.revokeObjectURL(url)
}
</script>

<style scoped>
.scene-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
}

.scene-body {
  display: flex;
  gap: 16px;
  min-height: 440px;
}

.tree-pane {
  width: 260px;
  flex-shrink: 0;
  border: 1px solid var(--app-border-light);
  border-radius: 8px;
  padding: 10px;
  background: var(--app-surface-2);
  overflow-y: auto;
  max-height: 480px;
}

.pane-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--app-text-muted);
  margin-bottom: 10px;
  padding: 0 4px;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 0;
  padding-right: 8px;
}

.tree-icon {
  flex-shrink: 0;
  color: var(--app-accent);
  font-size: 14px;
}

.tree-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}

.tree-pane :deep(.el-tree-node__content) {
  height: 34px;
  border-radius: 6px;
}

.tree-pane :deep(.el-tree-node.is-current > .el-tree-node__content) {
  background: var(--app-accent-soft);
}

.detail-pane {
  flex: 1;
  min-width: 0;
  border: 1px solid var(--app-border-light);
  border-radius: 8px;
  padding: 16px;
  background: var(--app-surface-2);
  overflow-y: auto;
  max-height: 480px;
}

.detail-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.detail-head h4 {
  margin: 0;
  font-size: 15px;
  color: var(--app-text);
}

.detail-path {
  font-size: 12px;
  color: var(--app-text-muted);
}

.kw-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 16px 0 10px;
}

.kw-table {
  margin-bottom: 8px;
}

.kw-table .scene-desc {
  max-width: 100%;
}

.kw-table :deep(.cell) {
  overflow: visible;
}

.scene-empty {
  min-height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
