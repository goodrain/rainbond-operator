import {
  getClusterInfo,
  getState,
  getIsAdmin,
  getClusterInitConfig,
  putClusterConfig,
  installCluster,
  detectionCluster,
  getClusterInstallResults,
  getClusterInstallResultsState,
  getAccessAddress,
  deleteUnloadingPlatform,
  putInit,
  putRecord,
  queryNode,
  putGenerateAdmin,
  Login
} from '@/api/installProcess'
import util from '@/libs/util.js'

const installProcess = {
  state: {},
  mutations: {},
  actions: {
    Login ({ commit }, resdata) {
      return new Promise((resolve, reject) => {
        Login(resdata)
          .then(response => {
            const tokenStr = response.data.token
            util.cookies.set('token', tokenStr)
            resolve(response)
          })
          .catch(error => {
            reject(error)
          })
      })
    },
    putInit ({ commit }, resdata) {
      return new Promise((resolve, reject) => {
        putInit(resdata)
          .then(response => {
            resolve(response)
          })
          .catch(error => {
            reject(error)
          })
      })
    },
    putRecord ({ commit }, resdata) {
      return new Promise((resolve, reject) => {
        putRecord(resdata)
          .then(response => {
            resolve(response)
          })
          .catch(error => {
            reject(error)
          })
      })
    },

    fetchState ({ commit }, resdata) {
      return new Promise((resolve, reject) => {
        getState(resdata)
          .then(response => {
            resolve(response)
          })
          .catch(error => {
            reject(error)
          })
      })
    },
    fetchIsAdmin ({ commit }, resdata) {
      return new Promise((resolve, reject) => {
        getIsAdmin(resdata)
          .then(response => {
            resolve(response)
          })
          .catch(error => {
            reject(error)
          })
      })
    },

    fetchGenerateAdmin ({ commit }, resdata) {
      return new Promise((resolve, reject) => {
        putGenerateAdmin(resdata)
          .then(response => {
            resolve(response)
          })
          .catch(error => {
            reject(error)
          })
      })
    },

    getClusterInitConfig ({ commit }, resdata) {
      return new Promise((resolve, reject) => {
        getClusterInitConfig(resdata)
          .then(response => {
            resolve(response)
          })
          .catch(error => {
            reject(error)
          })
      })
    },

    fetchClusterInfo ({ commit }, resdata) {
      return new Promise((resolve, reject) => {
        getClusterInfo(resdata)
          .then(response => {
            resolve(response)
          })
          .catch(error => {
            reject(error)
          })
      })
    },
    putClusterInfo ({ commit }, resdata) {
      return new Promise((resolve, reject) => {
        putClusterConfig(resdata)
          .then(response => {
            resolve(response)
          })
          .catch(error => {
            reject(error)
          })
      })
    },
    installCluster ({ commit }, resdata) {
      return new Promise((resolve, reject) => {
        installCluster(resdata)
          .then(response => {
            resolve(response)
          })
          .catch(error => {
            reject(error)
          })
      })
    },
    detectionCluster ({ commit }, resdata) {
      return new Promise((resolve, reject) => {
        detectionCluster(resdata)
          .then(response => {
            resolve(response)
          })
          .catch(error => {
            reject(error)
          })
      })
    },

    fetchClusterInstallResults ({ commit }, resdata) {
      return new Promise((resolve, reject) => {
        getClusterInstallResults(resdata)
          .then(response => {
            resolve(response)
          })
          .catch(error => {
            reject(error)
          })
      })
    },
    fetchClusterInstallResultsState ({ commit }, resdata) {
      return new Promise((resolve, reject) => {
        getClusterInstallResultsState(resdata)
          .then(response => {
            resolve(response)
          })
          .catch(error => {
            reject(error)
          })
      })
    },
    fetchAccessAddress ({ commit }, resdata) {
      return new Promise((resolve, reject) => {
        getAccessAddress(resdata)
          .then(response => {
            resolve(response)
          })
          .catch(error => {
            reject(error)
          })
      })
    },
    deleteUnloadingPlatform ({ commit }, resdata) {
      return new Promise((resolve, reject) => {
        deleteUnloadingPlatform(resdata)
          .then(response => {
            localStorage.clear()
            util.cookies.remove('token')
            resolve(response)
          })
          .catch(error => {
            reject(error)
          })
      })
    },
    queryNode ({ commit }, resdata) {
      return new Promise((resolve, reject) => {
        queryNode(resdata)
          .then(response => {
            resolve(response)
          })
          .catch(error => {
            reject(error)
          })
      })
    }
  }
}
export default installProcess
