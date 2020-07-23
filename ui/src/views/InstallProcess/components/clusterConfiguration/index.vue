<template>
  <div v-loading="loading">
    <el-alert
      v-if="onlyRegion"
      title="安装提示"
      type="info"
      description="当前集群不会安装控制台,集群安装完成后需要对接到已有控制台，此安装模式适用于对接Rainbond Cloud服务，或扩充多集群。"
      show-icon
    ></el-alert>

    <el-form
      :model="ruleForm"
      :rules="rules"
      @submit.native.prevent
      ref="ruleForm"
      label-width="40px"
      label-position="left"
      class="setruleForm"
      autocomplete="on"
    >
    <el-collapse v-model="activeNames" @change="handleChange">
      <!-- install mode -->
      <el-collapse-item name="installmode" class="d2-mb">
        <template slot="title">
          <div class="elCollapseHeader actives">
            <div>{{$t('page.install.config.installmode')}}</div>
            <p>选择适合自己的安装方式</p>
          </div>
        </template>
          <el-form-item
            prop="enableHA"
          >
            <el-radio-group
              class="d2-ml-35"
              v-model="ruleForm.enableHA"
              @change="installModeChange"
            >
              <el-radio class="d2-w-150" :label="false">
                {{ $t("page.install.config.minimize") }}
              </el-radio>
              <el-radio :label="true">{{ $t("page.install.config.ha") }}</el-radio>
            </el-radio-group>
            <div class="clues">{{ $t("page.install.config.installmodeDesc") }}</div>
          </el-form-item>

      </el-collapse-item>

      <!-- hub config -->
      <el-collapse-item name="hub" class="d2-mb">
        <template slot="title">
          <div class="elCollapseHeader actives">
            <div>{{$t('page.install.config.hub')}}</div>
            <p>{{ $t("page.install.config.hubDesc") }}</p>
          </div>
        </template>
        <el-form-item
          prop="imageHubInstall"
        >
          <el-radio-group class="d2-ml-35" v-model="ruleForm.imageHubInstall">
            <el-radio class="d2-w-150" :label="true">
              {{ ruleForm.enableHA?$t("page.install.config.hubInstallHA"):$t("page.install.config.hubInstall") }}
            </el-radio>
            <el-radio :label="false">
              {{ $t("page.install.config.hubProvide") }}
            </el-radio>
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
                autocomplete="off"
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
                show-password
                class="d2-input_inner"
                autocomplete="off"
              ></el-input>
            </el-form-item>
          </div>
        </el-form-item>
      </el-collapse-item>

      <!-- region db config -->
      <el-collapse-item name="regionDB" class="d2-mb">
        <template slot="title">
          <div class="elCollapseHeader actives">
            <div>{{$t('page.install.config.regionDB')}}</div>
            <p>{{ $t("page.install.config.regionDBDesc") }}</p>
          </div>
        </template>
        <el-form-item
          prop="installRegionDB"
        >
          <el-radio-group class="d2-ml-35" v-model="ruleForm.installRegionDB">
            <el-radio class="d2-w-150" :label="true">
              {{ $t("page.install.config.regionDBInstall") }}
            </el-radio>
            <el-radio :label="false">
              {{ $t("page.install.config.regionDBProvide") }}
            </el-radio>
          </el-radio-group>
          <div v-if="!ruleForm.installRegionDB" class="boxs">
            <span class="desc">
              {{ $t("page.install.config.regionDBProviderDesc") }}
            </span>
            <el-form-item
              :label="$t('page.install.config.regionDBAddress')"
              label-width="85px"
              class="d2-mt d2-form-item"
              prop="regionDatabaseHost"
            >
              <el-input
                v-model="ruleForm.regionDatabaseHost"
                class="d2-input_inner_url"
                style="width:240px"
              ></el-input>
              <span class="d2-w-20">:</span>

              <el-form-item
                prop="regionDatabasePort"
                style="width:300px;display: inline-block"
              >
                <el-input
                  v-model="ruleForm.regionDatabasePort"
                  class="d2-input_inner_url"
                  style="width:140px"
                  type="number"
                ></el-input>
              </el-form-item>
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
                show-password
                class="d2-input_inner"
              ></el-input>
            </el-form-item>
          </div>
        </el-form-item>
      </el-collapse-item>

      <!-- ui db config -->
      <el-collapse-item v-if="!onlyRegion" name="uiDB" class="d2-mb">
        <template slot="title">
          <div class="elCollapseHeader actives">
            <div>{{$t('page.install.config.uiDB')}}</div>
            <p>{{ $t("page.install.config.uiDBDesc") }}</p>
          </div>
        </template>
      <el-form-item
        prop="installUIDB"
      >
        <el-radio-group class="d2-ml-35" v-model="ruleForm.installUIDB">
          <el-radio class="d2-w-150" :label="true">
            {{ $t("page.install.config.uiDBInstall") }}
          </el-radio>
          <el-radio :label="false">
            {{ $t("page.install.config.uiDBProvide") }}
          </el-radio>
        </el-radio-group>
        <div v-if="!ruleForm.installUIDB" class="boxs">
          <span class="desc">
            {{ $t("page.install.config.uiDBProviderDesc") }}
          </span>
          <el-form-item
            :label="$t('page.install.config.uiDBAddress')"
            label-width="85px"
            class="d2-mt d2-form-item"
            prop="uiDatabaseHost"
          >
            <el-input
              v-model="ruleForm.uiDatabaseHost"
              class="d2-input_inner_url"
              style="width:240px"
            ></el-input>
            <span class="d2-w-20">:</span>

            <el-form-item
              prop="uiDatabasePort"
              style="width:300px;display: inline-block"
            >
              <el-input
                v-model="ruleForm.uiDatabasePort"
                class="d2-input_inner_url"
                style="width:140px"
                type="number"
              ></el-input>
            </el-form-item>
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
              show-password
              class="d2-input_inner"
            ></el-input>
          </el-form-item>
        </div>
      </el-form-item>
      </el-collapse-item>

      <!-- etcd config -->
      <el-collapse-item name="etcd" class="d2-mb">
        <template slot="title">
          <div class="elCollapseHeader actives">
            <div>{{$t('page.install.config.etcd')}}</div>
            <p>{{ $t("page.install.config.etcdDesc") }}</p>
          </div>
        </template>
        <el-form-item  prop="installETCD">
          <el-radio-group class="d2-ml-35" v-model="ruleForm.installETCD">
            <el-radio class="d2-w-150" :label="true">
              {{ ruleForm.enableHA ? $t("page.install.config.etcdInstallHA"):$t("page.install.config.etcdInstall") }}
            </el-radio>
            <el-radio :label="false">
              {{ $t("page.install.config.etcdProvide") }}
            </el-radio>
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
                <el-form-item  :prop="`items[${index}].endpoints`"
                :error="errorEndpointsMsg">
                  <el-input
                  placeholder="ETCD地址"
                  @blur="detection(index)"
                  v-model="ruleForm.etcdConfig.endpoints[index]"
                  class="d2-input_inner"
                  ></el-input>
                </el-form-item>
                <em
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
        </el-form-item>
      </el-collapse-item>

      <!-- gateway node config -->
      <el-collapse-item name="nodes" class="d2-mb">
        <template slot="title">
          <div class="elCollapseHeader actives d2-pt-5">
            <div>{{$t('page.install.config.gatewayNode')}}</div>
            <p class="clues">{{ $t("page.install.config.gatewayNodeDesc") }}</p>
            <p class="clues">
              提示：如果你无法搜索并选择一个网关 IP，请参考
              <a
                style="color:#409EFF"
                target="_black"
                href="https://www.rainbond.com/docs/user-operations/install/troubleshooting/#%E6%97%A0%E6%B3%95%E9%80%89%E6%8B%A9%E7%BD%91%E5%85%B3%E8%8A%82%E7%82%B9"
              >
                无法选择网关节点。
              </a>
            </p>
          </div>
        </template>
        <el-form-item  prop="nodes">
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
            ></el-option>
          </el-select>

        </el-form-item>
      </el-collapse-item>

      <!-- chaos node config -->
      <el-collapse-item name="chaosNodes" class="d2-mb">
        <template slot="title">
          <div class="elCollapseHeader actives">
            <div>{{$t('page.install.config.chaosNode')}}</div>
            <p>{{ $t("page.install.config.chaosNodeDesc") }}</p>
          </div>
        </template>
        <el-form-item
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
            ></el-option>
          </el-select>
        </el-form-item>
      </el-collapse-item>

      <!-- default app domain config -->
      <el-collapse-item name="HTTPDomain" class="d2-mb">
        <template slot="title">
          <div class="elCollapseHeader actives">
            <div>{{$t('page.install.config.appDefaultDomain')}}</div>
            <p>{{ $t("page.install.config.appDefaultDomainDesc") }}</p>
          </div>
        </template>
        <el-form-item
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

        </el-form-item>
      </el-collapse-item>

      <!-- eip config -->
      <el-collapse-item name="gatewayIP" class="d2-mb">
        <template slot="title">
          <div class="elCollapseHeader d2-pt-5 noactives">
            <div>{{$t('page.install.config.gatewayIP')}}</div>
            <p>{{ $t("page.install.config.gatewayIPDesc") }}</p>
          </div>
        </template>
        <el-form-item  prop="ips">
          <!-- <div class="boxs"> -->
            <div v-for="(item, indexs) in ruleForm.gatewayIngressIPs" :key="indexs" class="cen">
              <el-input v-model="ruleForm.gatewayIngressIPs[indexs]" class="d2-input_inner"></el-input>
              <!-- <em
                v-show="ruleForm.gatewayIngressIPs.length != 1"
                class="el-icon-remove-outline icon-f-22 d2-ml-16"
                @click.prevent="removeIP(indexs)"
              /> -->
            </div>
            <div class="clues" >以下场景，请务必填写网关公网IP：</div>
            <div class="clues" >- 当网关节点具备公网 IP 地址时，填写对应的公网地址。</div>
            <div class="clues" >- 当多个网关节点被统一负载均衡时，填写负载均衡的 IP 地址，如阿里云 SLB 服务地址。</div>
            <div class="clues" >- 当多个网关节点部署诸如 Keepalived 等基于 VIP 的高可用服务时，填写 VIP。</div>
            <!-- <el-button style="margin-top:1rem" size="small" @click="addIP">
              {{
              $t("page.install.config.gatewayIPAdd")
              }}
            </el-button> -->
          <!-- </div> -->
        </el-form-item>
      </el-collapse-item>

      <!-- share storage config -->
      <el-collapse-item name="shareStorage" class="d2-mb">
        <template slot="title">
          <div class="elCollapseHeader actives">
            <div>{{$t('page.install.config.shareStorage')}}</div>
            <p>{{ $t("page.install.config.shareStorageDesc") }}</p>
          </div>
        </template>
        <el-form-item
          prop="activeStorageType"
        >
          <el-radio-group
            @change="validShareStorage"
            v-model="ruleForm.activeStorageType"
            class="d2-ml-35"
          >
            <el-radio v-if="!ruleForm.enableHA" class="d2-w-150" :label="1">{{
              $t("page.install.config.newNFSServer")
            }}</el-radio>
            <el-radio :label="2">{{
              $t("page.install.config.selectStorage")
            }}</el-radio>
            <!-- <el-radio :label="3">{{
              $t("page.install.config.useAliNasFilesystem")
            }}</el-radio> -->
            <el-radio :label="4">{{
              $t("page.install.config.useAliNasSubpath")
            }}</el-radio>
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
              label-width="120px"
              class="d2-mt-form d2-form-item"
              v-if="clusterInitInfo.storageClasses"
            >
              <el-radio-group
                @change="validShareStorage"
                v-model="ruleForm.shareStorageClassName"
              >
                <el-radio
                  border
                  style="margin-bottom: 1rem;margin-left:10px"
                  size="medium"
                  :title="item.accessMode === 'Unknown' && '无法识别读写模式'"
                  v-for="item in clusterInitInfo.storageClasses"
                  v-show="item.accessMode !== 'ReadWriteOnce'"
                  :key="item.name"
                  :label="item.name"
                ></el-radio>
              </el-radio-group>
            </el-form-item>
          </div>

          <!-- useAliNasFilesystem-->
          <div v-show="ruleForm.activeStorageType == 3" class="boxs">
            <span class="desc">{{
              $t("page.install.config.nasFilesystem")
            }}</span>
            <el-form-item
              label="AccessKeyID"
              label-width="130px"
              class="d2-mt d2-form-item"
            >
              <el-input
                @change="validShareStorage"
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
                @change="validShareStorage"
                :placeholder="$t('page.install.config.accessKeySecret')"
                v-model="storage.RWX.csiPlugin.aliyunNas.accessKeySecret"
                class="d2-input_inner"
              ></el-input>
            </el-form-item>
            <el-form-item
              label="ZoneID"
              label-width="130px"
              class="d2-mt d2-form-item"
            >
              <el-input
                @change="validShareStorage"
                :placeholder="$t('page.install.config.zoneId')"
                v-model="storage.RWX.csiPlugin.aliyunNas.zoneId"
                class="d2-input_inner"
              ></el-input>
            </el-form-item>
            <el-form-item
              label="VPC ID"
              label-width="130px"
              class="d2-mt d2-form-item"
            >
              <el-input
                @change="validShareStorage"
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
                @change="validShareStorage"
                :placeholder="$t('page.install.config.vSwitchId')"
                v-model="storage.RWX.csiPlugin.aliyunNas.vSwitchId"
                class="d2-input_inner"
              ></el-input>
            </el-form-item>
          </div>

          <!-- useAliNasSubpath -->
          <div v-show="ruleForm.activeStorageType == 4" class="boxs">
            <span class="desc">{{ $t("page.install.config.nasSubpath") }}</span>
            <el-form-item
              label="Server"
              label-width="85px"
              class="d2-mt d2-form-item"
            >
              <el-input
                v-model="storage.RWX.csiPlugin.aliyunNas.server"
                @change="validShareStorage"
                :placeholder="$t('page.install.config.Server')"
                class="d2-input_inner_url"
              ></el-input>
              <div class="clues">
              示例：abcdefghij-12345.cn-huhehaote.nas.aliyuncs.com:/
              </div>
            </el-form-item>
          </div>

        </el-form-item>
      </el-collapse-item>

      <!-- block storage config -->
      <el-collapse-item name="blockStorage" class="d2-mb">
        <template slot="title">
          <div class="elCollapseHeader noactives">
            <div>{{$t('page.install.config.blockStorage')}}</div>
            <p>{{ $t("page.install.config.blockStorageDesc") }}</p>
          </div>
        </template>
        <el-form-item
          prop="activeBlockStorageType"
        >
          <el-radio-group
            @change="validBlockStorage"
            v-model="ruleForm.activeBlockStorageType"
            class="d2-ml-35"
          >
            <el-radio :label="0">{{
              $t("page.install.config.noStorage")
            }}</el-radio>
            <el-radio :label="1">{{
              $t("page.install.config.selectStorage")
            }}</el-radio>
            <el-radio :label="2">
              {{ $t("page.install.config.useAliDisk") }}
            </el-radio>
            <!-- <el-radio :label="3">{{ $t("page.install.config.useRBD") }}</el-radio> -->
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
              label-width="120px"
              v-if="clusterInitInfo.storageClasses"
              class="d2-mt-form d2-form-item"
            >
              <el-radio-group
                @change="validBlockStorage"
                v-model="ruleForm.blockStorageClassName"
              >
                <el-radio
                  border
                  style="margin-bottom: 1rem;margin-left:10px"
                  v-for="item in clusterInitInfo.storageClasses"
                  :title="item.accessMode === 'Unknown' && '无法识别读写模式'"
                  v-show="item.accessMode !== 'ReadWriteMany'"
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
                @change="validBlockStorage"
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
                @change="validBlockStorage"
                :placeholder="$t('page.install.config.accessKeySecret')"
                v-model="storage.RWO.csiPlugin.aliyunCloudDisk.accessKeySecret"
                class="d2-input_inner"
              ></el-input>
            </el-form-item>
            <el-form-item
              label="RegionID"
              label-width="130px"
              class="d2-mt d2-form-item"
            >
              <el-input
                @change="validBlockStorage"
                :placeholder="$t('page.install.config.regionID')"
                v-model="storage.RWO.csiPlugin.aliyunCloudDisk.region_id"
                class="d2-input_inner"
              ></el-input>
            </el-form-item>
            <el-form-item
              label="ZoneID"
              label-width="130px"
              class="d2-mt d2-form-item"
            >
              <el-input
                @change="validBlockStorage"
                :placeholder="$t('page.install.config.zoneId')"
                v-model="storage.RWO.csiPlugin.aliyunCloudDisk.zoneId"
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
                @change="validBlockStorage"
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
                @change="validBlockStorage"
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
                @change="validBlockStorage"
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
                @change="validBlockStorage"
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
                @change="validBlockStorage"
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
                @change="validBlockStorage"
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
                @change="validBlockStorage"
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
                @change="validBlockStorage"
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
                @change="validBlockStorage"
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
                @change="validBlockStorage"
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
                @change="validBlockStorage"
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
                @change="validBlockStorage"
                v-model="
                  storage.RWO.storageClassParameters.rbdParameters.imageFeatures
                "
                class="d2-input_inner"
              ></el-input>
            </el-form-item>
          </div>
        </el-form-item>
      </el-collapse-item>
    </el-collapse>
      <div style="width:1100px;text-align:center;">
        <el-button type="primary" @click="submitForm('ruleForm')">
          {{ $t("page.install.config.next") }}
        </el-button>
      </div>
    </el-form>
  </div>
</template>

<script>
import { validateDomain, validateDomainPort, validateIpPort, validateIp } from '@/libs/validate'

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
        return item.indexOf('127.0.0.1') > -1 || !regIp.test(item)
      })

      if (gatewayIngressIPs.length > 0) {
        if (gatewayIngressIPs.length === 1 && gatewayIngressIPs[0] === '') {
          callback()
        } else if (arr.length >= 1) {
          if (arr[0].indexOf('127.0.0.1') > -1) {
            callback(new Error('不可以添加 127.0.0.1'))
          } else {
            callback(new Error(this.$t('page.install.config.portValidation')))
          }
        } else {
          callback()
        }
      } else {
        callback()
      }
    }

    const validateAddressIP = (rule, value, callback) => {
      if ((!validateDomain(value) && !validateDomainPort(value)) && !validateIp(value)) {
        callback(new Error('合法格式:rm-xxxxxxxx.mysql.rds.aliyuncs.com 或 192.168.1.1'))
      } else {
        callback()
      }
    }
    let validateport = (rule, value, callback) => {
      let regPort = /^([0-9]|[1-9]\d{1,3}|[1-5]\d{4}|6[0-5]{2}[0-3][0-5])$/
      let isRegPort = regPort.test(value)
      if (value === '') {
        callback(new Error(this.$t('page.install.config.dbPortValidation')))
      } else if (!isRegPort) {
        callback(new Error(this.$t('page.install.config.portValidation')))
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
    let validateDomains = (rule, value, callback) => {
      if (value.substr(0, 4) === 'http') {
        callback(new Error(this.$t('page.install.config.nohttpValidation')))
      } else if ((!validateDomain(value) && !validateDomainPort(value)) && !validateIpPort(value)) {
        callback(new Error('合法格式:hub.docker.com或192.168.1.1:5000'))
      } else {
        callback()
      }
    }
    //        const domainRules = (rule, value, callback) => {
    //   if (value === '') {
    //     callback(new Error('自定义域名不能为空'));
    //   } else if (!validateDomain(value)) {
    //     callback(new Error('域名格式：www.baidu.com'));
    //   } else {
    //     callback();
    //   }
    // }

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
    let validateShareStorage = (rule, value, callback) => {
      if (value === 2 && this.ruleForm.shareStorageClassName === '') {
        callback(
          new Error(this.$t('page.install.config.storageClassValidation'))
        )
        return
      }
      if (value === 3) {
        const nas = this.storage.RWX.csiPlugin.aliyunNas
        if (
          nas.zoneId === '' ||
          nas.vpcId === '' ||
          nas.vSwitchId === '' ||
          nas.accessKeyID === '' ||
          nas.accessKeySecret === ''
        ) {
          callback(new Error(this.$t('page.install.config.nasValidation')))
          return
        }
      }
      if (value === 4) {
        if (this.storage.RWX.csiPlugin.aliyunNas.server === '') {
          callback(new Error(this.$t('page.install.config.nasValidation')))
          return
        }
      }
      callback()
    }
    let validateBlockStorage = (rule, value, callback) => {
      if (value === 1 && this.ruleForm.blockStorageClassName === '') {
        callback(
          new Error(this.$t('page.install.config.storageClassValidation'))
        )
      }
      if (value === 2) {
        const disk = this.storage.RWO.csiPlugin.aliyunCloudDisk
        if (
          disk.region_id === '' ||
          disk.zoneId === '' ||
          disk.accessKeyID === '' ||
          disk.accessKeySecret === ''
        ) {
          callback(new Error(this.$t('page.install.config.diskValidation')))
          return
        }
      }
      if (value === 3) {
        const rbd = this.storage.RWO.storageClassParameters.rbdParameters
        if (
          rbd.monitors === '' ||
          rbd.adminId === '' ||
          rbd.adminSecretName === '' ||
          rbd.adminSecretNamespace === '' ||
          rbd.pool === '' ||
          rbd.userId === '' ||
          rbd.userSecretName === '' ||
          rbd.userSecretNamespace === '' ||
          rbd.fsType === '' ||
          rbd.imageFormat === '' ||
          rbd.imageFeatures === ''
        ) {
          callback(new Error(this.$t('page.install.config.rbdValidation')))
          return
        }
      }
      callback()
    }
    return {
      activeNames: ['nodes', 'chaosNodes', 'gatewayIP'],
      errorEndpointsMsg: null,
      queryGatewayNodeloading: false,
      queryChaosNodeloading: false,
      upLoading: false,
      loading: true,
      onlyRegion: false,
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
              vSwitchId: '',
              server: ''
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
              maxVolumePerNode: 15,
              type: 'cloud_ssd',
              zoneId: '',
              region_id: ''
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
            validator: validateDomains,
            required: true,
            trigger: 'blur'
          }
        ],
        regionDatabaseHost: [
          {
            required: true,
            message: this.$t('page.install.config.dbAddrValidation'),
            trigger: 'blur'
          },
          {
            validator: validateAddressIP,
            required: true,
            trigger: 'blur'
          }
        ],
        regionDatabasePort: [
          {
            validator: validateport,
            type: 'Number',
            required: true,
            trigger: 'change'
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
          },
          {
            validator: validateAddressIP,
            required: true,
            trigger: 'blur'
          }
        ],
        uiDatabasePort: [
          {
            validator: validateport,
            type: 'Number',
            required: true,
            trigger: 'change'
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
            validator: validateShareStorage,
            required: true,
            trigger: 'blur'
          }
        ],
        activeBlockStorageType: [
          {
            validator: validateBlockStorage,
            required: false,
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
            required: false,
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
    handleChange (val) {
      console.log(val)
    },
    detection (index) {
      this.errorEndpointsMsg = null
      if (index || index == 0) {
        const value = this.ruleForm.etcdConfig.endpoints[index]
        if ((!validateDomain(value) && !validateDomainPort(value)) && !validateIpPort(value)) {
          this.errorEndpointsMsg = '合法格式:etcd-0:2379 或 192.168.1.1:2379'
        }
      }
    },
    fetchErrMessage (err) {
      return err && typeof err === 'object' ? JSON.stringify(err) : ''
    },
    validShareStorage (value, item) {
      const info = this.clusterInitInfo
      const arr =
        info &&
        info.storageClasses &&
        info.storageClasses.length > 0 &&
        info.storageClasses
      if (arr) {
        arr.map(item => {
          if (item.name === value && item.accessMode === 'Unknown') {
            this.openNfsMessage(
              'ReadWriteMany',
              'activeStorageType',
              'shareStorageClassName'
            )
          }
        })
      } else {
        this.$refs.ruleForm.validateField('activeStorageType')
      }
    },
    validBlockStorage (value) {
      const info = this.clusterInitInfo
      const arr =
        info &&
        info.storageClasses &&
        info.storageClasses.length > 0 &&
        info.storageClasses
      if (arr) {
        arr.map(item => {
          if (item.name === value && item.accessMode === 'Unknown') {
            this.openNfsMessage(
              'ReadWriteOnce',
              'activeBlockStorageType',
              'blockStorageClassName'
            )
          }
        })
      } else {
        this.$refs.ruleForm.validateField('activeBlockStorageType')
      }
    },
    openNfsMessage (text, checkName, formName) {
      this.$confirm(`请务必确认该存储的读写模式支持 ${text}"?`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
        center: true
      })
        .then(() => {
          this.$refs.ruleForm.validateField(checkName)
        })
        .catch(() => {
          this.ruleForm[formName] = ''
          this.$refs.ruleForm.validateField(checkName)
        })
    },
    installModeChange (value) {
      if (value && this.ruleForm.activeStorageType === 1) {
        this.ruleForm.activeStorageType = 2
        this.activeNames.push('shareStorage')
      } else {
        this.ruleForm.activeStorageType = 1
      }
    },
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
            this.onlyRegion = res.data.only_install_region
          }
        })
        .catch(err => {
          const message = this.fetchErrMessage(err)
          this.$emit('onhandleErrorRecord', 'failure', `${message}`)
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
        if (valid && !this.errorEndpointsMsg) {
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
              port: Number(this.ruleForm.regionDatabasePort),
              username: this.ruleForm.regionDatabaseUsername,
              password: this.ruleForm.regionDatabasePassword
            }
          }
          if (!this.ruleForm.installUIDB) {
            obj.uiDatabase = {
              host: this.ruleForm.uiDatabaseHost,
              port: Number(this.ruleForm.uiDatabasePort),
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
          let ips = []
          this.ruleForm.gatewayIngressIPs.map(item => {
            if (item !== '') {
              ips.push(item)
            }
          })
          obj.gatewayIngressIPs = this.unique(ips)
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
          this.installCluster(obj)
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
    installCluster (parameter) {
      let obj = {}
      obj.rainbondvolumes = {}
      // share storage nfs
      if (this.ruleForm.activeStorageType === 1) {
        obj.rainbondvolumes.RWX = {
          csiPlugin: {
            nfs: {}
          }
        }
      }
      // share storage class
      if (this.ruleForm.activeStorageType === 2) {
        obj.rainbondvolumes.RWX = {
          storageClassName: this.ruleForm.shareStorageClassName
        }
      }
      // share nas filesystem
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
              accessKeySecret: this.storage.RWX.csiPlugin.aliyunNas
                .accessKeySecret
            }
          }
        }
      }
      // share nas subpath
      if (this.ruleForm.activeStorageType === 4) {
        obj.rainbondvolumes.RWX = {
          storageClassParameters: {
            parameters: {
              server: this.storage.RWX.csiPlugin.aliyunNas.server,
              volumeAs: 'subpath'
            }
          },
          csiPlugin: {
            aliyunNas: {
              accessKeyID: '',
              accessKeySecret: ''
            }
          }
        }
      }
      // block storage class
      if (this.ruleForm.activeBlockStorageType === 1) {
        obj.rainbondvolumes.RWO = {
          storageClassName: this.ruleForm.blockStorageClassName
        }
      }
      // ali disk
      if (this.ruleForm.activeBlockStorageType === 2) {
        obj.rainbondvolumes.RWO = {
          storageClassParameters: {
            parameters: {
              zoneId: this.storage.RWO.csiPlugin.aliyunCloudDisk.zoneId,
              region_id: this.storage.RWO.csiPlugin.aliyunCloudDisk.region_id,
              type: this.storage.RWO.csiPlugin.aliyunCloudDisk.type
            }
          },
          csiPlugin: {
            aliyunCloudDisk: {
              accessKeyID: this.storage.RWO.csiPlugin.aliyunCloudDisk
                .accessKeyID,
              accessKeySecret: this.storage.RWO.csiPlugin.aliyunCloudDisk
                .accessKeySecret,
              maxVolumePerNode: this.storage.RWO.csiPlugin.aliyunCloudDisk
                .maxVolumePerNode
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
      this.$store
        .dispatch('putClusterInfo', Object.assign({}, parameter, obj))
        .then(res => {
          if (res && res.code === 200) {
            this.$emit('onhandleStartRecord')
            this.$emit('onResults')
          } else {
            this.handleCancelLoading()
          }
        })
        .catch(err => {
          this.handleCancelLoading()
          const message = this.fetchErrMessage(err)
          this.$emit('onhandleErrorRecord', 'failure', `${message}`)
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
            const message = this.fetchErrMessage(err)
            this.$emit('onhandleErrorRecord', 'failure', `${message}`)
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
            const message = this.fetchErrMessage(err)
            this.$emit('onhandleErrorRecord', 'failure', `${message}`)
          })
      }
    }
  }
}
</script>

<style  lang="scss" scoped>
.elCollapseHeader{
  padding-left:15px;
  background: rgb(242, 245, 245) !important;
  font-family: PingFangSC-Regular;
  div,p{line-height: 20px;margin: 0;}
  div{
    font-size: 16px;
  }
  p{
    font-size: 12px;
    color: #999999;
  }
}

// .actives{
//   div{
//     &:before {
//     content: '*';
//     color: #F56C6C;
//     margin-right: 8px;
//     }
//   }
// p{margin-left: 16px;}
// }
.actives,.noactives{
  div,p{margin-left: 16px;}
}
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
.setruleForm{
.el-collapse-item__content{
  padding-bottom: 0px;
  padding-top: 20px;
}
.el-collapse-item__header{
  min-height: 48px;
  height: auto !important;
  background: rgb(242, 245, 245) !important;
}
.el-collapse-item__header,.el-collapse-item__wrap{
    border: 1px solid rgb(221, 228, 230) !important;
  }
.el-collapse-item__wrap{
  border-top: 0 !important;
  }

  .el-collapse{
    border: none !important;
  }
}
.d2-mt-form {
  margin-top: 20px;
  .el-form-item__label {
    margin-top: 8px;
  }
}
.d2-form-item {
  .el-form-item__label {
    line-height: 25px;
  }
  .el-form-item__content {
    line-height: 25px;
  }
}
.d2-input_inner {
  width: 400px;
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
