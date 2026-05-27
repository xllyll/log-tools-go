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

/** newest first: no suffix, then ascending rotation number */
export function compareLogRotationNames(a, b) {
  const ia = logRotationIndex(a)
  const ib = logRotationIndex(b)
  if (ia !== ib) return ia - ib
  return a.localeCompare(b, 'zh-CN')
}

/** @param {{ name: string }[]} files */
export function sortLogFilesByRotation(files) {
  return [...files].sort((a, b) => compareLogRotationNames(a.name, b.name))
}

/** @param {File[]} files */
export function sortPendingFilesByRotation(files) {
  return [...files].sort((a, b) => compareLogRotationNames(a.name, b.name))
}

/** @param {string[]} ids @param {{ id: string, name: string }[]} fileList */
export function sortFileIdsByRotation(ids, fileList) {
  const nameOf = (id) => fileList.find((f) => f.id === id)?.name || ''
  return [...ids].sort((a, b) => compareLogRotationNames(nameOf(a), nameOf(b)))
}
