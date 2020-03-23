package main

import "encoding/hex"

func McHexDigest(zdata []byte) []byte {
	data := make([]byte, len(zdata))
	copy(data, zdata)
	negative := int8(data[0]) < 0
	if negative {
		data = performTwosCompliment(data)
	}
	var digest []byte = make([]byte, 40)
	hex.Encode(digest, data)

	start := 0

	for i := 0; i < len(digest); i++ {
		if digest[i] == byte('0') {
			start = i + 1
		} else {
			break
		}
	}

	digest = digest[start:len(digest)]

	if negative {
		var realDigest []byte = []byte{byte('-')}
		realDigest = append(realDigest, digest...)
		return realDigest
	}

	return digest
}

func performTwosCompliment(buf []byte) []byte {
	carry := true
	var i int = 0
	var newByte byte
	var value byte
	for i = len(buf) - 1; i >= 0; i-- {
		value = buf[i]
		newByte = ^value & 255
		if carry {
			carry = newByte == 255
			buf[i] = newByte + 1
		} else {
			buf[i] = newByte
		}
	}

	return buf
}
