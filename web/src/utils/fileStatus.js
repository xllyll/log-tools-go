export function statusType(s) {
  if (s === 'ready') return 'success'
  if (s === 'uploaded') return 'info'
  if (s === 'failed') return 'danger'
  return 'warning'
}

export function statusLabel(s) {
  const map = {
    uploaded: '未入库',
    parsing: '入库中',
    inserting: '入库中',
    ready: '已入库',
    failed: '失败',
  }
  return map[s] || s
}

export function isProcessing(status) {
  return status === 'parsing' || status === 'inserting'
}

export function isViewable(f) {
  return f.status === 'uploaded' || f.status === 'ready' || f.status === 'failed'
}

export function canIngest(f) {
  return f.status === 'uploaded' || f.status === 'failed'
}
