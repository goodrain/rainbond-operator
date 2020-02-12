<template>
  <div v-loading="loading">
    <el-form
      v-if="ruleForm"
      :model="ruleForm"
      :rules="rules"
      @submit.native.prevent
      hide-required-asterisk
      ref="ruleForm"
      label-width="170px"
      class="demo-ruleForm"
    >
      <el-form-item label="镜像仓库">
        <el-collapse
          class="setbr d2-w-868"
          v-model="activeImageHubNamesCollapse"
          accordion
          @change="handleChangeImageHubNames"
        >
          <el-collapse-item class="clcolor" name="2">
            <template slot="title">
              <el-radio-group
                class="d2-ml-35"
                v-model="activeImageHubNames"
                @change="changeImageHubRadio"
              >
                <el-radio class="d2-w-150" :label="'1'">新安装默认镜像仓库</el-radio>
                <el-radio :label="'2'">提供已有的镜像仓库</el-radio>
              </el-radio-group>
            </template>
            <div v-show="activeImageHubNames!='1'">
              <el-form-item label="域名" label-width="85px" class="d2-mt d2-form-item">
                <el-input v-model="ruleForm.imageHub.domain" class="d2-input_inner"></el-input>
              </el-form-item>
              <el-form-item label="空间名称" label-width="85px" class="d2-mt d2-form-item">
                <el-input v-model="ruleForm.imageHub.namespace" class="d2-input_inner"></el-input>
              </el-form-item>
              <el-form-item label="账户" label-width="85px" class="d2-mt d2-form-item">
                <el-input v-model="ruleForm.imageHub.username" class="d2-input_inner"></el-input>
              </el-form-item>
              <el-form-item label="密码" label-width="85px" class="d2-mt d2-form-item">
                <el-input v-model="ruleForm.imageHub.password" class="d2-input_inner"></el-input>
              </el-form-item>
            </div>
          </el-collapse-item>
        </el-collapse>
        <div class="clues">镜像仓库用于应用运行镜像存储和集群公共镜像存储，若你的Kubernetes集群雨安装本地镜像仓库，请提供</div>
      </el-form-item>
      <el-form-item label="数据中心数据库">
        <el-collapse
          class="setbr d2-w-868"
          v-model="activeregionDatabaseNamesCollapse"
          accordion
          @change="handleChangeRegionDatabaseNames"
        >
          <el-collapse-item class="clcolor" name="2">
            <template slot="title">
              <el-radio-group
                class="d2-ml-35"
                v-model="activeregionDatabaseNames"
                @change="changeregionDatabaseRadio"
              >
                <el-radio class="d2-w-150" :label="'1'">新安装数据库</el-radio>
                <el-radio :label="'2'">提供已有的数据仓库</el-radio>
              </el-radio-group>
            </template>
            <div v-show="activeregionDatabaseNames!='1'">
              <el-form-item label="地址" label-width="85px" class="d2-mt d2-form-item" prop="address">
                <el-input
                  v-model="ruleForm.regionDatabase.host"
                  class="d2-input_inner_url d2-w-150"
                ></el-input>
                <span class="d2-w-20">:</span>
                <el-input v-model="ruleForm.regionDatabase.port" class="d2-input_inner_url d2-w-80"></el-input>
              </el-form-item>
              <el-form-item label="账户" label-width="85px" class="d2-mt d2-form-item">
                <el-input v-model="ruleForm.regionDatabase.username" class="d2-input_inner"></el-input>
              </el-form-item>
              <el-form-item label="密码" label-width="85px" class="d2-mt d2-form-item">
                <el-input v-model="ruleForm.regionDatabase.password" class="d2-input_inner"></el-input>
              </el-form-item>
            </div>
          </el-collapse-item>
        </el-collapse>

        <div class="clues">数据中心数据库记录数据中心原数据，若使用其他数据库（比如RDS）请提供可访问信息</div>
      </el-form-item>
      <el-form-item label="UI数据库">
        <el-collapse
          class="setbr d2-w-868"
          v-model="activeUiDatabaseNamesCollapse"
          accordion
          @change="handleChangeUiDatabaseNames"
        >
          <el-collapse-item class="clcolor" name="2">
            <template slot="title">
              <el-radio-group
                class="d2-ml-35"
                v-model="activeUiDatabaseNames"
                @change="changeUiDatabaseRadio"
              >
                <el-radio class="d2-w-150" :label="'1'">新安装UI数据库</el-radio>
                <el-radio :label="'2'">提供已有的UI数据库</el-radio>
              </el-radio-group>
            </template>
            <div v-show="activeUiDatabaseNames!='1'">
              <el-form-item
                label="地址"
                label-width="85px"
                class="d2-mt d2-form-item"
                prop="uiDatabase"
              >
                <el-input v-model="ruleForm.uiDatabase.host" class="d2-input_inner_url d2-w-150"></el-input>
                <span class="d2-w-20">:</span>
                <el-input v-model="ruleForm.uiDatabase.port" class="d2-input_inner_url d2-w-80"></el-input>
              </el-form-item>
              <el-form-item label="账户" label-width="85px" class="d2-mt d2-form-item">
                <el-input v-model="ruleForm.uiDatabase.username" class="d2-input_inner"></el-input>
              </el-form-item>
              <el-form-item label="密码" label-width="85px" class="d2-mt d2-form-item">
                <el-input v-model="ruleForm.uiDatabase.password" class="d2-input_inner"></el-input>
              </el-form-item>
            </div>
          </el-collapse-item>
        </el-collapse>
        <div class="clues">UI数据库记录数据中心原数据，若使用其他数据库（比如RDS请提供可访问信息）</div>
      </el-form-item>
      <el-form-item label="ETCD">
        <el-collapse
          class="setbr d2-w-868"
          v-model="activeETCDNamesCollapse"
          accordion
          @change="handleChangeETCDNames"
        >
          <el-collapse-item class="clcolor" name="2">
            <template slot="title">
              <el-radio-group class="d2-ml-35" v-model="activeETCDNames" @change="changeETCDRadio">
                <el-radio class="d2-w-150" :label="'1'">新安装ETCD</el-radio>
                <el-radio :label="'2'">提供已有的ETCD</el-radio>
              </el-radio-group>
            </template>
            <div v-show="activeETCDNames!='1'" v-if="ruleForm.etcdConfig">
              <el-form-item label="节点列表" label-width="85px" class="d2-mt d2-form-item">
                <div
                  v-for="(item, index) in ruleForm.etcdConfig.endpoints"
                  :key="index"
                  class="cen"
                  :style="{
                    marginTop: index===0?'0':'20px'
                }"
                >
                  <el-input v-model="ruleForm.etcdConfig.endpoints[index]" class="d2-input_inner"></el-input>
                  <i class="el-icon-circle-plus-outline icon-f-22 d2-ml-16" @click="addEndpoints"></i>
                  <i
                    v-show="ruleForm.etcdConfig.endpoints.length!=1"
                    class="el-icon-remove-outline icon-f-22 d2-ml-16"
                    @click.prevent="removeEndpoints(index)"
                  ></i>
                </div>
              </el-form-item>

              <el-form-item label="TLS" label-width="85px" class="d2-mt d2-form-item">
                <el-switch v-model="ruleForm.etcdConfig.useTLS"></el-switch>
              </el-form-item>

              <div v-show="ruleForm.etcdConfig.useTLS">
                <el-form-item label="机构证书" label-width="85px" class="d2-mt d2-form-item">
                  <el-input
                    type="textarea"
                    v-if="ruleForm.etcdConfig.certInfo"
                    v-model="ruleForm.etcdConfig.certInfo.caFile"
                    class="d2-input_inner"
                  ></el-input>
                </el-form-item>
                <el-form-item label="证书" label-width="85px" class="d2-mt d2-form-item">
                  <el-input
                    type="textarea"
                    v-if="ruleForm.etcdConfig.certInfo"
                    v-model="ruleForm.etcdConfig.certInfo.certFile"
                    class="d2-input_inner"
                  ></el-input>
                </el-form-item>
                <el-form-item label="证书私钥" label-width="85px" class="d2-mt d2-form-item">
                  <el-input
                    type="textarea"
                    v-if="ruleForm.etcdConfig.certInfo"
                    v-model="ruleForm.etcdConfig.certInfo.keyFile"
                    class="d2-input_inner"
                  ></el-input>
                </el-form-item>
              </div>
            </div>
          </el-collapse-item>
        </el-collapse>
        <div class="clues">Rainbon各组件依赖ETCD服务，若不提供则默认安装</div>
      </el-form-item>
      <el-form-item
        label="网关安装节点"
        prop="nodes"
        v-if="clusterInfo&&clusterInfo.nodeAvailPorts&&clusterInfo.nodeAvailPorts.length>0"
      >
        <div class="boxs">
          <el-checkbox-group v-model="setgatewayNodes" class="cr_maxcheckbox">
            <el-checkbox
              v-for="(item, index) in clusterInfo.nodeAvailPorts"
              :key="index"
              class="cr_checkbox"
              :label="item.nodeIP"
              border
            ></el-checkbox>
          </el-checkbox-group>
        </div>

        <div class="clues">Rainbond网关服务默认安装到集群所有合适的管理节点，你可以选择配置，网关服务将占用宿主机80/443等端口</div>
      </el-form-item>
      <el-form-item label="分配默认域名">
        <div class="boxs">
          <el-switch v-model="HTTPDomainSwitch"></el-switch>
          <el-input
            style="margin-left:10px"
            v-if="!HTTPDomainSwitch"
            v-model="ruleForm.HTTPDomain"
            placeholder="请输入自定义域名"
            class="d2-input_inner"
          ></el-input>
        </div>
        <div class="clues">默认域名是指Rainbond 为HTTP类应用动态分配的多级域名，默认域名在非离线安装模式下将动态创建公网DNS泛解析记录</div>
      </el-form-item>

      <el-form-item :label="'网关外网IP'" prop="ips">
        <div class="boxs">
          <div v-for="(item, indexs) in ruleForm.gatewayIngressIPs" :key="indexs" class="cen">
            <el-input v-model="ruleForm.gatewayIngressIPs[indexs]" class="d2-input_inner"></el-input>
            <i class="el-icon-circle-plus-outline icon-f-22 d2-ml-16" @click="addIP"></i>
            <i
              v-show="ruleForm.gatewayIngressIPs.length!=1"
              class="el-icon-remove-outline icon-f-22 d2-ml-16"
              @click.prevent="removeIP(indexs)"
            ></i>
          </div>
        </div>

        <div class="clues">默认域名默认解析到所有网关节点的IP地址上，若指定则仅解析到指定</div>
      </el-form-item>
      <el-form-item label="共享存储驱动">
        <el-collapse
          class="setbr d2-w-868"
          v-model="activeStorageNamesCollapse"
          accordion
          @change="handleChangeStorageNames"
        >
          <el-collapse-item class="clcolor" name="2">
            <template slot="title">
              <el-radio-group
                v-model="activeStorageNames"
                class="d2-ml-35"
                @change="changeStorageRadio"
              >
                <el-radio class="d2-w-150" :label="'1'">新部署NFS-Server</el-radio>
                <el-radio :label="'2'">选择已有的共享存储驱动</el-radio>
              </el-radio-group>
            </template>
            <div
              v-show="activeStorageNames!='1'"
              v-if="clusterInfo&&clusterInfo.storage&&clusterInfo.storage.length>0"
            >
              <el-form-item label="存储名称" label-width="85px" class="d2-mt d2-form-item">
                <el-radio-group v-model="ruleForm.storage.name">
                  <el-radio
                    v-for="(item) in clusterInfo.storage"
                    :key="item.name"
                    :label="item.name"
                  ></el-radio>
                </el-radio-group>
              </el-form-item>
            </div>
          </el-collapse-item>
        </el-collapse>
      </el-form-item>

      <!-- <el-form-item label="fsTab信息">
        <el-collapse class="setbr d2-w-868" v-model="activeFstabLineNames">
          <el-collapse-item class="clcolor" name="2">
            <template slot="title">
              <el-radio-group
                v-model="ruleForm.rainbondShareStorage.fstabLine.default"
                class="d2-ml-35"
                @change="changeFstabLineRadio"
              >
                <el-radio :label="true">新部署NFS-Server</el-radio>
                <el-radio :label="shared">选择集群已有共享存储驱动</el-radio>
                <el-radio :label="false">对接外部存储</el-radio>
              </el-radio-group>
      </template>-->
      <!-- <div v-show="ruleForm.rainbondShareStorage.fstabLine.default!=='true'" style="padding:20px">
              <el-form-item
                label="存储名称"
                label-width="85px"
                class="d2-mt d2-form-item"
                v-if="ruleForm.rainbondShareStorage.stabLine.default==='shared'"
              >
                <el-radio-group v-model="ruleForm.rainbondShareStorage.fstabLine.storageClassName">
                  <el-radio
                    v-for="(item) in ruleForm.storage.opts"
                    :key="item.name"
                    :label="item.name"
                  ></el-radio>
                </el-radio-group>
              </el-form-item>
              <div v-else>
                <el-form-item label="存储设备" label-width="100px" class="d2-mt d2-form-item">
                  <el-input v-model="ruleForm.rainbondShareStorage.fstabLine.fileSystem" class="d2-input_inner_url"></el-input>
                  <div class="clues">存储设备说明</div>
                </el-form-item>
                <el-form-item label="挂载位置" label-width="100px" class="d2-mt d2-form-item">
                  <el-input v-model="ruleForm.rainbondShareStorage.fstabLine.mountPoint" class="d2-input_inner_url"></el-input>
                  <div class="clues">挂载位置说明</div>
                </el-form-item>
                <el-form-item label="文件系统类型" label-width="100px" class="d2-mt d2-form-item">
                  <el-input v-model="ruleForm.rainbondShareStorage.fstabLine.type" class="d2-input_inner_url"></el-input>
                  <div class="clues">文件系统类型说明</div>
                </el-form-item>
                <el-form-item label="挂载参数" label-width="100px" class="d2-mt d2-form-item">
                  <el-input v-model="ruleForm.rainbondShareStorage.fstabLine.options" class="d2-input_inner_url"></el-input>
                  <div class="clues">挂载参数说明</div>
                </el-form-item>
                <el-form-item label="dump备份" label-width="100px" class="d2-mt d2-form-item">
                  <el-input v-model="ruleForm.rainbondShareStorage.fstabLine.dump" class="d2-input_inner_url"></el-input>
                  <div class="clues">dump备份说明</div>
                </el-form-item>
                <el-form-item label="检查文件系统" label-width="100px" class="d2-mt d2-form-item">
                  <el-input v-model="ruleForm.rainbondShareStorage.fstabLine.pass" class="d2-input_inner_url"></el-input>
                  <div class="clues">检查文件系统说明</div>
                </el-form-item>
      </div>-->

      <!-- <el-row :gutter="20">
                <el-col :span="6">
                  <el-input v-model="ruleForm.fstabLine.fileSystem" class="d2-input_inner_url"></el-input>
                  <div class="clues">存储设备</div>
                </el-col>
                <el-col :span="4">
                  <el-input
                    disabled
                    v-model="ruleForm.fstabLine.mountPoint"
                    class="d2-input_inner_url"
                  ></el-input>
                  <div class="clues">挂载位置</div>
                </el-col>

                <el-col :span="4">
                  <el-input v-model="ruleForm.fstabLine.type" class="d2-input_inner_url"></el-input>
                  <div class="clues">文件系统类型</div>
                </el-col>

                <el-col :span="4">
                  <el-input v-model="ruleForm.fstabLine.options" class="d2-input_inner_url"></el-input>
                  <div class="clues">挂载参数</div>
                </el-col>

                <el-col :span="3">
                  <el-input  v-model="ruleForm.fstabLine.dump" class="d2-input_inner_url"></el-input>
                  <div class="clues">dump备份</div>
                </el-col>

                <el-col :span="3">
                  <el-input  v-model="ruleForm.fstabLine.pass" class="d2-input_inner_url"></el-input>
                  <div class="clues">检查文件系统</div>
                </el-col>
      </el-row>-->
      <!-- </div>
          </el-collapse-item>
        </el-collapse>
      </el-form-item>-->
      <div style="width:1100px;text-align:center;">
        <el-button type="primary" @click="submitForm('ruleForm')">配置就绪,开始安装</el-button>
      </div>
    </el-form>
  </div>
</template>

<script>
export default {
  name: 'clusterConfiguration',
  props: {
    clusterInfo: {
      type: Object,
      default: () => {}
    }
  },
  data () {
    let validateNodes = (rule, value, callback) => {
      if (this.setgatewayNodes.length === 0) {
        callback(new Error('请至少选择一个网关安装节点'))
      } else {
        callback()
      }
    }

    let validateIPs = (rule, value, callback) => {
      let regIp = /^(\d|[1-9]\d|1\d{2}|2[0-4]\d|25[0-5])\.(\d|[1-9]\d|1\d{2}|2[0-4]\d|25[0-5])\.(\d|[1-9]\d|1\d{2}|2[0-4]\d|25[0-5])\.(\d|[1-9]\d|1\d{2}|2[0-4]\d|25[0-5])$/
      let gatewayIngressIPs = this.ruleForm.gatewayIngressIPs
      let arr = gatewayIngressIPs.filter(item => {
        return !regIp.test(item)
      })

      if (gatewayIngressIPs.length > 0) {
        if (gatewayIngressIPs.length === 1 && gatewayIngressIPs[0] === '') {
          callback()
        } else if (arr.length >= 1) {
          callback(new Error('格式不对，请重新输入'))
        } else {
          callback()
        }
      } else {
        callback()
      }
    }
    let reg = /^([0-9]|[1-9]\d{1,3}|[1-5]\d{4}|6[0-4]\d{4}|65[0-4]\d{2}|655[0-2]\d|6553[0-5])$/

    let validateAddress = (rule, value, callback) => {
      let str = this.ruleForm.regionDatabase.port
      let ress = reg.test(str)

      if (!ress && str !== '') {
        callback(new Error('格式不对，请重新输入'))
      } else {
        callback()
      }
    }
    let validUiDateBase = (rule, value, callback) => {
      let str = this.ruleForm.uiDatabase.port
      let ress = reg.test(str)

      if (!ress && str !== '') {
        callback(new Error('格式不对，请重新输入'))
      } else {
        callback()
      }
    }

    return {
      upLoading: false,
      loading: true,
      HTTPDomainSwitch: true,
      ruleForm: {
        imageHub: {
          domain: '',
          namespace: '',
          username: '',
          password: ''
        },
        regionDatabase: { host: '', port: '', username: '', password: '' },
        uiDatabase: { host: '', port: '', username: '', password: '' },
        etcdConfig: {
          endpoints: [''],
          certInfo: { caFile: '', certFile: '', keyFile: '' }
        },
        HTTPDomain: '',
        gatewayIngressIPs: [''],
        storage: { name: '' },
        rainbondShareStorage: { storageClassName: '', fstabLine: {} }
      },
      activeImageHubNames: '1',
      activeImageHubNamesCollapse: '1',
      activeregionDatabaseNames: '1',
      activeregionDatabaseNamesCollapse: '1',
      activeUiDatabaseNames: '1',
      activeUiDatabaseNamesCollapse: '1',

      activeETCDNames: '1',
      activeETCDNamesCollapse: '1',

      activeStorageNames: '1',
      activeStorageNamesCollapse: '1',

      activeFstabLineNames: '1',
      activeFstabLineNamesCollapse: '1',

      setgatewayNodes: [],
      fileList: [],
      rules: {
        nodes: [
          {
            validator: validateNodes,
            type: 'array',
            required: true,
            trigger: 'change'
          }
        ],
        address: [
          {
            validator: validateAddress,
            type: 'string',
            required: true,
            trigger: 'change'
          }
        ],
        uiDatabase: [
          {
            validator: validUiDateBase,
            type: 'string',
            required: true,
            trigger: 'change'
          }
        ],
        ips: [
          {
            validator: validateIPs,
            type: 'array',
            required: true,
            trigger: 'change'
          }
        ]
      },
      fstabLineType: [
        {
          value: 'nfs',
          label: 'nfs'
        },
        {
          value: 'gfs',
          label: 'gfs'
        },
        {
          value: 'xfs',
          label: 'xfs'
        }
      ],
      fstabLineOptions: [
        {
          value: 'defaults',
          label: 'defaults'
        },
        {
          value: 'auto',
          label: 'auto'
        }
      ]
    }
  },
  created () {
    this.fetchClusterInfo()
  },
  methods: {
    handleChangeImageHubNames () {
      this.activeImageHubNamesCollapse = this.activeImageHubNames
    },
    handleChangeRegionDatabaseNames () {
      this.activeregionDatabaseNamesCollapse = this.activeregionDatabaseNames
    },
    handleChangeUiDatabaseNames () {
      this.activeUiDatabaseNamesCollapse = this.activeUiDatabaseNames
    },
    handleChangeETCDNames () {
      this.activeETCDNamesCollapse = this.activeETCDNames
    },
    handleChangeStorageNames () {
      this.activeStorageNamesCollapse = this.activeStorageNames
    },

    changeImageHubRadio (value) {
      this.activeImageHubNamesCollapse = value
      this.activeImageHubNames = value
    },
    changeregionDatabaseRadio (value) {
      this.activeregionDatabaseNames = value
      this.activeregionDatabaseNamesCollapse = value
    },
    changeUiDatabaseRadio (value) {
      this.activeUiDatabaseNames = value
      this.activeUiDatabaseNamesCollapse = value
    },
    changeETCDRadio (value) {
      this.activeETCDNames = value
      this.activeETCDNamesCollapse = value
    },
    changeStorageRadio (value) {
      this.activeStorageNames = value
      this.activeStorageNamesCollapse = value
    },
    // changeFstabLineRadio (value) {
    //   this.activeFstabLineNames = value
    //   if (value === '1') {
    //     this.ruleForm.rainbondShareStorage.fstabLine.fileSystem = ''
    //     this.ruleForm.rainbondShareStorage.fstabLine.mountPoint = '/grdata'
    //     this.ruleForm.rainbondShareStorage.fstabLine.type = ''
    //     this.ruleForm.rainbondShareStorage.fstabLine.options = ''
    //     this.ruleForm.rainbondShareStorage.fstabLine.dump = 0
    //     this.ruleForm.rainbondShareStorage.fstabLine.pass = 0
    //   }
    // },
    removeIP (index) {
      this.ruleForm.gatewayIngressIPs.splice(index, 1)
    },
    addIP () {
      this.ruleForm.gatewayIngressIPs.push('')
    },
    addEndpoints () {
      this.ruleForm.etcdConfig.endpoints.push('')
    },
    removeEndpoints (index) {
      this.ruleForm.etcdConfig.endpoints.splice(index, 1)
    },
    fetchClusterInfo () {
      this.$store.dispatch('fetchClusterInfo').then(res => {
        if (res && res.data) {
          this.loading = false
          // this.ruleForm = res.data;
          if (res.data.HTTPDomain && res.data.HTTPDomain !== '') {
            this.HTTPDomainSwitch = false
            this.ruleForm.HTTPDomain = res.data.HTTPDomain
          }
          if (
            res.data.gatewayIngressIPs &&
            res.data.gatewayIngressIPs.length > 0
          ) {
            this.ruleForm.gatewayIngressIPs = res.data.gatewayIngressIPs
          }
          let arr = []
          if (this.clusterInfo && this.clusterInfo.nodeAvailPorts.length > 0) {
            this.clusterInfo.nodeAvailPorts.map(item => {
              const { nodeIP } = item
              arr.push(nodeIP)
            })
          } else if (
            res.data.gatewayNodes &&
            res.data.gatewayNodes.length > 0
          ) {
            res.data.gatewayNodes.map(item => {
              const { nodeIP } = item
              arr.push(nodeIP)
            })
          }
          this.setgatewayNodes = arr
        }
      })
    },
    submitForm (formName) {
      this.$refs[formName].validate(valid => {
        if (valid) {
          this.loading = true
          // this.ruleForm.rainbondShareStorage.fstabLine.dump = Number(
          //   this.ruleForm.fstabLine.dump
          // );
          // this.ruleForm.rainbondShareStorage.fstabLine.pass = Number(
          //   this.ruleForm.fstabLine.pass
          // );

          let obj = {}
          if (this.setgatewayNodes.length > 0) {
            let arr = []
            this.setgatewayNodes.map(item => {
              arr.push({ nodeIP: item })
            })
            obj.gatewayNodes = arr
          }
          if (this.activeImageHubNames === '2') {
            obj.imageHub = this.ruleForm.imageHub
          }
          if (this.activeregionDatabaseNames === '2') {
            obj.regionDatabase = this.ruleForm.regionDatabase
            obj.regionDatabase.port = Number(obj.regionDatabase.port)
          }
          if (this.activeUiDatabaseNames === '2') {
            obj.uiDatabase = this.ruleForm.uiDatabase
            obj.uiDatabase.port = Number(obj.uiDatabase.port)
          }

          if (this.activeETCDNames === '2') {
            obj.etcdConfig = this.ruleForm.etcdConfig
          }
          if (this.activeStorageNames === '2') {
            obj.storage = this.ruleForm.storage
          }
          if (
            this.ruleForm.gatewayIngressIPs &&
            this.ruleForm.gatewayIngressIPs.length > 0 &&
            this.ruleForm.gatewayIngressIPs[0] !== ''
          ) {
            obj.gatewayIngressIPs = this.ruleForm.gatewayIngressIPs
          }

          if (!this.HTTPDomainSwitch) {
            obj.HTTPDomain = this.ruleForm.HTTPDomain
          }

          this.$store
            .dispatch('fixClusterInfo', obj)
            .then(res => {
              if (res && res.code === 200) {
                this.addCluster()
              } else {
                this.handleCancelLoading()
              }
            })
            .catch(err => {
              this.handleCancelLoading()
              console.log(err)
            })
        } else {
          this.handleCancelLoading()

          console.log('error submit!!')
          return false
        }
      })
    },
    addCluster () {
      this.$store
        .dispatch('addCluster')
        .then(en => {
          if (en && en.code === 200) {
            this.$emit('onResults')
          } else {
            this.handleCancelLoading()
          }
        })
        .catch(err => {
          this.handleCancelLoading()

          console.log(err)
        })
    },
    handleCancelLoading () {
      this.loading = false
    }
  }
}
</script>

<style rel="stylesheet/scss" lang="scss" scoped>
.boxs {
  border: 1px solid #dcdfe6 !important;
  line-height: 40px;
  min-height: 40px;
  padding-left: 35px;
  width: 870px;
  box-sizing: border-box;
  overflow-y: auto;
}
.d2-w-150 {
  width: 150px;
}
.d2-w-800 {
  width: 800px;
}
.upload-demo {
  text-align: center;
}
.d2-w-20 {
  width: 20px;
  display: inline-block;
  text-align: center;
}
.setbr {
  border: 1px solid #dcdfe6;
}
.d2-w-868 {
  width: 868px !important;
}
.cen {
  display: flex;
  align-items: center;
}
.d2-ml-35 {
  margin-left: 35px;
}

.clues {
  font-family: PingFangSC;
  font-size: 16px;
  color: #cccccc;
}
.icon-f-22 {
  font-size: 22px;
}
.d2-ml-16 {
  margin-left: 16px;
}
.addbr {
  font-size: 21px;
  color: #606266;
  height: 39px;
  line-height: 39px;
  border: 1px solid #dcdfe6;
  display: flex;
  align-items: center;
}
</style>
<style lang="scss" >
.d2-form-item {
  .el-form-item__label {
    line-height: 25px;
  }
  .el-form-item__content {
    line-height: 25px;
  }
}
.d2-input_inner {
  width: 250px;
  .el-input__inner {
    height: 25px;
    line-height: 25px;
  }
}

.d2-input_inner_url {
  .el-input__inner {
    height: 25px;
    line-height: 25px;
  }
}
.d2-w-150 {
  width: 150px;
}
.d2-w-80 {
  width: 80px;
}

.clcolor,
.clcolors {
  .el-collapse-item__header {
    border-color: #dcdfe6 !important;
    height: 39px;
    line-height: 39px;
    width: 868px !important;
  }
  .el-collapse-item__wrap {
    border-bottom: 1px solid #dcdfe6 !important;
  }
}
.clcolors {
  .el-collapse-item__header {
    width: 868px !important;
  }
}
.cr_checkbox {
  margin-left: 10px;
  margin-bottom: 0 !important;
  margin-right: 10px !important;
  padding: 2px 20px 2px 10px !important;
  height: 25px !important;
  .el-checkbox__input {
    display: none !important;
  }
}
.cr_maxcheckbox {
  margin-left: -10px;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  min-height: 40px;
}
</style>
