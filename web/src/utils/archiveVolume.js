const rePartExt = /^(.+)\.part(\d+)\.(rar|7z|zip)$/i
const reSplitExt = /^(.+)\.(7z|zip)\.(\d+)$/i
const reRarContin = /^(.+)\.r(\d{2})$/i

export function parseVolumeFilename(filename) {
  const base = (filename || '').split(/[/\\]/).pop() || ''
  let m = base.match(rePartExt)
  if (m) {
    return {
      key: m[1],
      archiveExt: m[3].toLowerCase(),
      partNum: parseInt(m[2], 10) || 1,
      isVolumePart: true,
    }
  }
  m = base.match(reSplitExt)
  if (m) {
    return {
      key: m[1],
      archiveExt: m[2].toLowerCase(),
      partNum: parseInt(m[3], 10) || 1,
      isVolumePart: true,
    }
  }
  m = base.match(reRarContin)
  if (m) {
    return {
      key: m[1],
      archiveExt: 'rar',
      partNum: parseInt(m[2], 10) + 2,
      isVolumePart: true,
    }
  }
  const dot = base.lastIndexOf('.')
  if (dot > 0) {
    const ext = base.slice(dot).toLowerCase()
    if (ext === '.rar' || ext === '.7z' || ext === '.zip') {
      return {
        key: base.slice(0, -ext.length),
        archiveExt: ext.slice(1),
        partNum: 1,
        isVolumePart: false,
      }
    }
  }
  return { key: '', archiveExt: '', partNum: 0, isVolumePart: false }
}

/**
 * 将 Jira 附件列表中的分卷合并为一行展示。
 * @param {Array<{ id, filename, size, mime_type, content_url }>} list
 */
export function groupJiraAttachments(list) {
  if (!list?.length) return []
  const buckets = new Map()
  const standalone = []

  for (const item of list) {
    const { key, archiveExt, partNum, isVolumePart } = parseVolumeFilename(item.filename)
    if (isVolumePart) {
      const bk = `${key}\0${archiveExt}`
      if (!buckets.has(bk)) {
        buckets.set(bk, { key, archiveExt, parts: [] })
      }
      buckets.get(bk).parts.push({ ...item, _partNum: partNum })
    } else {
      standalone.push(item)
    }
  }

  const rows = []
  for (const b of buckets.values()) {
    b.parts.sort((a, c) => a._partNum - c._partNum)
    if (b.parts.length > 1) {
      const totalSize = b.parts.reduce((s, p) => s + (p.size || 0), 0)
      rows.push({
        id: `volume:${b.key}.${b.archiveExt}`,
        filename: `${b.key}.${b.archiveExt}`,
        size: totalSize,
        mime_type: b.parts[0].mime_type,
        isVolumeGroup: true,
        partCount: b.parts.length,
        volumeParts: b.parts,
      })
    } else {
      const only = b.parts[0]
      delete only._partNum
      standalone.push(only)
    }
  }

  rows.push(...standalone)
  rows.sort((a, b) => (a.filename || '').localeCompare(b.filename || ''))
  return rows
}

/** 导入选中项：分卷组展开为全部 part 附件 */
export function flattenSelectedForImport(selected) {
  const out = []
  for (const row of selected) {
    if (row.isVolumeGroup && row.volumeParts?.length) {
      out.push(...row.volumeParts)
    } else {
      out.push(row)
    }
  }
  return out
}

function pushVolumeBucket(buckets, key, archiveExt, file, partNum) {
  const bk = `${key}\0${archiveExt}`
  if (!buckets.has(bk)) {
    buckets.set(bk, { key, archiveExt, parts: [] })
  }
  buckets.get(bk).parts.push({ file, _partNum: partNum })
}

/**
 * 将待上传文件按分卷分组，用于本地上传。
 * @param {File[]} fileList
 * @returns {Array<{ isVolumeGroup: boolean, files: File[], label: string, incomplete?: boolean }>}
 */
export function groupUploadFiles(fileList) {
  if (!fileList?.length) return []
  const buckets = new Map()
  const jobs = []

  for (const file of fileList) {
    const { key, archiveExt, partNum, isVolumePart } = parseVolumeFilename(file.name)
    if (isVolumePart) {
      pushVolumeBucket(buckets, key, archiveExt, file, partNum)
    } else {
      jobs.push({ isVolumeGroup: false, files: [file], label: file.name })
    }
  }

  for (const b of buckets.values()) {
    b.parts.sort((a, c) => a._partNum - c._partNum)
    const files = b.parts.map((p) => p.file)
    const label = `${b.key}.${b.archiveExt}`
    if (b.parts.length > 1) {
      jobs.push({ isVolumeGroup: true, files, label })
    } else {
      jobs.push({ isVolumeGroup: true, files, label, incomplete: true })
    }
  }
  return jobs
}
