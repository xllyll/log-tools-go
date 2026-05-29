export function escapeHtml(text) {
  return String(text)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function escapeRegExp(s) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function collectRanges(text, keywords, useRegex, caseSensitive = false) {
  const ranges = []
  const flags = caseSensitive ? 'g' : 'gi'
  for (const kw of keywords) {
    if (!kw) continue
    let re
    if (useRegex) {
      try {
        re = new RegExp(kw, flags)
      } catch {
        continue
      }
    } else {
      re = new RegExp(escapeRegExp(kw), flags)
    }
    let m
    const src = String(text)
    while ((m = re.exec(src)) !== null) {
      ranges.push({ start: m.index, end: m.index + m[0].length })
      if (m[0].length === 0) {
        re.lastIndex += 1
        if (re.lastIndex > src.length) break
      }
    }
  }
  if (!ranges.length) return []
  ranges.sort((a, b) => a.start - b.start || b.end - a.end)
  const merged = [ranges[0]]
  for (let i = 1; i < ranges.length; i++) {
    const cur = ranges[i]
    const last = merged[merged.length - 1]
    if (cur.start <= last.end) {
      last.end = Math.max(last.end, cur.end)
    } else {
      merged.push(cur)
    }
  }
  return merged
}

function rangeOverlaps(range, start, end) {
  return range.start < end && range.end > start
}

function renderSegments(src, sceneRanges, searchRanges) {
  if (!sceneRanges.length && !searchRanges.length) return escapeHtml(src)

  const points = new Set([0, src.length])
  for (const r of [...sceneRanges, ...searchRanges]) {
    points.add(r.start)
    points.add(r.end)
  }
  const sorted = [...points].sort((a, b) => a - b)

  let out = ''
  for (let i = 0; i < sorted.length - 1; i++) {
    const start = sorted[i]
    const end = sorted[i + 1]
    if (start >= end) continue
    const inScene = sceneRanges.some((r) => rangeOverlaps(r, start, end))
    const inSearch = searchRanges.some((r) => rangeOverlaps(r, start, end))
    let part = escapeHtml(src.slice(start, end))
    if (inScene) part = `<strong class="scene-kw-bold">${part}</strong>`
    if (inSearch) part = `<mark class="kw-highlight">${part}</mark>`
    out += part
  }
  return out
}

/**
 * 日志行高亮：搜索词 mark 背景；场景匹配关键字加粗
 * @param {string} text
 * @param {string[]} searchKeywords
 * @param {boolean} useRegex
 * @param {{ keyword: string, mode?: number, case_sensitive?: number }[]} sceneKeywords
 */
export function highlightLogLine(text, searchKeywords, useRegex = false, sceneKeywords = []) {
  const src = text == null ? '' : String(text)
  const searchKws = (searchKeywords || []).map((k) => String(k).trim()).filter(Boolean)
  const sceneRanges = []
  for (const item of sceneKeywords || []) {
    const kw = item?.keyword
    if (!kw) continue
    const useRe = item.mode === 1 || item.mode === '1' || item.mode === 'regex'
    const cs = item.case_sensitive === 1 || item.case_sensitive === '1'
    for (const r of collectRanges(src, [kw], useRe, cs)) {
      sceneRanges.push(r)
    }
  }
  const searchRanges = searchKws.length ? collectRanges(src, searchKws, useRegex) : []
  return renderSegments(src, sceneRanges, searchRanges)
}

/** 在已转义 HTML 安全的前提下，为搜索关键词加 <mark> 高亮 */
export function highlightKeywords(text, keywords, useRegex = false) {
  return highlightLogLine(text, keywords, useRegex, [])
}
