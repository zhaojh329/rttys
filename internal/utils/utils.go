/* SPDX-License-Identifier: MIT */
/*
 * Author: Jianhui Zhao <zhaojh329@gmail.com>
 */

package utils

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"io"

	"github.com/google/uuid"
)

func GenUniqueID() string {
	hash := md5.Sum([]byte(uuid.New().String()))
	return hex.EncodeToString(hash[:16])
}

func ParseTLV(data []byte) map[uint8][]byte {
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
