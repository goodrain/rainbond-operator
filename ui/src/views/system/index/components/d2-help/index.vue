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
  data() {
    return {
      text: "开始安装",
      loading: false
    };
  },
  created() {
    this.handleState();
  },
  beforeDestroy() {
    this.timers && clearInterval(this.timers);
  },
  methods: {
    handleState() {
      this.$store.dispatch("fetchState").then(res => {
        if (res && res.code === 200 && res.data.final_status) {
          switch (res.data.final_status) {
            case "Initing":
              this.text = "集群初始化中";
              this.loading = true;
              this.handleInit();
              this.timers = setTimeout(() => {
                this.handleState();
              }, 5000);
              break;
            case "Setting":
              this.handleClick();
              break;
            case "Installing":
              this.handleClick();
              break;
            case "UnInstalling":
              this.timers = setTimeout(() => {
                this.handleState();
              }, 5000);
              this.loading = true;
              this.text = "卸载中";
              break;
            default:
              this.text = "开始安装";
              this.loading = false;
              this.timers && clearInterval(this.timers);
              break;
          }
        }
      });
    },
    handleInit() {
      this.$store.dispatch("fetchInit").then(res => {
        if (res && res.code === 200) {
          this.handleClick();
        } else if (res && res.code == 400) {
          this.loading = true;
          this.text = "卸载中";
          this.handleState();
        }
      });
    },
    handleClick() {
      this.$router.push({
        name: "InstallProcess"
      });
    }
  }
};
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
