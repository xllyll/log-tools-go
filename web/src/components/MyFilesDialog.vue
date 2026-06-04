<template>
  <el-dialog
    v-model="visible"
    title="我的文件"
    width="min(1200px, 94vw)"
    top="4vh"
    destroy-on-close
    class="my-files-dialog"
  >
    <FileListPanel
      ref="panelRef"
      :selection-items="selectionItems"
      :list-version="listVersion"
      :selected-ids="selectedIds"
      dialog-mode
      @update:selected-ids="emit('update:selectedIds', $event)"
      @select-change="emit('select-change', $event)"
      @removed="emit('removed', $event)"
      @ingested="emit('ingested', $event)"
      @need-poll="emit('need-poll')"
      @folders-loaded="emit('folders-loaded', $event)"
      @files-loaded="emit('files-loaded', $event)"
    />
    <template #footer>
      <el-button @click="visible = false">关闭</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import FileListPanel from './FileListPanel.vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  selectionItems: { type: Array, default: () => [] },
  selectedIds: { type: Array, default: () => [] },
  listVersion: { type: Number, default: 0 },
})

const emit = defineEmits([
  'update:modelValue',
  'update:selectedIds',
  'select-change',
  'removed',
  'ingested',
  'need-poll',
  'folders-loaded',
  'files-loaded',
])

const panelRef = ref(null)

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

watch(visible, async (v) => {
  if (!v) return
  await nextTick()
  await panelRef.value?.refresh?.()
})

defineExpose({ refresh: () => panelRef.value?.refresh?.() })
</script>

<style>
.my-files-dialog .el-dialog__body {
  padding: 12px 16px 8px;
  height: min(72vh, 720px);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
</style>
