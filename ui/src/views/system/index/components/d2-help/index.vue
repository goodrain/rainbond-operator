<template>
  <div>
    <el-button type="primary"  @click="handleClick" plain round>
      <!-- <d2-icon name="question-circle-o" class="d2-mr-5"/> -->
      开始安装
    </el-button>
  </div>
</template>

<script>
export default {
  methods: {
    handleClick () {
      this.$router.push({
        name: 'InstallProcess'
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
  transition: opacity .2s;
}
</style>
