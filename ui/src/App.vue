<template>
  <el-config-provider :locale="locale">
    <router-view/>
  </el-config-provider>
</template>

<script setup>
import { locale } from './element-plus'
</script>

<style>
  html, body, #app {
    background-color: #555;
    margin-bottom: 0;
  }
</style>
