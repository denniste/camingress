import axios from 'axios'

// 统一 API 客户端
// 若配置了 VITE_API_KEY, 自动附加 Bearer 鉴权头 (与后端 VIDEHUB_API_KEY 对应)
const api = axios.create({ baseURL: '' })

api.interceptors.request.use((config) => {
  const key = import.meta.env.VITE_API_KEY
  if (key) {
    config.headers.Authorization = `Bearer ${key}`
  }
  return config
})

export default api
