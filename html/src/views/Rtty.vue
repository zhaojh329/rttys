<template>
    <div style="height: 100%; padding: 5px">
        <div ref="terminal" style="height: 100%"></div>
        <Modal v-model="upfile.modal" width="380" :mask-closable="false" @on-cancel="cancelUpfile">
            <p slot="header"><span>{{ $t('Upload file to device') }}</span></p>
            <Upload :before-upload="beforeUpload" action="">
                <Button type="primary">{{ $t('Please select the file to upload') }}</Button>
            </Upload>
            <div v-if="upfile.file !== null">{{ upfile.file.name }}</div>
            <div slot="footer">
                <Button type="primary" size="large" long @click="doUpload">{{ $t('Click to upload') }}</Button>
            </div>
        </Modal>
    </div>
</template>

<script>

import { Terminal } from 'xterm'
import 'xterm/lib/xterm.css'
import * as fit from 'xterm/lib/addons/fit/fit'
import * as overlay from '../plugins/overlay'
import RttyFile from '../plugins/rtty-file'

Terminal.applyAddon(fit);
Terminal.applyAddon(overlay);

const LoginErrorOffline = 0x01;
const LoginErrorBusy = 0x02;

export default {
    name: 'Rtty',
    data() {
        return {
            upfile: {modal: false, file: null},
        }
    },
    methods: {
        logout() {
            if (this.ws) {
                this.ws.close();
                delete this.ws;
            }

            if (this.term) {
                this.term.destroy();
                delete this.term;
            }

            this.$router.push('/');
        },
        beforeUpload (file) {
            if (file.size > 500 * 1024 * 1024) {
                this.$Message.warning(this.$t('Cannot be greater than 500MB'));
                return false;
            }

            if (file.name.length > 255) {
                this.$Message.warning(this.$t('The file name too long'));
                return false;
            }

            this.upfile.file = file;
            return false;
        },
        cancelUpfile() {
            this.term.focus();
            this.rf.cancel();
        },
        doUpload() {
            if (!this.upfile.file) {
                this.$Message.error(this.$t('Select the file to upload'));
                return;
            }

            this.upfile.modal = false;
            this.term.focus();
            this.rf.sendFile(this.upfile.file);
        },
        wsSendData(type, data) {
            this.ws.send(Buffer.concat([Buffer.from([type]), Buffer.from(data)]));
        }
    },
    mounted() {
        let devid = this.$route.query.devid;
        let protocol = (location.protocol === 'https:') ? 'wss://' : 'ws://';

        this.username = this.$route.query.username;
        this.password = this.$route.query.password;

        let ws = new WebSocket(protocol + location.host + process.env.BASE_URL + 'ws?devid=' + devid + '&sid=' + sessionStorage.getItem('rtty-sid'));

        ws.onopen = () => {
            ws.binaryType = 'arraybuffer';
            this.ws = ws;

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
                    let msg = {type: "winsize", cols: size.cols, rows: size.rows};
                    ws.send(JSON.stringify(msg));
                    term.showOverlay(size.cols + 'x' + size.rows);
                }, 500);
            });

            this.term = term;
            this.rf = new RttyFile(data => {
                this.ws.send(data);
            }, () => {
                this.upfile.modal = true;
            });
        };

        ws.onmessage = (ev) => {
            let term = this.term;

            if (typeof ev.data === 'string') {
                let msg = JSON.parse(ev.data);
                if (msg.type === "login") {
                    if (msg.err === LoginErrorOffline) {
                        this.$Message.error(this.$t('Device offline'));
                        this.logout();
                        return;
                    } else if (msg.err === LoginErrorBusy) {
                        this.$Message.error(this.$t('Sessions is full'));
                        this.logout();
                        return;
                    }

                    msg = {type: 'winsize', cols: term.cols, rows: term.rows};
                    ws.send(JSON.stringify(msg));

                    term.on('data', (data) => {
                        this.wsSendData(0, data);
                    });
                } else if (msg.type === 'logout') {
                    this.logout();
                }
            } else {
                let data = Buffer.from(ev.data);
                let isFileMsg = data[0] === 1;

                if (isFileMsg) {
                    this.rf.recvFileMsg(data.slice(1));
                    return;
                }

                data = data.slice(1).toString();

                if (!this.recvTTYCnt)
                    this.recvTTYCnt = 0;
                this.recvTTYCnt++;

                if (this.recvTTYCnt < 4) {
                    if (data.match('login:') && this.username && this.username !== '') {
                        this.wsSendData(0, this.username + '\n');
                        return;
                    }

                    if (data.match('Password:') && this.password && this.password !== '') {
                        this.wsSendData(0, this.password + '\n');
                        return;
                    }
                }

                term.write(data);
            }
        };

        ws.onerror = () => {
            this.$Message.error(this.$t('Connect failed'));
            this.logout();
        };

        ws.onclose = () => {
            this.logout();
        };
    }
};
</script>

<style>
.xterm .xterm-viewport {
    overflow: auto;
}
</style>
