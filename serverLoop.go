package main

import (
	"bytes"
	"compress/zlib"
	"crypto/cipher"
	"github.com/google/uuid"
	"io/ioutil"
	"net"
	"sync"
)

type ChunkWorkerCommand int
const (
	CW_NEW_CHUNK	ChunkWorkerCommand = 0
	CW_DIE			ChunkWorkerCommand = 1
)

type Session struct {
	stage 					int
	connection 				net.Conn

	// player info
	playerUsername  		string
	encryptionVerifyToken	[]byte
	loginStage				int

	// Encryption related
	clientBoundCryptoStream cipher.Stream
	serverBoundCryptoStream cipher.Stream
	encryptionEnabled       bool

	// Player info
	playerUUID				uuid.UUID
	playerUUIDDashed		string

	xPosition				float64
	yPosition				float64
	zPosition				float64

	xChunk					int
	zChunk					int

	// Compression info
	compressionEnabled 		bool
	compressionThreshold	int

	// Packet mutex
	packetLock				sync.Mutex
	_kc						chan bool
}

type ChunkWorkerPacket struct {
	s 		*Session
	command ChunkWorkerCommand
}

func (s *Session) disconnect() {
	// TODO: Kill chunk goroutine
	s.connection.Close()
	s._kc <- true
}

func (s *Session) sendPacket(id byte, data []byte) {
	s.packetLock.Lock()
	packet := make([]byte, 0)
	if s.compressionEnabled == false {
		packet, _ = writeVarInt(len(data) + 1)
		packet = append(packet, id)
		packet = append(packet, data...)
	} else {
		// Unlike no compression, this stuff gets compressed...
		compressedPacket := []byte{id}
		compressedPacket = append(compressedPacket, data...)


		if len(compressedPacket) < s.compressionThreshold {
			dataLen, _ := writeVarInt(0)
			compressedPacket = append(dataLen, compressedPacket...)
		} else {
			dataLen, _ := writeVarInt(len(compressedPacket))
			var b bytes.Buffer
			compressedWriter := zlib.NewWriter(&b)
			compressedWriter.Write(compressedPacket)
			compressedWriter.Close()

			compressedPacket = b.Bytes()

			compressedPacket = append(dataLen, compressedPacket...)
		}

		compressedPacketSize, _ := writeVarInt(len(compressedPacket))
		packet = append(compressedPacketSize, compressedPacket...)
	}



	if s.encryptionEnabled == true {
		s.clientBoundCryptoStream.XORKeyStream(packet, packet)
	}

	_, _ = s.connection.Write(packet)
	s.packetLock.Unlock()
}

func serverLoop(connection net.Conn) {
	stopChannel := make(chan bool)
	dataChannel := make(chan []byte)

	defer func() {
		recover()
		stopChannel <- true
		connection.Close()
	}()

	startedKeepAlive := false

	session := Session{
		stage:                   0,
		connection:              connection,
		playerUsername:          "",
		encryptionVerifyToken:   nil,
		loginStage:              0,
		clientBoundCryptoStream: nil,
		serverBoundCryptoStream: nil,
		encryptionEnabled:       false,
		playerUUID:              uuid.UUID{},
		playerUUIDDashed:        "",
		xPosition:               0,
		yPosition:               0,
		zPosition:               0,
		compressionEnabled:      false,
		compressionThreshold:    0,
	}

	for {
		pointer := 0
		packet := make([]byte, 32768)
		bytesRead, err := connection.Read(packet)
		packet = packet[:bytesRead]

		if err != nil {
			panic(err)
		}

		var packetId int
		var packetData []byte

		if session.encryptionEnabled == true {
			session.serverBoundCryptoStream.XORKeyStream(packet, packet)
		}

		if session.compressionEnabled != true {
			var packetLength int
			packetLength, pointer = readVarInt(packet, pointer)
			packetId, pointer = readVarInt(packet, pointer)
			packetData = packet[pointer : packetLength + pointer - 1]
		} else {
			var packetLength int
			packetLength, pointer = readVarInt(packet, pointer)
			var decompressedLength int
			var size int
			decompressedLength, size = readVarInt(packet, pointer)
			size = size - pointer
			pointer += size

			if decompressedLength == 0 {
				packetId, pointer = readVarInt(packet, pointer)
				packetData = packet[pointer : packetLength + pointer - size - 1]
			} else {
				packetLength -= size
				compressedData := packet[pointer : pointer+packetLength]

				reader, _ := zlib.NewReader(bytes.NewReader(compressedData))
				packetData, _ = ioutil.ReadAll(reader)

				packetId, pointer = readVarInt(packetData, 0)
				packetData = packetData[pointer:]
			}
		}

		switch session.stage {
		case 0:
			handshakeHandler(&session, packetId, packetData)
			continue
		case 1:
			handshakeHandler(&session, packetId, packetData)
			continue
		case 2:
			loginHandler(&session, packetId, packetData)
			continue
		case 3:
			if startedKeepAlive == false {
				go doKeepAlive(&session, stopChannel, dataChannel)
				startedKeepAlive = true
				continue
			}

			if packetId == 15 {
				dataChannel	 <- packetData
				continue
			}
		}
	}
}
