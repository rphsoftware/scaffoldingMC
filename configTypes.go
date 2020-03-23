package main

type JSONConfigVector struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type JSONConfigPreviewArea struct {
	Start JSONConfigVector `json:"start"`
	End   JSONConfigVector `json:"end"`
}

type JSONIconSettings struct {
	Enabled			bool					`json:"enabled"`
	Path			string					`json:"path"`
}

type JSONConfig struct {
	PreviewArea 	JSONConfigPreviewArea 	`json:"previewArea"`
	RadiusToSend 	int					  	`json:"radiusToSend"`
	OnlineMode		bool					`json:"onlineMode"`
	Port 			int 					`json:"port"`
	RegionFiles		string					`json:"regionFiles"`
	ShowCredits		bool					`json:"showServerCredits"`
	Motd			string					`json:"motd"`
	Icon  			JSONIconSettings		`json:"icon"`
	TimeOfDay		int						`json:"timeOfDay"`
}

var config JSONConfig

type LUTEntry struct {
	File 			string
	Offset 			int
	Size 			int
}