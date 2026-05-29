const LOCAL_KEY = 'log_tools_scene_config'

/** 0 关键字 1 正则 */
export const KW_MODE_WORD = 0
export const KW_MODE_REGEX = 1
/** 0 不区分大小写 1 区分 */
export const KW_CASE_IGNORE = 0
export const KW_CASE_SENSITIVE = 1

export function normalizeKeywordMode(mode) {
  if (mode === 1 || mode === '1' || mode === 'regex') return KW_MODE_REGEX
  return KW_MODE_WORD
}

export function normalizeCaseSensitive(v) {
  if (v === 1 || v === '1' || v === true) return KW_CASE_SENSITIVE
  return KW_CASE_IGNORE
}

export function normalizeSceneKeyword(kw) {
  if (!kw) return kw
  kw.mode = normalizeKeywordMode(kw.mode)
  kw.case_sensitive = normalizeCaseSensitive(kw.case_sensitive ?? kw.caseSensitive)
  delete kw.caseSensitive
  return kw
}

export function normalizeSceneConfig(config) {
  if (!config?.modules) return config
  for (const mod of config.modules) {
    for (const scene of mod.scenes || []) {
      for (const kw of scene.keywords || []) {
        normalizeSceneKeyword(kw)
      }
    }
  }
  return config
}

/** 判断日志行是否命中该场景关键字规则 */
export function keywordMatchesContent(content, rule) {
  const keyword = rule?.keyword
  if (!keyword) return false
  const regex = normalizeKeywordMode(rule.mode) === KW_MODE_REGEX
  const cs = normalizeCaseSensitive(rule.case_sensitive) === KW_CASE_SENSITIVE
  if (regex) {
    try {
      return new RegExp(keyword, cs ? '' : 'i').test(content)
    } catch {
      return false
    }
  }
  if (cs) return content.includes(keyword)
  return content.toLowerCase().includes(keyword.toLowerCase())
}

export const defaultSceneConfig = () => ({
  modules: [
    {
      name: 'DeviceService',
      scenes: [
        {
          name: 'System问题分析',
          keywords: [
              {
                  keyword: "--------- beginning of main",
                  desc: "系统启动",
                  mode: 0,
                  case_sensitive: 0,
                  color: "#ff3333"
              },
              {
                  keyword: "00 09 0a",
                  desc: "系统重启原因",
                  mode: 0,
                  case_sensitive: 0,
                  color: "#0de8d2"
              },
              {
                  keyword: "onLocalAccChanged: false",
                  desc: "ACC关闭",
                  mode: 0,
                  case_sensitive: 0,
                  color: "#ffda33"
              },
              {
                  keyword: "onLocalAccChanged: true",
                  desc: "ACC打开",
                  mode: 0,
                  case_sensitive: 0,
                  color: "#3396ff"
              },
              {
                  keyword: "BluetoothManagerService: ACTION_QB_POWERON",
                  desc: "唤醒",
                  mode: 0,
                  case_sensitive: 0,
                  color: "#030a32"
              },
              {
                  keyword: "BluetoothManagerService: ACTION_QB_POWEROFF",
                  desc: "休眠",
                  mode: 0,
                  case_sensitive: 0,
                  color: "#026d02"
              },
              {
                  keyword: "ActivityTaskManager: START u0",
                  desc: "打开应用",
                  mode: 0,
                  case_sensitive: 0,
                  color: "#75b5b1"
              }
          ]
        },
      ],
    },
  ],
})

export function loadLocalScene() {
  try {
    const raw = localStorage.getItem(LOCAL_KEY)
    if (raw) return normalizeSceneConfig(JSON.parse(raw))
  } catch (_) {}
  return defaultSceneConfig()
}

export function saveLocalScene(config) {
  localStorage.setItem(LOCAL_KEY, JSON.stringify(config))
}

export function cloneSceneConfig(config) {
  return normalizeSceneConfig(JSON.parse(JSON.stringify(config || defaultSceneConfig())))
}

function mergeSceneKeywords(targetScene, incomingScene) {
  if (!incomingScene?.keywords?.length) return
  if (!targetScene.keywords) targetScene.keywords = []
  const seen = new Set(targetScene.keywords.map((k) => k.keyword).filter(Boolean))
  for (const kw of incomingScene.keywords) {
    if (!kw?.keyword || seen.has(kw.keyword)) continue
    targetScene.keywords.push(normalizeSceneKeyword(JSON.parse(JSON.stringify(kw))))
    seen.add(kw.keyword)
  }
}

/** 将远程配置合并到本地：同模块/场景名则追加关键词，否则新增模块或场景 */
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
      const existing = targetMod.scenes.find((s) => s.name === scene.name)
      if (!existing) {
        targetMod.scenes.push(JSON.parse(JSON.stringify(scene)))
        continue
      }
      mergeSceneKeywords(existing, scene)
    }
  }
  return normalizeSceneConfig(out)
}

export function shortDeviceLabel(deviceId) {
  if (!deviceId) return '未知'
  return deviceId.length > 8 ? `${deviceId.slice(0, 8)}…` : deviceId
}

export function emptyKeyword() {
  return { keyword: '', desc: '', mode: KW_MODE_WORD, case_sensitive: KW_CASE_IGNORE, color: '#409eff' }
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

/** 搜索侧：模块下拉选项 */
export function buildModuleSelectOptions(config) {
  return (config?.modules || []).map((mod, mi) => ({
    value: mi,
    label: mod.name?.trim() || `模块 ${mi + 1}`,
  }))
}

/** 搜索侧：当前模块下的场景选项（第二个下拉） */
export function buildSceneSelectOptionsForModule(config, moduleIndex) {
  const mi = moduleIndex
  if (!Number.isInteger(mi) || mi < 0 || mi >= (config?.modules?.length || 0)) return []
  const mod = config.modules[mi]
  return (mod.scenes || []).map((scene, si) => ({
    value: `${mi}:${si}`,
    label: scene.name?.trim() || `场景 ${si + 1}`,
  }))
}

/** 去掉配置中已不存在的场景 key */
export function pruneSceneKeys(config, keys) {
  const allowed = new Set()
  for (const g of buildSceneSelectGroups(config)) {
    for (const o of g.options) allowed.add(o.key)
  }
  return (keys || []).filter((k) => allowed.has(k))
}

/** 搜索侧树形选择：模块为父节点（不可选），场景为叶子 */
export function buildSceneSelectTree(config) {
  const tree = []
  ;(config?.modules || []).forEach((mod, mi) => {
    const children = (mod.scenes || []).map((scene, si) => ({
      value: `${mi}:${si}`,
      label: scene.name?.trim() || `场景 ${si + 1}`,
    }))
    if (!children.length) return
    tree.push({
      value: `mod:${mi}`,
      label: mod.name?.trim() || `模块 ${mi + 1}`,
      disabled: true,
      children,
    })
  })
  return tree
}

function pushSceneKeywords(specs, meta, scene) {
  for (const kw of scene.keywords || []) {
    if (!kw.keyword) continue
    normalizeSceneKeyword(kw)
    const item = {
      keyword: kw.keyword,
      mode: kw.mode,
      case_sensitive: kw.case_sensitive,
    }
    specs.push({ ...item })
    meta.push({ ...item, desc: kw.desc, color: kw.color })
  }
}

/** 从已选场景收集查询规格（selected 为 mi:si 或兼容旧版场景名） */
export function collectSceneKeywords(config, selected) {
  const specs = []
  const meta = []
  if (!config?.modules?.length || !selected?.length) return { specs, meta }

  const selectedSet = new Set(selected)
  const usedKeys = new Set()

  for (const item of selected) {
    if (typeof item === 'string' && item.includes(':')) {
      const [mi, si] = item.split(':').map((n) => parseInt(n, 10))
      const scene = config.modules[mi]?.scenes?.[si]
      if (!scene || usedKeys.has(item)) continue
      usedKeys.add(item)
      pushSceneKeywords(specs, meta, scene)
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
      pushSceneKeywords(specs, meta, scene)
    })
  })

  return { specs, meta }
}

/** 场景 desc 标签样式：文字用关键词配置色，背景/圆角由 .scene-desc 统一 */
export function sceneDescStyle(color) {
  if (!color) return {}
  return { color }
}

function sceneKeywordHits(content, meta) {
  const hits = []
  for (const m of meta) {
    if (!m.keyword) continue
    if (keywordMatchesContent(content, m)) {
      hits.push({
        keyword: m.keyword,
        mode: normalizeKeywordMode(m.mode),
        case_sensitive: normalizeCaseSensitive(m.case_sensitive),
        desc: m.desc,
        color: m.color,
      })
    }
  }
  return hits
}

/** 前端为匹配行附加 desc 与颜色 */
export function decorateEntries(entries, meta) {
  if (!meta.length) return entries
  return entries.map((e) => {
    const content = e.content || e.message || ''
    const matched = sceneKeywordHits(content, meta)
    const first = matched[0]
    return {
      ...e,
      color: first?.color || e.color || '',
      scene_desc: first?.desc || '',
      scene_match_keywords: matched.map(({ keyword, mode, case_sensitive }) => ({
        keyword,
        mode,
        case_sensitive,
      })),
      display: content,
    }
  })
}
