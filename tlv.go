package main

import (
	"bytes"
	"encoding/binary"
	"io"
)

func parseTLV(data []byte) map[uint8][]byte {
	if len(data) < 3 {
		return nil
	}

	tlvs := map[uint8][]byte{}

	reader := bytes.NewReader(data)

	for reader.Len() > 0 {
		typ, _ := reader.ReadByte()

		var length uint16
		err := binary.Read(reader, binary.BigEndian, &length)
		if err != nil {
			return nil
		}

		value := make([]byte, length)

		_, err = io.ReadFull(reader, value)
		if err != nil {
			return nil
		}

		tlvs[typ] = value
	}

	return tlvs
}
