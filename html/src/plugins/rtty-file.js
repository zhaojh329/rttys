const MsgTypeFileStartDownload = 0x00;
const MsgTypeFileInfo = 0x01;
const MsgTypeFileData = 0x02;
const MsgTypeFileCanceled = 0x03;
const blk_size = 4096;         /* 4KB */

function RttyFile(wsSend, on_download) {
    this.buffer = [];
    this.canceled = false;

    this.recvFileMsg = function(data) {
        let type = data[0];

        data = data.slice(1);

        switch (type) {
        case MsgTypeFileStartDownload:
            on_download();
            break;

        case MsgTypeFileInfo:
            this.name = data.toString();
            this.buffer = [];
            break;

        case MsgTypeFileData:
            if (data.length === 0) {
                let blob = new Blob(this.buffer);
                let url = URL.createObjectURL(blob);
                let el = document.createElement('a');
                el.style.display = 'none';
                el.href = url;
                el.download = this.name;
                document.body.appendChild(el);
                el.click();
                document.body.removeChild(el);
                this.buffer = [];
            } else {
                this.buffer.push(data);
            }
            break;

        case MsgTypeFileCanceled:
            this.buffer = [];
            this.canceled = true;
            break;
        }

        return true;
    }

    this.cancel = function() {
        let b = Buffer.from([1, MsgTypeFileCanceled]);
        wsSend(b);
    }

    this.sendInfo = function() {
        let buf = [];

        buf.push(Buffer.from([1, MsgTypeFileInfo]));

        let sb = new Buffer(4);
        sb.writeUInt32BE(this.file.size, 0);
        buf.push(sb);

        buf.push(Buffer.from(this.file.name));

        wsSend(Buffer.concat(buf));
    }

    this.sendData = function(data) {
        let b = Buffer.concat([Buffer.from([1, MsgTypeFileData]), Buffer.from(data)]);
        wsSend(b);
    }

    this.readFile = function(offset, size) {
        let blob = this.file.slice(offset, offset + size);
        this.fr.readAsArrayBuffer(blob);
    }

    this.sendFile = function(file) {
        this.fr = new FileReader();
        this.file = file;
        this.canceled = false;

        this.sendInfo();

        let offset = 0;

        this.fr.onload = e => {

            this.sendData(e.target.result);

            offset += e.loaded;

            if (this.canceled)
                return;

            if (offset < file.size) {
                this.readFile(offset, blk_size);
                return;
            }

            this.sendData([]);
        };

        this.readFile(offset, blk_size);
    }
}

export default RttyFile;
