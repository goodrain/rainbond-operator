<template>
  <div>
    <el-button
      type="primary"
      @click="handleInit"
      plain
      round
      v-loading="loading"
      style="background:#409EFF;color:#ffffff"
    >
      <!-- <d2-icon name="question-circle-o" class="d2-mr-5"/> -->
      {{text}}
    </el-button>
  </div>
</template>

<script>
export default {
  data () {
    return {
      text: this.$t('page.overview.install'),
      loading: false,
      recordInfo: {
        install_id: '',
        version: '',
        status: 'uninstall',
        eid: '',
        message: '',
        testMode: false
      }
    }
  },
  created () {
    this.handleState()
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
            this.recordInfo.message = ''
            this.recordInfo.testMode = testMode
          }

          if (reasons) {
            this.recordInfo.message = reasons.toString()
            this.handleRecord()
          }
          switch (final_status) {
            case 'Initing':
              this.text = this.$t('page.overview.init')
              this.loading = true
              break
            case 'Setting':
              this.handleRouter('InstallProcess')
              break
            case 'Installing':
              this.handleRouter('InstallProcess')
              break
            case 'Running':
              this.handleRouter('successfulInstallation')
              break
            case 'UnInstalling':
              this.loading = true
              this.text = this.$t('page.overview.uninstall')
              break
            default:
              this.text = this.$t('page.overview.install')
              this.loading = false
              this.timers && clearInterval(this.timers)
              break
          }
        }
      })
      this.timer = setTimeout(() => {
        this.handleState()
      }, 8000)
    },
    handleInit () {
      this.$store.dispatch('putInit').then(res => {
        if (res && res.code === 200) {
          this.text = this.$t('page.overview.init')
          this.loading = true
          this.handleState()
        } else if (res && res.code === 400) {
          this.loading = true
          this.text = this.$t('page.overview.uninstall')
          this.handleState()
        }
      })
    },
    handleRecord () {
      if (!this.recordInfo.testMode) {
        this.$store.dispatch('putRecord', this.recordInfo)
      }
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
.d2-help--qr-info {
  background-color: #f4f4f5;
  color: #909399;
  width: 100%;
  padding: 8px 16px;
  margin: 0;
  box-sizing: border-box;
  border-radius: 4px;
  position: relative;
  overflow: hidden;
  opacity: 1;
  display: flex;
  align-items: center;
  transition: opacity 0.2s;
}
</style>
<style lang="scss" >
.el-loading-mask {
  background-color: rgba(255, 255, 255, 0.5) !important;
}
</style>
