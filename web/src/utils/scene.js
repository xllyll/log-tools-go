const LOCAL_KEY = 'log_tools_scene_config'

export const defaultSceneConfig = () => ({
  modules: [
    {
      name: 'DeviceService',
      scenes: [
        {
          name: 'System问题分析',
          keywords: [
            { keyword: '--------- beginning of main', desc: '系统启动', mode: 'word', color: '#ff3333' },
            { keyword: '00 09 0a', desc: '系统重启原因', mode: 'word', color: '#33ff33' },
          ],
        },
      ],
    },
  ],
})

export function loadLocalScene() {
  try {
    const raw = localStorage.getItem(LOCAL_KEY)
    if (raw) return JSON.parse(raw)
  } catch (_) {}
  return defaultSceneConfig()
}

export function saveLocalScene(config) {
  localStorage.setItem(LOCAL_KEY, JSON.stringify(config))
}

export function cloneSceneConfig(config) {
  return JSON.parse(JSON.stringify(config || defaultSceneConfig()))
}

/** 将场景库配置合并到本地（按模块名、场景名去重） */
export function mergeSceneConfig(local, remote) {
  const out = cloneSceneConfig(local?.modules?.length ? local : defaultSceneConfig())
  if (!remote?.modules?.length) return out
  for (const mod of remote.modules) {
    let targetMod = out.modules.find((m) => m.name === mod.name)
    if (!targetMod) {
      out.modules.push(cloneSceneConfig({ modules: [mod] }).modules[0])
      continue
    }
    if (!targetMod.scenes) targetMod.scenes = []
    for (const scene of mod.scenes || []) {
      if (!targetMod.scenes.some((s) => s.name === scene.name)) {
        targetMod.scenes.push(JSON.parse(JSON.stringify(scene)))
      }
    }
  }
  return out
}

export function shortDeviceLabel(deviceId) {
  if (!deviceId) return '未知'
  return deviceId.length > 8 ? `${deviceId.slice(0, 8)}…` : deviceId
}

export function emptyKeyword() {
  return { keyword: '', desc: '', mode: 'word', color: '#409eff' }
}

/** 搜索侧下拉：模块分组，仅场景可选 */
export function buildSceneSelectGroups(config) {
  const groups = []
  ;(config?.modules || []).forEach((mod, mi) => {
    const moduleName = mod.name?.trim() || `模块 ${mi + 1}`
    const options = []
    ;(mod.scenes || []).forEach((scene, si) => {
      const label = scene.name?.trim() || `场景 ${si + 1}`
      options.push({ key: `${mi}:${si}`, label })
    })
    if (options.length) groups.push({ moduleName, options })
  })
  return groups
}

function pushSceneKeywords(keywords, meta, scene) {
  for (const kw of scene.keywords || []) {
    if (!kw.keyword) continue
    keywords.push(kw.keyword)
    meta.push({ keyword: kw.keyword, desc: kw.desc, color: kw.color, mode: kw.mode || 'word' })
  }
}

/** 从已选场景收集 keywords（selected 为 mi:si 或兼容旧版场景名） */
export function collectSceneKeywords(config, selected) {
  const keywords = []
  const meta = []
  if (!config?.modules?.length || !selected?.length) return { keywords, meta }

  const selectedSet = new Set(selected)
  const usedKeys = new Set()

  for (const item of selected) {
    if (typeof item === 'string' && item.includes(':')) {
      const [mi, si] = item.split(':').map((n) => parseInt(n, 10))
      const scene = config.modules[mi]?.scenes?.[si]
      if (!scene || usedKeys.has(item)) continue
      usedKeys.add(item)
      pushSceneKeywords(keywords, meta, scene)
    }
  }

  // 兼容旧数据：仅保存了场景名称
  config.modules.forEach((mod, mi) => {
    ;(mod.scenes || []).forEach((scene, si) => {
      const name = scene.name?.trim()
      if (!name || !selectedSet.has(name)) return
      const key = `${mi}:${si}`
      if (usedKeys.has(key)) return
      usedKeys.add(key)
      pushSceneKeywords(keywords, meta, scene)
    })
  })

  return { keywords, meta }
}

/** 场景 desc 标签样式：文字用关键词配置色，背景/圆角由 .scene-desc 统一 */
export function sceneDescStyle(color) {
  if (!color) return {}
  return { color }
}

/** 前端为匹配行附加 desc 与颜色 */
export function decorateEntries(entries, meta) {
  if (!meta.length) return entries
  return entries.map((e) => {
    const content = e.content || e.message || ''
    let sceneDesc = ''
    let color = e.color || ''
    for (const m of meta) {
      const hit =
        m.mode === 'regex'
          ? new RegExp(m.keyword, 'i').test(content)
          : content.includes(m.keyword)
      if (hit) {
        sceneDesc = m.desc
        color = m.color || color
        break
      }
    }
    return {
      ...e,
      color,
      scene_desc: sceneDesc,
      display: content,
    }
  })
}
