<template>
  <el-card :header="signup ? $t('Sign up') : $t('Authorization Required')" class="login-container">
    <el-form ref="login" :model="formData" :rules="ruleValidate" label-width="100px" label-position="left">
      <el-form-item :label="$t('Username')" prop="username">
        <el-input v-model="formData.username" prefix-icon="el-icon-user-solid" :placeholder="$t('Enter username...')"
                  @keyup.enter.native="handleSubmit"/>
      </el-form-item>
      <el-form-item :label="$t('Password')" prop="password">
        <el-input v-model="formData.password" prefix-icon="el-icon-lock" :placeholder="$t('Enter password...')"
                  show-password @keyup.enter.native="handleSubmit"/>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" style="width: 70%" @click="handleSubmit">{{ signup ? $t('Sign up') : $t('Sign in') }}</el-button>
        <el-button type="warning" @click="reset">{{ $t('Reset') }}</el-button>
      </el-form-item>
    </el-form>
    <p v-if="signup" style="text-align: center">{{ $t('Already have an account?') }}<a href="/login" style="text-decoration: none; color: #1c7cd6">{{ $t('Sign in') }}</a></p>
    <p v-else style="text-align: center">{{ $t('New to Rttys?') }}<a href="/login?signup=1" style="text-decoration: none; color: #1c7cd6">{{ $t('Sign up') }}</a></p>
  </el-card>
</template>

<script lang="ts">
  import {Component, Vue} from 'vue-property-decorator'
  import {Form as ElForm} from 'element-ui/types/element-ui'

  @Component
  export default class Login extends Vue {
    signup = false;

    formData = {
      username: '',
      password: ''
    };

    ruleValidate = {
      username: [{required: true, trigger: 'blur', message: ''}],
      password: [{required: true, trigger: 'blur', message: ''}]
    };

    handleSubmit() {
      (this.$refs['login'] as ElForm).validate(valid => {
        if (valid) {
          const params = {
            username: this.formData.username,
            password: this.formData.password
          };

          if (this.signup) {
            this.axios.post('/signup', params).then(() => {
              this.reset();
              this.signup = false;
              this.$router.push('/login');
            }).catch(() => {
              this.reset();
              this.$message.error(this.$t('Sign up Fail.').toString());
            });
          } else {
            this.axios.post('/signin', params).then(res => {
              sessionStorage.setItem('rttys-sid', res.data.sid);
              sessionStorage.setItem('rttys-username', res.data.username);
              sessionStorage.setItem('rttys-admin', res.data.admin);
              this.$router.push('/');
            }).catch(() => {
              this.$message.error(this.$t('Signin Fail! username or password wrong.').toString());
            });
          }
        }
      });
    }

    reset() {
      (this.$refs['login'] as ElForm).resetFields();
    }

    mounted() {
      this.ruleValidate['username'][0].message = this.$t('username is required').toString();
      this.ruleValidate['password'][0].message = this.$t('password is required').toString();
    }

    created() {
      this.signup = this.$route.query.signup === '1';
      sessionStorage.removeItem('rttys-sid');
    }
  }
</script>

<style scoped>
  .login-container {
    width: 500px;
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
  }
</style>
