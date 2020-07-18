import {
  getClusterInfo,
  getState,
  getDetectionState,
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
  putrestartpackage
} from '@/api/installProcess'

const installProcess = {
  state: {},
  mutations: {},
  actions: {
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
    restartpackage ({ commit }, resdata) {
      return new Promise((resolve, reject) => {
        putrestartpackage(resdata)
          .then(response => {
            resolve(response)
          })
          .catch(error => {
            reject(error)
          })
      })
    },

    fetchState (_, resdata) {
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
    fetchDetectionState (_, resdata) {
      return new Promise((resolve, reject) => {
        getDetectionState(resdata)
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
