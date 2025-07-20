<template>
  <el-card class="login">
    <template #header>
      {{ $t('Authorization Required') }}
    </template>
    <el-form :model="formValue" size="large" @submit.prevent="handleSubmit">
      <el-form-item prop="password">
        <el-input autofocus type="password" v-model="formValue.password" prefix-icon="lock" :placeholder="$t('Enter password...')" show-password/>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="loading" @click="handleSubmit" class="login-button">{{ $t('Sign in') }}</el-button>
      </el-form-item>
    </el-form>
  </el-card>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import axios from 'axios'

const { t } = useI18n()
const router = useRouter()

const loading = ref(false)
const formValue = reactive({
  password: ''
})

const handleSubmit = () => {
  const params = {
    password: formValue.password
  }

  axios.post('/signin', params).then(() => {
    router.push('/')
  }).catch(() => {
    ElMessage.error(t('Signin Fail! password wrong.'))
  })
}
</script>

<style scoped>
.header {
  text-align: center;
}

.login {
  width: 400px;
  top: 40%;
  left: 50%;
  position: fixed;
  transform: translate(-50%, -50%);
}

.login-button {
  width: 100%;
}
</style>
