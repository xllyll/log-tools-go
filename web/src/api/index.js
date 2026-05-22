import axios from 'axios'
import { getDeviceId } from '../utils/device'

const http = axios.create({
  baseURL: '/api',
  timeout: 120000,
})

http.interceptors.request.use((config) => {
  config.headers['X-Device-ID'] = getDeviceId()
  return config
})

export const api = {
  upload(file, onProgress) {
    const fd = new FormData()
    fd.append('file', file)
    return http.post('/upload', fd, {
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: onProgress,
    })
  },
  listFiles() {
    return http.get('/files')
  },
  fileStatus(id) {
    return http.get(`/files/${id}`)
  },
  deleteFile(id) {
    return http.delete(`/files/${id}`)
  },
  batchDelete(ids) {
    return http.post('/files/batch-delete', { ids })
  },
  queryLogs(body) {
    return http.post('/logs/query', body)
  },
  logContext(params) {
    return http.get('/logs/context', { params })
  },
  saveScene(data) {
    return http.post('/scenes', data)
  },
  listScenes() {
    return http.get('/scenes')
  },
  jiraAttachments(key, config) {
    return http.post(`/jira/issues/${key}/attachments`, config)
  },
  jiraImport(body) {
    return http.post('/jira/import', body)
  },
}
