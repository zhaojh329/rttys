/*
 * Copyright (C) 2017 Jianhui Zhao <jianhuizhao329@gmail.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
    "log"
    "bytes"
    "encoding/binary"
)

const RTTY_PROTOCOL_VERSION = 1

const (
    _ = iota
    RTTY_PACKET_LOGIN
    RTTY_PACKET_LOGINACK
    RTTY_PACKET_LOGOUT
    RTTY_PACKET_TTY
    RTTY_PACKET_ANNOUNCE
    RTTY_PACKET_UPFILE
)

const (
    _ = iota
    RTTY_ATTR_SID
    RTTY_ATTR_CODE
    RTTY_ATTR_DATA
    RTTY_ATTR_NAME
    RTTY_ATTR_SIZE
)

type RttyPacketInfo struct {
    version byte
    typ byte
    sid string
    code byte
    description string
    data []byte
    name string
    size uint32
}

type RttyPacket struct {
    w *bytes.Buffer
}

func (pkt *RttyPacket)Put(typ byte, length uint16, data []byte) {
    pkt.w.WriteByte(typ)
    binary.Write(pkt.w, binary.BigEndian, length)
    pkt.w.Write(data)
}

func (pkt *RttyPacket)PutU8(typ byte, val byte) {
    pkt.w.WriteByte(typ)
    binary.Write(pkt.w, binary.BigEndian, uint16(1))
    binary.Write(pkt.w, binary.BigEndian, val)
}

func (pkt *RttyPacket)PutU16(typ byte, val uint16) {
    pkt.w.WriteByte(typ)
    binary.Write(pkt.w, binary.BigEndian, uint16(2))
    binary.Write(pkt.w, binary.BigEndian, val)
}

func (pkt *RttyPacket)PutU32(typ byte, val uint32) {
    pkt.w.WriteByte(typ)
    binary.Write(pkt.w, binary.BigEndian, uint16(4))
    binary.Write(pkt.w, binary.BigEndian, val)
}

func (pkt *RttyPacket)PutString(typ byte, str string) {
    b := []byte(str)
    b = append(b, 0)
    pkt.Put(typ, uint16(len(str) + 1), b)
}

func (pkt *RttyPacket)Bytes() []byte {
    return pkt.w.Bytes()
}

func (pkt *RttyPacket)Init(typ byte) {
    pkt.w.Reset()
    pkt.w.WriteByte(RTTY_PROTOCOL_VERSION)
    pkt.w.WriteByte(typ)
}

func (pkt *RttyPacketInfo)Dump() {
    log.Println("version: ", pkt.version)
    log.Println("type: ", pkt.typ)
    log.Println("sid: ", pkt.sid)
    log.Println("code: ", pkt.code)

    if pkt.typ == RTTY_PACKET_UPFILE {
        log.Println("size: ", pkt.size)
        log.Println("name: ", pkt.name)
        log.Println("data: ", pkt.data)
    }

    if pkt.typ == RTTY_PACKET_TTY {
        log.Println("data: ", string(pkt.data))
    }
}

func rttyPacketNew(typ byte) *RttyPacket {
    pkt := new(RttyPacket)
    pkt.w = new(bytes.Buffer)
    pkt.w.WriteByte(RTTY_PROTOCOL_VERSION)
    pkt.w.WriteByte(typ)
    return pkt
}

func rttyPacketGetString(b *bytes.Buffer, length uint16) string {
    data := make([]byte, length - 1)
    b.Read(data)
    b.ReadByte()
    return string(data)
}

func rttyPacketParse(data []byte) *RttyPacketInfo {
    info := new(RttyPacketInfo)

    info.version = data[0]
    info.typ = data[1]

    b := bytes.NewBuffer(data[2:])

    for {
        var typ byte
        var length uint16
        var err error

        typ, err = b.ReadByte()
        if err != nil {
            break
        }

        err = binary.Read(b, binary.BigEndian, &length)
        if err != nil {
            break
        }

        if typ == RTTY_ATTR_SID {
            info.sid = rttyPacketGetString(b, length)
        } else if typ == RTTY_ATTR_DATA {
            info.data = make([]byte, length)
            b.Read(info.data)
        } else if typ == RTTY_ATTR_CODE {
            info.code, _ = b.ReadByte()
        } else if typ == RTTY_ATTR_NAME {
            info.name = rttyPacketGetString(b, length)
        } else if typ == RTTY_ATTR_SIZE {
            binary.Read(b, binary.BigEndian, &info.size)
        }
    }

    return info
}
