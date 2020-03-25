package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unicode/utf8"
)

func readVarInt(data []byte, offset int) (int, int) {
	var result int = 0
	var numRead uint = 0
	var read int = 0
	var value int = 0
	for {
		read = int(data[offset])
		value = read & 127
		result |= value << (7 * numRead)

		numRead++
		offset++

		if numRead > 5 {
			panic("Malformed varint")
		}

		if (read & 128) == 0 {
			return result, offset
		}
	}
}

func writeVarInt(value int) ([]byte, int) {
	var result = make([]byte, 5)
	var size int = 0
	for {
		temp := byte(value & 127)
		value = value >> 7
		if value != 0 {
			temp |= 128
		}
		result[size] = temp
		size++
		if value == 0 {
			return result[:size], size
		}
	}
}

func readString(data []byte, offset int) (string, int) {
	size, offset := readVarInt(data, offset)
	output := ""
	for i := 0; i < size; i++ {
		bruh := data[offset:]
		r, size := utf8.DecodeRune(bruh)
		output += string(r)
		offset += size
	}

	return output, offset
}

func writeString(value string) ([]byte, int) {
	result, _ := writeVarInt(len(value))
	result = append(result, []byte(value)...)

	return result, len(result)
}

func readByteArray(data []byte, offset int) ([]byte, int) {
	size, offset := readVarInt(data, offset)
	output := make([]byte, size)
	for i := 0; i < size; i++ {
		output[i] = data[offset]
		offset++
	}

	return output, offset
}

func float64ToByte(f float64) []byte {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, f)
	if err != nil {
		fmt.Println("binary.Write failed:", err)
	}
	return buf.Bytes()
}

func float32ToByte(f float32) []byte {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, f)
	if err != nil {
		fmt.Println("binary.Write failed:", err)
	}
	return buf.Bytes()
}
