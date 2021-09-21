<template>
  <Card :title="signup ? $t('Sign up') : $t('Authorization Required')" class="login-container">
    <Form ref="login" :model="formData" :rules="ruleValidate" :label-width="100" label-position="left">
      <FormItem :label="$t('Username')" prop="username">
        <Input v-model="formData.username" prefix="ios-person-outline" :placeholder="$t('Enter username...')" @on-enter="handleSubmit"/>
      </FormItem>
      <FormItem :label="$t('Password')" prop="password">
        <Input type="password" v-model="formData.password" prefix="ios-lock-outline" :placeholder="$t('Enter password...')" password @on-enter="handleSubmit"/>
      </FormItem>
      <FormItem>
        <Button type="primary" size="large" long @click="handleSubmit">{{ signup ? $t('Sign up') : $t('Sign in') }}</Button>
      </FormItem>
    </Form>
    <p v-if="signup" style="text-align: center">{{ $t('Already have an account?') }}<a href="/login" style="text-decoration: none; color: #1c7cd6">{{ $t('Sign in') }}</a></p>
    <p v-else style="text-align: center">{{ $t('New to Rttys?') }}<a href="/login?signup=1" style="text-decoration: none; color: #1c7cd6">{{ $t('Sign up') }}</a></p>
  </Card>
</template>

<script>
export default {
  name: 'Login',
  data() {
    return {
      signup: false,
        formData: {
        username: '',
        password: ''
      },
      ruleValidate: {
        username: [{required: true, trigger: 'blur', message: this.$t('username is required')}],
        password: [{required: true, trigger: 'blur', message: this.$t('password is required')}]
      }
    }
  },
  methods: {
    handleSubmit() {
      (this.$refs['login']).validate(valid => {
        if (valid) {
          const params = {
            username: this.formData.username,
            password: this.formData.password
          };

          if (this.signup) {
            this.axios.post('/signup', params).then(() => {
              this.signup = false;
              this.$router.push('/login');
            }).catch(() => {
              this.$Message.error(this.$t('Sign up Fail.').toString());
            });
          } else {
            this.axios.post('/signin', params).then(res => {
              sessionStorage.setItem('rttys-sid', res.data.sid);
              sessionStorage.setItem('rttys-username', res.data.username);
              sessionStorage.setItem('rttys-admin', res.data.admin);
              this.$router.push('/');
            }).catch(() => {
              this.$Message.error(this.$t('Signin Fail! username or password wrong.').toString());
            });
          }
        }
      });
    }
  },
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
