<template>
    <div ref="terminal" class="terminal-container"></div>
</template>

<script>

import { Terminal } from 'xterm'
import * as fit from 'xterm/lib/addons/fit/fit';
import 'xterm/lib/xterm.css'
import { Base64 } from 'js-base64';
import * as Socket from 'simple-websocket';

export default {
    name: 'Home',
    mounted: function () {
        Terminal.applyAddon(fit);
        var term;
        var sid;
        var container = this.$refs['terminal'];
        var socket = new Socket('ws://192.168.3.33:5912/ws/browser?did=qq')
        socket.on('connect', function () {
            term = new Terminal();
        })

        socket.on('data', function (data) {
            var resp = JSON.parse(data);
                    
            var type = resp.type
            if (type == 'login') {
                if (resp.err) {
                    //$('#msg').text(resp.err);
                    return;
                }
                sid = resp.sid;
                term.open(container);
                term.fit();

                term.on('data', function(data) {
                    socket.send(JSON.stringify({type: 'data', did: 'qq', sid: sid, data: Base64.encode(data)}));
                });
            } else if (type == 'data') {
                var data = Base64.decode(resp.data);
                term.write(data);
            } else if (type == 'logout') {
                socket.close();
            }
        });
    }
}
</script>

<style scoped>
    .terminal-container {
        height: 100%;
    }
</style>
