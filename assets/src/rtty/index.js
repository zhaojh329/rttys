export const RTTY_PROTOCOL_VERSION = 1

export const RTTY_PACKET_LOGIN      = 1
export const RTTY_PACKET_LOGINACK   = 2
export const RTTY_PACKET_LOGOUT     = 3
export const RTTY_PACKET_TTY        = 4
export const RTTY_PACKET_ANNOUNCE   = 5
export const RTTY_PACKET_UPFILE     = 6

export const RTTY_ATTR_SID          = 1
export const RTTY_ATTR_CODE         = 2
export const RTTY_ATTR_DATA         = 3
export const RTTY_ATTR_NAME         = 4
export const RTTY_ATTR_SIZE         = 5

export let parsePacket = function(buf) {
    let pkt = { };

    pkt.version = buf.readUInt8(0);
    pkt.typ = buf.readUInt8(1);

    let i = 2;
    while (i < buf.length) {
        let typ = buf.readUInt8(i);
        let length = buf.readUInt16BE(i + 1);

        i += 3;
        if (typ == RTTY_ATTR_SID)
            pkt.sid = buf.toString('utf8', i, i + length - 1);
        else if (typ == RTTY_ATTR_CODE)
            pkt.code = buf.readUInt8(i);
        else if (typ == RTTY_ATTR_DATA)
            pkt.data = buf.slice(i, i + length);

        i += length;
    }

    return pkt
}

function addAttr(buf, typ, data) {
    let tmp, length

    if (typeof data == 'string') {
        length = data.length + 1;
        tmp = Buffer.alloc(3 + length);
    } else {
        length = data.byteLength;
        tmp = Buffer.alloc(3);
    }

    tmp.writeUInt8(typ, 0);
    tmp.writeUInt16BE(length, 1);

    if (typeof data == 'string') {
        tmp.write(data, 3);
        return Buffer.concat([buf, tmp]);
    } else {
        return Buffer.concat([buf, tmp, Buffer.from(data)]);
    }
}

function addAttrU8(buf, typ, val) {
    let tmp = Buffer.alloc(3 + 1);
    tmp.writeUInt8(typ, 0);
    tmp.writeUInt16BE(1, 1);
    tmp.writeUInt8(val, 3);
    return Buffer.concat([buf, tmp]);
}

function addAttrU32(buf, typ, val) {
    let tmp = Buffer.alloc(3 + 4);
    tmp.writeUInt8(typ, 0);
    tmp.writeUInt16BE(4, 1);
    tmp.writeUInt32BE(val, 3);
    return Buffer.concat([buf, tmp]);
}

export let newPacket = function(typ, attr) {
	let buf = Buffer.from([RTTY_PROTOCOL_VERSION, typ]);

    if (attr.sid)
        buf = addAttr(buf, RTTY_ATTR_SID, attr.sid)

    if (attr.code)
        buf = addAttrU8(buf, RTTY_ATTR_CODE, attr.code)

    if (attr.data)
        buf = addAttr(buf, RTTY_ATTR_DATA, attr.data)

    if (attr.name)
        buf = addAttr(buf, RTTY_ATTR_NAME, attr.name)

    if (attr.size)
        buf = addAttrU32(buf, RTTY_ATTR_SIZE, attr.size)

	return buf;
}