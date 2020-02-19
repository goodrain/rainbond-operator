import request from '@/plugin/axios'

//  获取全局状态
export function getState () {
  return request({
    url:
      // 'http://doc.goodrain.org/mock/48/cluster/status',
      '/cluster/status',
    method: 'get'
  })
}

//  获取全局状态
export function putInit () {
  return request({
    url:
      // 'http://doc.goodrain.org/mock/48/cluster/init',
      '/cluster/init',
    method: 'post'
  })
}

//  保存安装记录
export function putRecord (data) {
  return request({
    url:
      'https://log.rainbond.com/log/install',
      // '/log/install',
    method: 'post',
    data
  })
}

//  获取集群配置信息
export function getClusterInfo () {
  return request({
    url:
      // 'http://doc.goodrain.org/mock/48/cluster/configs',
      '/cluster/configs',
    method: 'get'
  })
}

//  查询安装检测结果
export function detectionCluster () {
  return request({
    url:
      // 'http://doc.goodrain.org/mock/48/cluster/install/status',
      '/cluster/install/status',
    method: 'get'
  })
}

//  修改集群配置信息
export function putClusterInfo (data) {
  return request({
    url:
      // 'http://doc.goodrain.org/mock/48/cluster/configs',
      '/cluster/configs',
    method: 'PUT',
    data
  })
}
//  添加集群配置信息
export function addsCluster () {
  return request({
    url:
      // 'http://doc.goodrain.org/mock/48/cluster/install',
      '/cluster/install',
    method: 'post'
  })
}
//  安装集群配置结果
export function getClusterInstallResults () {
  return request({
    url:
      // 'http://doc.goodrain.org/mock/48/cluster/install/status',
      '/cluster/install/status',
    method: 'get'
  })
}
//  安装集群配置结果
export function getClusterInstallResultsState (params) {
  return request({
    url:
      //  'http://doc.goodrain.org/mock/48/cluster/components',
      '/cluster/components',
    method: 'get',
    params: {
      isInit: params ? params.isInit : false
    }
  })
}
//  访问地址
export function getAccessAddress () {
  return request({
    url:
      //  'http://doc.goodrain.org/mock/48/cluster/address',
      '/cluster/address',
    method: 'get'
  })
}
//  平台安装包卸载
export function deleteUnloadingPlatform () {
  return request({
    url:
      //  'http://doc.goodrain.org/mock/48/cluster/uninstall',
      '/cluster/uninstall',
    method: 'DELETE'
  })
}
