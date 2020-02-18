<template>
  <d2-container type="full">
    <div class="d2-ml-115 d2-w-1100">
      <el-collapse class="clbr" v-model="activeName" accordion>
        <el-collapse-item name="cluster" class="installationStepTitle" title="集群安装配置">
          <cluster-configuration
            :clusterInfo="clusterInfo"
            @onResults="handlePerform('startrRsults')"
            @onhandleErrorRecord="handleRecord('failure')"
            class="d2-mt"
          ></cluster-configuration>
        </el-collapse-item>
        <el-collapse-item
          v-if="resultShow"
          class="installationStepTitle"
          title="安装"
          name="startrRsults"
        >
          <install-results
            @onhandleErrorRecord="handleRecord('failure')"
            @onhandleUninstallRecord="handleRecord('uninstall')"
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
      clusterInfo: null,
      recordInfo: {
        install_id: '',
        version: '',
        status: 'uninstall',
        eid: ''
      }
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
      this.$store.dispatch('fetchState').then(res => {
        if (res && res.code === 200 && res.data.final_status) {
          if (res.data.clusterInfo) {
            this.recordInfo.install_id = res.data.clusterInfo.installID
            this.recordInfo.version = res.data.clusterInfo.installVersion
            this.recordInfo.eid = res.data.clusterInfo.enterpriseID
          }

          this.clusterInfo = res.data.clusterInfo
          switch (res.data.final_status) {
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
          this.timer = setTimeout(() => {
            this.handleState()
          }, 10000)
        } else {
          this.handleRouter('index')
        }
      })
    },
    handleRecord (states) {
      this.recordInfo.status = states
      this.$store.dispatch('putRecord', this.recordInfo).then(res => {
        console.log('res', res)
      })
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
<style lang="scss" >
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
    border-bottom: 1px solid #409eff;
  }
}
</style>
