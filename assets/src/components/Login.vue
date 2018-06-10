<template>
    <div @keydown.enter="handleSubmit">
        <Card class="login-container">
            <p slot="title">{{ $t('Authorization Required') }}</p>
            <Form ref="form" :model="form" :rules="ruleValidate">
                <FormItem prop="username">
                    <Input type="text" v-model="form.username" size="large" auto-complete="off" :placeholder="$t('Enter username...')">
                        <Icon type="ios-person-outline" slot="prepend"></Icon>
                    </Input>
                </FormItem>
                <FormItem>
                    <Input type="password" v-model="form.password" size="large" auto-complete="off" :placeholder="$t('Enter password...')">
                        <Icon type="ios-locked-outline" slot="prepend"></Icon>
                    </Input>
                </FormItem>
                <FormItem>
                    <Button type="primary" long size="large" icon="log-in" @click="handleSubmit">{{ $t('Login') }}</Button>
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
                        
                        const params = new URLSearchParams();
                        params.append('username', this.form.username);
                        params.append('password', this.form.password);
                        this.$http.post('/login', params).then(res => {
                            sessionStorage.setItem('rtty-sid', res)
                            this.$router.push('/');
                        }).catch(err => {
                            this.$Message.error(this.$t('Login Fail! username or password wrong.'));
                        });
                    }
                });
            }
        }
    }
</script>

<style>
    .login-container {
        width: 400px;
        height: 240px;
        position: absolute;
        top: 50%;
        left: 50%;
        margin-left: -200px;
        margin-top: -120px;
    }
</style>
