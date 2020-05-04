<template>
  <el-dialog :visible.sync="dialogVisible" width="30%" :before-close="handleClose">
    <div slot="title">
      安装包下载失败，请下载
      <a
        target="_blank"
        href="https://rainbond-pkg.oss-cn-shanghai.aliyuncs.com/offline/5.2/rainbond.images.2020-04-16-v5.2.0-release.tgz"
        download="filename"
        style="color:#3489ff"
      >安装包</a>
      、成功后将其上传。
    </div>
    <el-upload
      class="upload-demo"
      :action="api"
      :data="uploadObj"
      :on-preview="handlePreview"
      :on-remove="handleRemove"
      :before-remove="beforeRemove"
      :on-success="handleSuccess"
      :on-error="handleAvatarError"
      multiple
      :limit="1"
      :on-exceed="handleExceed"
      :file-list="fileList"
      element-loading-text="正在上传中。。。请稍等"
    >
      <el-button size="small" type="primary">
        上传
        <i class="el-icon-upload el-icon--right"></i>
      </el-button>
    </el-upload>
    <el-progress v-show="showProgress" :percentage="progressLength" :stroke-width="2"></el-progress>
    <span slot="footer" class="dialog-footer" v-show="upLoading">
      <el-button size="small" type="primary" :loading="nextLoading" @click="submitForm()">完成</el-button>
    </span>
  </el-dialog>
</template>

<script>
var baseDomain = process.env.VUE_APP_API
if (baseDomain === '/') {
  baseDomain = window.location.origin
}
export default {
  name: 'clusterConfiguration',
  props: {
    dialogVisible: {
      type: Boolean,
      default: false
    },
    nextLoading: {
      type: Boolean,
      default: false
    }
  },
  data () {
    return {
      progressLength: 0,
      showProgress: false,
      upLoading: false,
      api: `${baseDomain}/uploads`,
      uploadObj: { file_type: 'install_file' },
      loading: true,
      setgatewayNodes: [],
      fileList: []
    }
  },
  created () {},
  methods: {
    submitForm () {
      this.$emit('onhandleClone')
    },
    handleRemove (file, fileList) {
      this.fileList = []
    },
    handlePreview (file) {
      console.log(file)
    },
    handleAvatarError () {
      this.upLoading = false
    },
    handleExceed (files, fileList) {
      this.$message.warning(
        `当前限制选择 1 个文件，本次选择了 ${
          files.length
        } 个文件，共选择了 ${files.length + fileList.length} 个文件`
      )
      this.upLoading = false
    },

    beforeRemove (file, fileList) {
      return this.$confirm(`确定移除 ${file.name}？`)
    },
    handleSuccess (response, file) {
      this.upLoading = true
      this.$emit('onSubmitLoads')
    },
    handleClose (done) {
      this.$confirm('确认关闭？')
        .then(_ => {
          done()
        })
        .catch(_ => {})
    }
  }
}
</script>

<style rel="stylesheet/scss" lang="scss" scoped>
.upload-demo {
  text-align: center;
}
</style>
