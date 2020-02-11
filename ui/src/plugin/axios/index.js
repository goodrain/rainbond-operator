// import store from '@/store'
import axios from 'axios'
import { Message } from 'element-ui'
// import util from '@/libs/util'
import qs from 'qs'

function handleResponseCode (res) {
  Message({
    message: res.msg,
    type: 'error',
    duration: 3 * 1000
  })
  return Promise.reject(res)
}

// 创建一个错误
// function errorCreate (msg) {
//   const error = new Error(msg)
//   errorLog(error)
//   throw error
// }

// 记录和显示错误
// function errorLog (error) {
//   // 添加到日志
//   store.dispatch('d2admin/log/push', {
//     message: '数据请求异常',
//     type: 'danger',
//     meta: {
//       error
//     }
//   })
//   // 打印到控制台
//   if (process.env.NODE_ENV === 'development') {
//     util.log.danger('>>>>>> Error >>>>>>')
//     console.log(error)
//   }
//   // 显示提示
//   Message({
//     message: error.message,
//     type: 'error',
//     duration: 5 * 1000
//   })
// }
// 创建一个 axios 实例
const service = axios.create({
  baseURL: process.env.VUE_APP_API,
  timeout: 90000 // 请求超时时间
})
axios.defaults.withCredentials = true

// 请求拦截器
service.interceptors.request.use(
  config => {
    // 在请求发送之前做一些处理
    // const token = util.cookies.get('token')
    // if (token) {
    // 让每个请求携带token-- ['X-Token']为自定义key 请根据实际情况自行修改

    // config.headers = {
    //   Accept: 'application/json',
    //   'Content-Type': 'application/json; charset=utf-8',
    //   'Access-Control-Allow-Origin':'*',
    //   ...config.headers
    // }
    // config.headers['Authorization'] = token
    if (config.method === 'PUT') {
      // const qs = requrie('qs')
      config.data = qs.stringify(config.data)
    }
    // }

    return config
  },
  error => {
    // 发送失败
    console.log(error)
    return Promise.reject(error)
  }
)

// 响应拦截器
service.interceptors.response.use(
  response => {
    // dataAxios 是 axios 返回数据中的 data
    const dataAxios = response.data
    // 这个状态码是和后端约定的
    const { code } = dataAxios

    // 根据 code 进行判断
    if (code === undefined) {
      // 如果没有 code 代表这不是项目后端开发的接口 比如可能是 D2Admin 请求最新版本
      return dataAxios
    } else {
      if (code >= 300 && code <= 500) {
        return handleResponseCode(dataAxios)
      } else {
        return dataAxios
      }
    }
  },
  error => {
    console.log('error', error)

    if (error.response && error.response.data) {
      const dataAxios = error.response.data
      return handleResponseCode(dataAxios)
    }
    Message({
      message: '数据请求发生错误',
      type: 'error',
      duration: 3 * 1000
    })
    return Promise.reject(error.response)
  }
)

export default service
