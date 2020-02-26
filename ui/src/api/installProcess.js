import request from '@/plugin/axios'
import setting from '../setting'
//  获取全局状态
export function getState () {
  return request({
    url: `${setting.apiHost}/cluster/status`,
    method: 'get'
  })
}

export function getClusterInitConfig () {
  return request({
    url: `${setting.apiHost}/cluster/status-info`,
    method: 'get'
  })
}

//  获取全局状态
export function putInit () {
  return request({
    url: `${setting.apiHost}/cluster/init`,
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
    url: `${setting.apiHost}/cluster/configs`,
    method: 'get'
  })
}

//  查询安装检测结果
export function detectionCluster () {
  return request({
    url: `${setting.apiHost}/cluster/install/status`,
    method: 'get'
  })
}

//  修改集群配置信息
export function putClusterConfig (data) {
  return request({
    url: `${setting.apiHost}/cluster/configs`,
    method: 'PUT',
    data
  })
}
//
export function installCluster () {
  return request({
    url: `${setting.apiHost}/cluster/install`,
    method: 'post'
  })
}
//  安装集群配置结果
export function getClusterInstallResults () {
  return request({
    url: `${setting.apiHost}/cluster/install/status`,
    method: 'get'
  })
}
//  安装集群配置结果
export function getClusterInstallResultsState (params) {
  return request({
    url: `${setting.apiHost}/cluster/components`,
    method: 'get',
    params: {
      isInit: params ? params.isInit : false
    }
  })
}
//  访问地址
export function getAccessAddress () {
  return request({
    url: `${setting.apiHost}/cluster/address`,
    method: 'get'
  })
}
//  平台安装包卸载
export function deleteUnloadingPlatform () {
  return request({
    url: `${setting.apiHost}/cluster/uninstall`,
    method: 'DELETE'
  })
}

export function queryNode (params) {
  return request({
    url: params.mock ? 'http://doc.goodrain.org/mock/48/cluster/nodes' : `${setting.apiHost}/cluster/nodes`,
    method: 'GET',
    params: params
  })
}
