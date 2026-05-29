import { getDeviceId } from '../utils/device'

const baseURL = import.meta.env.VITE_APP_BASE_API || '/api'

/**
 * Jira 导入（SSE），onProgress({ percent, current, total, filename, phase, message })
 */
export async function jiraImportStream(body, onProgress) {
  const res = await fetch(`${baseURL}/jira/import/stream`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Device-ID': getDeviceId(),
    },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    const text = await res.text()
    let msg = text
    try {
      const j = JSON.parse(text)
      msg = j.error || j.message || text
    } catch (_) {}
    throw new Error(msg || `请求失败 ${res.status}`)
  }
  const reader = res.body?.getReader()
  if (!reader) {
    throw new Error('浏览器不支持流式响应')
  }
  const decoder = new TextDecoder()
  let buffer = ''
  let result = null

  const dispatch = (block) => {
    let event = 'message'
    let data = ''
    for (const line of block.split('\n')) {
      if (line.startsWith('event:')) {
        event = line.slice(6).trim()
      } else if (line.startsWith('data:')) {
        data += line.slice(5).trim()
      }
    }
    if (!data) return
    const payload = JSON.parse(data)
    if (event === 'progress' && onProgress) {
      onProgress(payload)
    } else if (event === 'error') {
      throw new Error(payload.message || '导入失败')
    } else if (event === 'done') {
      result = payload
    }
  }

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const parts = buffer.split('\n\n')
    buffer = parts.pop() || ''
    for (const block of parts) {
      if (block.trim()) dispatch(block)
    }
  }
  if (buffer.trim()) dispatch(buffer)
  if (!result) {
    throw new Error('导入未完成')
  }
  return result
}
