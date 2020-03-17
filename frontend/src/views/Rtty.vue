<template>
  <div>
    <div ref="terminal" :style="{height: termHeight + 'px'}" @contextmenu.prevent="showContextmenu"/>
    <el-dialog ref="dialog" :visible.sync="file.modal" :title="$t('Upload file to device')" width="350px"
               @close="onUploadDialogClosed">
      <el-upload ref="upload" action="" :auto-upload="false" :http-request="doUploadFile">
        <el-button slot="trigger" size="small" type="primary">{{ $t("Select file") }}</el-button>
        <el-button style="margin-left: 10px;" size="small" type="success" @click="submitUploadFile">{{ $t('Upload') }}
        </el-button>
      </el-upload>
    </el-dialog>
    <contextmenu ref="contextmenu" :menus="contextmenus" @click="onContextmenuClick"/>
  </div>
</template>

<script lang="ts">
  import {Component, Prop, Vue} from 'vue-property-decorator'
  import {HttpRequestOptions} from 'element-ui/types/upload'
  import {Upload as ElUpload} from 'element-ui'
  import {Terminal, IDisposable} from 'xterm'
  import {FitAddon} from 'xterm-addon-fit'
  import {OverlayAddon} from '@/plugins/xterm-addon-overlay'
  import 'xterm/css/xterm.css'
  import {Contextmenu} from '@/components/contextmenu'
  import ClipboardEx from '@/plugins/clipboard'

  const LoginErrorOffline = 0x01;
  const LoginErrorBusy = 0x02;
  const MsgTypeFileStartDownload = 0x00;
  const MsgTypeFileInfo = 0x01;
  const MsgTypeFileData = 0x02;
  const MsgTypeFileCanceled = 0x03;

  @Component
  export default class Rtty extends Vue {
    @Prop(String) devid!: string;
    disposables: IDisposable[] = [];
    file = {
      name: '',
      modal: false,
      recving: false,
      accepted: false,
      buffer: [] as Uint8Array[]
    };
    socket: WebSocket | undefined;
    term: Terminal | undefined;
    fitAddon: FitAddon | undefined;
    resizeDelay: NodeJS.Timeout | undefined;
    termHeight = 0;
    contextmenus = [
      {name: 'copy', caption: this.tr('Copy - Ctrl+Insert')},
      {name: 'paste', caption: this.tr('Paste - Shift+Insert')},
      {name: 'Clear Scrollback', caption: this.tr('Clear Scrollback')},
      {name: 'Font Size+', caption: this.tr('Font Size+')},
      {name: 'Font Size-', caption: this.tr('Font Size-')}
    ];

    tr(key: string): string {
      if (this.$t)
        return this.$t(key).toString();
      return '';
    }

    showContextmenu(e: MouseEvent) {
      (this.$refs.contextmenu as Contextmenu).show(e);
    }

    onContextmenuClick(name: string) {
      if (name === 'copy') {
        ClipboardEx.write(this.term?.getSelection() || '');
      } else if (name === 'paste') {
        ClipboardEx.read().then(text => this.term?.paste(text));
      } else if (name === 'Clear Scrollback') {
        this.term?.clear();
      } else if (name === 'Font Size+') {
        const size = this.term?.getOption('fontSize');
        if (size)
          this.updateFontSize(size + 1);
      } else if (name === 'Font Size-') {
        const size = this.term?.getOption('fontSize');
        if (size && size > 12)
          this.updateFontSize(size - 1);
      }
      this.term?.focus();
    }

    updateFontSize(size: number) {
      this.term?.setOption('fontSize', size);
      this.fitAddon?.fit();
      this.axios.post(`/fontsize/${this.devid}`, {size});
    }

    onUploadDialogClosed() {
      this.term?.focus();
      if (this.file.accepted)
        return;
      this.sendFileData(MsgTypeFileCanceled, null);
    }

    submitUploadFile() {
      (this.$refs.upload as ElUpload).submit();
    }

    sendFileInfo(file: File) {
      const b = Buffer.alloc(4 + file.name.length);
      b.writeUInt32BE(file.size, 0);
      b.write(file.name, 4);
      this.sendFileData(MsgTypeFileInfo, b);
    }

    readFileBlob(fr: FileReader, file: File, offset: number, size: number) {
      const blob = file.slice(offset, offset + size);
      fr.readAsArrayBuffer(blob);
    }

    doUploadFile(options: HttpRequestOptions) {
      this.file.accepted = true;
      this.file.modal = false;

      this.sendFileInfo(options.file);

      const fr = new FileReader();
      let offset = 0;

      fr.onload = e => {
        if (!this.file.recving)
          return;
        offset += e.loaded;

        this.sendFileData(MsgTypeFileData, Buffer.from(fr.result as ArrayBuffer));

        if (offset < options.file.size) {
          this.readFileBlob(fr, options.file, offset, 4096);
          return;
        }
        this.sendFileData(MsgTypeFileData, null);
      };
      this.readFileBlob(fr, options.file, offset, 4096);
    }

    sendTermData(data: string): void {
      this.sendData(Buffer.concat([Buffer.from([0]), Buffer.from(data)]));
    }

    sendFileData(type: number, data: Uint8Array | null): void {
      const buf = [];
      buf.push(Buffer.from([1, type]));
      if (data !== null)
        buf.push(Buffer.from(data));
      this.sendData(Buffer.concat(buf));
    }

    sendData(data: Uint8Array) {
      const socket = this.socket as WebSocket;
      if (!socket)
        return;
      if (socket.readyState !== 1)
        return;
      socket.send(data);
    }

    parseFileMsg(msg: Uint8Array) {
      const type = msg[0];

      msg = msg.slice(1);

      switch (type) {
        case MsgTypeFileStartDownload:
          this.file.modal = true;
          this.file.recving = true;
          this.file.accepted = false;
          this.term?.blur();
          break;
        case MsgTypeFileInfo:
          this.file.name = msg.toString();
          this.file.buffer = [];
          break;
        case MsgTypeFileData:
          if (msg.length === 0) {
            const blob = new Blob(this.file.buffer, {type : 'application/octet-stream'});
            const url = URL.createObjectURL(blob);
            const el = document.createElement('a');
            el.style.display = 'none';
            el.href = url;
            el.download = this.file.name;
            document.body.appendChild(el);
            el.click();
            document.body.removeChild(el);
            this.file.buffer = [];
          } else {
            this.file.buffer.push(msg);
          }
          break;
        case MsgTypeFileCanceled:
          this.file.buffer = [];
          this.file.recving = false;
          break;
      }
    }

    fitTerm() {
      this.termHeight = document.documentElement.clientHeight - 10;

      this.$nextTick(() => {
        if (this.resizeDelay)
          clearTimeout(this.resizeDelay);
        this.resizeDelay = setTimeout(() => {
          this.fitAddon?.fit();
        }, 200);
      });
    }

    mounted() {
      const protocol = (location.protocol === 'https:') ? 'wss://' : 'ws://';

      const term = new Terminal({
        cursorBlink: true,
        fontSize: 16
      });
      this.disposables.push({dispose: () => term.dispose()});
      this.term = term;

      const fitAddon = new FitAddon();
      this.fitAddon = fitAddon;
      term.loadAddon(fitAddon);

      const overlayAddon = new OverlayAddon();
      term.loadAddon(overlayAddon);

      term.open(this.$refs['terminal'] as HTMLElement);
      term.focus();

      window.addEventListener('resize', this.fitTerm);

      const socket = new WebSocket(protocol + location.host + `/connect/${this.devid}`);
      this.disposables.push({dispose: () => socket.close()});
      socket.binaryType = 'arraybuffer';
      this.socket = socket;

      socket.addEventListener('open', () => {
        this.axios.get(`/fontsize/${this.devid}`).then(r => {
          this.term?.setOption('fontSize', r.data.size);
          this.fitTerm();
        });
      });

      socket.addEventListener('close', () => this.dispose());
      socket.addEventListener('error', () => this.dispose());

      socket.addEventListener('message', ev => {
        const data: ArrayBuffer | string = ev.data;

        if (typeof data === 'string') {
          const msg = JSON.parse(data);
          if (msg.type === 'login') {
            if (msg.err === LoginErrorOffline) {
              this.$message.error(this.$t('Device offline').toString());
              this.dispose();
              return;
            } else if (msg.err === LoginErrorBusy) {
              this.$message.error(this.$t('Sessions is full').toString());
              this.dispose();
              return;
            }
          } else if (msg.type === 'logout') {
            this.dispose();
          }
        } else {
          let data = Buffer.from(ev.data);
          const isFileMsg = data[0] === 1;

          data = data.slice(1);

          if (isFileMsg) {
            this.parseFileMsg(data);
            return;
          }

          term.write(data.toString());
        }
      });

      term.onData(data => {
        this.sendTermData(data);
      });

      term.onBinary(data => {
        this.sendTermData(data);
      });

      term.onResize(size => {
        const msg = {type: 'winsize', cols: size.cols, rows: size.rows};
        socket.send(JSON.stringify(msg));
        overlayAddon.show(term.cols + 'x' + term.rows);
      });
    }

    dispose() {
      this.$router.push('/');
    }

    destroyed() {
      window.removeEventListener('resize', this.fitTerm);
      this.disposables.forEach(d => d.dispose());
      this.socket = undefined;
      this.term = undefined;
    }
  }
</script>

<style>
  .xterm .xterm-viewport {
    overflow: auto;
  }
</style>
