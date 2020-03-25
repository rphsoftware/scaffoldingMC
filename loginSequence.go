package main

import (
	"crypto/aes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io/ioutil"
	"net/http"
	"time"
)

type YggdrasilResponse struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func sendLoginSuccess(s *Session, offline bool) {
	if offline == true {
		theUuid := uuid.New()
		s.playerUUIDDashed = theUuid.String()
		s.playerUUID = theUuid
	} else {
		s.playerUUIDDashed = s.playerUUID.String()
	}

	compressionPacket, _ := writeVarInt(128)
	s.sendPacket(0x03, compressionPacket)

	s.compressionEnabled = true
	s.compressionThreshold = 128

	packet, _ := writeString(s.playerUUIDDashed)
	usernamePart, _ := writeString(s.playerUsername)

	packet = append(packet, usernamePart...)
	s.sendPacket(0x02, packet)

	// Send join game
	joinPacket := []byte{
		0, 0, 0, 0, // Player entity ID (always 0)
		3,          // Gamemode 3
		0, 0, 0, 0, // Overworld TODO: Support other dimensions
		0, 0, 0, 0, 0, 0, 0, 0, 0, // i have no idea what this is for...
	}
	levelType, _ := writeString("default")
	renderDistance, _ := writeVarInt(config.RadiusToSend)
	joinPacket = append(joinPacket, levelType...)
	joinPacket = append(joinPacket, renderDistance...)
	joinPacket = append(joinPacket, 0, 1)

	s.sendPacket(0x26, joinPacket)

	// Send server brand

	// TODO: Replace with real code, this is just temporary!!!!!!!!!!!

	positionBuffer := float64ToByte(60)
	rotBuffer := float32ToByte(60)

	posPacket := make([]byte, 0)
	posPacket = append(posPacket, positionBuffer...)
	positionBuffer = float64ToByte(80)
	posPacket = append(posPacket, positionBuffer...)
	positionBuffer = float64ToByte(60)
	posPacket = append(posPacket, positionBuffer...)
	posPacket = append(posPacket, rotBuffer...)
	posPacket = append(posPacket, rotBuffer...)
	posPacket = append(posPacket, 0, 15)

	s.sendPacket(54, posPacket)
	s.sendPacket(0x41, []byte{3, 3})

	for i := 0; i < 20; i++ {
		for j := 0; j < 20; j++ {
			s.sendPacket(0x22, chunkCache[578][i][j])
			fmt.Println(i, j)
			time.Sleep(time.Millisecond * 30)
			//s.sendPacket(0x41, []byte{byte(i), byte(j)})
		}

	}

	//	fmt.Println(hex.Dump(chunkCache[578][3][3]))

	s.sendPacket(0x41, []byte{3, 3})

	s.sendPacket(54, posPacket)

	s.stage = 3
}

func loginStart(s *Session, packet []byte) {
	var ptr = 0
	s.playerUsername, ptr = readString(packet, ptr)

	if config.OnlineMode == false {
		// Offline mode
		sendLoginSuccess(s, true)
	} else {
		s.loginStage = 1
		s.encryptionVerifyToken = make([]byte, 4)
		rand.Read(s.encryptionVerifyToken)

		publicKeyLength, _ := writeVarInt(len(EncodedPublicKey))
		verifyTokenLength, _ := writeVarInt(len(s.encryptionVerifyToken))

		encryptionRequestPacket := []byte{0}
		encryptionRequestPacket = append(encryptionRequestPacket, publicKeyLength...)
		encryptionRequestPacket = append(encryptionRequestPacket, EncodedPublicKey...)
		encryptionRequestPacket = append(encryptionRequestPacket, verifyTokenLength...)
		encryptionRequestPacket = append(encryptionRequestPacket, s.encryptionVerifyToken...)

		s.sendPacket(1, encryptionRequestPacket)
	}
}

func loginVerify(s *Session, packet []byte) {
	var ptr = 0
	encryptedSharedSecret, ptr := readByteArray(packet, ptr)
	encryptedVerifyToken, ptr := readByteArray(packet, ptr)

	decryptedVerifyToken, err := PrivateKey.Decrypt(rand.Reader, encryptedVerifyToken, nil)
	if err != nil {
		s.disconnect()
	}

	if string(decryptedVerifyToken) != string(s.encryptionVerifyToken) {
		s.disconnect()
	}

	// Shared secret decryption
	decryptedSharedSecret, err := PrivateKey.Decrypt(rand.Reader, encryptedSharedSecret, nil)

	// Generate hash to send to yggdrasil
	hash := sha1.New()

	hash.Write(decryptedSharedSecret)
	hash.Write(EncodedPublicKey)

	hashSum := hash.Sum(nil)
	hashString := string(McHexDigest(hashSum))

	// Send request to yggdrasil
	resp, _ := http.Get(
		"https://sessionserver.mojang.com/session/minecraft/hasJoined?username=" +
			s.playerUsername +
			"&serverId=" +
			hashString)
	body, _ := ioutil.ReadAll(resp.Body)

	var response YggdrasilResponse
	json.Unmarshal(body, &response)

	if len(response.Id) < 8 { // Safe to say we have a suspicious one going on.
		s.disconnect()
	}

	// Create encryption instances for both stream ciphers
	s.encryptionEnabled = true

	encryptionBlock, _ := aes.NewCipher(decryptedSharedSecret)
	decryptionBlock, _ := aes.NewCipher(decryptedSharedSecret)

	s.clientBoundCryptoStream = NewCFB8(encryptionBlock, decryptedSharedSecret, false)
	s.serverBoundCryptoStream = NewCFB8(decryptionBlock, decryptedSharedSecret, true)

	s.playerUUID, _ = uuid.Parse(response.Id)

	sendLoginSuccess(s, false)
}

func loginHandler(s *Session, packetId int, packet []byte) {
	if s.loginStage == 0 && packetId == 0 {
		loginStart(s, packet)
	} else if s.loginStage == 1 && packetId == 1 {
		loginVerify(s, packet)
	} else {
		s.disconnect()
	}
}
