<template>
  <el-card class="login">
    <template #header>
      {{ $t('Authorization Required') }}
    </template>
    <el-form ref="form" :model="formValue" :rules="rules" label-width="80px" label-suffix=":" size="large">
      <el-form-item :label="$t('Password')" prop="password">
        <el-input type="password" v-model="formValue.password" prefix-icon="lock" :placeholder="$t('Enter password...')" @keyup.enter="handleSubmit" show-password/>
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
      },
      rules: {
        password: {
          trigger: 'blur',
          message: () => this.$t('password is required')
        }
      }
    }
  },
  methods: {
    handleSubmit() {
      this.$refs.form.validate(valid => {
        if (valid) {
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
      })
    }
  },
  created() {
    sessionStorage.removeItem('rttys-sid')
  }
}
</script>

<style scoped>
.header {
  text-align: center;
}

.login {
  width: 500px;
  top: 40%;
  left: 50%;
  position: fixed;
  transform: translate(-50%, -50%);
}

.login-button {
  width: 100%;
}

.copyright {
  text-align: right;
  font-size: 1.2em;
  color: #888;
}
</style>
