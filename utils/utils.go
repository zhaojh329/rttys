/*
 * MIT License
 *
 * Copyright (c) 2019 Jianhui Zhao <zhaojh329@gmail.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
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
