const LOCAL_KEY = 'log_tools_search_keyword_history'
const MAX_ITEMS = 30

export function normalizeSearchHistoryText(text) {
  return String(text ?? '').trim()
}

export function loadSearchKeywordHistory() {
  try {
    const raw = localStorage.getItem(LOCAL_KEY)
    if (!raw) return []
    const arr = JSON.parse(raw)
    if (!Array.isArray(arr)) return []
    return arr
      .filter((item) => item && typeof item.text === 'string' && normalizeSearchHistoryText(item.text))
      .map((item) => ({
        id: item.id || `${Date.now()}-${Math.random()}`,
        text: normalizeSearchHistoryText(item.text),
      }))
      .slice(0, MAX_ITEMS)
  } catch {
    return []
  }
}

export function saveSearchKeywordHistory(list) {
  localStorage.setItem(LOCAL_KEY, JSON.stringify((list || []).slice(0, MAX_ITEMS)))
}

/** 新增一条（相同内容移到最前） */
export function pushSearchKeywordHistory(text, list) {
  const entry = normalizeSearchHistoryText(text)
  if (!entry) return list || []
  const prev = (list || []).filter((item) => item.text !== entry)
  const next = [{ id: `${Date.now()}`, text: entry }, ...prev]
  saveSearchKeywordHistory(next)
  return next
}

export function removeSearchKeywordHistory(id, list) {
  const next = (list || []).filter((item) => item.id !== id)
  saveSearchKeywordHistory(next)
  return next
}

export function clearSearchKeywordHistory() {
  saveSearchKeywordHistory([])
  return []
}

/** 标签展示：单行截断，多行显示首行+… */
export function formatSearchHistoryLabel(text, maxLen = 36) {
  const lines = String(text).split('\n').map((s) => s.trim()).filter(Boolean)
  const one = lines.length > 1 ? `${lines[0]}…(+${lines.length - 1})` : lines[0] || text
  if (one.length <= maxLen) return one
  return `${one.slice(0, maxLen)}…`
}
