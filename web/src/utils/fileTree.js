import { displayFileName } from './fileDisplay'
import { compareLogRotationNames } from './logSort'

function isFolderItem(item) {
  return item?.entry_type === 'folder'
}

/**
 * @param {{ id: string, entry_type?: string, parent_id?: string, name: string }[]} items
 */
export function buildFileTree(items) {
  const folders = (items || []).filter(isFolderItem)
  const files = (items || []).filter((i) => !isFolderItem(i))
  const folderNodes = new Map()
  const roots = []

  const ensureFolderNode = (folder) => {
    if (folderNodes.has(folder.id)) return folderNodes.get(folder.id)
    const node = {
      id: `folder:${folder.id}`,
      folderId: folder.id,
      type: 'folder',
      label: folder.name || folder.original_name,
      children: [],
    }
    folderNodes.set(folder.id, node)
    return node
  }

  for (const folder of folders) {
    ensureFolderNode(folder)
  }

  for (const folder of folders) {
    const node = folderNodes.get(folder.id)
    if (!node) continue
    const parentId = folder.parent_id
    if (parentId && folderNodes.has(parentId)) {
      folderNodes.get(parentId).children.push(node)
    } else {
      roots.push(node)
    }
  }

  for (const file of files) {
    const node = {
      id: file.id,
      type: 'file',
      label: displayFileName(file),
      file,
      children: [],
    }
    const parentId = file.parent_id
    if (parentId && folderNodes.has(parentId)) {
      folderNodes.get(parentId).children.push(node)
    } else {
      roots.push(node)
    }
  }

  sortTreeNodes(roots)
  return roots
}

function sortTreeNodes(nodes) {
  nodes.sort((a, b) => {
    if (a.type !== b.type) return a.type === 'folder' ? -1 : 1
    if (a.type === 'file' && b.type === 'file') {
      return compareLogRotationNames(a.label, b.label)
    }
    return a.label.localeCompare(b.label, 'zh-CN')
  })
  for (const n of nodes) {
    if (n.children?.length) sortTreeNodes(n.children)
  }
}

/** All selectable ids in tree: folders + viewable files. */
export function collectTreeSelectableIds(nodes, isViewable) {
  const ids = []
  const walk = (list) => {
    for (const n of list || []) {
      if (n.type === 'folder') {
        ids.push(n.folderId)
      } else if (n.file && isViewable(n.file)) {
        ids.push(n.id)
      }
      if (n.children?.length) walk(n.children)
    }
  }
  walk(nodes)
  return ids
}

/** 删除文件夹时一并剔除其所有子项（与后端级联删除一致，用于列表即时刷新）。 */
export function expandRemovedItemIds(items, deletedIds) {
  const removed = new Set(deletedIds || [])
  if (!removed.size) return removed
  let changed = true
  while (changed) {
    changed = false
    for (const item of items || []) {
      if (!item?.id || removed.has(item.id)) continue
      if (item.parent_id && removed.has(item.parent_id)) {
        removed.add(item.id)
        changed = true
      }
    }
  }
  return removed
}

/** Keep only log file ids (exclude folders) for preview/query. */
export function filterLogFileIds(selectedIds, items) {
  const fileIds = new Set((items || []).filter((i) => i.entry_type !== 'folder').map((i) => i.id))
  return selectedIds.filter((id) => fileIds.has(id))
}

/** Collect file ids under a tree node (folder includes descendants). */
export function collectFileIds(node) {
  if (!node) return []
  if (node.type === 'file') return [node.id]
  const ids = []
  for (const ch of node.children || []) {
    ids.push(...collectFileIds(ch))
  }
  return ids
}
