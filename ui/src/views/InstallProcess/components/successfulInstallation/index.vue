<template>
  <d2-container type="full">
    <div class="d2-ml-115 d2-w-1100" v-loading="loading">
      <el-card class="d2-mb">
        <div class="d2-h-30">
          <el-col :span="4">访问地址</el-col>
          <el-col :span="16" class="d2-text-center">
            <a :href="accessAddress" target="_blank" class="successLink">{{accessAddress}}</a>
          </el-col>
          <el-col :span="4" class="d2-text-center">
            <el-button size="small" type="primary" @click="dialogVisibles = true">卸载</el-button>
          </el-col>
        </div>
      </el-card>

      <el-card shadow="hover" v-show="componentList">
        <span>
          <rainbond-component :componentList="componentList"></rainbond-component>
        </span>
      </el-card>
    </div>

    <el-dialog title="你卸载Rainbond的原因" :visible.sync="dialogVisibles" width="30%">
      <el-form
        :inline="true"
        @submit.native.prevent
        :model="uninstallForm"
        ref="uninstallForm"
        size="small"
        label-width="88px"
      >
        <el-row :gutter="20">
          <el-col :span="24" class="table-cell-title">
            <el-form-item class="bor" label prop="checkList">
              <el-checkbox-group v-model="uninstallForm.checkList" style="width:390px">
                <el-checkbox label="安装复杂"></el-checkbox>
                <el-checkbox label="上手困难"></el-checkbox>
                <el-checkbox label="界面不美观"></el-checkbox>
                <el-checkbox label="找不到想要的功能"></el-checkbox>
                <el-checkbox label="没有理由"></el-checkbox>
              </el-checkbox-group>
            </el-form-item>
          </el-col>
          <el-col :span="24" class="table-cell-title">
            <el-form-item label="其他原因" class="d2-mt d2-form-item">
              <el-input type="textarea" v-model="uninstallForm.otherReasons" style="width:290px"></el-input>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <span slot="footer" class="dialog-footer">
        <el-button @click="dialogVisibles = false">取 消</el-button>
        <el-button type="primary" @click="onhandleDelete">卸 载</el-button>
      </span>
    </el-dialog>
  </d2-container>
</template>

<script>
import RainbondComponent from '../installResults/rainbondComponent'

export default {
  name: 'successfulInstallation',
  components: {
    RainbondComponent
  },
  data () {
    return {
      dialogVisibles: false,
      uninstallForm: {
        checkList: [],
        otherReasons: ''
      },
      activeName: 'rsultSucess',
      componentList: [],
      loading: true,
      accessAddress: '',
      recordInfo: {
        install_id: '',
        version: '',
        status: 'complete',
        eid: '',
        message: '',
        testMode: false
      }
    }
  },
  created () {
    this.handleState()
    this.fetchAccessAddress()
    this.fetchClusterInstallResultsState(true)
  },
  beforeDestroy () {
    this.timers && clearInterval(this.timers)
  },
  methods: {
    handleState () {
      this.timers && clearInterval(this.timers)

      this.$store.dispatch('fetchState').then(res => {
        if (res && res.code === 200 && res.data.final_status) {
          const { clusterInfo, final_status, reasons, testMode } = res.data
          if (clusterInfo) {
            this.recordInfo.install_id = clusterInfo.installID
            this.recordInfo.version = clusterInfo.installVersion
            this.recordInfo.eid = clusterInfo.enterpriseID
            this.recordInfo.testMode = testMode
            this.recordInfo.message = ''
          }
          if (reasons) {
            this.recordInfo.status = final_status
            this.recordInfo.message = reasons
            this.handleRecord()
          }

          switch (final_status) {
            case 'Initing':
              this.handleRouter('index')
              break
            case 'Waiting':
              this.handleRouter('index')
              break
            case 'Installing':
              this.handleRouter('InstallProcess')
              break
            case 'Setting':
              this.handleRouter('InstallProcess')
              break
            case 'UnInstalling':
              this.handleRouter('index')
              break
            case 'Running':
              this.recordInfo.status = 'complete'
              this.handleRecord()
              break
            default:
              break
          }
        }
      })
      this.timer = setTimeout(() => {
        this.handleState()
      }, 10000)
    },
    handleRouter (name) {
      this.$router.push({
        name
      })
    },
    handleRecord () {
      if (!this.recordInfo.testMode) {
        this.$store.dispatch('putRecord', this.recordInfo)
      }
    },
    onhandleDelete () {
      this.$store
        .dispatch('deleteUnloadingPlatform')
        .then(res => {
          if (res && res.code === 200) {
            this.dialogVisibles = false
            let message = this.uninstallForm.otherReasons
              ? this.uninstallForm.otherReasons + ','
              : ''
            if (this.uninstallForm.checkList.length > 0) {
              this.uninstallForm.checkList.map(item => {
                message += ',' + item
              })
            }
            this.recordInfo.message = message
            this.recordInfo.status = 'uninstall'
            this.handleRecord()
            this.$notify({
              type: 'success',
              title: '卸载',
              message: '卸载成功'
            })
            this.handleRouter('index')
          }
        })
        .catch(_ => {
          this.dialogVisibles = false
          this.recordInfo.message = ''
          this.recordInfo.status = 'failure'
          this.handleRecord()
        })
    },
    fetchAccessAddress () {
      this.$store.dispatch('fetchAccessAddress').then(res => {
        if (res && res.code === 200) {
          this.accessAddress = res.data
        }
      })
    },
    fetchClusterInstallResultsState (isloading) {
      this.$store.dispatch('fetchClusterInstallResultsState').then(res => {
        if (isloading) {
          this.loading = false
        }
        if (res && res.code === 200) {
          this.componentList = res.data
        }
        this.timers = setTimeout(() => {
          this.fetchClusterInstallResultsState()
        }, 8000)
      })
    }
  }
}
</script>
<style lang="scss" scoped>
.d2-h-30 {
  height: 30px;
  line-height: 30px;
}
.clbr {
  border: none;
}
.d2-ml-115 {
  margin-left: 115px;
}
.el-icon-circle-check {
  color: #67c23a;
  font-size: 22px;
  margin-right: 20px;
}
.d2-w-1100 {
  width: 1100px;
  margin: 0 auto;
}
.successLink {
  text-align: center;
  margin: 0;
  padding: 0;
  color: #409eff;
}
.installSuccess {
  font-size: 22px;
  margin-top: 20px;
  line-height: 22px;
  text-align: center;
  color: #67c23a;
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
  }
}
</style>
