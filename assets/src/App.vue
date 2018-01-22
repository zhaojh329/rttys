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
                <FormItem>
                    <Button type="primary" long size="large" icon="log-in" @click="handleSubmit">Login</Button>
                </FormItem>
            </Form>
        </Card>
        <div ref="terminal" :style="{display: termOn ? 'block' : 'none', height: '100%'}"></div>
        <Spin size="large" fix v-if="terminal_loading"></Spin>
        <context-menu class="right-menu" :target="contextMenuTarget" :show="contextMenuVisible" @update:show="showMenu">
            <a href="javascript:;" @click="openUpModal">Upload file to device</a>
            <a href="javascript:;" @click="downFile">Download file from device</a>
        </context-menu>
        <Modal v-model="upmodal" width="360">
            <p slot="header">
                <span>Upload file to device</span>
            </p>
            <Upload :before-upload="beforeUpload" action="//jsonplaceholder.typicode.com/posts/">
                <Button type="ghost" icon="Upload">Select the file to upload</Button>
            </Upload>
            <div v-if="file !== null">Upload file: {{ file.name }}</div>
            <div slot="footer">
                <Button type="primary" size="large" long :loading="modal_loading" @click="doUpload">{{ modal_loading ? 'Uploading' : 'Click to upload' }}</Button>
            </div>
        </Modal>
    </div>
</template>

<script>

import * as Socket from 'simple-websocket';
import { Terminal } from 'xterm'
import 'xterm/lib/xterm.css'
import * as fit from 'xterm/lib/addons/fit/fit';

export default {
    data() {
        return {
            contextMenuTarget: document.body,
            contextMenuVisible: false,
            termOn: false,
            terminal_loading: false,
            modal_loading: false,
            upmodal: false,
            file: null,
            filePos: 0,
            fileStep: 512,
            ws: null,
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
        /* ucs-2 string to base64 encoded ascii */
        utoa(str) {
            return window.btoa(unescape(encodeURIComponent(str)));
        },
        /* base64 encoded ascii to ucs-2 string */
        atou(str) {
            return decodeURIComponent(escape(window.atob(str)));
        },
        beforeUpload (file) {
            this.file = file;
            return false;
        },
        readFile(fr) {
            var blob = this.file.slice(this.filePos, this.filePos + this.fileStep);
            fr.readAsArrayBuffer(blob);
        },
        arrayBufferToBase64(buffer) {
            var binary = '';
            var bytes = new Uint8Array(buffer);
            var len = bytes.byteLength;
            for (var i = 0; i < len; i++) {
            binary += String.fromCharCode(bytes[ i ]);
            }
            return this.utoa(binary);
        },
        doUpload () {
            this.modal_loading = true;
            this.filePos = 0;
            var fr = new FileReader();
            fr.onload = (e) => {
                var msg = {
                    type: 'upfile',
                    sid: this.sid,
                    name: this.file.name,
                    data: this.arrayBufferToBase64(fr.result)
                };
                this.ws.send(JSON.stringify(msg));

                this.filePos += e.loaded;

                if (this.filePos < this.file.size) {
                    /* Control the client read speed based on the current buffer */
                    if (this.ws.bufferedAmount > this.fileStep * 10) {
                        setTimeout(() => {
                            this.readFile(fr);
                        }, 3);
                    } else {
                        this.readFile(fr);
                    }
                } else {
                    msg.data = '';
                    this.ws.send(JSON.stringify(msg));

                    this.file = null;
                    this.upmodal = false;
                    this.modal_loading = false;
                    this.$Message.success('Upload Success');
                }
            };

            this.readFile(fr);
        },
        showMenu(show) {
            if (!this.termOn)
                show = false;
            this.contextMenuVisible = show;
        },
        openUpModal () {
            this.contextMenuVisible = false;
            this.upmodal = true;
            this.modal_loading = false;
        },
        downFile () {
            alert('downFile');
            this.contextMenuVisible = false
        },
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
                cursorBlink: true,
                lineHeight: 1.1
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
                        this.terminal_loading = false;

                        if (resp.err) {
                             this.$Message.error(resp.err);
                            this.logout(null, term);
                            return;
                        }
                        this.ws = ws;
                        this.sid = resp.sid;
                        term.on('data', (data)=> {
                            data = JSON.stringify({type: 'data', sid: this.sid, data: this.utoa(data)});
                            ws.send(data);
                        });
                    } else if (type == 'data') {
                        this.recvCnt++;
                        var data = this.atou(resp.data);

                        if (this.recvCnt < 4) {
                            if (data.match('login:') && this.username != '') {
                                data = JSON.stringify({type: 'data', sid: this.sid, data: this.utoa(this.username + '\n')});
                                ws.send(data);
                                return;
                            }

                            if (data.match('Password:') && this.password != '') {
                                data = JSON.stringify({type: 'data', sid: this.sid, data: this.utoa(this.password + '\n')});
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
            this.$refs['form'].validate((valid) => {
                if (valid) {
                    this.terminal_loading = true;
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
            this.terminal_loading = true;
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
        height: 200px;
        top: 50%;
        left: 50%;
        margin-left: -200px;
        margin-top: -120px;
    }

    .right-menu {
        position: fixed;
        background: #fff;
        border-radius: 3px;
        z-index: 999;
        display: none;
    }

    .right-menu a {
        width: 150px;
        height: 28px;
        line-height: 28px;
        text-align: left;
        display: block;
        color: #1a1a1a;
        border: solid 1px rgba(0, 0, 0, .2);
    }

    .right-menu a:hover {
        background: #42b983;
        color: #fff;
    }
</style>
