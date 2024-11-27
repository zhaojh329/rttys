<template>
  <el-card class="login">
    <template #header>
      {{ signup ? $t('Sign up') : $t('Authorization Required') }}
    </template>
    <el-form ref="form" :model="formValue" :rules="rules" label-width="80px" label-suffix=":" size="large">
      <el-form-item :label="$t('Username')" prop="username">
        <el-input v-model="formValue.username" prefix-icon="user" :placeholder="$t('Enter username...')" @keyup.enter="handleSubmit" autofocus/>
      </el-form-item>
      <el-form-item :label="$t('Password')" prop="password">
        <el-input type="password" v-model="formValue.password" prefix-icon="lock" :placeholder="$t('Enter password...')" @keyup.enter="handleSubmit" show-password/>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="loading" @click="handleSubmit" class="login-button">{{ signup ? $t('Sign up') : $t('Sign in') }}</el-button>
      </el-form-item>
    </el-form>
    <el-divider/>
    <p v-if="signup" style="text-align: center">{{ $t('Already have an account?') }}
      <a href="#" @click.prevent="toggleSignup" style="text-decoration: none; color: #1c7cd6">{{ $t('Sign in') }}</a>
    </p>
    <p v-else-if="allowSignup" style="text-align: center">{{ $t('New to Rttys?') }}
      <a href="#" @click.prevent="toggleSignup" style="text-decoration: none; color: #1c7cd6">{{ $t('Sign up') }}</a>
    </p>
  </el-card>
</template>

<script>
export default {
  data() {
    return {
      signup: false,
      allowSignup: false,
      loading: false,
      formValue: {
        username: '',
        password: ''
      },
      rules: {
        username: {
          required: true,
          trigger: 'blur',
          message: () => this.$t('username is required')
        },
        password: {
          required: true,
          trigger: 'blur',
          message: () => this.$t('password is required')
        }
      }
    }
  },
  methods: {
    toggleSignup() {
      this.signup = !this.signup
    },
    handleSubmit() {
      this.$refs.form.validate(valid => {
        if (valid) {
          const params = {
            username: this.formValue.username,
            password: this.formValue.password
          }

          if (this.signup) {
            this.axios.post('/signup', params).then(() => {
              this.signup = false
              this.$router.push('/login')
            }).catch(() => {
              this.$message.error(this.$t('Sign up Fail.'))
            })
          } else {
            this.axios.post('/signin', params).then(res => {
              sessionStorage.setItem('rttys-sid', res.data.sid)
              sessionStorage.setItem('rttys-username', res.data.username)
              sessionStorage.setItem('rttys-admin', res.data.admin)
              this.$router.push('/')
            }).catch(() => {
              this.$message.error(this.$t('Signin Fail! username or password wrong.'))
            })
          }
        }
      })
    }
  },
  created() {
    sessionStorage.removeItem('rttys-sid')
    this.axios.get('/allowsignup').then(response => {
      this.allowSignup = response.data.allow
    }).catch(() => {
      this.allowSignup = false
    })
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
