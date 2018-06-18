<template>
    <div id="rtty">
        <div ref="terminal" class="terminal" @contextmenu="$vuecontextmenu()"></div>
        <VueContextMenu :menulists="menulists" @contentmenu-click="contentmenuClick"></VueContextMenu>
        <Modal v-model="upfile.modal" width="380" :mask-closable="false" @on-cancel="cancelUpfile">
            <p slot="header"><span>{{ $t('Upload file to device') }}</span></p>
            <Upload v-if="!upfile.loading" :before-upload="beforeUpload" action="">
                <Button type="ghost" icon="Upload">{{ $t('Select the file to upload') }}</Button>
            </Upload>
            <Progress v-if="upfile.loading" :percent="upfile.percent"></Progress>
            <div v-if="upfile.file !== null">{{ $t('upfile-info', {name: upfile.file.name}) }}</div>
            <div slot="footer">
                <Button type="primary" size="large" long :loading="upfile.loading"
                @click="doUpload">{{ upfile.loading ? $t('Uploading') : $t('Click to upload') }}</Button>
            </div>
        </Modal>
        <Modal v-model="downfile.modal" width="700" :mask-closable="false" @on-cancel="cancelDownfile">
            <p slot="header"><span>{{ $t('Download file from device') }}</span></p>
            <Input v-if="!downfile.downing" v-model="downfile.filter" icon="search"
                @on-change="handleFilterDownFile" :placeholder="$t('Please enter the filter key...')">
                <span slot="prepend">{{ downfile.pathname }}</span>
            </Input>
            <Table :loading="downfile.loading" v-if="!downfile.downing" :columns="filelistTitle" height="400" :data="downfile.filelistFiltered" @on-row-dblclick="filelistDblclick"></Table>
            <Progress v-if="downfile.downing" :percent="downfile.percent"></Progress>
            <div slot="footer"></div>
        </Modal>
    </div>
</template>

<script>

import * as Socket from 'simple-websocket';
import { Terminal } from 'xterm'
import 'xterm/lib/xterm.css'
import * as fit from 'xterm/lib/addons/fit/fit';
import * as overlay from '@/overlay';
import Utf8ArrayToStr from '@/utf8array_str'

Terminal.applyAddon(fit);
Terminal.applyAddon(overlay);

const Pbf = require('pbf');
const rttyMsg = require('@/rtty.proto').rtty_message;

function rttyMsgInit(type, msg) {
    let pbf = new Pbf();

    msg.version = 2;
    msg.type = rttyMsg.Type[type].value;
    rttyMsg.write(msg, pbf);

    return pbf.finish();
}

export default {
    name: 'Rtty',
    data() {
        return {
            menulists: [
                {
                    name: 'upfile',
                    caption: this.$t('Upload file to device')
                },{
                    name: 'downfile',
                    caption: this.$t('Download file from device')
                },{
                    name: 'increasefontsize',
                    caption: this.$t('Increase font size')
                },{
                    name: 'decreasefontsize',
                    caption: this.$t('Decrease font size')
                }
            ],
            filelistTitle: [
                {
                    title: this.$t('Name'),
                    key: 'name',
                    render: (h, params) => {
                        if (params.row.dir)
                            return h('div', [
                                h('Icon', {props: {type: 'folder', color: '#FFE793', size: 20}}),
                                h('strong', ' ' + params.row.name)
                            ]);
                        else
                            return h('span', params.row.name);
                    }
                }, {
                    title: this.$t('Size'),
                    key: 'size',
                    sortable: true,
                    render: (h, params) => {
                        return h('span', params.row.size && '%1024mB'.format(params.row.size));
                    }
                }, {
                    title: this.$t('modification'),
                    key: 'mtime',
                    sortable: true,
                    render: (h, params) => {
                        if (params.row.mtim)
                            return h('span', new Date(params.row.mtim * 1000).toLocaleString());
                    }
                }
            ],
            upfile: {modal: false, file: null, step: 8192, pos: 0, percent: 0, loading: false},
            downfile: {modal: false, loading: true, pathname: '/', filelist: [], filelistFiltered: [], downing: false, percent: 0, filter: ''},
        }
    },
    methods: {
        logout() {
            if (this.ws) {
                this.ws.destroy();
                delete this.ws;
            }

            if (this.term) {
                this.term.destroy();
                delete this.term;
            }

            this.$router.push('/');
        },
        contentmenuClick(name) {
            let changeFontSize = 0;

            if (!this.term)
                return;

            if (name == 'upfile') {
                this.upfile = {modal: true, file: null, step: 8192, pos: 0, percent: 0, loading: false};
            } else if (name == 'downfile') {
                this.downfile = {modal: true, loading: true, path: [], pathname: '/', filelist: [], downing: false, percent: 0, filter: ''};
                let msg = rttyMsgInit('DOWNFILE', {sid: this.sid});
                this.ws.send(msg);
            } else if (name == 'increasefontsize') {
                changeFontSize = 1;
            } else if (name == 'decreasefontsize') {
                changeFontSize = -1;
            }

            window.setTimeout(() => {
                let size = this.term.getOption('fontSize');

                this.term.setOption('fontSize', size + changeFontSize);
                this.term.fit();
                this.term.focus();
                this.term.refresh();
            }, 50);
        },
        beforeUpload (file) {
            this.upfile.file = file;
            this.upfile.lf = true;
            return false;
        },
        readFile(fr) {
            var blob = this.upfile.file.slice(this.upfile.pos, this.upfile.pos + this.upfile.step);
            fr.readAsArrayBuffer(blob);
        },
        cancelUpfile() {
            if (!this.upfile.loading)
                return;
            this.upfile.canceled = true;
            this.$Message.info(this.$t('Upload canceled'));

            let msg = rttyMsgInit('UPFILE', {sid: this.sid, code: rttyMsg.FileCode.CANCELED.value});
            this.ws.send(msg);
        },
        doUpload () {
            if (!this.upfile.file) {
                this.$Message.error(this.$t('Select the file to upload'));
                return;
            }

            this.upfile.loading = true;
            
            var fr = new FileReader();
            fr.onload = (e) => {
                if (this.upfile.canceled)
                    return;

                let msg = rttyMsgInit('UPFILE', {sid: this.sid, code: rttyMsg.FileCode.FILEDATA.value, data: Buffer.from(fr.result)});
                this.ws.send(msg);

                this.upfile.pos += e.loaded;
                this.upfile.percent = Math.round(this.upfile.pos / this.upfile.file.size * 100);

                if (this.upfile.pos < this.upfile.file.size) {
                    /* Control the client read speed based on the current buffer and server */
                    if (this.ws.bufferedAmount > this.upfile.pos * 10 || this.upfile.ratelimit) {
                        this.upfile.ratelimit = false;

                        setTimeout(() => {
                            this.readFile(fr);
                        }, 100);
                    } else {
                        this.readFile(fr);
                    }
                } else {
                    this.upfile.modal = false;
                    this.$Message.info(this.$t('Upload success'));
                }
            };

            let msg = rttyMsgInit('UPFILE', {sid: this.sid, name: this.upfile.file.name, size: this.upfile.file.size, code: rttyMsg.FileCode.START.value});
            this.ws.send(msg);

            this.readFile(fr);
        },
        cancelDownfile() {
            if (this.downfile.downing) {
                let msg = rttyMsgInit('DOWNFILE', {sid: this.sid, code: rttyMsg.FileCode.CANCELED.value});
                this.ws.send(msg);

                this.$Message.info(this.$t('Download canceled'));
            }
        },
        handleFilterDownFile() {
            this.downfile.filelistFiltered = this.downfile.filelist.filter(d => {
                return d.name.indexOf(this.downfile.filter) > -1;
            });
        },
        filelistDblclick(row, index) {
            let attr = {sid: this.sid};

            this.downfile.filter = '';

            if (row.name == '..') {
                if (this.downfile.path.length < 1)
                    return;
                this.downfile.path.pop();
            } else {
                this.downfile.path.push(row.name);
            }

            this.downfile.pathname = '/' + this.downfile.path.join('/');

            if (row.dir) {
                this.downfile.loading = true;
                if (!this.downfile.pathname.endsWith('/'))
                    this.downfile.pathname = this.downfile.pathname + '/';
            } else {
                this.downfile.received = 0;
                this.downfile.size = row.size;
                this.downfile.downing = true;
            }

            attr.name = this.downfile.pathname;

            let msg = rttyMsgInit('DOWNFILE', attr);
            this.ws.send(msg);
        }
    },
    mounted() {
        let devid = this.$route.query.devid;
        let protocol = (location.protocol === 'https:') ? 'wss://' : 'ws://';

        this.username = this.$route.query.username;
        this.password = this.$route.query.password;

        let ws = new Socket(protocol + location.host + '/ws?devid=' + devid);
        this.ws = ws;

        ws.on('connect', () => {
            let term = new Terminal({
                cursorBlink: true,
                fontSize: 16
            });

            term.open(this.$refs['terminal']);
            term.fit();
            term.focus();
            term.showOverlay(term.cols + 'x' + term.rows);

            window.addEventListener('resize', () => {
                clearTimeout(window.resizedFinished);
                window.resizedFinished = setTimeout(() => {
                    term.fit();
                }, 250);
            });

            term.on('resize', (size) => {
                setTimeout(() => {
                    let msg = rttyMsgInit('WINSIZE', {sid: this.sid, cols: size.cols, rows: size.rows});
                    ws.send(msg);
                    term.showOverlay(size.cols + 'x' + size.rows);
                }, 500);
            });

            this.term = term;

            ws.on('data', (data) => {
                let pbf = new Pbf(data);
                let msg = rttyMsg.read(pbf);

                if (msg.type == rttyMsg.Type.LOGINACK.value) {
                    if (msg.code == rttyMsg.LoginCode.OFFLINE.value) {
                        this.$Message.error(this.$t('Device offline'));
                        this.logout();
                        return;
                    }

                    this.sid = msg.sid;

                    msg = rttyMsgInit('WINSIZE', {sid: this.sid, cols: term.cols, rows: term.rows});
                    ws.send(msg);

                    term.on('data', (data) => {
                        let msg = rttyMsgInit('TTY', {sid: this.sid, data: Buffer.from(data)});
                        ws.send(msg);
                    });
                } else if (msg.type == rttyMsg.Type.TTY.value) {
                    let data = Utf8ArrayToStr(msg.data);

                    if (!this.recvTTYCnt)
                        this.recvTTYCnt = 0;
                    this.recvTTYCnt++;

                    if (this.recvTTYCnt < 4) {
                        if (data.match('login:') && this.username && this.username != '') {
                            let msg = rttyMsgInit('TTY', {sid: this.sid, data: Buffer.from(this.username + '\n')});
                            ws.send(msg);
                            return;
                        }

                        if (data.match('Password:') && this.password && this.password != '') {
                            let msg = rttyMsgInit('TTY', {sid: this.sid, data: Buffer.from(this.password + '\n')});
                            ws.send(msg);
                            return;
                        }
                    }

                    term.write(data);
                } else if (msg.type == rttyMsg.Type.UPFILE.value) {
                    if (msg.code == rttyMsg.FileCode.RATELIMIT.value) {
                        /* Need reduce the sending rate */
                        this.upfile.ratelimit = true;
                    }
                } else if (msg.type == rttyMsg.Type.DOWNFILE.value) {
                    let code = msg.code;
                    if (code == rttyMsg.FileCode.START.value) {
                        this.downfile.loading = false;
                        this.downfile.filelist = msg.filelist;
                        this.handleFilterDownFile();
                    }
                    else if (code == rttyMsg.FileCode.FILEDATA.value) {
                        if (!this.downfile.data)
                            this.downfile.data = new Blob([msg.data]);
                        else
                            this.downfile.data = new Blob([this.downfile.data, msg.data]);
                        this.downfile.received += msg.data.byteLength;
                        this.downfile.percent = Math.round(this.downfile.received / this.downfile.size * 100);
                    } else if (code == rttyMsg.FileCode.END.value) {
                        let url = URL.createObjectURL(this.downfile.data);
                        let a = document.createElement('a');
                        a.download = this.downfile.pathname;
                        a.href = url;
                        a.click();
                        URL.revokeObjectURL(url);
                        this.downfile.modal = false;
                        this.downfile.downing = false;
                        this.$Message.info(this.$t('Download Finish'));
                    }
                }
            });
        });

        ws.on('error', () => {
            this.$Message.error(this.$t('Connect failed'));
            this.logout();
        });

        ws.on('close', () => {
            this.logout();
        });
    }
}
</script>

<style>
	#rtty {
	    width: 100%;
	    height: 100%;
    }

    .terminal {
        height: 100%;
        padding: 10px;
    }
</style>
