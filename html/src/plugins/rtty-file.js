const blk_size = 8912;         /* 8KB */

function RttyFile(ws, term, opt) {
    this.state = '';
    this.ws = ws;
    this.term = term;
    this.cache = [];
    this.buffer = [];

    this.to_term = function(octets) {
        this.term.write(Buffer.from(octets).toString());
    }

    this.detect = function(input) {
        let type = '';

        if (input.byteLength < 3)
            return '';

        input = new Uint8Array(input);

        let pos = input.indexOf(0xB6);
        if (pos < 0)
            return '';
        
        if (pos > input.length - 3)
            return '';
    
        if (input[pos + 1] != 0xBC)
            return '';
        
        type = String.fromCharCode(input[pos + 2]);
        if (type != 's' && type != 'r')
            return '';
    
        if (pos > 0)
            this.to_term(input.slice(0, pos));
    
        if (type == 's')
            this.cache = Array.prototype.slice.call(input.slice(pos + 3));
        else
            this.to_term(input.slice(pos + 3));

        this.start_ts = new Date().getTime() / 1000;
        return type;
    }

    this.consume = function(input) {
        if (this.state == '') {
            let t = this.detect(input);
            if (t == '') {
                this.to_term(Buffer.from(input));
                return;
            }
            if (t == 'r') {
                this.state = 'send_pending';
                opt.on_detect('r');
            } else if (t == 's') {
                this.state = 'recving';
                opt.on_detect('s');
            }
        } else if (this.state == 'recving' || this.state == 'abort_recv') {
            this.recvFile(input);
        } else if (this.state == 'sending') {
            input = new Uint8Array(input);
            if (input.length == 3) {
                if (input[0] == 0xB6 && input[1] == 0xBC && String.fromCharCode(input[2]) == 'e') {
                    this.state = 'abort';
                    return;
                }
            }
            this.to_term(Buffer.from(input));
        } else {
            this.to_term(Buffer.from(input));
        }
    }

    
    this.readFile = function(offset, size) {
        let blob = this.file.slice(offset, offset + size);
        this.fr.readAsArrayBuffer(blob);
    }

    this.sendEof = function() {
        let b = new Uint8Array([0x03]);
        this.ws.send(b);
        this.state = '';
    }

    this.abort = function() {
        this.state = 'abort';
    }

    this.abortRecv = function() {
        let b = new Uint8Array([0x03]);
        this.ws.send(b);
        this.state = 'abort_recv';
    }

    this.sendInfo = function(file) {
        let b = Buffer.alloc(6 + file.name.length);
    
        b[0] = 0x01;    /* packet type: file info */
        b[1] = file.name.length;
        b.write(file.name, 2);
        b.writeUInt32BE(file.size, 2 + file.name.length);
        this.ws.send(b);
        this.file = file;
    }

    this.sendData = function(data) {
        let b = Buffer.alloc(3);
        let piece = new Uint8Array(data);
    
        b[0] = 0x02;    /* packet type: file data */
        b.writeUInt16BE(piece.length, 1);
        this.ws.send(b);
        this.ws.send(piece);
    }

    this.sendFile = function(file) {
        this.fr = new FileReader();
    
        this.state = 'sending';
        let offset = 0;
    
        this.sendInfo(file);
    
        this.fr.onload = (e) => {
            
            this.sendData(e.target.result);
    
            offset += e.loaded;
    
            if (this.state != 'abort' && offset < file.size) {
                this.readFile(offset, blk_size);
                return;
            }
    
            this.sendEof();
        };
    
        this.readFile(offset, blk_size);
    }

    this.recvFile = function(input) {
        input = Array.prototype.slice.call(new Uint8Array(input));
        this.cache.push.apply(this.cache, input);
    
        while (this.cache.length > 0) {
            let type = this.cache[0];

            switch (type) {
            case 0x01:  /* file info */ {
                if (this.cache.length < 2)
                    return;
                let nl = this.cache[1];
                if (this.cache.length < nl + 2)
                    return;
                this.cache.splice(0, 2);
    
                this.name = Buffer.from(this.cache.splice(0, nl)).toString();
                this.size = Buffer.from(this.cache.splice(0, 4)).readUInt32BE(0);
                this.offset = 0;
                this.buffer = [];
                break;
            }
            case 0x02:  /* file data */ {
                if (this.cache.length < 3)
                    return;

                let dl = Buffer.from(this.cache.slice(1,3)).readUInt16BE(0);
                if (this.cache.length < dl + 3)
                    return;

                this.cache.splice(0, 3);
                this.buffer.push(new Uint8Array(this.cache.splice(0, dl)));
                this.offset += dl;

                let now_ts = new Date().getTime() / 1000;

                let unit = 'K';
                let offset = this.offset / 1024;

                if (offset / 1024 > 0) {
                    offset /= 1024;
                    unit = 'M';
                }
                this.term.write('  %d%%    %.2f %sB     %.3fs\r'.format(this.offset / this.size * 100, offset, unit, now_ts - this.start_ts));
                break;
            }
            case 0x03:  /* file eof */ {
                this.cache = [];
    
                this.term.write('\n');

                if (this.state == 'abort_recv') {
                    this.state = '';
                    return;
                }

                this.state = '';
    
                let blob = new Blob(this.buffer);
                let url = URL.createObjectURL(blob);
    
                let el = document.createElement("a");
                el.style.display = "none";
                el.href = url;
                el.download = this.name;
                document.body.appendChild(el);
                el.click();
                document.body.removeChild(el);
                break;
            }
            default:
                // console.error('invalid type:' + type);
                return;
            }
        }
    }
}

export default RttyFile;
