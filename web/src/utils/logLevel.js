/** 日志级别颜色（浅色/深色模式通用） */
const LEVEL_COLORS = {
  VERBOSE: '#8b949e',
  DEBUG: '#58a6ff',
  INFO: '#3fb950',
  WARN: '#d29922',
  ERROR: '#f85149',
  FATAL: '#da3633',
  ASSERT: '#a371f7',
}

const LEVEL_SHORT = {
  VERBOSE: 'V',
  DEBUG: 'D',
  INFO: 'I',
  WARN: 'W',
  ERROR: 'E',
  FATAL: 'F',
  ASSERT: 'A',
}

export function normalizeLevel(level) {
  const u = (level || 'INFO').toUpperCase()
  if (u.startsWith('WARN')) return 'WARN'
  return LEVEL_COLORS[u] ? u : 'INFO'
}

export function levelColor(level) {
  return LEVEL_COLORS[normalizeLevel(level)] || LEVEL_COLORS.INFO
}

export function levelShort(level) {
  return LEVEL_SHORT[normalizeLevel(level)] || 'I'
}
