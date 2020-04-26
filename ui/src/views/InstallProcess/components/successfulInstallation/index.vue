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
      <el-alert
        v-if="username"
        class="d2-mb"
        title="下次访问进行登录校验、请记录保存该用户名、密码、只能显示一次，所以需谨慎，账户密码生成之后不会再生成，也不会在展示"
        type="warning"
        :closable="false"
      ></el-alert>
      <el-card class="d2-mb" v-if="username">
        <div class="d2-h-30">
          <el-col :span="10">账户 : {{username}}</el-col>
          <el-col :span="10">密码 : {{password}}</el-col>
          <el-col :span="4">
            <el-button size="small" type="primary" @click.native.prevent="handleRouter('successfulLogin')">登录</el-button>
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
import util from '@/libs/util'
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
      username: '',
      password: '',
      accessAddress: '',
      recordInfo: {
        install_id: '',
        version: '',
        status: 'complete',
        eid: '',
        message: ''
      }
    }
  },
  created () {
    this.handleState()
    this.handleIsAdmin()
    this.fetchAccessAddress()
    this.fetchClusterInstallResultsState(true)
  },
  beforeDestroy () {
    this.timers && clearInterval(this.timers)
  },
  methods: {
    handleIsAdmin () {
      const token = util.cookies.get('token')
      this.$store.dispatch('fetchIsAdmin').then(res => {
        if (res && res.code === 200 && res.data && res.data.answer) {
          if (token) {
            return null
          }
          this.handleRouter('successfulLogin')
        } else {
          this.handleGenerateAdmin()
        }
      })
    },
    handleGenerateAdmin () {
      this.$store.dispatch('fetchGenerateAdmin').then(res => {
        if (res && res.code === 200 && res.data) {
          this.username = res.data.username
          this.password = res.data.password
        }
      })
    },
    handleState () {
      this.$store
        .dispatch('fetchState')
        .then(res => {
          if (res && res.code === 200 && res.data.final_status) {
            if (res.data.clusterInfo) {
              this.recordInfo.install_id = res.data.clusterInfo.installID
              this.recordInfo.version = res.data.clusterInfo.installVersion
              this.recordInfo.eid = res.data.clusterInfo.enterpriseID
              this.recordInfo.message = ''
            }

            switch (res.data.final_status) {
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
        .catch(err => {
          if (err && (err === 50004 || err === 50005 || err === 50006)) {
            this.handleRouter('successfulLogin')
          }
        })
    },
    handleRouter (name) {
      this.$router.push({
        name
      })
    },
    handleRecord () {
      this.$store.dispatch('putRecord', this.recordInfo).then(res => {
        console.log('res', res)
      })
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
