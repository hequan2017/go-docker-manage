import axios from 'axios'
import { ElMessage } from 'element-plus'

const http = axios.create({ baseURL: '/api', timeout: 35000 })

http.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

http.interceptors.response.use(
  (resp) => {
    const body = resp.data
    if (body && body.code !== undefined && body.code !== 0) {
      ElMessage.error(body.msg || '请求失败')
      return Promise.reject(new Error(body.msg))
    }
    return body?.data !== undefined ? body.data : body
  },
  (err) => {
    const msg = err.response?.data?.msg || err.message || '网络错误'
    if (err.response?.status === 401) {
      localStorage.removeItem('token')
      if (location.pathname !== '/login') location.href = '/login'
    }
    ElMessage.error(msg)
    return Promise.reject(err)
  },
)

export default http
