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

function collectRanges(text, keywords, useRegex) {
  const ranges = []
  for (const kw of keywords) {
    if (!kw) continue
    let re
    if (useRegex) {
      try {
        re = new RegExp(kw, 'gi')
      } catch {
        continue
      }
    } else {
      re = new RegExp(escapeRegExp(kw), 'gi')
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

/** 在已转义 HTML 安全的前提下，为搜索关键词加 <mark> 高亮 */
export function highlightKeywords(text, keywords, useRegex = false) {
  const src = text == null ? '' : String(text)
  const kws = (keywords || []).map((k) => String(k).trim()).filter(Boolean)
  if (!kws.length) return escapeHtml(src)

  const ranges = collectRanges(src, kws, useRegex)
  if (!ranges.length) return escapeHtml(src)

  let out = ''
  let pos = 0
  for (const { start, end } of ranges) {
    if (start > pos) {
      out += escapeHtml(src.slice(pos, start))
    }
    out += `<mark class="kw-highlight">${escapeHtml(src.slice(start, end))}</mark>`
    pos = end
  }
  if (pos < src.length) {
    out += escapeHtml(src.slice(pos))
  }
  return out
}
