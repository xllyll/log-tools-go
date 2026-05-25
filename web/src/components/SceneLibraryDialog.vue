<template>
  <el-dialog v-model="visible" title="场景库" width="720px" destroy-on-close class="scene-library-dialog" @open="loadList">
    <div class="lib-toolbar">
      <el-button size="small" :loading="loading" @click="loadList">
        <el-icon><Refresh /></el-icon>
        刷新
      </el-button>
    </div>

    <el-table v-loading="loading" :data="items" border size="small" max-height="360" empty-text="场景库暂无分享">
      <el-table-column prop="title" label="名称" min-width="140" show-overflow-tooltip />
      <el-table-column label="发布者" width="110">
        <template #default="{ row }">
          <span>{{ shortDeviceLabel(row.device_id) }}</span>
          <el-tag v-if="row.is_mine" size="small" type="success" effect="plain" class="mine-tag">我</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="规模" width="100">
        <template #default="{ row }">{{ row.module_count }} 模块 / {{ row.scene_count }} 场景</template>
      </el-table-column>
      <el-table-column label="更新时间" width="150">
        <template #default="{ row }">{{ formatTime(row.updated_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="200" align="center" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" size="small" @click="pullMerge(row)">合并</el-button>
          <el-button link type="warning" size="small" @click="pullReplace(row)">覆盖</el-button>
          <el-button v-if="row.is_mine" link type="danger" size="small" @click="removeItem(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <p class="lib-hint">在「场景配置」中上传分享；在此合并或覆盖拉取他人场景包。</p>
  </el-dialog>
</template>

<script setup>
import { computed, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { api } from '../api'
import { getDeviceId } from '../utils/device'
import { cloneSceneConfig, mergeSceneConfig, shortDeviceLabel } from '../utils/scene'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  /** 待上传的当前场景配置 */
  config: { type: Object, required: true },
})

const emit = defineEmits(['update:modelValue', 'apply'])

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const items = ref([])
const loading = ref(false)
const myDeviceId = getDeviceId()

defineExpose({ loadList })

function formatTime(t) {
  if (!t) return ''
  return new Date(t).toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

async function loadList() {
  loading.value = true
  try {
    const { data } = await api.listSceneLibrary()
    if (!data.success) throw new Error(data.error)
    items.value = (data.data || []).map((row) => ({
      ...row,
      is_mine: row.is_mine || row.device_id === myDeviceId,
    }))
  } catch (e) {
    ElMessage.error(e.response?.data?.error || e.message)
  } finally {
    loading.value = false
  }
}

async function fetchConfig(id) {
  const { data } = await api.getSceneLibrary(id)
  if (!data.success) throw new Error(data.error)
  return data.data?.config
}

async function pullMerge(row) {
  try {
    const remote = await fetchConfig(row.id)
    if (!remote?.modules?.length) {
      ElMessage.warning('该场景包无有效配置')
      return
    }
    emit('apply', mergeSceneConfig(props.config, remote))
    ElMessage.success(`已合并「${row.title}」`)
    visible.value = false
  } catch (e) {
    ElMessage.error(e.response?.data?.error || e.message)
  }
}

async function pullReplace(row) {
  try {
    await ElMessageBox.confirm(`将用「${row.title}」覆盖当前编辑中的全部场景配置，是否继续？`, '覆盖配置', {
      type: 'warning',
    })
    const remote = await fetchConfig(row.id)
    if (!remote?.modules?.length) {
      ElMessage.warning('该场景包无有效配置')
      return
    }
    emit('apply', cloneSceneConfig(remote))
    ElMessage.success(`已覆盖为「${row.title}」`)
    visible.value = false
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error(e.response?.data?.error || e.message)
    }
  }
}

async function removeItem(row) {
  try {
    await ElMessageBox.confirm(`确定从场景库删除「${row.title}」？`, '删除', { type: 'warning' })
    const { data } = await api.deleteSceneLibrary(row.id)
    if (!data.success) throw new Error(data.error)
    ElMessage.success('已删除')
    await loadList()
  } catch (e) {
    if (e !== 'cancel' && e !== 'close') {
      ElMessage.error(e.response?.data?.error || e.message)
    }
  }
}
</script>

<style scoped>
.lib-toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.lib-hint {
  margin: 10px 0 0;
  font-size: 12px;
  color: var(--app-text-muted);
}

.mine-tag {
  margin-left: 4px;
  vertical-align: middle;
}
</style>
