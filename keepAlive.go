package main

import (
	"crypto/rand"
	"time"
)

func doKeepAlive(session *Session, stop <-chan bool, keepAliveConfirm <-chan []byte) {
	ongoingKeepalive := false
	kaId := make([]byte, 8)
	for {
		select {
		case <-time.After(8 * time.Second):
			if ongoingKeepalive {
				session.disconnect()
				return
			} else {
				rand.Reader.Read(kaId)
				session.sendPacket(33, kaId)
				ongoingKeepalive = true
			}
		case <-stop:
			return

		case data := <-keepAliveConfirm:
			data = data[0:8]
			if string(data) == string(kaId) {
				ongoingKeepalive = false
			} else {
				session.disconnect()
				return
			}
		}
	}
}
