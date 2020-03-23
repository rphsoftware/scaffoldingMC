package main

import "encoding/json"

type Color string

const (
	BLACK        Color = "black"
	DARK_BLUE    Color = "dark_blue"
	DARK_GREEN   Color = "dark_green"
	DARK_AQUA    Color = "dark_aqua"
	DARK_RED     Color = "dark_red"
	DARK_PURPLE  Color = "dark_purple"
	GOLD         Color = "gold"
	GRAY         Color = "gray"
	DARK_GRAY    Color = "dark_gray"
	BLUE         Color = "blue"
	GREEN        Color = "green"
	AQUA         Color = "aqua"
	RED          Color = "red"
	LIGHT_PURPLE Color = "light_purple"
	YELLOW       Color = "yellow"
	WHITE        Color = "white"
)

type Chat struct {
	Text string `json:"text"`

	// formatting
	Bold          bool  `json:"bold"`
	Italic        bool  `json:"italic"`
	Underlined    bool  `json:"underlined"`
	Strikethrough bool  `json:"strikethrough"`
	Obfuscated    bool  `json:"obfuscated"`
	Color         Color `json:"color"`

	// children
	Children []Chat `json:"children"`
}

type ServerInfoVersion struct {
	Name     string `json:"name"`
	Protocol int    `json:"protocol"`
}

type ServerInfoPlayersSample struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

type ServerInfoPlayers struct {
	Max    int                       `json:"max"`
	Online int                       `json:"online"`
	Sample []ServerInfoPlayersSample `json:"sample"`
}

type ServerInfo struct {
	Version     ServerInfoVersion `json:"version"`
	Players     ServerInfoPlayers `json:"players"`
	Description Chat              `json:"description"`
	Favicon     string            `json:"favicon"`
}

func handshakeHandler(s *Session, packetId int, packet []byte) {
	if s.stage == 0 {
		protocolVersion, ptr := readVarInt(packet, 0)

		_, ptr = readString(packet, ptr)
		ptr += 2

		nextState, ptr := readVarInt(packet, ptr)
		s.stage = nextState

		_, ok := registry[protocolVersion]

		if nextState == 2 && ok != true {
			disconnectPacket, size := writeString("{\"text\":\"§3K§2i§3l§4l §5y§6o§7u§8r§9s§3e§al§bf §cr§ee§ft§3a§2r§3d§4!\"}")
			disconnectPacketSize, _ := writeVarInt(size + 1)
			adisconnectPacket := append(disconnectPacketSize, 0)
			adisconnectPacket = append(adisconnectPacket, disconnectPacket...)
			s.connection.Write(adisconnectPacket)
			s.connection.Close()
			panic("Unsupported version!")
		}

		return
	}
	if s.stage == 1 {
		if packetId == 0 {
			newData := ServerInfo{
				Version:     ServerInfoVersion{
					Protocol: 	578,
					Name: 		"1.15.2",
				},
				Players:     ServerInfoPlayers{
					100,
					0,
					[]ServerInfoPlayersSample{},
				},
				Description: Chat{
					Text: config.Motd,
				},
			}
			if config.Icon.Enabled == true {
				newData.Favicon = "data:image/png;base64," + icon
			}

			encoded, err := json.Marshal(newData)
			if err != nil {
				panic(err)
			}

			data, _ := writeString(string(encoded))

			s.sendPacket(0, data)
		}
		if packetId == 1 {
			s.sendPacket(1, packet)
			s.connection.Close()
		}
	}
}
