<template>
    <div @keydown.enter="handleSubmit">
        <Card class="login-container">
            <p slot="title">Login</p>
            <Form ref="form" :model="form" :rules="ruleValidate">
                <FormItem prop="id">
                    <Input type="text" v-model="form.id" size="large" auto-complete="off" placeholder="Enter device ID...">
                        <Icon type="ios-person-outline" slot="prepend"></Icon>
                    </Input>
                </FormItem>
                <FormItem>
                    <Button type="primary" long size="large" icon="log-in" @click="handleSubmit">Login</Button>
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
                    id: ''
                },
                ruleValidate: {
                    id: [
                        {required: true, trigger: 'blur', message: 'Device ID is required'}
                    ]
                }
            }
        },

        methods: {
            handleSubmit() {
                this.$refs['form'].validate((valid) => {
                    if (valid) {
                        this.$router.push('/home');
                    }
                });
            }
        }
    }
</script>

<style scoped>
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
