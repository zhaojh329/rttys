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
        <Modal v-model="filelist_modal" width="700" :mask-closable="false">
            <p slot="header"><span>Please select file to download</span></p>
            <Tag>{{'/' + ((downfile_path.length > 0) ? (downfile_path.join('/') + '/') : '')}}</Tag>
            <Table :columns="filelist_title" height="400" :data="filelist" @on-row-dblclick="filelistDblclick"></Table>
            <div slot="footer"></div>
        </Modal>
        <Modal v-model="downfile_modal" width="360" :mask-closable="false" @on-cancel="cancelDownfile">
            <p slot="header"><span>Download file from device</span></p>
            <Progress :percent="Math.round(downfile_info.received / downfile_info.size * 100)"></Progress>
            <div slot="footer">
                <Button type="primary" size="large" long</Button>
            </div>
        </Modal>
    </div>
</template>

<script>

import * as Socket from 'simple-websocket';
import { Terminal } from 'xterm'
import 'xterm/lib/xterm.css'
import * as fit from 'xterm/lib/addons/fit/fit';
import axios from 'axios'
import * as rtty from './rtty'

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
            downfile_modal: false,
            downfile_path: [],
            downfile_info: {},
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
            devId: '',
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
                                    this.devId = params.row.id;
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
                        if (params.row.dir)
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
                    sortable: true,
                    render: (h, params) => {
                        if (params.row.mtim)
                            return new Date(params.row.mtim * 1000).toLocaleString();
                    }
                }
            ],
            filelist: []
        }
    },

    methods: {
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

                let pkt = rtty.newPacket(rtty.RTTY_PACKET_UPFILE, {sid: this.sid, code: 1, data: fr.result});
                this.ws.send(pkt);

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

            let pkt = rtty.newPacket(rtty.RTTY_PACKET_UPFILE, {sid: this.sid, name: this.file.name, size: this.file.size, code: 0});
            this.ws.send(pkt);
            this.readFile(fr);
        },
        cancelUpfile() {
            if (!this.modal_loading)
                return;
            this.cancel_upfile = true;
            this.$Message.info("Upload canceled");

            let pkt = rtty.newPacket(rtty.RTTY_PACKET_UPFILE, {sid: this.sid, code: 2});
            this.ws.send(pkt);
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
            let attr = {sid: this.sid};

            if (row.dir) {
                if (row.name == '..')
                    this.downfile_path.pop();
                else
                    this.downfile_path.push(row.name);

                attr.name = '/' + ((this.downfile_path.length > 0) ? (this.downfile_path.join('/') + '/') : '');
            } else {
                if (this.downfile_path.length > 0)
                    attr.name = '/' + this.downfile_path.join('/') + '/' + row.name;
                else
                    attr.name = '/' + row.name;

                this.downfile_info = {name: row.name, size: row.size, received: 0};
                this.filelist_modal = false;
                this.downfile_modal = true;
            }

            let pkt = rtty.newPacket(rtty.RTTY_PACKET_DOWNFILE, attr);
            this.ws.send(pkt);
        },
        downFile () {
            this.contextMenuVisible = false;
            this.filelist_modal = true;
            this.downfile_path = [];

            let pkt = rtty.newPacket(rtty.RTTY_PACKET_DOWNFILE, {sid: this.sid});
            this.ws.send(pkt);
        },
        cancelDownfile() {
            let pkt = rtty.newPacket(rtty.RTTY_PACKET_DOWNFILE, {sid: this.sid, code: 1});
            this.ws.send(pkt);
            this.$Message.info("Download canceled");
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

            var ws = new Socket(protocol + location.host + '/ws?devid=' + this.devId);
            ws.on('connect', () => {
                ws.on('data', (data) => {
                    let pkt = rtty.parsePacket(data);

                    if (pkt.typ == rtty.RTTY_PACKET_LOGINACK) {
                        this.terminal_loading = false;

                        if (pkt.code != 0) {
                             this.$Message.error('Device offline');
                            this.logout(null, term);
                            return;
                        }
                        this.ws = ws;
                        this.sid = pkt.sid;
                        term.on('data', (data) => {
                            let pkt = rtty.newPacket(rtty.RTTY_PACKET_TTY, {sid: this.sid, data: Buffer.from(data)});
                            ws.send(pkt);
                        });
                    } else if (pkt.typ == rtty.RTTY_PACKET_TTY) {
                        this.recvCnt++;
                        var data = pkt.data.toString();

                        if (this.recvCnt < 4) {
                            if (data.match('login:') && this.username != '') {
                                let pkt = rtty.newPacket(rtty.RTTY_PACKET_TTY, {sid: this.sid, data: this.username + '\n'});
                                ws.send(pkt);
                                return;
                            }

                            if (data.match('Password:') && this.password != '') {
                                let pkt = rtty.newPacket(rtty.RTTY_PACKET_TTY, {sid: this.sid, data: this.password + '\n'});
                                ws.send(pkt);
                                return;
                            }
                        }
                        term.write(data);
                    } else if (pkt.typ == rtty.RTTY_PACKET_DOWNFILE) {
                        let code = pkt.code;
                        if (code == 0)
                            this.filelist = JSON.parse(pkt.data.toString());
                        else if (code == 1) {
                            if (!this.downfile_info.data)
                                this.downfile_info.data = new Blob([pkt.data]);
                            else
                                this.downfile_info.data = new Blob([this.downfile_info.data, pkt.data]);
                            this.downfile_info.received += pkt.data.byteLength;
                        } else if (code == 2) {
                            let url = URL.createObjectURL(this.downfile_info.data);
                            let a = document.createElement('a');
                            a.download = this.downfile_info.name;
                            a.href = url;
                            a.click();
                            URL.revokeObjectURL(url);
                            this.downfile_modal = false;
                        }
                    }
                });

                ws.on('error', ()=> {
                    this.logout(null, term);
                });

                ws.on('close', ()=> {
                    this.logout(null, term);
                });
            })
        }
    },
    mounted() {
        var devId = this.getQueryString('id');
        var username = this.getQueryString('username');
        var password = this.getQueryString('password');

        if (username)
            this.username = username;
        if (password)
            this.password = password;

        if (devId) {
            this.terminal_loading = true;
            this.termOn = true;
            this.devId = devId;
            window.setTimeout(this.login, 200);
        }

        window.setInterval(() => {
            if (this.termOn)
                return;
            axios.get('/devs').then((res => {
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
