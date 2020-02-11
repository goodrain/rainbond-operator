<template>
  <div>
    <div class="result" v-loading="loading">
      <el-row :gutter="12">
        <el-col :span="24" class="d2-mt" v-for="(item,index) in installList" :key="item.stepName">
          <el-card
            v-if="item.stepName!=='step_install_component'&&item.stepName!=='step_prepare_hub'"
            shadow="hover"
          >
            <install-component
              :item="item"
              :index="index"
              :dialogVisible="dialogVisible"
              @onhandleDialogVisible="dialogVisible=true"
            ></install-component>
          </el-card>

          <el-card v-else class="box-card" shadow="hover">
            <div slot="header" class="clearfix">
              <install-component
                :item="item"
                :index="index"
                :dialogVisible="dialogVisible"
                @onhandleDialogVisible="dialogVisible=true"
              ></install-component>
            </div>

            <rainbond-component
              v-if="item.stepName==='step_install_component'&&item.status!=='status_waiting'"
              :componentList="componentList"
            ></rainbond-component>
            <rainbond-component v-else :componentList="mirrorComponentList"></rainbond-component>
          </el-card>
        </el-col>
        <el-col :span="24" class="d2-f-16 d2-text-cen d2-mt">
          <el-button type="primary" @click="onhandleDelete">卸载</el-button>
        </el-col>
      </el-row>
    </div>
    <Uploads
      :nextLoading="nextLoading"
      :dialogVisible="dialogVisible"
      @onSubmitLoads="onSubmitLoads"
      @onhandleClone="dialogVisible=false"
    />
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
    return {
      nextLoading: false,
      dialogVisible: false,
      installList: [],
      loading: true,
      componentState: {
        Running: '成功',
        Waiting: '等待',
        Terminated: '停止'
      },
      componentList: [],
      mirrorComponentList: []
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
    onhandleDelete () {
      this.$confirm('确定要卸载吗？')
        .then(_ => {
          this.$store.dispatch('deleteUnloadingPlatform').then(res => {
            if (res && res.code === 200) {
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
        })
        .catch(_ => {})
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
      this.addCluster()
    },
    fetchClusterInstallResults () {
      this.timer = setTimeout(() => {
        this.fetchClusterInstallResults()
      }, 10000)
      this.$store.dispatch('fetchClusterInstallResults').then(res => {
        if (res) {
          this.loading = false
          this.installList = res.data.statusList
          let arrs = res.data.statusList
          if (arrs && arrs.length > 0) {
            arrs.map(item => {
              const { stepName, status } = item
              if (stepName === 'step_download' && status === 'status_failed') {
                this.dialogVisible = true
                this.timer && clearInterval(this.timer)
              }
            })
          }
        } else {
          this.loading = false
        }
      })
    },
    fetchClusterInstallResultsState () {
      this.timers = setTimeout(() => {
        this.fetchClusterInstallResultsState()
      }, 10000)
      this.$store.dispatch('fetchClusterInstallResultsState').then(res => {
        this.componentList = res.data
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
    }
  }
}
</script>
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
