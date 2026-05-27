/** @param {string} filename */
export function logRotationIndex(filename) {
  let name = (filename || '').split(/[/\\]/).pop()
  const lower = name.toLowerCase()
  for (const ext of ['.7z', '.zip', '.rar', '.gz']) {
    if (lower.endsWith(ext)) {
      name = name.slice(0, -ext.length)
      break
    }
  }
  const dot = name.lastIndexOf('.')
  if (dot < 0) return -1
  const numPart = name.slice(dot + 1)
  if (!/^\d+$/.test(numPart)) return -1
  return parseInt(numPart, 10)
}

/**
 * 解析轮转日志名，如 logcat.log.004.log → 组 logcat.log.log、序号 4；
 * logcat.log.log → 同组、序号 -1（基础文件排最后）。
 * @param {string} filename
 * @returns {{ groupKey: string, rotation: number, full: string }}
 */
export function parseLogRotationName(filename) {
  let full = (filename || '').split(/[/\\]/).pop() || ''
  const lower = full.toLowerCase()
  for (const ext of ['.7z', '.zip', '.rar', '.gz']) {
    if (lower.endsWith(ext)) {
      full = full.slice(0, -ext.length)
      break
    }
  }
  const m = full.match(/^(.+)\.(\d+)\.([^.\\/]+)$/)
  if (m) {
    return {
      groupKey: `${m[1]}.${m[3]}`,
      rotation: parseInt(m[2], 10),
      full,
    }
  }
  return { groupKey: full, rotation: -1, full }
}

/**
 * 默认排序：组名 A→Z；同组内轮转号大的在前（004→001），无序号基础文件最后。
 */
export function compareLogFileDefault(a, b) {
  const pa = parseLogRotationName(a)
  const pb = parseLogRotationName(b)
  const byGroup = pa.groupKey.localeCompare(pb.groupKey, 'zh-CN')
  if (byGroup !== 0) return byGroup
  if (pa.rotation !== pb.rotation) return pb.rotation - pa.rotation
  return pa.full.localeCompare(pb.full, 'zh-CN')
}

export function compareLogRotationNames(a, b) {
  const ia = logRotationIndex(a)
  const ib = logRotationIndex(b)
  if (ia !== ib) return ia - ib
  return a.localeCompare(b, 'zh-CN')
}

/** @param {{ name: string, original_name?: string }[]} files */
export function sortLogFilesByRotation(files) {
  return [...files].sort((a, b) =>
    compareLogRotationNames(a.original_name || a.name, b.original_name || b.name),
  )
}

/** @param {File[]} files */
export function sortPendingFilesByRotation(files) {
  return [...files].sort((a, b) => compareLogRotationNames(a.name, b.name))
}

function fileDisplayName(fileList, id) {
  const f = fileList.find((x) => x.id === id)
  return f?.original_name || f?.name || ''
}

/** @param {string[]} ids @param {{ id: string, name: string, original_name?: string }[]} fileList */
export function sortFileIdsByRotation(ids, fileList) {
  return [...ids].sort((a, b) =>
    compareLogRotationNames(fileDisplayName(fileList, a), fileDisplayName(fileList, b)),
  )
}

/** @param {string[]} ids @param {{ id: string, name: string, original_name?: string }[]} fileList */
export function sortFileIdsByDefault(ids, fileList) {
  return [...ids].sort((a, b) =>
    compareLogFileDefault(fileDisplayName(fileList, a), fileDisplayName(fileList, b)),
  )
}

/** @param {string[]} ids @param {{ id: string, name: string, original_name?: string }[]} fileList */
export function sortFileIdsByName(ids, fileList, desc = false) {
  return [...ids].sort((a, b) => {
    const cmp = fileDisplayName(fileList, a).localeCompare(fileDisplayName(fileList, b), 'zh-CN')
    return desc ? -cmp : cmp
  })
}

export const LOG_FILE_SORT_OPTIONS = [
  { value: 'default', label: '默认' },
  { value: 'selection', label: '选择顺序' },
  { value: 'rotation', label: '轮转序号' },
  { value: 'name-asc', label: '文件名 A→Z' },
  { value: 'name-desc', label: '文件名 Z→A' },
]

/**
 * @param {string[]} ids 已按选择顺序过滤后的日志文件 id
 * @param {{ id: string, name: string, original_name?: string }[]} fileList
 * @param {'default'|'selection'|'rotation'|'name-asc'|'name-desc'} mode
 */
export function orderLogFileIds(ids, fileList, mode) {
  if (!ids?.length || ids.length <= 1) return ids || []
  switch (mode) {
    case 'selection':
      return ids
    case 'rotation':
      return sortFileIdsByRotation(ids, fileList)
    case 'name-asc':
      return sortFileIdsByName(ids, fileList, false)
    case 'name-desc':
      return sortFileIdsByName(ids, fileList, true)
    default:
      return sortFileIdsByDefault(ids, fileList)
  }
}
