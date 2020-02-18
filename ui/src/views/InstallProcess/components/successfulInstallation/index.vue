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
            <el-button size="small" type="primary" @click="onhandleDelete">卸载</el-button>
          </el-col>
        </div>
      </el-card>

      <el-card shadow="hover" v-show="componentList">
        <span>
          <rainbond-component :componentList="componentList"></rainbond-component>
        </span>
      </el-card>
    </div>
  </d2-container>
</template>

<script>
import RainbondComponent from "../installResults/rainbondComponent";

export default {
  name: "successfulInstallation",
  components: {
    RainbondComponent
  },
  data() {
    return {
      activeName: "rsultSucess",
      componentList: [],
      loading: true,
      accessAddress: "",
      recordInfo: {
        install_id: "",
        version: "",
        status: "complete",
        eid: ""
      }
    };
  },
  created() {
    this.handleState();
    this.fetchAccessAddress();
    this.fetchClusterInstallResultsState(true);
  },
  beforeDestroy() {
    this.timers && clearInterval(this.timers);
  },
  methods: {
    handleState() {
      this.$store.dispatch("fetchState").then(res => {
        if (res && res.code === 200 && res.data.final_status) {
          if (res.data.clusterInfo) {
            this.recordInfo.install_id = res.data.clusterInfo.installID;
            this.recordInfo.version = res.data.clusterInfo.installVersion;
            this.recordInfo.eid = res.data.clusterInfo.enterpriseID;
          }

          switch (res.data.final_status) {
            case "Initing":
              this.handleRouter("index");
              break;
            case "Waiting":
              this.handleRouter("index");
              break;
            case "Installing":
              this.handleRouter("InstallProcess");
              break;
            case "Setting":
              this.handleRouter("InstallProcess");
              break;
            case "UnInstalling":
              this.handleRouter("index");
              break;
            case "Running":
              this.recordInfo.status = "complete";
              this.handleRecord();
              break;
            default:
              break;
          }
        }
      });
    },
    handleRouter(name) {
      this.$router.push({
        name
      });
    },
    handleRecord() {
      this.$store.dispatch("putRecord", this.recordInfo).then(res => {
        console.log("res", res);
      });
    },
    onhandleDelete() {
      this.$confirm("确定要卸载吗？")
        .then(_ => {
          this.$store.dispatch("deleteUnloadingPlatform").then(res => {
            if (res && res.code === 200) {
              this.recordInfo.status = "uninstall";
              this.handleRecord();
              this.$notify({
                type: "success",
                title: "卸载",
                message: "卸载成功"
              });
              this.handleRouter("index");
            }
          });
        })
        .catch(_ => {
          this.recordInfo.status = "failure";
          this.handleRecord();
        });
    },
    fetchAccessAddress() {
      this.$store.dispatch("fetchAccessAddress").then(res => {
        if (res && res.code === 200) {
          this.accessAddress = res.data;
        }
      });
    },
    fetchClusterInstallResultsState(isloading) {
      this.$store.dispatch("fetchClusterInstallResultsState").then(res => {
        if (isloading) {
          this.loading = false;
        }
        if (res && res.code === 200) {
          this.componentList = res.data;
        }
        this.timers = setTimeout(() => {
          this.fetchClusterInstallResultsState();
        }, 8000);
      });
    }
  }
};
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
    border-bottom: 1px solid #409eff;
  }
}
</style>
