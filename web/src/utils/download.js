export function triggerBlobDownload(blob, filename) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename || 'download'
  a.click()
  URL.revokeObjectURL(url)
}

export function filteredLogFilename(baseName) {
  const base = (baseName || 'log').replace(/[/\\?%*:|"<>]/g, '_')
  const dot = base.lastIndexOf('.')
  if (dot > 0) {
    return `${base.slice(0, dot)}.filtered${base.slice(dot)}`
  }
  return `${base}.filtered.log`
}
