<template>
  <div v-loading="loading">
    <el-form
      :model="ruleForm"
      :rules="rules"
      @submit.native.prevent
      ref="ruleForm"
      label-width="140px"
      label-position="left"
      class="demo-ruleForm"
    >
      <!-- install mode -->
      <el-form-item
        :label="$t('page.install.config.installmode')"
        prop="enableHA"
      >
        <el-radio-group class="d2-ml-35" v-model="ruleForm.enableHA">
          <el-radio class="d2-w-150" :label="false">{{
            $t("page.install.config.minimize")
          }}</el-radio>
          <el-radio :label="true">{{ $t("page.install.config.ha") }}</el-radio>
        </el-radio-group>
        <div class="clues">{{ $t("page.install.config.installmodeDesc") }}</div>
      </el-form-item>
      <!-- hub config -->
      <el-form-item
        :label="$t('page.install.config.hub')"
        prop="imageHubInstall"
      >
        <el-radio-group class="d2-ml-35" v-model="ruleForm.imageHubInstall">
          <el-radio class="d2-w-150" :label="true">{{
            $t("page.install.config.hubInstall")
          }}</el-radio>
          <el-radio :label="false">{{
            $t("page.install.config.hubProvide")
          }}</el-radio>
        </el-radio-group>
        <div v-if="!ruleForm.imageHubInstall" class="boxs">
          <el-form-item
            prop="imageHubDomain"
            :label="$t('page.install.config.hubDomain')"
            label-width="85px"
            class="d2-mt d2-form-item"
          >
            <el-input
              v-model="ruleForm.imageHubDomain"
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
          <el-form-item
            prop="hubnamespace"
            :label="$t('page.install.config.hubNamespace')"
            label-width="85px"
            class="d2-mt d2-form-item"
          >
            <el-input
              v-model="ruleForm.imageHubNamespace"
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
          <el-form-item
            prop="hubuser"
            :label="$t('page.install.config.hubUser')"
            label-width="85px"
            class="d2-mt d2-form-item"
          >
            <el-input
              v-model="ruleForm.imageHubUsername"
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
          <el-form-item
            prop="hubpassword"
            :label="$t('page.install.config.hubPassword')"
            label-width="85px"
            class="d2-mt d2-form-item"
          >
            <el-input
              v-model="ruleForm.imageHubPassword"
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
        </div>
        <div class="clues">{{ $t("page.install.config.hubDesc") }}</div>
      </el-form-item>
      <!-- region db config -->
      <el-form-item
        :label="$t('page.install.config.regionDB')"
        prop="installRegionDB"
      >
        <el-radio-group class="d2-ml-35" v-model="ruleForm.installRegionDB">
          <el-radio class="d2-w-150" :label="true">{{
            $t("page.install.config.regionDBInstall")
          }}</el-radio>
          <el-radio :label="false">{{
            $t("page.install.config.regionDBProvide")
          }}</el-radio>
        </el-radio-group>
        <div v-if="!ruleForm.installRegionDB" class="boxs">
          <span class="desc">{{
            $t("page.install.config.regionDBProviderDesc")
          }}</span>
          <el-form-item
            :label="$t('page.install.config.regionDBAddress')"
            label-width="85px"
            class="d2-mt d2-form-item"
            prop="regionDatabaseHost"
          >
            <el-input
              v-model="ruleForm.regionDatabaseHost"
              class="d2-input_inner_url"
              style="width:200px"
            ></el-input>
            <span class="d2-w-20">:</span>
            <el-input
              v-model="ruleForm.regionDatabasePort"
              class="d2-input_inner_url"
              style="width:80px"
              type="number"
            ></el-input>
          </el-form-item>
          <el-form-item
            :label="$t('page.install.config.regionDBUser')"
            label-width="85px"
            class="d2-mt d2-form-item"
            prop="regionDatabaseUsername"
          >
            <el-input
              v-model="ruleForm.regionDatabaseUsername"
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
          <el-form-item
            :label="$t('page.install.config.regionDBPassword')"
            label-width="85px"
            class="d2-mt d2-form-item"
            prop="regionDatabasePassword"
          >
            <el-input
              v-model="ruleForm.regionDatabasePassword"
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
        </div>
        <div class="clues">
          {{ $t("page.install.config.regionDBDesc") }}
        </div>
      </el-form-item>
      <!-- ui db config -->
      <el-form-item :label="$t('page.install.config.uiDB')" prop="installUIDB">
        <el-radio-group class="d2-ml-35" v-model="ruleForm.installUIDB">
          <el-radio class="d2-w-150" :label="true">{{
            $t("page.install.config.uiDBInstall")
          }}</el-radio>
          <el-radio :label="false">{{
            $t("page.install.config.uiDBProvide")
          }}</el-radio>
        </el-radio-group>
        <div v-if="!ruleForm.installUIDB" class="boxs">
          <span class="desc">{{
            $t("page.install.config.uiDBProviderDesc")
          }}</span>
          <el-form-item
            :label="$t('page.install.config.uiDBAddress')"
            label-width="85px"
            class="d2-mt d2-form-item"
            prop="uiDatabaseHost"
          >
            <el-input
              v-model="ruleForm.uiDatabaseHost"
              class="d2-input_inner_url"
              style="width:200px"
            ></el-input>
            <span class="d2-w-20">:</span>
            <el-input
              v-model="ruleForm.uiDatabasePort"
              class="d2-input_inner_url"
              style="width:80px"
              type="number"
            ></el-input>
          </el-form-item>
          <el-form-item
            :label="$t('page.install.config.uiDBUser')"
            label-width="85px"
            class="d2-mt d2-form-item"
            prop="uiDatabaseUsername"
          >
            <el-input
              v-model="ruleForm.uiDatabaseUsername"
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
          <el-form-item
            :label="$t('page.install.config.uiDBPassword')"
            label-width="85px"
            class="d2-mt d2-form-item"
            prop="uiDatabasePassword"
          >
            <el-input
              v-model="ruleForm.uiDatabasePassword"
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
        </div>
        <div class="clues">{{ $t("page.install.config.uiDBDesc") }}</div>
      </el-form-item>
      <!-- etcd config -->
      <el-form-item :label="$t('page.install.config.etcd')" prop="installETCD">
        <el-radio-group class="d2-ml-35" v-model="ruleForm.installETCD">
          <el-radio class="d2-w-150" :label="true">{{
            $t("page.install.config.etcdInstall")
          }}</el-radio>
          <el-radio :label="false">{{
            $t("page.install.config.etcdProvide")
          }}</el-radio>
        </el-radio-group>
        <div v-if="!ruleForm.installETCD" class="boxs">
          <el-form-item
            :label="$t('page.install.config.etcdEndpoint')"
            label-width="120px"
            class="d2-mt d2-form-item"
          >
            <div
              v-for="(item, index) in ruleForm.etcdConfig.endpoints"
              :key="index"
              class="cen"
              :prop="'endpoints.' + index + '.value'"
              :style="{
                marginTop: index === 0 ? '0' : '20px'
              }"
            >
              <el-input
                v-model="ruleForm.etcdConfig.endpoints[index]"
                class="d2-input_inner"
              ></el-input>
              <i
                v-if="ruleForm.etcdConfig.endpoints.length > 1"
                style="margin-left:1rem;font-size:16px"
                class="el-icon-remove-outline"
                @click.prevent="removeEndpoints(index)"
              />
            </div>
          </el-form-item>
          <el-button
            style="margin:1rem 0 0 120px"
            size="small"
            @click="addEndpoints"
            >{{ $t("page.install.config.etcdEndpointAdd") }}</el-button
          >
          <el-form-item
            :label="$t('page.install.config.etcdTLS')"
            label-width="120px"
            class="d2-mt d2-form-item"
          >
            <el-switch v-model="ruleForm.etcdConfig.useTLS"></el-switch>
          </el-form-item>

          <div v-show="ruleForm.etcdConfig.useTLS">
            <el-form-item
              :label="$t('page.install.config.etcdCA')"
              label-width="120px"
              class="d2-mt d2-form-item"
            >
              <el-input
                type="textarea"
                v-if="ruleForm.etcdConfig.certInfo"
                v-model="ruleForm.etcdConfig.certInfo.caFile"
                class="d2-input_inner"
              ></el-input>
            </el-form-item>
            <el-form-item
              :label="$t('page.install.config.etcdCert')"
              label-width="120px"
              class="d2-mt d2-form-item"
            >
              <el-input
                type="textarea"
                v-if="ruleForm.etcdConfig.certInfo"
                v-model="ruleForm.etcdConfig.certInfo.certFile"
                class="d2-input_inner"
              ></el-input>
            </el-form-item>
            <el-form-item
              :label="$t('page.install.config.etcdCertKey')"
              label-width="120px"
              class="d2-mt d2-form-item"
            >
              <el-input
                type="textarea"
                v-if="ruleForm.etcdConfig.certInfo"
                v-model="ruleForm.etcdConfig.certInfo.keyFile"
                class="d2-input_inner"
              ></el-input>
            </el-form-item>
          </div>
        </div>
        <div class="clues">{{ $t("page.install.config.etcdDesc") }}</div>
      </el-form-item>
      <!-- gateway node config -->
      <el-form-item :label="$t('page.install.config.gatewayNode')" prop="nodes">
        <el-select
          v-model="setgatewayNodes"
          multiple
          filterable
          style="width:868px"
          remote
          :loading="queryGatewayNodeloading"
          :placeholder="$t('page.install.config.ipsearch')"
          :remote-method="queryGatewayNode"
          default-first-option
        >
          <el-option
            v-for="item in optionGatewayNodes"
            :key="item"
            :label="item"
            :value="item"
          >
          </el-option>
        </el-select>

        <div class="clues">
          {{ $t("page.install.config.gatewayNodeDesc") }}
        </div>
      </el-form-item>
      <!-- chaos node config -->
      <el-form-item
        :label="$t('page.install.config.chaosNode')"
        prop="chaosNodes"
      >
        <el-select
          v-model="setChaosNodes"
          multiple
          filterable
          style="width:868px"
          :placeholder="$t('page.install.config.ipsearch')"
          remote
          :loading="queryChaosNodeloading"
          :remote-method="queryChaosNode"
          default-first-option
        >
          <el-option
            v-for="item in optionChaosNodes"
            :key="item"
            :label="item"
            :value="item"
          >
          </el-option>
        </el-select>
        <div class="clues">
          {{ $t("page.install.config.chaosNodeDesc") }}
        </div>
      </el-form-item>
      <!-- default app domain config -->
      <el-form-item
        :label="$t('page.install.config.appDefaultDomain')"
        prop="HTTPDomain"
      >
        <el-switch v-model="ruleForm.HTTPDomainSwitch"></el-switch>
        <el-input
          style="margin-left:10px"
          v-if="!ruleForm.HTTPDomainSwitch"
          v-model="ruleForm.HTTPDomain"
          :placeholder="$t('page.install.config.appDefaultDomainPlaceholder')"
          class="d2-input_inner"
        ></el-input>
        <div class="clues">
          {{ $t("page.install.config.appDefaultDomainDesc") }}
        </div>
      </el-form-item>
      <!-- eip config -->
      <el-form-item :label="$t('page.install.config.gatewayIP')" prop="ips">
        <div class="boxs">
          <div
            v-for="(item, indexs) in ruleForm.gatewayIngressIPs"
            :key="indexs"
            class="cen"
          >
            <el-input
              v-model="ruleForm.gatewayIngressIPs[indexs]"
              class="d2-input_inner"
            ></el-input>
            <i
              v-show="ruleForm.gatewayIngressIPs.length != 1"
              class="el-icon-remove-outline icon-f-22 d2-ml-16"
              @click.prevent="removeIP(indexs)"
            ></i>
          </div>
          <el-button style="margin-top:1rem" size="small" @click="addIP">{{
            $t("page.install.config.gatewayIPAdd")
          }}</el-button>
        </div>

        <div class="clues">
          {{ $t("page.install.config.gatewayIPDesc") }}
        </div>
      </el-form-item>
      <!-- share storage config -->
      <el-form-item
        :label="$t('page.install.config.shareStorage')"
        prop="activeStorageType"
      >
        <el-radio-group v-model="ruleForm.activeStorageType" class="d2-ml-35">
          <el-radio v-if="!ruleForm.enableHA" class="d2-w-150" :label="1">
            {{ $t("page.install.config.newNFSServer") }}
          </el-radio>
          <el-radio :label="2">
            {{ $t("page.install.config.selectStorage") }}
          </el-radio>
          <el-radio :label="3">
            {{ $t("page.install.config.useAliNas") }}
          </el-radio>
        </el-radio-group>
        <!-- storage class -->
        <div v-show="ruleForm.activeStorageType == 2" class="boxs">
          <span
            v-if="
              !clusterInitInfo.storageClasses ||
                clusterInitInfo.storageClasses.length === 0
            "
            class="desc"
            >{{ $t("page.install.config.noStorageClass") }}</span
          >
          <el-form-item
            label="StorageClass"
            label-width="85px"
            class="d2-mt d2-form-item"
            v-if="clusterInitInfo.storageClasses"
          >
            <el-radio-group v-model="ruleForm.shareStorageClassName">
              <el-radio
                v-for="item in clusterInitInfo.storageClasses"
                :key="item.name"
                :label="item.name"
              ></el-radio>
            </el-radio-group>
          </el-form-item>
        </div>
        <!-- nas -->
        <div v-show="ruleForm.activeStorageType == 3" class="boxs">
          <span class="desc">{{ $t("page.install.config.nasDesc") }}</span>
          <el-form-item
            label="AccessKeyID"
            label-width="130px"
            class="d2-mt d2-form-item"
          >
            <el-input
              :placeholder="$t('page.install.config.accessKeyID')"
              v-model="storage.RWX.csiPlugin.aliyunNas.accessKeyID"
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
          <el-form-item
            label="AccessKeySecret"
            label-width="130px"
            class="d2-mt d2-form-item"
          >
            <el-input
              :placeholder="$t('page.install.config.zoneId')"
              v-model="storage.RWX.csiPlugin.aliyunNas.zoneId"
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
          <el-form-item
            label="AccessKeyID"
            label-width="130px"
            class="d2-mt d2-form-item"
          >
            <el-input
              :placeholder="$t('page.install.config.vpcId')"
              v-model="storage.RWX.csiPlugin.aliyunNas.vpcId"
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
          <el-form-item
            :label="$t('page.install.config.vSwitchId')"
            label-width="130px"
            class="d2-mt d2-form-item"
          >
            <el-input
              :placeholder="$t('page.install.config.vSwitchId')"
              v-model="storage.RWX.csiPlugin.aliyunNas.vSwitchId"
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
        </div>
        <div class="clues">
          {{ $t("page.install.config.shareStorageDesc") }}
        </div>
      </el-form-item>
      <!-- block storage config -->
      <el-form-item
        :label="$t('page.install.config.blockStorage')"
        prop="activeBlockStorageType"
      >
        <el-radio-group
          v-model="ruleForm.activeBlockStorageType"
          class="d2-ml-35"
        >
          <el-radio :label="0">
            {{ $t("page.install.config.noStorage") }}
          </el-radio>
          <el-radio :label="1">
            {{ $t("page.install.config.selectStorage") }}
          </el-radio>
          <el-radio :label="2">{{
            $t("page.install.config.useAliDisk")
          }}</el-radio>
          <el-radio :label="3">{{ $t("page.install.config.useRBD") }}</el-radio>
        </el-radio-group>
        <!-- storage class -->
        <div v-show="ruleForm.activeBlockStorageType == 1" class="boxs">
          <span
            v-if="
              !clusterInitInfo.storageClasses ||
                clusterInitInfo.storageClasses.length === 0
            "
            class="desc"
            >{{ $t("page.install.config.noStorageClass") }}</span
          >
          <el-form-item
            label="StorageClass"
            label-width="85px"
            v-if="clusterInitInfo.storageClasses"
            class="d2-mt d2-form-item"
          >
            <el-radio-group v-model="ruleForm.blockStorageClassName">
              <el-radio
                v-for="item in clusterInitInfo.storageClasses"
                :key="item.name"
                :label="item.name"
              ></el-radio>
            </el-radio-group>
          </el-form-item>
        </div>
        <!-- ali disk -->
        <div v-if="ruleForm.activeBlockStorageType == 2" class="boxs">
          <span class="desc">{{ $t("page.install.config.aliDiskDesc") }}</span>
          <el-form-item
            label="AccessKeyID"
            label-width="130px"
            class="d2-mt d2-form-item"
          >
            <el-input
              :placeholder="$t('page.install.config.accessKeyID')"
              v-model="storage.RWO.csiPlugin.aliyunCloudDisk.accessKeyID"
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
          <el-form-item
            label="AccessKeySecret"
            label-width="130px"
            class="d2-mt d2-form-item"
          >
            <el-input
              :placeholder="$t('page.install.config.accessKeySecret')"
              v-model="storage.RWO.csiPlugin.aliyunCloudDisk.accessKeySecret"
              class="d2-input_inner"
            ></el-input>
          </el-form-item>

          <el-form-item
            label="ZoneID"
            label-width="130px"
            class="d2-mt d2-form-item"
          >
            <el-input
              :placeholder="$t('page.install.config.zoneId')"
              v-model="storage.RWO.csiPlugin.aliyunCloudDisk.zoneId"
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
          <el-form-item
            label="VPC ID"
            label-width="130px"
            class="d2-mt d2-form-item"
          >
            <el-input
              :placeholder="$t('page.install.config.vpcId')"
              v-model="storage.RWO.csiPlugin.aliyunCloudDisk.vpcId"
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
          <el-form-item
            :label="$t('page.install.config.vSwitchId')"
            label-width="130px"
            class="d2-mt d2-form-item"
          >
            <el-input
              :placeholder="$t('page.install.config.vSwitchId')"
              v-model="storage.RWO.csiPlugin.aliyunCloudDisk.vSwitchId"
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
        </div>
        <!-- ceph rbd -->
        <div v-if="ruleForm.activeBlockStorageType == 3" class="boxs">
          <span class="desc">{{ $t("page.install.config.rbdDesc") }}</span>
          <el-form-item
            label="Provisioner"
            label-width="170px"
            class="d2-mt d2-form-item"
          >
            <el-input
              v-model="storage.RWO.provisioner"
              :disabled="true"
              class="d2-input_inner"
              value="kubernetes.io/rbd"
            ></el-input>
          </el-form-item>
          <el-form-item
            label="Monitors"
            label-width="170px"
            class="d2-mt d2-form-item"
          >
            <el-input
              :placeholder="$t('page.install.config.rbdmonitors')"
              v-model="
                storage.RWO.storageClassParameters.rbdParameters.monitors
              "
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
          <el-form-item
            label="AdminID"
            label-width="170px"
            class="d2-mt d2-form-item"
          >
            <el-input
              v-model="storage.RWO.storageClassParameters.rbdParameters.adminId"
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
          <el-form-item
            label="AdminSecretName"
            label-width="170px"
            class="d2-mt d2-form-item"
          >
            <el-input
              v-model="
                storage.RWO.storageClassParameters.rbdParameters.adminSecretName
              "
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
          <el-form-item
            label="AdminSecretNamespace"
            label-width="170px"
            class="d2-mt d2-form-item"
          >
            <el-input
              v-model="
                storage.RWO.storageClassParameters.rbdParameters
                  .adminSecretNamespace
              "
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
          <el-form-item
            label="Pool"
            label-width="170px"
            class="d2-mt d2-form-item"
          >
            <el-input
              v-model="storage.RWO.storageClassParameters.rbdParameters.pool"
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
          <el-form-item
            label="User ID"
            label-width="170px"
            class="d2-mt d2-form-item"
          >
            <el-input
              v-model="storage.RWO.storageClassParameters.rbdParameters.userId"
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
          <el-form-item
            label="UserSecretName"
            label-width="170px"
            class="d2-mt d2-form-item"
          >
            <el-input
              v-model="
                storage.RWO.storageClassParameters.rbdParameters.userSecretName
              "
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
          <el-form-item
            label="UserSecretNamespace"
            label-width="170px"
            class="d2-mt d2-form-item"
          >
            <el-input
              v-model="
                storage.RWO.storageClassParameters.rbdParameters
                  .userSecretNamespace
              "
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
          <el-form-item
            label="FSType"
            label-width="170px"
            class="d2-mt d2-form-item"
          >
            <el-input
              v-model="storage.RWO.storageClassParameters.rbdParameters.fsType"
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
          <el-form-item
            label="ImageFormat"
            label-width="170px"
            class="d2-mt d2-form-item"
          >
            <el-input
              v-model="
                storage.RWO.storageClassParameters.rbdParameters.imageFormat
              "
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
          <el-form-item
            label="ImageFeatures"
            label-width="170px"
            class="d2-mt d2-form-item"
          >
            <el-input
              v-model="
                storage.RWO.storageClassParameters.rbdParameters.imageFeatures
              "
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
        </div>
        <div class="clues">
          {{ $t("page.install.config.blockStorageDesc") }}
        </div>
      </el-form-item>
      <div style="width:1100px;text-align:center;">
        <el-button type="primary" @click="submitForm('ruleForm')"
          >{{$t('page.install.config.startInstall')}}</el-button
        >
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
    let validateGatewayNodes = (rule, value, callback) => {
      if (this.setgatewayNodes.length === 0) {
        callback(new Error('请至少选择一个网关安装节点'))
      } else {
        callback()
      }
    }
    let validateChaosNodes = (rule, value, callback) => {
      if (this.setChaosNodes.length === 0) {
        callback(new Error('请至少选择一个构建服务安装节点'))
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
        callback(new Error(this.$t('page.install.config.invalidValidation')))
      } else {
        callback()
      }
    }
    let validateDomain = (rule, value, callback) => {
      console.log(value)
      if (value.substr(0, 4) === 'http') {
        callback(new Error(this.$t('page.install.config.nohttpValidation')))
        return
      }
      callback()
    }
    let validUiDateBase = (rule, value, callback) => {
      let str = this.ruleForm.uiDatabase.port
      let ress = reg.test(str)

      if (!ress && str !== '') {
        callback(new Error(this.$t('page.install.config.invalidValidation')))
      } else {
        callback()
      }
    }
    let validateHTTPDomain = (rule, value, callback) => {
      if (value.length < 1 && !this.ruleForm.HTTPDomainSwitch) {
        callback(new Error(this.$t('page.install.config.domainValidation')))
        return
      }
      callback()
    }
    return {
      queryGatewayNodeloading: false,
      queryChaosNodeloading: false,
      upLoading: false,
      loading: true,
      clusterInitInfo: {
        storageClasses: []
      },
      storage: {
        RWX: {
          csiPlugin: {
            aliyunNas: {
              accessKeyID: '',
              accessKeySecret: '',
              volumeAs: 'filesystem',
              zoneId: '',
              vpcId: '',
              vSwitchId: ''
            }
          }
        },
        RWO: {
          provisioner: 'kubernetes.io/rbd',
          storageClassParameters: {
            rbdParameters: {
              monitors: '',
              adminId: 'kube',
              adminSecretName: 'ceph-secret',
              adminSecretNamespace: 'kube-system',
              pool: 'kube',
              userId: 'kube',
              userSecretName: 'ceph-secret-user',
              userSecretNamespace: 'default',
              fsType: 'ext4',
              imageFormat: '2',
              imageFeatures: 'layering'
            }
          },
          csiPlugin: {
            aliyunCloudDisk: {
              accessKeyID: '',
              accessKeySecret: '',
              maxVolumePerNode: 30,
              volumeAs: 'filesystem',
              zoneId: '',
              vpcId: '',
              vSwitchId: ''
            }
          }
        }
      },
      ruleForm: {
        enableHA: false,
        labelPosition: 'left',
        imageHubInstall: true,
        imageHubDomain: '',
        imageHubNamespace: '',
        imageHubUsername: '',
        imageHubPassword: '',
        installRegionDB: true,
        regionDatabaseHost: '',
        regionDatabasePort: 3306,
        regionDatabaseUsername: '',
        regionDatabasePassword: '',
        installUIDB: true,
        uiDatabaseHost: '',
        uiDatabasePort: 3306,
        uiDatabaseUsername: '',
        uiDatabasePassword: '',
        installETCD: true,
        etcdConfig: {
          endpoints: [''],
          certInfo: { caFile: '', certFile: '', keyFile: '' }
        },
        HTTPDomain: '',
        HTTPDomainSwitch: true,
        gatewayIngressIPs: [''],
        activeStorageType: 1,
        shareStorageClassName: '',
        activeBlockStorageType: 0,
        blockStorageClassName: ''
      },
      setgatewayNodes: [],
      optionGatewayNodes: [],
      setChaosNodes: [],
      optionChaosNodes: [],
      fileList: [],
      rules: {
        imageHubInstall: [
          {
            required: true,
            message: this.$t('page.install.config.hubInstallValidation'),
            trigger: 'blur'
          }
        ],
        installRegionDB: [
          {
            required: true,
            message: this.$t('page.install.config.dbValidation'),
            trigger: 'blur'
          }
        ],
        installUIDB: [
          {
            required: true,
            message: this.$t('page.install.config.dbValidation'),
            trigger: 'blur'
          }
        ],
        installETCD: [
          {
            required: true,
            message: this.$t('page.install.config.etcdValidation'),
            trigger: 'blur'
          }
        ],
        imageHubDomain: [
          {
            required: true,
            message: this.$t('page.install.config.hubDomainValidation'),
            trigger: 'blur'
          },
          {
            validator: validateDomain,
            required: true,
            trigger: 'blur'
          }
        ],
        regionDatabaseHost: [
          {
            required: true,
            message: this.$t('page.install.config.dbAddrValidation'),
            trigger: 'blur'
          }
        ],
        regionDatabasePort: [
          {
            required: true,
            message: this.$t('page.install.config.dbPortValidation'),
            trigger: 'blur'
          }
        ],
        regionDatabaseUsername: [
          {
            required: true,
            message: this.$t('page.install.config.dbUserValidation'),
            trigger: 'blur'
          }
        ],
        regionDatabasePassword: [
          {
            required: true,
            message: this.$t('page.install.config.dbPasswordValidation'),
            trigger: 'blur'
          }
        ],
        uiDatabaseHost: [
          {
            required: true,
            message: this.$t('page.install.config.dbAddrValidation'),
            trigger: 'blur'
          }
        ],
        uiDatabasePort: [
          {
            required: true,
            message: this.$t('page.install.config.dbPortValidation'),
            trigger: 'blur'
          }
        ],
        uiDatabaseUsername: [
          {
            required: true,
            message: this.$t('page.install.config.dbUserValidation'),
            trigger: 'blur'
          }
        ],
        uiDatabasePassword: [
          {
            required: true,
            message: this.$t('page.install.config.dbPasswordValidation'),
            trigger: 'blur'
          }
        ],
        activeStorageType: [
          {
            required: true,
            message: this.$t('page.install.config.shareStorageValidation'),
            trigger: 'blur'
          }
        ],
        enableHA: [
          {
            message: this.$t('page.install.config.installModeValidation'),
            required: true,
            trigger: 'blur'
          }
        ],
        nodes: [
          {
            validator: validateGatewayNodes,
            type: 'array',
            required: true,
            trigger: 'change'
          }
        ],
        chaosNodes: [
          {
            validator: validateChaosNodes,
            type: 'array',
            required: true,
            trigger: 'change'
          }
        ],
        HTTPDomain: [
          {
            required: true,
            validator: validateHTTPDomain,
            trigger: 'blur'
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
      }
    }
  },
  created () {
    this.fetchClusterInfo()
    this.fetchClusterInitConfig()
  },
  methods: {
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
      this.$store
        .dispatch('fetchClusterInfo')
        .then(res => {
          if (res && res.data) {
            if (res.data.HTTPDomain && res.data.HTTPDomain !== '') {
              this.ruleForm.HTTPDomainSwitch = false
              this.ruleForm.HTTPDomain = res.data.HTTPDomain
            }
            if (
              res.data.gatewayIngressIPs &&
              res.data.gatewayIngressIPs.length > 0
            ) {
              this.ruleForm.gatewayIngressIPs = res.data.gatewayIngressIPs
            }
          }
        })
        .catch(err => {
          this.$emit('onhandleErrorRecord')
          console.log(err)
        })
    },
    fetchClusterInitConfig () {
      this.$store.dispatch('getClusterInitConfig').then(res => {
        if (res && res.code === 200) {
          this.clusterInitInfo = res.data
          if (
            res.data.gatewayAvailableNodes &&
            res.data.gatewayAvailableNodes.masterNodes
          ) {
            let valueOp = []
            let valueSet = []
            res.data.gatewayAvailableNodes.masterNodes.map(item => {
              valueOp.push(item.internalIP)
              valueSet.push(item.internalIP)
            })
            this.optionGatewayNodes = valueOp
            this.setgatewayNodes = valueSet
          }
          if (
            res.data.gatewayAvailableNodes &&
            res.data.gatewayAvailableNodes.specifiedNodes
          ) {
            let values = []
            let op = this.optionGatewayNodes
            res.data.gatewayAvailableNodes.specifiedNodes.map(item => {
              values.push(item.internalIP)
              op.push(item.internalIP)
            })
            this.setgatewayNodes = values
            this.optionGatewayNodes = this.unique(op)
          }
          if (
            res.data.chaosAvailableNodes &&
            res.data.chaosAvailableNodes.masterNodes
          ) {
            let valueOp = []
            let valueSet = []
            res.data.chaosAvailableNodes.masterNodes.map(item => {
              valueOp.push(item.internalIP)
              valueSet.push(item.internalIP)
            })
            this.setChaosNodes = valueSet
            this.optionChaosNodes = valueOp
          }
          if (
            res.data.chaosAvailableNodes &&
            res.data.chaosAvailableNodes.specifiedNodes
          ) {
            let values = []
            let op = this.optionChaosNodes
            res.data.chaosAvailableNodes.specifiedNodes.map(item => {
              values.push(item.internalIP)
              op.push(item.internalIP)
            })
            this.setChaosNodes = values
            this.optionChaosNodes = this.unique(op)
          }
          this.loading = false
        }
      })
    },
    unique (arr) {
      return Array.from(new Set(arr))
    },
    submitForm (formName) {
      this.$refs[formName].validate(valid => {
        if (valid) {
          console.log('valid success')
          this.loading = true
          let obj = {}
          if (!this.ruleForm.imageHubInstall) {
            obj.imageHub = {
              domain: this.ruleForm.imageHubDomain,
              namespace: this.ruleForm.imageHubNamespace,
              username: this.ruleForm.imageHubUsername,
              password: this.ruleForm.imageHubPassword
            }
          }
          if (!this.ruleForm.installRegionDB) {
            obj.regionDatabase = {
              host: this.ruleForm.regionDatabaseHost,
              port: this.ruleForm.regionDatabasePort,
              username: this.ruleForm.regionDatabaseUsername,
              password: this.ruleForm.regionDatabasePassword
            }
          }
          if (!this.ruleForm.installUIDB) {
            obj.uiDatabase = {
              host: this.ruleForm.uiDatabaseHost,
              port: this.ruleForm.uiDatabasePort,
              username: this.ruleForm.uiDatabaseUsername,
              password: this.ruleForm.uiDatabasePassword
            }
          }
          if (!this.ruleForm.installETCD) {
            obj.etcdConfig = this.ruleForm.etcdConfig
          }
          if (!this.ruleForm.HTTPDomainSwitch) {
            obj.HTTPDomain = this.ruleForm.HTTPDomain
          }
          obj.gatewayIngressIPs = this.ruleForm.gatewayIngressIPs
          let gatewayNodes = []
          this.setgatewayNodes.map(item => {
            gatewayNodes.push({ internalIP: item })
          })
          obj.nodesForGateway = gatewayNodes
          let chaosNodes = []
          this.setChaosNodes.map(item => {
            chaosNodes.push({ internalIP: item })
          })
          obj.nodesForChaos = chaosNodes
          obj.enableHA = this.ruleForm.enableHA
          console.log(obj)
          this.$store
            .dispatch('putClusterInfo', obj)
            .then(res => {
              if (res && res.code === 200) {
                this.$emit('onhandleStartRecord')
                this.installCluster()
              } else {
                this.handleCancelLoading()
              }
            })
            .catch(err => {
              this.handleCancelLoading()
              this.$emit('onhandleErrorRecord')
              console.log(err)
            })
        } else {
          this.handleCancelLoading()
          this.$message({
            message: this.$t('page.install.config.formInvalid'),
            type: 'warning'
          })
          return false
        }
      })
    },
    installCluster () {
      let obj = {}
      obj.rainbondvolumes = {}
      // share storage nfs
      if (this.ruleForm.activeStorageType === 1) {
        obj.rainbondvolumes.RWX = {
          nfs: {}
        }
      }
      // share storage class
      if (this.ruleForm.activeStorageType === 2) {
        obj.rainbondvolumes.RWX = {
          storageClassName: this.ruleForm.shareStorageClassName
        }
      }
      // share nas
      if (this.ruleForm.activeStorageType === 3) {
        obj.rainbondvolumes.RWX = {
          storageClassParameters: {
            parameters: {
              zoneId: this.storage.RWX.csiPlugin.aliyunNas.zoneId,
              volumeAs: this.storage.RWX.csiPlugin.aliyunNas.volumeAs,
              vpcId: this.storage.RWX.csiPlugin.aliyunNas.vpcId,
              vSwitchId: this.storage.RWX.csiPlugin.aliyunNas.vSwitchId
            }
          },
          csiPlugin: {
            aliyunNas: {
              accessKeyID: this.storage.RWX.csiPlugin.aliyunNas.accessKeyID,
              accessKeySecret: this.storage.RWX.csiPlugin.aliyunNas.accessKeySecret
            }
          }
        }
      }
      // block storage class
      if (this.ruleForm.activeBlockStorageType === 1) {
        obj.rainbondvolumes.RWO = {
          storageClassName: this.ruleForm.shareStorageClassName
        }
      }
      // ali disk
      if (this.ruleForm.activeBlockStorageType === 2) {
        obj.rainbondvolumes.RWO = {
          storageClassParameters: {
            parameters: {
              zoneId: this.storage.RWO.csiPlugin.aliyunCloudDisk.zoneId,
              volumeAs: this.storage.RWO.csiPlugin.aliyunCloudDisk.volumeAs,
              vpcId: this.storage.RWO.csiPlugin.aliyunCloudDisk.vpcId,
              vSwitchId: this.storage.RWO.csiPlugin.aliyunCloudDisk.vSwitchId
            }
          },
          csiPlugin: {
            aliyunCloudDisk: {
              accessKeyID: this.storage.RWO.csiPlugin.aliyunCloudDisk.accessKeyID,
              accessKeySecret: this.storage.RWO.csiPlugin.aliyunCloudDisk.accessKeySecret,
              maxVolumePerNode: this.storage.RWO.csiPlugin.aliyunCloudDisk.maxVolumePerNode
            }
          }
        }
      }
      // rbd
      if (this.ruleForm.activeBlockStorageType === 3) {
        obj.rainbondvolumes.RWO = {
          provisioner: this.storage.RWO.provisioner,
          storageClassParameters: {
            parameters: this.storage.RWO.storageClassParameters.rbdParameters
          }
        }
      }
      console.log(obj)
      this.$store
        .dispatch('installCluster', obj)
        .then(en => {
          if (en && en.code === 200) {
            this.$emit('onResults')
          } else {
            this.$emit('onhandleErrorRecord')
            this.handleCancelLoading()
          }
        })
        .catch(_ => {
          this.handleCancelLoading()
          this.$emit('onhandleErrorRecord')
        })
    },
    handleCancelLoading () {
      this.loading = false
    },
    queryGatewayNode (query) {
      if (query.length > 2) {
        this.queryGatewayNodeloading = true
        this.$store
          .dispatch('queryNode', { query, rungateway: true })
          .then(res => {
            if (res && res.data) {
              let values = this.optionGatewayNodes
              res.data.map(item => {
                values.push(item.internalIP)
              })
              this.optionGatewayNodes = this.unique(values)
            }
            this.queryGatewayNodeloading = false
          })
          .catch(err => {
            this.queryGatewayNodeloading = false
            this.$emit('onhandleErrorRecord')
            console.log(err)
          })
      }
    },
    queryChaosNode (query) {
      if (query.length > 2) {
        this.queryChaosNodeloading = true
        this.$store
          .dispatch('queryNode', { query })
          .then(res => {
            if (res && res.data) {
              let values = this.optionChaosNodes
              res.data.map(item => {
                values.push(item.internalIP)
              })
              this.optionChaosNodes = this.unique(values)
            }
            this.queryChaosNodeloading = false
          })
          .catch(err => {
            this.queryChaosNodeloading = false
            this.$emit('onhandleErrorRecord')
            console.log(err)
          })
      }
    }
  }
}
</script>

<style rel="stylesheet/scss" lang="scss" scoped>
.boxs {
  border: 1px solid #dcdfe6 !important;
  line-height: 40px;
  min-height: 40px;
  padding: 16px;
  width: 870px;
  box-sizing: border-box;
  overflow-y: auto;
}
.d2-w-150 {
  min-width: 150px;
}
.d2-w-800 {
  min-width: 800px;
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

.clues {
  font-family: PingFangSC-Regular;
  font-size: 14px;
  color: #999999;
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
<style lang="scss">
.d2-form-item {
  .el-form-item__label {
    line-height: 25px;
  }
  .el-form-item__content {
    line-height: 25px;
  }
}
.d2-input_inner {
  width: 300px;
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
  min-width: 150px;
}
.d2-w-80 {
  min-width: 80px;
}

.clcolor,
.clcolors {
  .el-collapse-item__header {
    border-color: #dcdfe6 !important;
    height: 39px;
    line-height: 39px;
    padding-left: 16px;
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
.table-body {
  padding: 1rem;
}
.desc {
  color: #999999;
  border-left: 2px solid #409eff;
  padding-left: 16px;
}
</style>
