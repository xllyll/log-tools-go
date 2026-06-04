<template>
  <div class="panel-card search-card log-search-panel">
    <el-form label-position="top" size="default">
      <el-form-item>
        <template #label>
          <div class="search-kw-label-row">
            <span>关键词</span>
            <div class="search-kw-checks">
              <el-checkbox v-model="useRegex" class="search-regex-check">正则匹配</el-checkbox>
              <el-checkbox v-model="keywordCaseSensitive" class="search-regex-check">区分大小写</el-checkbox>
            </div>
          </div>
        </template>
        <el-input v-model="keywords" type="textarea" :rows="2" placeholder="输入关键词或正则..." />
        <div v-if="searchKeywordHistory.length" class="search-history">
          <span class="search-history-label">历史</span>
          <div class="search-history-tags">
            <el-tag
              v-for="item in searchKeywordHistory"
              :key="item.id"
              class="search-history-tag"
              size="small"
              effect="plain"
              closable
              :title="item.text"
              @click="applySearchHistory(item)"
              @close.stop="onRemoveSearchHistory(item.id)"
            >
              {{ formatSearchHistoryLabel(item.text) }}
            </el-tag>
          </div>
          <el-button link type="danger" size="small" class="search-history-clear" @click="onClearSearchHistory">
            清空
          </el-button>
        </div>
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
        <div v-if="selectedSceneTags.length" class="selected-scenes">
          <div class="selected-scenes-head">
            <span class="selected-scenes-label">已选场景</span>
            <el-button link type="danger" size="small" class="selected-scenes-clear" @click="clearAllScenes">
              清空
            </el-button>
          </div>
          <div class="selected-scenes-tags">
            <el-tag
              v-for="item in selectedSceneTags"
              :key="item.key"
              class="selected-scene-tag"
              size="small"
              effect="plain"
              closable
              :title="item.display"
              @close="removeSceneKey(item.key)"
            >
              {{ item.display }}
            </el-tag>
          </div>
        </div>
      </el-form-item>
      <el-button type="primary" :loading="loading" class="full-btn" @click="onSearch">
        <el-icon><Search /></el-icon>
        查询日志
      </el-button>
    </el-form>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { Search } from '@element-plus/icons-vue'
import {
  buildModuleSelectOptions,
  buildSceneSelectOptionsForModule,
  buildSceneSelectGroups,
  pruneSceneKeys,
} from '../utils/scene'
import {
  clearSearchKeywordHistory,
  formatSearchHistoryLabel,
  loadSearchKeywordHistory,
  pushSearchKeywordHistory,
  removeSearchKeywordHistory,
} from '../utils/searchHistory'

const keywords = defineModel('keywords', { type: String, default: '' })
const useRegex = defineModel('useRegex', { type: Boolean, default: false })
const keywordCaseSensitive = defineModel('keywordCaseSensitive', { type: Boolean, default: false })
const sceneKeys = defineModel('sceneKeys', { type: Array, default: () => [] })

const props = defineProps({
  sceneConfig: { type: Object, required: true },
  loading: { type: Boolean, default: false },
})

const emit = defineEmits(['search'])

const activeModuleIndex = ref(null)
const searchKeywordHistory = ref(loadSearchKeywordHistory())

const moduleSelectOptions = computed(() => buildModuleSelectOptions(props.sceneConfig))

const currentModuleSceneOptions = computed(() =>
  buildSceneSelectOptionsForModule(props.sceneConfig, activeModuleIndex.value),
)

const selectedSceneTags = computed(() => {
  const keyMeta = new Map()
  for (const g of buildSceneSelectGroups(props.sceneConfig)) {
    for (const o of g.options) {
      keyMeta.set(o.key, {
        label: o.label,
        moduleName: g.moduleName,
        display: `${g.moduleName} / ${o.label}`,
      })
    }
  }
  return sceneKeys.value.map((key) => {
    const meta = keyMeta.get(key)
    return {
      key,
      display: meta?.display || key,
    }
  })
})

const currentModuleSceneKeys = computed({
  get() {
    const mi = activeModuleIndex.value
    if (mi == null) return []
    const prefix = `${mi}:`
    return sceneKeys.value.filter((k) => k.startsWith(prefix))
  },
  set(keysForModule) {
    const mi = activeModuleIndex.value
    if (mi == null) return
    const prefix = `${mi}:`
    const other = sceneKeys.value.filter((k) => !k.startsWith(prefix))
    sceneKeys.value = [...other, ...keysForModule]
  },
})

watch(
  () => props.sceneConfig,
  (cfg) => {
    sceneKeys.value = pruneSceneKeys(cfg, sceneKeys.value)
    const validMods = new Set(buildModuleSelectOptions(cfg).map((o) => o.value))
    if (activeModuleIndex.value != null && !validMods.has(activeModuleIndex.value)) {
      activeModuleIndex.value = null
    }
  },
  { deep: true },
)

function recordSearchKeywordHistory() {
  const text = keywords.value
  if (!String(text ?? '').trim()) return
  searchKeywordHistory.value = pushSearchKeywordHistory(text, searchKeywordHistory.value)
}

function applySearchHistory(item) {
  keywords.value = item.text
}

function onRemoveSearchHistory(id) {
  searchKeywordHistory.value = removeSearchKeywordHistory(id, searchKeywordHistory.value)
}

function onClearSearchHistory() {
  searchKeywordHistory.value = clearSearchKeywordHistory()
}

function removeSceneKey(key) {
  sceneKeys.value = sceneKeys.value.filter((k) => k !== key)
}

function clearAllScenes() {
  sceneKeys.value = []
}

function onSearch() {
  recordSearchKeywordHistory()
  emit('search')
}
</script>

<style scoped>
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

.search-kw-checks {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
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

.search-history {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-top: 8px;
  width: 100%;
}

.search-history-label {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--app-text-muted);
  line-height: 24px;
}

.search-history-tags {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.search-history-tag {
  cursor: pointer;
  max-width: 100%;
}

.search-history-tag :deep(.el-tag__content) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 200px;
}

.search-history-clear {
  flex-shrink: 0;
  padding: 0;
  height: 24px;
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

.selected-scenes {
  margin-top: 10px;
  width: 100%;
}

.selected-scenes-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 6px;
}

.selected-scenes-label {
  font-size: 12px;
  color: var(--app-text-muted);
}

.selected-scenes-clear {
  padding: 0;
  height: auto;
}

.selected-scenes-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.selected-scene-tag :deep(.el-tag__content) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 220px;
}

.full-btn {
  width: 100%;
  margin-top: 4px;
}
</style>
