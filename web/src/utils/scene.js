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

/** 从已选场景名收集 keywords，供服务端 OR 查询 */
export function collectSceneKeywords(config, selectedSceneNames) {
  const keywords = []
  const meta = []
  if (!config?.modules || !selectedSceneNames?.length) return { keywords, meta }

  for (const mod of config.modules) {
    for (const scene of mod.scenes || []) {
      if (!selectedSceneNames.includes(scene.name)) continue
      for (const kw of scene.keywords || []) {
        keywords.push(kw.keyword)
        meta.push({ keyword: kw.keyword, desc: kw.desc, color: kw.color, mode: kw.mode || 'word' })
      }
    }
  }
  return { keywords, meta }
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
    const suffix = sceneDesc ? ` [${sceneDesc}]` : ''
    return {
      ...e,
      color,
      scene_desc: sceneDesc,
      display: content + suffix,
    }
  })
}
