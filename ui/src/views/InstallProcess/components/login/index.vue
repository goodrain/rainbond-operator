<template>
  <div class="publicSubmit">
    <div class="login-container">
      <div class="container clearfix">
        <div class="loginBox">
          <el-form
            autocomplete="on"
            :model="loginForm"
            :rules="loginRules"
            ref="loginForm"
            label-position="left"
          >
            <div class="loginForm">
              <div class="head">
                <a class="active" @click="handleLogins">账号登录</a>
              </div>

              <div class="content">
                <div id="login">
                  <div class="form-group">
                    <div class="form-group">
                      <div class="input-group">
                        <svg aria-hidden="true" class="icon input-group-text">
                          <use xlink:href="#icon-name">
                            <svg id="icon-name" viewBox="0 0 1024 1024" width="100%" height="100%">
                              <path
                                d="M870.410034 772.355608c-19.685219-46.668693-48.021246-88.545603-83.957573-124.488073-35.941447-35.936327-77.818356-64.164846-124.487049-83.956549-0.416721-0.209896-0.833443-0.317404-1.251188-0.525253 64.898971-46.876542 107.091238-123.234838 107.091238-209.385573 0-142.71016-115.631465-258.34265-258.347769-258.34265-142.712208 0-258.343674 115.632489-258.343674 258.34265 0 86.151759 42.187147 162.509031 107.086119 209.493081-0.416721 0.207849-0.833443 0.311261-1.251188 0.520134-46.668693 19.689315-88.545603 47.918857-124.48193 83.961668-35.94247 35.94247-64.170989 77.81938-83.962692 124.488073-19.373958 45.626378-29.690627 94.064345-30.732942 143.854864-0.102389 4.691443 3.64708 8.547395 8.336474 8.547395h62.50308a8.314973 8.314973 0 0 0 8.332378-8.128626c2.085654-80.416977 34.380021-155.732958 91.465724-212.819684 59.064873-59.063849 137.50268-91.566065 221.049675-91.566064 83.545971 0 161.989921 32.502216 221.052746 91.566064 57.08775 57.08775 89.382117 132.402707 91.461628 212.819684 0.106484 4.582911 3.748444 8.128626 8.337498 8.128626h62.50308c4.684275 0 8.43272-3.855952 8.332379-8.547395-1.045387-49.790519-11.355912-98.22951-30.736014-143.962372zM509.457693 533.17702c-47.813397 0-92.813157-18.645976-126.668949-52.501768s-52.502792-78.856576-52.502792-126.675092c0-47.811349 18.647-92.812133 52.502792-126.667925s78.856576-52.501768 126.668949-52.501768c47.817492 0 92.818276 18.645976 126.674068 52.501768 33.855792 33.855792 52.502792 78.856576 52.502792 126.667925 0 47.818516-18.647 92.8193-52.502792 126.675092s-78.856576 52.501768-126.674068 52.501768z m0 0"
                              />
                            </svg>
                          </use>
                        </svg>

                        <el-form-item prop="username" :error="errorUserMsg">
                          <input
                            name="username"
                            type="text"
                            v-model="loginForm.username"
                            autocomplete="on"
                            placeholder="请输入用户名"
                            class="form-control ui-input"
                          />
                        </el-form-item>
                      </div>
                      <div class="help-block"></div>
                    </div>
                  </div>

                  <div class="form-group">
                    <div class="input-group">
                      <svg aria-hidden="true" class="icon input-group-text">
                        <use xlink:href="#icon-pass">
                          <svg id="icon-pass" viewBox="0 0 1024 1024" width="100%" height="100%">
                            <path
                              d="M611.139744 96.566721c-174.373814 0-315.720159 141.346345-315.720159 315.720159 0 73.013263 24.823076 140.204713 66.365175 193.689411l-42.683731 42.687827-64.702386-64.701362c-3.220119-3.220119-8.518726-3.220119-11.841233 0l-41.333227 41.334251c-3.221143 3.221143-3.221143 8.513606 0 11.836114l64.700337 64.701362-46.631833 46.633881-64.702385-64.701362c-3.221143-3.220119-8.51463-3.220119-11.835091 0l-41.33425 41.334251c-3.221143 3.221143-3.221143 8.515654 0 11.841233l64.701361 64.701362-67.822164 67.814997c-3.220119 3.220119-3.220119 8.518726 0 11.734749l43.931848 43.9298c3.221143 3.221143 8.51975 3.221143 11.733726 0L417.348969 661.748348c53.586062 41.441759 120.783656 66.258691 193.791799 66.258691 174.374838 0 315.725279-141.345321 315.725279-315.720159 0-174.373814-141.351465-315.720159-315.726303-315.720159z m167.417538 483.136672c-44.655735 44.764266-104.164974 69.37847-167.417538 69.37847-63.24642 0-122.755659-24.614203-167.411394-69.37847-44.762219-44.654711-69.377446-104.164974-69.377445-167.416513 0-63.24642 24.615227-122.755659 69.377445-167.416513 44.655735-44.757099 104.164974-69.371302 167.411394-69.371303 63.25154 0 122.760779 24.615227 167.417538 69.371303 44.762219 44.661878 69.377446 104.170093 69.377445 167.416513 0 63.25154-24.615227 122.761803-69.377445 167.416513z m0 0"
                            />
                          </svg>
                        </use>
                      </svg>
                      <el-form-item prop="password" :error="errorPasswordMsg">
                        <input
                          type="password"
                          id="password"
                          maxlength="16"
                          @keyup.enter="handleLogin"
                          v-model="loginForm.password"
                          autocomplete="on"
                          placeholder="请输入密码"
                          class="form-control ui-input"
                        />
                      </el-form-item>
                    </div>
                    <div class="help-block"></div>
                  </div>
                  <div class="form-group remember"></div>

                  <el-form-item style="margin: 30px auto">
                    <el-button
                      style="width: 100%"
                      type="primary"
                      :loading="loading"
                      @click.native.prevent="handleLogin"
                    >登录</el-button>
                  </el-form-item>
                </div>
              </div>
            </div>
          </el-form>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'Login',
  data () {
    // 定义用户名校验规则
    const validateUsername = (rule, value, callback) => {
      if (!value.trim()) {
        callback(new Error('用户名不能为空'))
      } else {
        callback()
      }
    }
    // 定义密码校验规则
    const validatePass = (rule, value, callback) => {
      if (value.length < 8) {
        callback(new Error('密码不能小于8位'))
      }
      if (value.length > 16) {
        callback(new Error('密码不能大于16位'))
      } else {
        callback()
      }
    }
    return {
      errorUserMsg: null,
      errorPasswordMsg: null,
      loginForm: {
        // username: window.localStorage.userName,
        username: '',
        password: ''
      },
      loginRules: {
        username: [
          { required: true, trigger: 'blur', validator: validateUsername }
        ],
        password: [{ required: true, trigger: 'blur', validator: validatePass }]
      },
      loading: false,
      dialogVisible: false,
      Address: false
    }
  },
  methods: {
    // 登录
    handleLogin () {
      this.$refs.loginForm.validate(valid => {
        if (valid) {
          this.errorUserMsg = null
          this.errorPasswordMsg = null
          this.loading = true
          this.$store
            .dispatch('Login', this.loginForm)
            .then(res => {
              this.$message({
                message: '登录成功',
                type: 'success'
              })
              this.loading = false
              this.$router.push('/cluster')
            })
            .catch(err => {
              this.loading = false
              if (err) {
                switch (err) {
                  case 50001:
                    this.$message.warning('生成账户失败')
                    break
                  case 50002:
                    this.$message.warning('不可重复生成管理员')
                    break
                  case 50003:
                    this.errorPasswordMsg = '密码错误'
                    break
                  case 50007:
                    this.errorUserMsg = '用户未找到'
                    break
                }
              }
            })
        } else {
          return false
        }
      })
    },
    // 点击登录页面上的头，就给推一个登录的页面
    handleLogins () {
      this.$router.push({
        name: 'login',
        query: { redirect: this.$route.fullPath }
      })
    },
    handleQuickLogin () {
      this.$router.push({ name: 'quick' })
    }
  }
}
</script>

<style rel="stylesheet/scss" lang="scss" scoped>
.publicSubmit {
  .login-form-layout {
    position: absolute;
    left: 0;
    right: 0;
    width: 360px;
    margin: 140px auto;
    border-top: 10px solid #409eff;
  }
  .third-login legend {
    width: auto;
    padding: 0 10px;
    border: 0;
    font-size: 14px;
    color: #666;
  }
  .btn.btn-boot {
    background-color: #428bca;
    color: #fff;
    border-color: #428bca;
    transition: background-color 0.5;
  }
  .login-center-layout {
    width: auto;
    height: auto;
    max-width: 100%;
    max-height: 100%;
    margin-top: 200px;
  }
  .login-container {
    background-repeat: repeat-x;
    background-size: 100% 100%;
    -moz-background-size: 100% 100%;
    background-color: #fff;
    height: 100vh;
    overflow: auto;
    background-position: top;
    background-repeat: no-repeat;
    background-size: cover;
  }
  .loginForm {
    padding: 0 85px 25px;
    background-color: #fff;
    border-radius: 5px;
    text-align: center;
  }
  .loginForm .head {
    border-bottom: 1px solid #e7e6eb;
    margin-bottom: 30px;
    text-align: center;
    font-size: 0;
  }
  .loginForm .head a.active,
  .loginForm .head a:hover {
    color: #428bca;
  }
  .loginForm .head a {
    display: inline-block;
    text-align: center;
    padding: 20px 0;
    position: relative;
    margin: 0 15px;
    font-size: 14px;
  }
  .loginForm .input-group {
    position: relative;
    display: -webkit-box;
    display: -ms-flexbox;
    display: flex;
    -ms-flex-wrap: wrap;
    flex-wrap: wrap;
    -webkit-box-align: stretch;
    -ms-flex-align: stretch;
    align-items: stretch;
    width: 100%;
    -webkit-box-align: center;
    -ms-flex-align: center;
    align-items: center;
    border: 1px solid #e7e6eb;
    -ms-flex-wrap: nowrap;
    flex-wrap: nowrap;
  }
  .loginForm .input-group .icon {
    -webkit-box-flex: 0;
    -ms-flex: 0 1 45px;
    flex: 0 1 45px;
  }
  .input-group {
    position: relative;
    display: -webkit-box;
    display: -ms-flexbox;
    display: flex;
    -ms-flex-wrap: wrap;
    flex-wrap: wrap;
    -webkit-box-align: stretch;
    -ms-flex-align: stretch;
    align-items: stretch;
    width: 100%;
  }
  .loginForm .icon {
    width: 1em;
    height: 1em;
    vertical-align: -0.15em;
    fill: currentColor;
    overflow: hidden;
    color: #428bca;
    font-size: 20px;
  }
  .input-group > .form-control:not(:first-child),
  .input-group > .nf:not(:first-child) {
    border-top-left-radius: 0;
    border-bottom-left-radius: 0;
  }
  .form-control {
    display: block;
    width: 100%;
    padding: 0.375rem 0.75rem;
    font-size: 1rem;
    line-height: 1.5;
    color: #1e3948;
    background-color: #fff;
    background-clip: inherit;
    border: 1px solid #ced4da;
    border-radius: 0.25rem;
    -webkit-transition: border-color 0.15s ease-in-out,
      box-shadow 0.15s ease-in-out;
    transition: border-color 0.15s ease-in-out, box-shadow 0.15s ease-in-out;
  }
  .loginForm .remember {
    color: #666;
    text-align: left;
  }
  .loginForm .remember input {
    margin-right: 5px;
  }
  .help-block {
    margin-top: 2px;
    margin-bottom: 5px;
    padding-bottom: 0;
    font-size: 10px;
    text-align: left;
    min-height: 16px;
    color: #a94442;
  }
  .loginForm .c-blue {
    color: #428bca;
  }
  .loginForm .btn-block {
    width: 100%;
    margin-top: 10px;
  }
  // .clearfix:after {
  //     display: block;
  //     content: "";
  //     clear: both;
  // }
  .loginBox {
    width: 500px;
    height: auto;
    display: block;
    text-align: center;
    margin: 150px auto 0;
    border: 1px solid #ebebeb;
    border-radius: 3px;
    transition: 0.2s;
    &:hover {
      box-shadow: 0 0 8px 0 rgba(232, 237, 250, 0.6),
        0 2px 4px 0 rgba(232, 237, 250, 0.5);
    }
  }
  .loginForm .btn-block {
    width: 100%;
    margin-top: 10px;
  }
  .third-login .field-title {
    margin-top: 38px;
    margin-bottom: 10px;
    padding: 0 5.5em;
    border: 0;
    border-top: 1px solid #e7e6eb !important;
    text-align: center;
  }
  .third-login a {
    margin-right: 10px;
  }
  .third-login a img {
    -webkit-filter: grayscale(1);
    filter: grayscale(1);
  }
  .third-login a:hover img {
    -webkit-filter: none;
    filter: none;
  }
  .loginForm .head a.active:after,
  .loginForm .head a:hover:after {
    content: "";
    width: 100%;
    height: 2px;
    background-color: #428bca;
    position: absolute;
    left: 0;
    bottom: 0;
  }
  input:-internal-autofill-selected {
    background-color: #e9effe !important;
    background-image: none !important;
    color: rgb(0, 0, 0) !important;
    width: 267px !important;
    background-clip: inherit !important;
  }
  .ui-input {
    background-color: #e9effe !important;
    border: none;
    border-radius: 0rem;
    width: 267px !important;
    background-clip: inherit !important;
  }
  .el-form-item {
    margin-bottom: 0px;
    width: 100%;
  }
  .initBox {
    position: absolute;
    left: 50%;
    top: 20%;
    width: 500px;
    display: block;
    text-align: center;
    float: right;
    margin: 50px;
    background: #ffffff;
    padding: 20px 20px 20px 0;
    text-align: center;
    z-index: 99;
  }
  .initTitle {
    font-size: 19px;
    height: 30px;
    line-height: 30px;
    margin-bottom: 40px;
    padding-bottom: 40px;
    border-bottom: 1px solid #eeeeee;
  }
  .initwd {
    width: 80%;
    margin: 0px auto;
    padding-bottom: 30px;
  }
}

@media screen and (min-width: 300px) and (max-width: 800px) {
  .publicSubmit {
    .initwd {
      width: 100%;
      padding-left: 10px;
    }
    .initTitle {
      font-size: 16px;
      height: 40px;
      line-height: 40px;
      margin-bottom: 5px;
      padding-bottom: 5px;
      border-bottom: 1px solid #eeeeee;
    }
    .initBox {
      position: inherit;
      left: 0%;
      top: 0%;
      width: 95%;
      display: block;
      text-align: center;
      float: inherit;
      margin: 100px auto 0;
      padding: 0px;
    }
    .container {
      width: 100%;
      margin: 0;
    }
    .loginBox {
      width: 95%;
      float: inherit;
      margin: 10px auto;
    }
    .loginForm {
      margin-top: 100px;
      padding: 0 15px 5px;
    }
    .wmax > div {
      width: 100% !important;
    }
    .loginForm .head {
      margin: 30px 0px;
    }
  }
  .detectionCode {
    & > div {
      width: 100% !important;
    }
    .loginForm {
      padding: 0px !important;
      margin-top: 0px !important;
      margin-bottom: 20px !important;
    }
    .loginForm .head {
      margin: 0px 0px;
    }
    .el-dialog__body {
      padding: 0px 5px 10px !important;
    }
  }
}
</style>
