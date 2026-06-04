import { displayFileName } from './fileDisplay'
import { compareLogRotationNames } from './logSort'

const ROOT_KEY = ''

function isFolderItem(item) {
  return item?.entry_type === 'folder'
}

function parentKey(parentId) {
  return parentId || ROOT_KEY
}

function sortFiles(files) {
  return [...files].sort((a, b) =>
    compareLogRotationNames(displayFileName(a), displayFileName(b)),
  )
}

function sortFolders(folders) {
  return [...folders].sort((a, b) => {
    const la = a.name || a.original_name || ''
    const lb = b.name || b.original_name || ''
    return la.localeCompare(lb, 'zh-CN')
  })
}

/**
 * 由文件夹列表构建索引，供懒加载树使用（文件按 parent_id 另行请求）。
 * @param {object[]} folders
 */
export function buildFolderIndex(folders) {
  const folderById = new Map()
  const childrenFoldersByParent = new Map()

  for (const item of folders || []) {
    if (!isFolderItem(item)) continue
    folderById.set(item.id, item)
    const pk = parentKey(item.parent_id)
    if (!childrenFoldersByParent.has(pk)) childrenFoldersByParent.set(pk, [])
    childrenFoldersByParent.get(pk).push(item)
  }

  for (const [k, list] of childrenFoldersByParent) {
    childrenFoldersByParent.set(k, sortFolders(list))
  }

  return { folderById, childrenFoldersByParent }
}

/** @deprecated 全量 items；新逻辑请用 buildFolderIndex + 按目录拉文件 */
export function buildFileIndex(items) {
  const folders = (items || []).filter(isFolderItem)
  const index = buildFolderIndex(folders)
  const filesByParent = new Map()
  for (const item of items || []) {
    if (isFolderItem(item)) continue
    const pk = parentKey(item.parent_id)
    if (!filesByParent.has(pk)) filesByParent.set(pk, [])
    filesByParent.get(pk).push(item)
  }
  for (const [k, list] of filesByParent) {
    filesByParent.set(k, sortFiles(list))
  }
  return { ...index, filesByParent }
}

/** 某目录下的子文件夹（懒加载树节点数据） */
export function getLazyFolderNodes(index, parentFolderId) {
  const pk = parentKey(parentFolderId)
  const folders = index.childrenFoldersByParent.get(pk) || []
  return folders.map((f) => ({
    id: `folder:${f.id}`,
    folderId: f.id,
    type: 'folder',
    label: f.name || f.original_name,
    isLeaf: !hasChildFolders(index, f.id),
  }))
}

export function hasChildFolders(index, folderId) {
  return (index.childrenFoldersByParent.get(folderId)?.length ?? 0) > 0
}

/** 当前目录下的文件列表（已排序） */
export function getFilesInFolder(index, parentFolderId) {
  const pk = parentKey(parentFolderId)
  return index.filesByParent.get(pk) || []
}

function folderDirectFileCount(index, folderId) {
  if (!folderId) return 0
  const f = index.folderById.get(folderId)
  return f?.child_file_count ?? 0
}

/** BFS：返回第一个直属含文件的文件夹 id，无则 ''（根目录） */
export function findFirstFolderWithFiles(index) {
  const queue = [ROOT_KEY]
  const seen = new Set([ROOT_KEY])
  while (queue.length) {
    const folderId = queue.shift()
    if (folderDirectFileCount(index, folderId) > 0) return folderId
    const pk = parentKey(folderId)
    const children = index.childrenFoldersByParent.get(pk) || []
    for (const f of children) {
      if (!seen.has(f.id)) {
        seen.add(f.id)
        queue.push(f.id)
      }
    }
  }
  return ROOT_KEY
}

/** 根目录懒加载：仅返回「根目录」节点，顶层文件夹在其下展开后加载 */
export function getRootTreeNodes(index, rootHasFiles = false) {
  const rootFolders = getLazyFolderNodes(index, '')
  const hasChildren = rootFolders.length > 0 || rootHasFiles
  return [
    {
      id: 'folder:__root__',
      folderId: '',
      type: 'folder',
      label: '根目录',
      isLeaf: !hasChildren,
    },
  ]
}

/** 全量可选项 id（文件夹 + 可预览文件），不构建整棵树 */
export function collectAllSelectableIds(items, isViewable) {
  const ids = []
  for (const item of items || []) {
    if (isFolderItem(item)) {
      ids.push(item.id)
    } else if (isViewable(item)) {
      ids.push(item.id)
    }
  }
  return ids
}

/** 合并轮询结果：仅更新已有项字段，结构变化时返回 true */
export function mergeFileItems(prev, next) {
  if (!prev?.length && next?.length) return { items: next, structureChanged: true }
  if (prev?.length !== next?.length) return { items: next, structureChanged: true }

  const prevMap = new Map(prev.map((i) => [i.id, i]))
  const nextMap = new Map(next.map((i) => [i.id, i]))

  for (const id of prevMap.keys()) {
    if (!nextMap.has(id)) return { items: next, structureChanged: true }
  }
  for (const id of nextMap.keys()) {
    if (!prevMap.has(id)) return { items: next, structureChanged: true }
  }

  let changed = false
  const merged = next.map((n) => {
    const o = prevMap.get(n.id)
    if (
      o.status !== n.status ||
      o.status_msg !== n.status_msg ||
      o.progress !== n.progress ||
      o.parsed_lines !== n.parsed_lines ||
      o.total !== n.total
    ) {
      changed = true
    }
    return n
  })
  return { items: merged, structureChanged: false, statusChanged: changed }
}
