<template>
  <el-config-provider :locale="locale">
    <router-view/>
  </el-config-provider>
</template>

<script setup>
import { locale } from './element-plus'
import { useI18n } from 'vue-i18n'
import { onMounted } from 'vue'

onMounted(() => {
  const i18n = useI18n()
  document.title = i18n.t('app.title')
})
</script>

<style>
  html, body, #app {
    background-color: #555;
    margin: 0;
  }
</style>
