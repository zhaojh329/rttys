<template>
    <div @keydown.enter="handleSubmit" class="login-container">
        <Card>
            <p slot="title">{{ $t('Authorization Required') }}</p>
            <Form ref="form" :model="form" :rules="ruleValidate">
                <FormItem prop="username">
                    <Input type="text" v-model="form.username" size="large" auto-complete="off" prefix="ios-person" :placeholder="$t('Enter username...')" />
                </FormItem>
                <FormItem>
                    <Input type="password" v-model="form.password" size="large" auto-complete="off" prefix="ios-lock" :placeholder="$t('Enter password...')" />
                </FormItem>
                <FormItem>
                    <Button type="primary" long size="large" icon="ios-log-in" @click="handleSubmit">{{ $t('Login') }}</Button>
                </FormItem>
            </Form>
        </Card>
    </div>
</template>

<script>
export default {
    name: 'Login',
    data() {
        return {
            form: {
                username: '',
                password: ''
            },
            ruleValidate: {
                username: [
                    {required: true, trigger: 'blur', message: this.$t('username is required')}
                ]
            }
        }
    },

    methods: {
        handleSubmit() {
            this.$refs['form'].validate((valid) => {
                if (valid) {
                    let params = {
                        username: this.form.username,
                        password: this.form.password
                    };
                    this.$axios.post('/signin', params).then(res => {
                        sessionStorage.setItem('rtty-sid', res);
                        this.$router.push('/');
                    }).catch(() => {
                        this.$Message.error(this.$t('Signin Fail! username or password wrong.'));
                    });
                }
            });
        }
    }
};
</script>

<style>
.login-container {
    width: 400px;
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
}
</style>
