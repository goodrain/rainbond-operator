<template>
  <d2-container type="full">
    <div class="d2-ml-115 d2-w-1100" v-loading="loading || upgradeLoading">
      <el-card class="d2-mb">
        <div class="d2-h-30">
          <el-col :span="5"
            >当前版本&nbsp;&nbsp;
            <span style="color:#409EFF">{{
              upVersionInfo.currentVersion
            }}</span>
          </el-col>
          <el-col :span="14" class="d2-text-center">
            访问地址&nbsp;&nbsp;
            <a :href="accessAddress" target="_blank" class="successLink">
              {{ accessAddress }}
            </a>
          </el-col>
          <el-col :span="5" class="d2-text-center">
            <el-button
              size="small"
              type="primary"
              @click="dialogVisibles = true"
            >
              卸载
            </el-button>
            <el-button
              v-if="upgradeableVersions"
              size="small"
              type="primary"
              @click="upgradeDialog = true"
            >
              升级
            </el-button>
          </el-col>
        </div>
      </el-card>

      <el-card shadow="hover" v-show="componentList">
        <span>
          <rainbond-component
            :componentList="componentList"
          ></rainbond-component>
        </span>
      </el-card>
    </div>

    <el-dialog
      title="你卸载Rainbond的原因"
      :visible.sync="dialogVisibles"
      width="30%"
    >
      <el-form
        :inline="true"
        @submit.native.prevent
        :rules="uninstallRules"
        :model="uninstallForm"
        ref="uninstallForm"
        size="small"
        label-width="88px"
      >
        <el-row :gutter="20">
          <el-col :span="24" class="table-cell-title">
            <el-form-item class="bor" label prop="checkList">
              <el-checkbox-group
                v-model="uninstallForm.checkList"
                style="width:390px"
              >
                <el-checkbox label="未感受到产品价值"></el-checkbox>
                <el-checkbox label="上手困难"></el-checkbox>
                <el-checkbox label="需求不满足"></el-checkbox>
              </el-checkbox-group>
            </el-form-item>
          </el-col>
          <el-col :span="24" class="table-cell-title">
            <el-form-item label="其他原因" class="d2-mt d2-form-item">
              <el-input
                type="textarea"
                v-model="uninstallForm.otherReasons"
                style="width:290px"
              ></el-input>
            </el-form-item>
          </el-col>
          <el-col :span="24" class="table-cell-title cen ">
            <el-popover
              placement="bottom-start"
              title=""
              width="200"
              trigger="hover"
            >
              <img
                :src="`${$baseUrl}image/contact.jpeg`"
                style="width:200px;height:200px"
              />
              <el-link :underline="false" slot="reference" type="primary"
                >如果安装受阻、请联系管理人员、寻求帮助。</el-link
              >
            </el-popover>
          </el-col>
        </el-row>
      </el-form>
      <span slot="footer" class="dialog-footer">
        <el-button @click="dialogVisibles = false">取 消</el-button>
        <el-button type="primary" @click="onhandleDelete">卸 载</el-button>
      </span>
    </el-dialog>

    <el-dialog title="版本升级" :visible.sync="upgradeDialog" width="30%">
      <el-form
        :inline="true"
        @submit.native.prevent
        :model="upVersionInfo"
        :rules="upVersionRules"
        ref="upVersionForm"
        size="small"
        label-width="100px"
      >
        <el-row :gutter="20">
          <el-col :span="24" class="table-cell-title cen ">
            <el-form-item label="当前版本" class="d2-mt d2-form-item">
              <el-input
                disabled
                v-model="upVersionInfo.currentVersion"
                style="width:290px"
              ></el-input>
            </el-form-item>
          </el-col>
          <el-col :span="24" class="table-cell-title">
            <el-form-item class="bor" label="可升级版本" prop="version">
              <el-select
                style="width:290px"
                v-model="upVersionInfo.version"
                :placeholder="$t('page.install.config.upgrade')"
              >
                <el-option
                  v-for="item in upVersionInfo.upgradeableVersions"
                  :key="item"
                  :label="item"
                  :value="item"
                ></el-option>
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <span slot="footer" class="dialog-footer">
        <el-button @click="upgradeDialog = false">取 消</el-button>
        <el-button
          type="primary"
          v-loading="upgradeLoading"
          @click="onhandleUpgrade"
          >升级</el-button
        >
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
    let validateCheckList = (rule, value, callback) => {
      if (
        this.uninstallForm.otherReasons === '' &&
        this.uninstallForm.checkList.length === 0
      ) {
        callback(new Error('请选择卸载原因'))
      } else {
        callback()
      }
    }
    return {
      uninstallRules: {
        checkList: [
          {
            required: true,
            type: 'array',
            trigger: 'change',
            validator: validateCheckList
          }
        ],
        otherReasons: [{ required: false }]
      },
      dialogVisibles: false,
      upgradeDialog: false,
      uninstallForm: {
        checkList: [],
        otherReasons: ''
      },
      upVersionRules: {
        version: [
          {
            required: true,
            message: this.$t('page.install.config.upgrade'),
            trigger: 'blur'
          }
        ]
      },
      upVersionInfo: {
        currentVersion: '',
        version: '',
        upgradeableVersions: []
      },
      activeName: 'rsultSucess',
      componentList: [],
      loading: true,
      upgradeLoading: false,
      upgradeableVersions: false,
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
    this.fetchAccessAddress()
    this.fetchUpVersions()
    this.fetchClusterInstallResultsState(true)
    this.handleState()
  },
  beforeDestroy () {
    this.timers && clearInterval(this.timers)
    this.timer && clearInterval(this.timer)
  },
  methods: {
    handleState () {
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
            this.recordInfo.message = reasons.toString()
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
              this.handleRouter('InstallProcess', 'start')
              break
            case 'Setting':
              this.handleRouter('InstallProcess', 'configuration')
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
    handleRouter (name, type) {
      if (name) {
        let obj = { name }
        if (type) {
          obj.params = { type }
        }
        this.$router.push(obj)
      }
    },
    handleRecord () {
      if (!this.recordInfo.testMode) {
        this.$store.dispatch('putRecord', this.recordInfo).then(() => {})
      }
    },
    onhandleDelete () {
      this.$refs.uninstallForm.validate(valid => {
        if (valid) {
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
        }
      })
    },
    onhandleUpgrade () {
      this.$refs.upVersionForm.validate(valid => {
        if (valid) {
          this.upgradeLoading = true
          this.$store
            .dispatch('upgradeVersion', { version: this.upVersionInfo.version })
            .then(res => {
              if (res && res.code === 200) {
                this.fetchUpVersions()
                this.fetchClusterInstallResultsState()
                this.upgradeLoading = false
                this.upgradeDialog = false
                this.$notify({
                  type: 'success',
                  title: '版本升级',
                  message: '升级开始, 请等待全部组件就绪'
                })
              }
            })
            .catch(err => {
              this.upgradeLoading = false
              let msg = ''
              const { code } = err
              if (code) {
                switch (code) {
                  case 1000:
                    msg = '版本格式有误'
                    break
                  case 1001:
                    msg = '当前版本不存在'
                    break
                  case 1002:
                    msg = '当前版本格式有误'
                    break
                  case 1003:
                    msg = '不支持降级'
                    break
                  case 1004:
                    msg = '版本不存在'
                    break
                  case 1005:
                    msg = '无法获取升级信息'
                    break
                  default:
                    break
                }
              }
              if (msg) {
                this.$message({
                  message: msg,
                  type: 'warning'
                })
              } else {
                this.upgradeDialog = false
              }
            })
        }
      })
    },
    fetchAccessAddress () {
      this.$store.dispatch('fetchAccessAddress').then(res => {
        if (res && res.code === 200) {
          this.recordInfo.message = res.data
          this.accessAddress = res.data
        }
      })
    },
    fetchUpVersions () {
      this.$store.dispatch('fetchUpVersions').then(res => {
        if (res && res.code === 200) {
          this.upVersionInfo = res.data
          const { upgradeableVersions } = res.data
          if (upgradeableVersions && upgradeableVersions.length > 0) {
            this.upgradeableVersions = true
            this.upVersionInfo.version = upgradeableVersions[0]
          } else {
            this.upgradeableVersions = false
          }
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
        }, 5000)
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
