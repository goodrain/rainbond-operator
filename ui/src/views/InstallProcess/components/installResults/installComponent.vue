<template>
  <el-row :gutter="12">
    <el-col :span="1" class="d2-f-16">
      <div class="el-step__icon is-text">
        <div class="el-step__icon-inner">{{index+1}}</div>
      </div>
    </el-col>
    <el-col :span="16" class="d2-f-16">{{phaseMap[item.stepName]}}</el-col>
    <el-col :span="6" class="d2-f-12">{{phaseDesc[item.stepName]}}</el-col>

    <el-col :span="1">
      <i v-if="item.status==='status_finished'" class="el-icon-circle-check success d2-f-20"></i>
      <i v-else-if="item.status==='status_failed'" class="el-icon-circle-close error d2-f-20"></i>
      <i v-else-if="item.status==='status_waiting'" class="el-icon-refresh-right loadings d2-f-20"></i>
      <i v-else class="el-icon-refresh d2-animation loadings d2-f-20"></i>
    </el-col>
    <el-progress :percentage="item.progress" class="d2-h-50 d2-mb"></el-progress>
    <div v-show="(item.reason ||item.message )">
      <el-col :span="24" class="description cen errorTitleColor">
        <el-button
          v-show="item.stepName==='step_download'&&item.status==='status_failed'"
          size="small"
          type="primary"
          @click="submit"
        >
          重新上传
          <i class="el-icon-upload el-icon--right"></i>
        </el-button>
      </el-col>
      <el-col v-show="item.reason" :span="2" class="description errorTitleColor">原因:</el-col>
      <el-col v-show="item.reason" :span="22" class="description">{{item.reason}}</el-col>
      <el-col v-show="item.message" :span="2" class="description errorTitleColor">消息:</el-col>
      <el-col v-show="item.message" :span="22" class="description">{{item.message}}</el-col>
    </div>
  </el-row>
</template>

<script>
export default {
  name: 'installComponent',
  props: {
    item: {
      type: Object,
      default: () => {}
    },
    index: {
      type: Number,
      default: 0
    },
    dialogVisible: {
      type: Boolean,
      default: () => false
    }
  },
  data () {
    return {
      phaseMap: {
        step_setting: '配置环境',
        step_prepare_hub: '安装镜像仓库',
        step_download: '下载安装包',
        step_unpacke: '解压安装包',
        step_handle_image: '处理镜像',
        step_install_component: '安装Rainbond组件'
      },
      phaseDesc: {
        step_setting: '准备环境、预计 30秒',
        step_prepare_hub: '准备存储、镜像仓库、预计 8 分钟',
        step_download: '下载所需的安装包、预计 3 分钟',
        step_unpacke: '解压基础镜像、预计 2 分钟',
        step_handle_image: '推送镜像到镜像仓库、预计 7 分钟',
        step_install_component: '安装所需的组件、预计 10 分钟'
      }
    }
  },
  methods: {
    submit () {
      this.$emit('onhandleDialogVisible')
    }
  }
}
</script>

<style rel="stylesheet/scss" lang="scss" scoped>
.cen {
  text-align: center;
}
.errorTitleColor {
  color: #303133 !important;
}
.d2-animation {
  animation: rotating 1s linear infinite;
}
.d2-f-12 {
  font-size: 14px;
  color: rgba(0, 0, 0, 0.45);
}
.description {
  font-size: 14px;
  line-height: 22px;
  color: rgba(0, 0, 0, 0.45);
  margin: 5px 0;
}
.d2-f-16 {
  font-size: 16px;
}
.d2-h-50 {
  height: 50px;
  line-height: 50px;
}
.d2-f-20 {
  font-size: 20px;
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
</style>
