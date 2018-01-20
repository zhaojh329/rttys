<template>
    <div id="app">
        <Card class="login-container" :style="{display: termOn ? 'none' : 'block'}">
            <p slot="title">Login</p>
            <Form ref="form" :model="form" :rules="ruleValidate">
                <FormItem prop="id">
                    <Input type="text" v-model="form.id" size="large" auto-complete="off" placeholder="Enter your device ID...">
                        <Icon type="social-tux" slot="prepend"></Icon>
                    </Input>
                </FormItem>
                <Alert v-show="msg" type="error">{{msg}}</Alert>
                <FormItem>
                    <Button type="primary" long size="large" icon="log-in" @click="handleSubmit">Login</Button>
                </FormItem>
            </Form>
        </Card>
        <div ref="terminal" class="terminal-container" :style="{display: termOn ? 'block' : 'none'}"></div>
        <Spin size="large" fix v-if="loading"></Spin>
    </div>
</template>

<script>

import * as Socket from 'simple-websocket';
import { Terminal } from 'xterm'
import 'xterm/lib/xterm.css'
import * as fit from 'xterm/lib/addons/fit/fit';
import { Base64 } from 'js-base64';

export default {
    data() {
        return {
            termOn: false,
            loading: false,
            msg: '',
            sid: '',
            recvCnt: 0,
            username: '',
            password: '',
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
        getQueryString(name) {
            var reg = new RegExp("(^|&)" + name + "=([^&]*)(&|$)", "i");
            var r = window.location.search.substr(1).match(reg);
            if (r != null)
                return unescape(r[2]);
            return null;
        },
        logout(ws, term) {
            this.termOn = false;

            if (ws)
                ws.destroy();
            if (term)
                term.destroy();
        },
        login() {
            Terminal.applyAddon(fit);
            var term = new Terminal({
                cursorBlink: true
            });
            term.open(this.$refs['terminal']);
            term.fit();
            term.focus();

            window.addEventListener("resize", function(event) {
                term.fit();
            });

            var protocol = 'ws://';
            if (location.protocol == 'https://')
                protocol = 'wss://';

            var ws = new Socket(protocol + location.host + '/ws/browser?did=' + this.form.id);
            ws.on('connect', ()=> {
                ws.on('data', (data)=>{
                    var resp = JSON.parse(data);
                    var type = resp.type;

                    if (type == 'login') {
                        this.loading = false;

                        if (resp.err) {
                            this.msg = resp.err;
                            this.logout(ws, term);
                            return;
                        }

                        this.sid = resp.sid;
                        term.on('data', (data)=> {
                            data = JSON.stringify({type: 'data', sid: this.sid, data: Base64.encode(data)});
                            ws.send(data);
                        });
                    } else if (type == 'data') {
                        this.recvCnt++;
                        var data = Base64.decode(resp.data);

                        if (this.recvCnt < 4) {
                            if (data.match('login:') && this.username != '') {
                                data = JSON.stringify({type: 'data', sid: this.sid, data: Base64.encode(this.username + '\n')});
                                ws.send(data);
                                return;
                            }

                            if (data.match('Password:') && this.password != '') {
                                data = JSON.stringify({type: 'data', sid: this.sid, data: Base64.encode(this.password + '\n')});
                                ws.send(data);
                                return;
                            }
                        }
                        term.write(data);
                    }
                });

                ws.on('close', ()=> {
                    this.logout(null, term);
                });
            })
        },
        handleSubmit() {
            this.msg = '';
            this.$refs['form'].validate((valid) => {
                if (valid) {
                    this.loading = true;
                    this.termOn = true;
                    window.setTimeout(this.login, 200);
                }
            });
        }
    },
    mounted() {
        var id = this.getQueryString('id');
        var username = this.getQueryString('username');
        var password = this.getQueryString('password');

        if (username)
            this.username = username;
        if (password)
            this.password = password;

        if (id) {
            this.loading = true;
            this.termOn = true;
            this.form.id = id;
            window.setTimeout(this.login, 200);
        }
    }
}
</script>

<style>
    html, body {
		width: 100%;
	    height: 100%;
        background-color: #555;
    }

	#app {
	    width: 100%;
	    height: 100%;
        background-color: #555;
    }

    .login-container {
        width: 400px;
        height: 240px;
        top: 50%;
        left: 50%;
        margin-left: -200px;
        margin-top: -120px;
    }

    .terminal-container {
        height: 100%;
    }
</style>
