const KEY = 'log_tools_theme'

export function getPreferredTheme() {
  const saved = localStorage.getItem(KEY)
  if (saved === 'light' || saved === 'dark') return saved
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export function applyTheme(theme) {
  const root = document.documentElement
  root.classList.toggle('dark', theme === 'dark')
  root.dataset.theme = theme
  localStorage.setItem(KEY, theme)
}

export function initTheme() {
  applyTheme(getPreferredTheme())
}
