<template>
  <div class="d2-page-cover">
    <div class="d2-page-cover__logo">
      <slot name="logo" />
    </div>
    <p class="d2-page-cover__title">{{$t('page.overview.title')}}</p>
    <p class="d2-page-cover__sub-title">{{$t('page.overview.description')}}</p>
    <p class="d2-page-cover__build-time">FINAL BUILD TIME {{$buildTime}}</p>
    <slot name="footer" />
  </div>
</template>

<style lang="scss" scoped>
.d2-page-cover {
  @extend %full;
  @extend %unable-select;
  display: flex;
  flex-flow: column nowrap;
  justify-content: center;
  align-items: center;
  .d2-page-cover__logo {
    img {
      width: 200px;
    }
  }
  .d2-page-cover__title {
    margin: 0px;
    margin-bottom: 20px;
    font-weight: bold;
    color: $color-text-main;
  }
  .d2-page-cover__sub-title {
    margin: 0px;
    margin-bottom: 5px;
    color: $color-text-normal;
  }
  .d2-page-cover__build-time {
    margin: 0px;
    margin-bottom: 10px;
    font-size: 12px;
    color: $color-text-placehoder;
  }
}
</style>
