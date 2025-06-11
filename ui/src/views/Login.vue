<template>
  <el-card class="login">
    <template #header>
      {{ $t('Authorization Required') }}
    </template>
    <el-form :model="formValue" size="large">
      <el-form-item prop="password">
        <el-input ref="passwordInput" type="password" v-model="formValue.password" prefix-icon="lock" :placeholder="$t('Enter password...')" show-password/>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="loading" @click="handleSubmit" class="login-button">{{ $t('Sign in') }}</el-button>
      </el-form-item>
    </el-form>
  </el-card>
</template>

<script>
export default {
  data() {
    return {
      loading: false,
      formValue: {
        password: ''
      }
    }
  },
  methods: {
    handleSubmit() {
      const params = {
        password: this.formValue.password
      }

      this.axios.post('/signin', params).then(res => {
        sessionStorage.setItem('rttys-sid', res.data.sid)
        this.$router.push('/')
      }).catch(() => {
        this.$message.error(this.$t('Signin Fail! password wrong.'))
      })
    }
  },
  created() {
    sessionStorage.removeItem('rttys-sid')
  },
  mounted() {
    this.$nextTick(() => {
      this.$refs.passwordInput.focus()
    })
  }
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
