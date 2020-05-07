<template>
  <d2-container type="full">
    <div class="d2-ml-115 d2-w-1100">
      <el-collapse class="clbr" v-model="activeName" accordion>
        <el-collapse-item
          name="cluster"
          class="installationStepTitle"
          :title="$t('page.install.config.title')"
        >
          <cluster-configuration
            :clusterInfo="recordInfo"
            @onResults="handlePerform('startrRsults')"
            @onhandleErrorRecord="handleRecord('failure')"
            @onhandleStartRecord="handleRecord('start')"
            class="d2-mt"
          ></cluster-configuration>
        </el-collapse-item>
        <el-collapse-item
          v-if="resultShow"
          class="installationStepTitle"
          :title="$t('page.install.install.title')"
          name="startrRsults"
        >
          <install-results
            @onhandleErrorRecord="handleRecord('failure')"
            @onhandleUninstallRecord="handleRecord"
          ></install-results>
        </el-collapse-item>
      </el-collapse>
    </div>
  </d2-container>
</template>

<script>
import ClusterConfiguration from './components/clusterConfiguration'
import InstallResults from './components/installResults'

export default {
  name: 'InstallProcess',
  components: {
    ClusterConfiguration,
    InstallResults
  },
  data () {
    return {
      activeName: 'cluster',
      resultShow: false,
      recordInfo: {
        install_id: '',
        version: '',
        status: 'uninstall',
        eid: '',
        message: '',
        testMode: false
      },
      clusterInitInfo: {}
    }
  },
  created () {
    document.documentElement.scrollTop = 0
    this.handleState()
  },
  beforeDestroy () {
    this.timer && clearInterval(this.timer)
  },
  methods: {
    handleState () {
      this.timer && clearInterval(this.timer)
      this.$store.dispatch('fetchState').then(res => {
        if (res && res.code === 200 && res.data.final_status) {
          const { clusterInfo, final_status, reasons, testMode } = res.data
          if (clusterInfo) {
            this.recordInfo.install_id = clusterInfo.installID
            this.recordInfo.version = clusterInfo.installVersion
            this.recordInfo.eid = clusterInfo.enterpriseID
            this.recordInfo.message = ''
            this.recordInfo.testMode = testMode
          }
          if (reasons) {
            this.handleRecord(final_status, reasons.toString())
          }
          switch (final_status) {
            case 'Initing':
              this.handleRouter('index')
              break
            case 'Waiting':
              this.handleRouter('index')
              break
            case 'Installing':
              this.handlePerform('startrRsults')
              break
            case 'Setting':
              this.handlePerform('cluster')
              break
            case 'Running':
              this.handleRouter('successfulInstallation')
              break
            case 'UnInstalling':
              this.handleRouter('index')
              break
            default:
              break
          }
        } else {
          this.handleRouter('index')
        }
      })
      this.timer = setTimeout(() => {
        this.handleState()
      }, 10000)
    },
    handleRecord (states, message) {
      if (!this.recordInfo.testMode) {
        this.recordInfo.status = states
        this.recordInfo.message = message || ''
        this.$store.dispatch('putRecord', this.recordInfo).then(() => {})
      }
    },
    handlePerform (name) {
      this.activeName = name
      this.resultShow = name === 'startrRsults'
    },
    handleRouter (name) {
      this.$router.push({
        name
      })
    }
  }
}
</script>
<style lang="scss" scoped>
.clbr {
  border: none;
}
.d2-ml-115 {
  margin-left: 115px;
}
.el-icon-circle-check {
  color: #67c23a;
  font-size: 50px;
  margin-right: 20px;
}
.d2-w-1100 {
  width: 1100px;
  margin: 0 auto;
}
</style>
<style lang="scss">
.installationStepTitle {
  .el-collapse-item__wrap {
    border: none;
  }
  .el-collapse-item__header {
    font-family: PingFangSC;
    font-size: 21px;
    color: #606266;
    height: 39px;
    line-height: 39px;
  }
}
</style>
