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
      <el-button size="small" :disabled="!hasSelection" type="danger" plain @click="removeSelected">
        <el-icon><Delete /></el-icon>
        删除选中
      </el-button>
      <el-divider direction="vertical" />
      <el-button size="small" @click="resetDefault">恢复示例</el-button>
      <el-button size="small" @click="importJson">导入 JSON</el-button>
      <el-button size="small" @click="exportJson">导出 JSON</el-button>
    </div>

    <div v-if="!draft.modules?.length" class="scene-empty">
      <el-empty description="暂无模块，点击「添加模块」开始配置" />
    </div>

    <div v-else class="scene-body">
      <div class="nav-pane">
        <div class="pane-title">模块</div>
        <el-select
          v-model="selectedModuleIndex"
          class="module-select"
          placeholder="选择模块"
          @change="onModuleChange"
        >
          <el-option
            v-for="(mod, mi) in draft.modules"
            :key="mi"
            :label="mod.name?.trim() || `模块 ${mi + 1}`"
            :value="mi"
          />
        </el-select>

        <div class="scene-list-head">
          <span class="pane-title scene-list-title">场景</span>
          <el-tag v-if="currentModule" size="small" effect="plain" round>
            {{ currentModule.scenes?.length || 0 }}
          </el-tag>
        </div>
        <el-scrollbar class="scene-list-scroll">
          <div v-if="!currentModule?.scenes?.length" class="scene-list-empty">暂无场景</div>
          <div
            v-for="(scene, si) in currentModule?.scenes || []"
            :key="si"
            :class="['scene-item', { active: selectedSceneIndex === si }]"
            @click="selectScene(si)"
          >
            <el-icon class="scene-item-icon"><Document /></el-icon>
            <span class="scene-item-name" :title="scene.name">{{ scene.name || `场景 ${si + 1}` }}</span>
            <el-tag size="small" effect="plain" round>{{ scene.keywords?.length || 0 }}</el-tag>
          </div>
        </el-scrollbar>
      </div>

      <div class="detail-pane">
        <el-empty v-if="selectedModuleIndex === null" description="请先添加并选择模块" />

        <!-- 模块编辑（未选场景时） -->
        <template v-else-if="selectedSceneIndex === null && currentModule">
          <div class="detail-head">
            <h4>模块设置</h4>
            <el-tag effect="plain">{{ currentModule.scenes?.length || 0 }} 个场景</el-tag>
          </div>
          <el-form label-position="top" size="default">
            <el-form-item label="模块名称">
              <el-input v-model="currentModule.name" placeholder="如 DeviceService" />
            </el-form-item>
          </el-form>
          <el-alert type="info" :closable="false" show-icon>
            在左侧场景列表中点击场景以编辑关键词，或点击「添加场景」新建。
          </el-alert>
        </template>

        <!-- 场景编辑 -->
        <template v-else-if="selectedSceneIndex !== null && currentScene">
          <div class="detail-head">
            <h4>场景设置</h4>
            <span class="detail-path">{{ currentModule?.name }} / {{ currentScene.name || '未命名' }}</span>
          </div>
          <el-form label-position="top" size="default">
            <el-form-item label="场景名称">
              <el-input v-model="currentScene.name" placeholder="如 System问题分析" />
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
      <el-button type="primary" :loading="confirming" @click="handleConfirm">确定</el-button>
    </template>

    <input ref="fileInput" type="file" accept=".json,application/json" hidden @change="onFileImport" />

    <SceneLibraryDialog ref="libraryRef" v-model="libraryVisible" :config="draft" @apply="onLibraryApply" />
  </el-dialog>
</template>

<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Document, Plus, Upload } from '@element-plus/icons-vue'
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
const selectedModuleIndex = ref(null)
const selectedSceneIndex = ref(null)
const confirming = ref(false)
const fileInput = ref(null)
const libraryVisible = ref(false)
const libraryRef = ref(null)
const libraryPublishing = ref(false)

const currentModule = computed(() => {
  if (selectedModuleIndex.value === null) return null
  return draft.value.modules?.[selectedModuleIndex.value] || null
})

const currentScene = computed(() => {
  if (selectedSceneIndex.value === null) return null
  return currentModule.value?.scenes?.[selectedSceneIndex.value] || null
})

const canAddScene = computed(() => selectedModuleIndex.value !== null)

const hasSelection = computed(
  () => selectedModuleIndex.value !== null || selectedSceneIndex.value !== null,
)

watch(
  () => props.config,
  (c) => {
    if (!visible.value) draft.value = cloneSceneConfig(c)
  },
  { deep: true },
)

function onOpen() {
  draft.value = cloneSceneConfig(props.config)
  resetSelection()
}

function onLibraryApply(cfg) {
  draft.value = cloneSceneConfig(cfg)
  resetSelection()
}

function resetSelection() {
  selectedModuleIndex.value = null
  selectedSceneIndex.value = null
  nextTick(() => selectFirstModule())
}

function selectFirstModule() {
  if (!draft.value.modules?.length) return
  selectedModuleIndex.value = 0
  selectedSceneIndex.value = null
}

function onModuleChange() {
  selectedSceneIndex.value = null
}

function selectScene(si) {
  selectedSceneIndex.value = si
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

function addModule() {
  if (!draft.value.modules) draft.value.modules = []
  draft.value.modules.push({ name: '新模块', scenes: [] })
  selectedModuleIndex.value = draft.value.modules.length - 1
  selectedSceneIndex.value = null
}

function addSceneToSelected() {
  if (selectedModuleIndex.value === null) {
    if (!draft.value.modules?.length) {
      ElMessage.warning('请先添加模块')
      return
    }
    selectedModuleIndex.value = 0
  }
  const mod = draft.value.modules[selectedModuleIndex.value]
  if (!mod.scenes) mod.scenes = []
  mod.scenes.push({ name: '新场景', keywords: [emptyKeyword()] })
  selectedSceneIndex.value = mod.scenes.length - 1
}

function removeSelected() {
  if (selectedModuleIndex.value === null) return
  if (selectedSceneIndex.value !== null) {
    ElMessageBox.confirm('确定删除该场景？', '提示', { type: 'warning' })
      .then(() => {
        const mi = selectedModuleIndex.value
        draft.value.modules[mi].scenes.splice(selectedSceneIndex.value, 1)
        selectedSceneIndex.value = null
      })
      .catch(() => {})
    return
  }
  ElMessageBox.confirm('确定删除该模块及其所有场景？', '提示', { type: 'warning' })
    .then(() => {
      const mi = selectedModuleIndex.value
      draft.value.modules.splice(mi, 1)
      if (!draft.value.modules.length) {
        selectedModuleIndex.value = null
        selectedSceneIndex.value = null
      } else {
        selectedModuleIndex.value = Math.min(mi, draft.value.modules.length - 1)
        selectedSceneIndex.value = null
      }
    })
    .catch(() => {})
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
      resetSelection()
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

async function handleConfirm() {
  const cfg = applyConfig()
  if (!cfg) return
  confirming.value = true
  try {
    saveLocalScene(cfg)
    const { data } = await api.uploadSharedScene(cfg)
    if (!data.success) throw new Error(data.error)
    //visible.value = false
    ElMessage.success('已保存到本地并上传服务器')
  } catch (e) {
    ElMessage.error(e.response?.data?.error || e.message || '保存失败')
  } finally {
    confirming.value = false
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
      resetSelection()
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

.nav-pane {
  width: 260px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--app-border-light);
  border-radius: 8px;
  padding: 10px;
  background: var(--app-surface-2);
  max-height: 480px;
  min-height: 440px;
}

.pane-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--app-text-muted);
  margin-bottom: 8px;
  padding: 0 4px;
}

.module-select {
  width: 100%;
  margin-bottom: 14px;
}

.scene-list-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}

.scene-list-title {
  margin-bottom: 0;
}

.scene-list-scroll {
  flex: 1;
  min-height: 0;
  height: 0;
}

.scene-list-scroll :deep(.el-scrollbar) {
  height: 100%;
}

.scene-list-empty {
  padding: 16px 8px;
  text-align: center;
  font-size: 12px;
  color: var(--app-text-muted);
}

.scene-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 8px;
  margin-bottom: 4px;
  border-radius: 6px;
  cursor: pointer;
  border: 1px solid transparent;
  transition: background 0.15s, border-color 0.15s;
}

.scene-item:hover {
  background: var(--app-accent-soft);
}

.scene-item.active {
  background: var(--app-accent-soft);
  border-color: var(--app-accent);
}

.scene-item-icon {
  flex-shrink: 0;
  font-size: 14px;
  color: var(--app-accent);
}

.scene-item-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
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
