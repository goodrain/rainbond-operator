import request from '@/plugin/axios'

//  获取全局状态
export function getState () {
  return request({
    url: `/cluster/status`,
    method: 'get'
  })
}
//  获取判断是否初始化管理员
export function getIsAdmin () {
  return request({
    url: `/user/generate`,
    method: 'get'
  })
}

//  用户登录
export function Login (data) {
  return request({
    url: `/user/login`,
    method: 'post',
    data
  })
}

export function getClusterInitConfig () {
  return request({
    url: `/cluster/status-info`,
    method: 'get'
  })
}

//  初始化管理员
export function putGenerateAdmin () {
  return request({
    url: '/user/generate',
    method: 'post'
  })
}
//  获取全局状态
export function putInit () {
  return request({
    url: `/cluster/init`,
    method: 'post'
  })
}

//  保存安装记录
export function putRecord (data) {
  return request({
    url: 'https://log.rainbond.com/log/install',
    method: 'post',
    data
  })
}

//  获取集群配置信息
export function getClusterInfo () {
  return request({
    url: `/cluster/configs`,
    method: 'get'
  })
}

//  查询安装检测结果
export function detectionCluster () {
  return request({
    url: `/cluster/install/status`,
    method: 'get'
  })
}

//  修改集群配置信息
export function putClusterConfig (data) {
  return request({
    url: `/cluster/configs`,
    method: 'PUT',
    data
  })
}
//
export function installCluster (data) {
  return request({
    url: `/cluster/install`,
    method: 'post',
    data
  })
}
//  安装集群配置结果
export function getClusterInstallResults () {
  return request({
    url: `/cluster/install/status`,
    method: 'get'
  })
}
//  安装集群配置结果
export function getClusterInstallResultsState (params) {
  return request({
    url: `/cluster/components`,
    method: 'get',
    params: {
      isInit: params ? params.isInit : false
    }
  })
}
//  访问地址
export function getAccessAddress () {
  return request({
    url: `/cluster/address`,
    method: 'get'
  })
}
//  平台安装包卸载
export function deleteUnloadingPlatform () {
  return request({
    url: `/cluster/uninstall`,
    method: 'DELETE'
  })
}

export function queryNode (params) {
  return request({
    url: params.mock ? 'http://doc.goodrain.org/mock/48/cluster/nodes' : `/cluster/nodes`,
    method: 'GET',
    params: params
  })
}
