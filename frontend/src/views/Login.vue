<template>
  <el-card :header="$t('Authorization Required')" class="login-container">
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
        <el-button type="primary" style="width: 70%" @click="handleSubmit">{{ $t('Login') }}</el-button>
        <el-button type="warning" @click="reset">{{ $t('Reset') }}</el-button>
      </el-form-item>
    </el-form>
  </el-card>
</template>

<script lang="ts">
  import {Component, Vue} from 'vue-property-decorator'
  import {Form as ElForm} from 'element-ui/types/element-ui'

  @Component
  export default class Login extends Vue {
    formData = {
      username: '',
      password: ''
    };

    ruleValidate = {
      username: [{required: true, trigger: 'blur', message: ''}]
    };

    handleSubmit() {
      (this.$refs['login'] as ElForm).validate(valid => {
        if (valid) {
          const params = {
            username: this.formData.username,
            password: this.formData.password
          };
          this.axios.post('/signin', params).then(res => {
            sessionStorage.setItem('rtty-sid', res.data);
            this.$router.push('/');
          }).catch(() => {
            this.$message.error(this.$t('Signin Fail! username or password wrong.').toString());
          });
        }
      });
    }

    reset() {
      (this.$refs['login'] as ElForm).resetFields();
    }

    mounted() {
      this.ruleValidate['username'][0].message = this.$t('username is required').toString();
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
