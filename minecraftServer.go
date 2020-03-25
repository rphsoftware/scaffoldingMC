package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"net"
	"strconv"
)

var PublicKey rsa.PublicKey
var PrivateKey *rsa.PrivateKey
var EncodedPublicKey []byte

func runServer() {
	if config.OnlineMode == true {
		log("Generating RSA keypair...")
		reader := rand.Reader
		bitSize := 1024

		privateKey, err := rsa.GenerateKey(reader, bitSize)
		if err != nil {
			log("Failed to generate RSA keypair!")
			panic("quit")
		}
		PrivateKey = privateKey
		PublicKey = PrivateKey.PublicKey
		EncodedPublicKey, err = x509.MarshalPKIXPublicKey(&PublicKey)
		log("Done")
	}
	log("Opening TCP socket")

	ln, err := net.Listen("tcp", ":"+strconv.Itoa(config.Port))
	if err != nil {
		log("Error occured while trying to open TCP port!")
		panic(err)
	}

	log("Ready")
	fmt.Println(writeVarInt(16384))
	for {
		connection, err := ln.Accept()
		if err != nil {
			log("An error occured while accepting connection", err)
			continue
		}

		go serverLoop(connection)
	}
}
