package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"strconv"
	"sync"
	"time"
)

var icon string = ""
var chunkCache = make(map[int]map[int]map[int][]byte)
var chunkLock sync.Mutex

var lastLog = time.Now()

func log(a ...interface{}) {
	a = append(a, []interface{}{"|", time.Since(lastLog)}...)
	lastLog = time.Now()
	fmt.Println(a...)
}

func main() {
	compressedChunks := make(map[int]map[int][]byte)
	log("Loading config...")
	file, err := ioutil.ReadFile("config.json")
	if err != nil {
		log("An error occured while reading the config.json file!", err)
		return
	}

	err = json.Unmarshal(file, &config)
	if err != nil {
		log("An error occured while reading the config.json file!", err)
		return
	}

	log("Loading region files into memory cache...")
	requiredRegionFiles := make(map[string][]int)

	var regionFiles = make(map[string][]byte)

	for i := config.PreviewArea.Start.X; i <= config.PreviewArea.End.X; i++ {
		for j := config.PreviewArea.Start.Y; j <= config.PreviewArea.End.Y; j++ {
			name := "r." + strconv.Itoa(int(math.Floor(float64(i)/32))) + "." + strconv.Itoa(int(math.Floor(float64(j)/32))) + ".mca"
			requiredRegionFiles[name] = []int{
				int(math.Floor(float64(i) / 32)),
				int(math.Floor(float64(j) / 32)),
			}
		}
	}

	for i, _ := range requiredRegionFiles {
		data, err := ioutil.ReadFile(config.RegionFiles + i)
		if err != nil {
			log("An error occured while reading", config.RegionFiles+i, err)
			return
		}
		regionFiles[i] = data
	}

	// TODO: Stream the WHOLE chunk generation process!
	log("Reading chunks from files...")

	for i := config.PreviewArea.Start.X; i <= config.PreviewArea.End.X; i++ {
		compressedChunkLine := make(map[int][]byte)
		for j := config.PreviewArea.Start.Y; j <= config.PreviewArea.End.Y; j++ {
			name := "r." + strconv.Itoa(int(math.Floor(float64(i)/32))) + "." + strconv.Itoa(int(math.Floor(float64(j)/32))) + ".mca"
			data := regionFiles[name]
			//offsetToEntry := int(math.Abs(( float64(i % 32) ) + ( float64(j % 32) * 32) * 4))
			var offsetToEntry int

			var ii = int(math.Abs(float64(i)))
			var jj = int(math.Abs(float64(j)))

			// Hack to make negative coords work. It sucks but it works!
			if i < 0 {
				ii = ii % 32
				ii = 32 - ii
			}

			if j < 0 {
				jj = jj % 32
				jj = 32 - jj
			}

			offsetToEntry = ((int(math.Abs(float64(ii))) % 32) + ((int(math.Abs(float64(jj))) % 32) * 32)) * 4
			entry := data[offsetToEntry : offsetToEntry+4]

			offset := (int(entry[0]) * 65536) + (int(entry[1]) * 256) + int(entry[2])
			offset *= 4096
			size := int(entry[3])
			size *= 4096

			compressedChunkLine[(j - config.PreviewArea.Start.Y)] = data[offset : offset+size]
		}
		compressedChunks[(i - config.PreviewArea.Start.X)] = compressedChunkLine
	}

	log("Chunks read into lookup table.")
	if config.Icon.Enabled {
		log("Loading icon...")
		iconData, err := ioutil.ReadFile(config.Icon.Path)
		if err != nil {
			panic("Failed to load icon file! Please disable it if you don't have one!")
		}
		icon = base64.StdEncoding.EncodeToString(iconData)
	}
	loadRegistry()
	log("Generating chunks for every registry version")
	chunkGeneration(compressedChunks)

	compressedChunks = nil

	log("Starting minecraft server")

	runServer()
}
