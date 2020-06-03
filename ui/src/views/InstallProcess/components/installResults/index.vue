<template>
  <div>
    <div class="result" v-loading="loading">
      <el-row :gutter="12">
        <el-col
          :span="24"
          class="d2-mt"
          v-for="(item, index) in installList"
          :key="item.stepName"
        >
          <el-card
            v-if="
              item.stepName !== 'step_install_component' &&
                item.stepName !== 'step_prepare_hub'
            "
            shadow="hover"
          >
            <install-component
              :item="item"
              :index="index"
              :dialogVisible="dialogVisible"
              @onhandleDialogVisible="dialogVisible = true"
            ></install-component>
          </el-card>

          <el-card v-else class="clearpadding" shadow="hover">
            <div slot="header">
              <install-component
                :item="item"
                :index="index"
                :dialogVisible="dialogVisible"
                @onhandleDialogVisible="dialogVisible = true"
              ></install-component>
            </div>
            <rainbond-component
              v-if="
                item.stepName === 'step_install_component' &&
                  item.status !== 'status_waiting' &&
                  componentList &&
                  componentList.length > 0
              "
              :componentList="componentList"
            ></rainbond-component>
            <rainbond-component
              v-if="
                item.stepName === 'step_prepare_hub' &&
                  item.status !== 'status_waiting' &&
                  mirrorComponentList &&
                  mirrorComponentList.length > 0
              "
              :componentList="mirrorComponentList"
            ></rainbond-component>
          </el-card>
        </el-col>
      </el-row>
    </div>
    <el-col :span="24" class="d2-f-16 d2-text-cen d2-mt">
      <el-button type="primary" @click="dialogVisibles = true">卸载</el-button>
    </el-col>
    <Uploads
      :nextLoading="nextLoading"
      :dialogVisible="dialogVisible"
      @onSubmitLoads="onSubmitLoads"
      @onhandleClone="dialogVisible = false"
    />

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
                <el-checkbox label="安装配置复杂"></el-checkbox>
                <el-checkbox label="安装受阻, 无法进行"></el-checkbox>
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
  </div>
</template>
<script>
import Uploads from '../upload'
import InstallComponent from './installComponent'
import RainbondComponent from './rainbondComponent'

export default {
  name: 'installResults',
  components: {
    Uploads,
    InstallComponent,
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
      nextLoading: false,
      dialogVisible: false,
      dialogVisibles: false,
      dialogVisibleNum: 0,
      installList: [],
      loading: true,
      componentState: {
        Running: '成功',
        Waiting: '等待',
        Terminated: '停止'
      },
      componentList: [],
      mirrorComponentList: [],
      uninstallForm: {
        checkList: [],
        otherReasons: ''
      }
    }
  },
  created () {
    this.loadData()
  },
  beforeDestroy () {
    this.timer && clearInterval(this.timer)
    this.timerdetection && clearInterval(this.timerdetection)
    this.timers && clearInterval(this.timers)
    this.timermirror && clearInterval(this.timermirror)
  },
  methods: {
    fetchErrMessage (err) {
      return err && typeof err === 'object' ? JSON.stringify(err) : ''
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
                    message += item + ','
                  })
                }
                this.$emit('onhandleUninstallRecord', 'uninstall', message)
                this.$notify({
                  type: 'success',
                  title: '卸载',
                  message: '卸载成功'
                })
                this.$router.push({
                  name: 'index'
                })
              }
            })
            .catch(err => {
              this.dialogVisibles = false
              const message = this.fetchErrMessage(err)
              this.$emit('onhandleErrorRecord', 'failure', `${message}`)
            })
        }
      })
    },
    loadData () {
      this.fetchClusterInstallResults()
      this.fetchClusterInstallResultsState()
      this.fetchClusterInstallMirrorWarehouse()
    },
    format (percentage) {
      return ''
    },
    onSubmitLoads () {
      this.installCluster()
    },
    fetchClusterInstallResults () {
      this.timer = setTimeout(() => {
        this.fetchClusterInstallResults()
      }, 10000)
      this.$store
        .dispatch('fetchClusterInstallResults')
        .then(res => {
          if (res) {
            this.loading = false
            this.installList = res.data.statusList
            let arrs = res.data.statusList
            if (arrs && arrs.length > 0) {
              arrs.map(item => {
                const { stepName, status } = item
                if (
                  stepName === 'step_download' &&
                  status === 'status_failed'
                ) {
                  if (this.dialogVisibleNum < 1) {
                    this.dialogVisible = true
                  }
                  this.dialogVisibleNum = 1
                  this.timer && clearInterval(this.timer)
                }
              })
            }
          } else {
            this.loading = false
          }
        })
        .catch(err => {
          const message = this.fetchErrMessage(err)
          this.$emit('onhandleErrorRecord', 'failure', `${message}`)
        })
    },
    fetchClusterInstallResultsState () {
      this.timers = setTimeout(() => {
        this.fetchClusterInstallResultsState()
      }, 10000)
      this.$store
        .dispatch('fetchClusterInstallResultsState')
        .then(res => {
          this.componentList = res.data
        })
        .catch(err => {
          const message = this.fetchErrMessage(err)
          this.$emit('onhandleErrorRecord', 'failure', `${message}`)
        })
    },
    fetchClusterInstallMirrorWarehouse () {
      this.timermirror = setTimeout(() => {
        this.fetchClusterInstallMirrorWarehouse()
      }, 10000)
      this.$store
        .dispatch('fetchClusterInstallResultsState', { isInit: true })
        .then(res => {
          if (res) {
            this.mirrorComponentList = res.data
          }
        })
        .catch(err => {
          const message = this.fetchErrMessage(err)
          this.$emit('onhandleErrorRecord', 'failure', `${message}`)
        })
    }
  }
}
</script>
<style rel="stylesheet/scss" lang="scss">
.clearpadding {
  .el-card__body {
    padding: 0 !important;
  }
}
</style>
<style rel="stylesheet/scss" lang="scss" scoped>
.d2-h-50 {
  height: 50px;
  line-height: 50px;
}

.errorTitleColor {
  color: #303133 !important;
}
.d2-text-cen {
  text-align: center;
}

.d2-f-14 {
  font-size: 14px;
}

.result {
  width: 1000px;
  min-height: 300px;
  margin: 0 auto;
  .d2-animation {
    animation: rotating 1s linear infinite;
  }
  .success {
    color: #52c41a;
  }
  .loadings {
    color: #303133;
  }
  .error {
    color: #f5222d;
  }
  .d2-f-20 {
    font-size: 20px;
  }
  .d2-f-16 {
    font-size: 16px;
  }
  .icon {
    font-size: 72px;
    line-height: 72px;
    margin-bottom: 24px;
  }

  .title {
    font-size: 24px;
    color: rgba(0, 0, 0, 0.85);
    font-weight: 500;
    line-height: 32px;
    margin-bottom: 16px;
  }

  .description {
    font-size: 14px;
    line-height: 22px;
    color: rgba(0, 0, 0, 0.45);
    margin: 5px 0;
  }

  .extra {
    background: #fafafa;
    padding: 24px 40px;
    border-radius: rgba(0, 0, 0, 0.45);
    text-align: left;
  }
}
</style>
