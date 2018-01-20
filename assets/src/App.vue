<template>
    <div id="app">
        <Card class="login-container" :style="{display: termOn ? 'none' : 'block'}">
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
        <div ref="terminal" class="terminal-container" :style="{display: termOn ? 'block' : 'none'}"></div>
    </div>
</template>

<script>

import * as Socket from 'simple-websocket';
import { Terminal } from 'xterm'
import 'xterm/lib/xterm.css'
import * as fit from 'xterm/lib/addons/fit/fit';
import { Base64 } from 'js-base64';


/**
 *  * Convert an Uint8Array into a string.
 *   *
 *    * @returns {String}
 *     */
function Decodeuint8arr(uint8array){
        return new TextDecoder("utf-8").decode(uint8array);
}

/**
 *  * Convert a string into a Uint8Array.
 *   *
 *    * @returns {Uint8Array}
 *     */
function Encodeuint8arr(myString){
        return new TextEncoder("utf-8").encode(myString);
}

export default {
    data() {
        return {
            termOn: false,
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
        login() {
            Terminal.applyAddon(fit);
            var term = new Terminal();
            term.open(this.$refs['terminal']);
            term.fit();

            var protocol = 'ws://';
            if (location.protocol == 'https://')
                protocol = 'wss://';

            var ws = new Socket(protocol + '192.168.0.100:5912' + '/ws/browser?did=' + this.form.id);
            ws.on('connect', ()=> {
                ws._ws.onmessage = (e)=>{
                    var resp = JSON.parse(e.data);
                    var type = resp.type;

                    if (type == 'login') {
                        var sid = resp.sid;

                        term.on('data', (data)=> {
                            data = JSON.stringify({type: 'data', did: this.form.id, sid: sid, data: Base64.encode(data)});
                            ws.send(data);
                        });
                    } else if (type == 'data') {
                        term.write(Base64.decode(resp.data));
                    } else if (type == 'logout') {
                        term.destroy();
                        this.termOn = false;
                    }
                };

                ws.on('destroy', ()=> {
                    term.destroy();
                    this.termOn = false;
                });
            })
        },
        handleSubmit() {
            this.$refs['form'].validate((valid) => {
                if (valid) {
                    this.termOn = true;
                    window.setTimeout(this.login, 200);
                }
            });
        }
    }
}
</script>

<style>
    html, body {
		width: 100%;
	    height: 100%;
    }

	#app {
	    width: 100%;
	    height: 100%;
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
