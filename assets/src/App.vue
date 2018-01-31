<template>
    <div id="app">
        <Table :loading="table_loading" :height="devlist_height" :columns="columns" :data="devlist" :style="{display: termOn ? 'none' : 'block', width: '100%'}"></Table>
        <div ref="terminal" class="terminal" :style="{display: termOn ? 'block' : 'none'}"></div>
        <Spin size="large" fix v-if="terminal_loading"></Spin>
        <context-menu class="right-menu" :target="contextMenuTarget" :show="contextMenuVisible" @update:show="showMenu">
            <a href="javascript:;" @click="openUpModal">Upload file to device</a>
            <a href="javascript:;" @click="downFile">Download file from device</a>
        </context-menu>
        <Modal v-model="upmodal" width="360" :mask-closable="false" @on-cancel="cancelUpfile">
            <p slot="header">
                <span>Upload file to device</span>
            </p>
            <Upload :before-upload="beforeUpload" action="">
                <Button type="ghost" icon="Upload">Select the file to upload</Button>
            </Upload>
            <Progress v-if="file !== null" :percent="file ? Math.round(filePos / file.size * 100) : 0"></Progress>
            <div v-if="file !== null">The file "{{ file.name }}" will be saved in the "/tmp/" directory of your device.</div>
            <div slot="footer">
                <Button type="primary" size="large" long :loading="modal_loading" @click="doUpload">{{ modal_loading ? 'Uploading' : 'Click to upload' }}</Button>
            </div>
        </Modal>
        <Modal v-model="filelist_modal" width="600" :mask-closable="false">
            <p slot="header">
                <span>Please select file to download</span>
            </p>
            <Tag>{{'/' + dl_path.join('/')}}</Tag>
            <Table :columns="filelist_title" height="300" :data="filelist" @on-row-dblclick="filelistDblclick"></Table>
            <div slot="footer"></div>
        </Modal>
    </div>
</template>

<script>

import * as Socket from 'simple-websocket';
import { Terminal } from 'xterm'
import 'xterm/lib/xterm.css'
import * as fit from 'xterm/lib/addons/fit/fit';
import axios from 'axios'

Terminal.applyAddon(fit);

export default {
    data() {
        return {
            contextMenuTarget: document.body,
            contextMenuVisible: false,
            devlist_height: document.body.offsetHeight,
            table_loading: true,
            termOn: false,
            terminal_loading: false,
            modal_loading: false,
            upmodal: false,
            filelist_modal: false,
            dl_path: [],
            file: null,
            filePos: 0,
            fileStep: 2048,
            cancel_upfile: false,
            ws: null,
            term: null,
            sid: '',
            recvCnt: 0,
            username: '',
            password: '',
            did: '',
            columns: [
                {
                    title: 'ID',
                    key: 'id',
                    sortType: 'asc',
                    sortable: true
                }, {
                    title: 'Uptime',
                    key: 'uptime',
                    sortable: true
                }, {
                    title: 'Description',
                    key: 'description'
                }, {
                    width: 150,
                    align: 'center',
                    render: (h, params) => {
                        return h('Button', {
                            props: {
                                type: 'primary'
                            },
                            on: {
                                click: () => {
                                    this.terminal_loading = true;
                                    this.termOn = true;
                                    this.did = params.row.id;
                                    window.setTimeout(this.login, 200);
                                }
                            }
                        }, 'Connect');
                    }
                }
            ],
            devlist: [ ],
            filelist_title: [
                {
                    title: 'Name',
                    key: 'name',
                    render: (h, params) => {
                        if (params.row.type == 'dir')
                        return h('div', [
                            h('Icon', {props: {type: 'folder', color: '#FFE793', size: 20}}),
                            h('strong', ' ' + params.row.name)
                        ]);
                        else
                            return params.row.name;
                    }
                }, {
                    title: 'Size',
                    key: 'size',
                    sortable: true,
                    render: (h, params) => {
                        let size = params.row.size;
                        let unit = 'B';

                        if (!size)
                            return;

                        if (size > 1024 * 1024 * 1024) {
                            size /= 1024.0 * 1024 * 1024;
                            unit = 'GB';
                        } else if (size > 1024 * 1024) {
                            size /= 1024.0 * 1024;
                            unit = 'MB';
                        } else if (size > 1024) {
                            size /= 1024.0;
                            unit = 'KB';
                        }
                        return size.toFixed(2) + ' ' + unit;
                    }
                }, {
                    title: 'modification',
                    key: 'mtim',
                    sortable: true
                }
            ],
            filelist: []
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
            this.filePos = 0;
            return false;
        },
        readFile(fr) {
            var blob = this.file.slice(this.filePos, this.filePos + this.fileStep);
            fr.readAsArrayBuffer(blob);
        },
        doUpload () {
            if (!this.file) {
                this.$Message.error('Please select file to upload.');
                return;
            }

            this.cancel_upfile = false;
            this.modal_loading = true;
            var fr = new FileReader();
            fr.onload = (e) => {
                if (this.cancel_upfile) {
                    this.file = null;
                    this.modal_loading = false;
                    this.upmodal = false;
                    return;
                }

                this.ws.send(fr.result);
                this.filePos += e.loaded;

                if (this.filePos < this.file.size) {
                    /* Control the client read speed based on the current buffer */
                    if (this.ws.bufferedAmount > this.fileStep * 10) {
                        setTimeout(() => {
                            this.readFile(fr);
                        }, 100);
                    } else {
                        this.readFile(fr);
                    }
                } else {
                    this.file = null;
                    this.modal_loading = false;
                    this.upmodal = false;
                    this.$Message.info("Upload success");
                }
            };

            var msg = {
                type: 'upfile',
                sid: this.sid,
                name: this.file.name,
                size: this.file.size
            };
            this.ws.send(JSON.stringify(msg));

            window.setTimeout(() => {
                this.readFile(fr);
            }, 100);
        },
        cancelUpfile() {
            if (!this.modal_loading)
                return;
            this.cancel_upfile = true;
            this.$Message.info("Upload canceled");
            var msg = {
                type: 'upfile',
                sid: this.sid,
                err: 'canceled'
            };
            this.ws.send(JSON.stringify(msg));
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
            this.file = null;
        },
        filelistDblclick(row, index) {
            if (row.type == 'dir') {
                if (row.name == '..')
                    this.dl_path.pop();
                else
                    this.dl_path.push(row.name);
                var msg = {
                    type: 'filelist',
                    sid: this.sid,
                    name: '/' + this.dl_path.join('/')
                };
                this.ws.send(JSON.stringify(msg));
            } else {
                var msg = {
                    type: 'downfile',
                    sid: this.sid
                };

                if (this.dl_path.length > 0)
                    msg.name = '/' + this.dl_path.join('/') + '/' + row.name;
                else
                    msg.name = '/' + row.name;
                this.ws.send(JSON.stringify(msg));
                this.filelist_modal = false;
                this.$Message.info("TODO");
            }
        },
        downFile () {
            this.contextMenuVisible = false;
            this.filelist_modal = true;
            this.dl_path = [];
            var msg = {
                type: 'filelist',
                sid: this.sid
            };
            this.ws.send(JSON.stringify(msg));
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
            var term = new Terminal({
                cursorBlink: true,
                fontSize: 18
            });
            term.open(this.$refs['terminal']);
            term.fit();
            term.focus();
            this.term = term;

            var protocol = 'ws://';
            if (location.protocol == 'https://')
                protocol = 'wss://';

            var ws = new Socket(protocol + location.host + '/ws/browser?did=' + this.did);
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
                    } else if (type == 'filelist') {
                        this.filelist = resp.list;
                    }
                });

                ws.on('close', ()=> {
                    this.logout(null, term);
                });
            })
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
            this.did = id;
            window.setTimeout(this.login, 200);
        }

        window.setInterval(() => {
            axios.get('/list').then((res => {
                this.table_loading = false;
                this.devlist = res.data;
            }));
        }, 3000);

        window.addEventListener("resize", () => {
            this.devlist_height = document.body.offsetHeight;
            if (this.termOn) {
                this.term.fit();
            }
        });
    }
}
</script>

<style>
    html, body {
		width: 100%;
	    height: 99%;
        background-color: #555;
    }

	#app {
	    width: 100%;
	    height: 100%;
        background-color: #555;
    }

    .terminal {
        height: 100%;
        margin-left: 5px;
        margin-top: 10px;
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
