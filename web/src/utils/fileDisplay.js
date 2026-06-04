/** User-visible file name (original upload name, not storage dedup name). */
export function displayFileName(file) {
  if (!file) return ''
  return file.original_name || file.name || ''
}

function folderDisplayName(folder) {
  if (!folder) return ''
  return (folder.original_name || folder.name || '').trim()
}

/**
 * 日志列表头展示路径：根目录下为文件名；在文件夹下为「文件夹/…/文件名」（至根目录）。
 * @param {object} file
 * @param {Map<string, object>|Record<string, object>} folderById
 */
export function buildLogFileDisplayPath(file, folderById) {
  if (!file) return ''
  const fileName = displayFileName(file)
  const segments = []
  let parentId = file.parent_id ?? file.parent_folder_id ?? ''
  parentId = parentId ? String(parentId).trim() : ''

  const getFolder = (id) => (folderById instanceof Map ? folderById.get(id) : folderById?.[id])

  const visited = new Set()
  while (parentId) {
    if (visited.has(parentId)) break
    visited.add(parentId)
    const folder = getFolder(parentId)
    if (!folder) break
    const seg = folderDisplayName(folder)
    if (seg) segments.unshift(seg)
    parentId = folder.parent_id ?? folder.parent_folder_id ?? ''
    parentId = parentId ? String(parentId).trim() : ''
  }

  return segments.length ? `${segments.join('/')}/${fileName}` : fileName
}
