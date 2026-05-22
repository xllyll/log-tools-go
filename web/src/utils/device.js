const KEY = 'log_tools_device_id'

export function getDeviceId() {
  let id = localStorage.getItem(KEY)
  if (!id) {
    id = crypto.randomUUID?.() || `dev_${Date.now()}_${Math.random().toString(36).slice(2)}`
    localStorage.setItem(KEY, id)
  }
  return id
}
