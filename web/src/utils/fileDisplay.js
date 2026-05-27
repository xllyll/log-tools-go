/** User-visible file name (original upload name, not storage dedup name). */
export function displayFileName(file) {
  if (!file) return ''
  return file.original_name || file.name || ''
}
