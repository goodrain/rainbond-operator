<template>
  <d2-container class="page">
    <d2-page-cover>
      <img
        class="logo d2-mb"
        slot="logo"
        name="d2-admin"
        :src="`${$baseUrl}image/rainbondLogo.png`"
        alt="loading"
      />
      <template slot="footer">
        <d2-help />
      </template>
    </d2-page-cover>
  </d2-container>
</template>

<script>
import D2Help from "./components/d2-help";
import D2PageCover from "./components/d2-page-cover";
export default {
  components: {
    D2Help,
    D2PageCover
  },
  data() {
    return {
      filename: __filename
    };
  },
  created() {
    console.log("代码版本:2-14:14:30");
  }
};
</script>

<style lang="scss" scoped>
.page {
  .logo {
    width: 120px;
    // background: #002140;
  }
  .btn-group {
    color: $color-text-placehoder;
    font-size: 12px;
    line-height: 12px;
    margin-top: 0px;
    margin-bottom: 20px;
    .btn-group__btn {
      color: $color-text-sub;
      &:hover {
        color: $color-text-main;
      }
      &.btn-group__btn--link {
        color: $color-primary;
      }
    }
  }
}
</style>
