<template>
  <div class="text item d2-p-20" v-show="componentList">
    <el-row :gutter="12" class="d2-mb">
      <el-col :span="14" class="d2-f-16">组件名称</el-col>
      <el-col :span="5" class="d2-f-16 d2-text-cen">组件副本数</el-col>
      <el-col :span="4" class="d2-f-16 d2-text-cen">已就绪副本数</el-col>
    </el-row>

    <el-collapse accordion>
      <el-collapse-item v-for="item in componentList" :key="item.name" class="componentTitle">
        <template slot="title">
          <el-col :span="15" class="d2-f-14">{{item.name}}</el-col>
          <el-col :span="4" class="d2-f-14 d2-text-cen">{{item.replicas}}</el-col>
          <el-col
            :span="5"
            class="d2-f-14 d2-text-cen"
            :style="{
                      color:item.replicas==item.readyReplicas?'#606266':'#333333'
                      }"
          >{{item.readyReplicas}}</el-col>
        </template>
        <div class="d2-mt">
          <div v-for="items in item.podStatus" :key="items.name">
            <div class="componentBox">
              <el-col :span="4" class="d2-f-14 minComponentColor">名称</el-col>
              <el-col :span="20" class="d2-f-14 descColor">{{items.name}}</el-col>
            </div>
            <div class="componentBox">
              <el-col :span="4" class="d2-f-14 minComponentColor">阶段</el-col>
              <el-col
                :span="20"
                class="d2-f-14 descColor"
                :style="{
                      color:componentColor[items.phase]
                      }"
              >{{items.phase}}</el-col>
            </div>

            <div class="componentBox" v-show="items.reason">
              <el-col :span="4" class="d2-f-14 minComponentColor">原因</el-col>
              <el-col :span="20" class="d2-f-14 descColor">{{items.reason}}</el-col>
            </div>
          </div>
        </div>
      </el-collapse-item>
    </el-collapse>
  </div>
</template>

<script>
export default {
  name: 'installComponent',
  props: {
    componentList: {
      type: Array,
      default: () => []
    }
  },
  data () {
    return {
      componentColor: {
        Pending: 'rgba(0, 0, 0, 0.45)',
        Running: '#52c41a',
        Waiting: 'rgba(0, 0, 0, 0.45)',
        Terminated: '#f5222d'
      }
    }
  },

  watch: {
    componentList (newValue, oldValue) {
      this.componentList = newValue
    }
  }
}
</script>

<style rel="stylesheet/scss" lang="scss" scoped>
.d2-p-20{
  padding: 20px;
}
.descColor {
  color: #606266;
}
.minComponentColor {
  color: #99a9bf !important;
}
.d2-f-16 {
  font-size: 16px;
}
.d2-text-cen {
  text-align: center;
}
.d2-f-14 {
  font-size: 14px;
}
.componentBox {
  min-height: 39px;
  line-height: 39px;
}
</style>

<style lang="scss" >
.componentTitle {
  .el-collapse-item__header {
    border-bottom: 1px solid #ebeef5 !important;
    width: 100% !important;
  }
  .el-collapse-item__header:hover {
    background: #f5f7fa;
  }
}
</style>
