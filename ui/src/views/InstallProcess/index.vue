<template>
  <d2-container type="full">
    <div class="d2-ml-115 d2-w-1100">
      <div v-if="activeName === 'configuration'">
        <p class="d2-f-24">{{ $t("page.install.config.title") }}</p>

        <cluster-configuration
          :clusterInfo="recordInfo"
          @onResults="handlePerform('detection')"
          @onhandleErrorRecord="handleRecord"
          @onhandleStartRecord="handleRecord('start')"
          class="d2-mt"
        ></cluster-configuration>
      </div>
      <div v-if="activeName === 'detection'">
        <p class="d2-f-24">{{ $t("page.install.config.detection") }}</p>
        <detection
          @onhandleErrorRecord="handleRecord"
          @onUpstep="handleUpstep"
          @onResults="handlePerform('start')"
        ></detection>
      </div>

      <div v-if="activeName === 'start'">
        <p class="d2-f-24">{{ $t("page.install.install.title") }}</p>
        <install-results
          @onhandleErrorRecord="handleRecord"
          @onhandleUninstallRecord="handleRecord"
        ></install-results>
      </div>
    </div>
  </d2-container>
</template>

<script>
import ClusterConfiguration from './components/clusterConfiguration'
import InstallResults from './components/installResults'
import Detection from './components/detection'

export default {
  name: 'InstallProcess',
  components: {
    ClusterConfiguration,
    InstallResults,
    Detection
  },
  data () {
    return {
      activeName:
        this.$route.params.type && this.$route.params.type !== ':type'
          ? this.$route.params.type
          : 'configuration',
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
              this.handlePerform('start')
              break
            case 'Setting':
              this.handlePerform('configuration')
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
    handleUpstep () {
      this.activeName = 'configuration'
      this.handlePerform('configuration')
    },
    handlePerform (name) {
      if (name !== 'start' && this.activeName === 'detection') {
        return null
      }
      this.activeName = name
      // alert(name)
      this.$router.push({
        name: 'InstallProcess',
        params: { type: name }
      })
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
.d2-f-24 {
  font-size: 24px;
  margin: 0;
}
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
