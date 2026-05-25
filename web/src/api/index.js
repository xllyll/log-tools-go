import axios from 'axios'
import { getDeviceId } from '../utils/device'

const http = axios.create({
  baseURL: import.meta.env.VITE_APP_BASE_API,
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
    }).then((res) => res)
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
  retryIngest(id) {
    return http.post(`/files/${id}/retry`)
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
  listSceneLibrary() {
    return http.get('/scene-library')
  },
  getSceneLibrary(id) {
    return http.get(`/scene-library/${id}`)
  },
  publishSceneLibrary(body) {
    return http.post('/scene-library', body)
  },
  deleteSceneLibrary(id) {
    return http.delete(`/scene-library/${id}`)
  },
  jiraAttachments(issueKey) {
    return http.get(`/jira/issues/${encodeURIComponent(issueKey)}/attachments`)
  },
  jiraImport(body) {
    return http.post('/jira/import', body)
  },
}
