package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	nbt "github.com/rphsoftware/go.nbt"
	"github.com/rphsoftware/mcpackedarray"
	"io/ioutil"
	"math"
	"runtime"
	"sync"
)

type HeightMap struct {
	MotionBlocking         []uint64 `nbt:"MOTION_BLOCKING"`
	MotionBlockingNoLeaves []int64  `nbt:"MOTION_BLOCKING_NO_LEAVES"`
	OceanFloor             []int64  `nbt:"OCEAN_FLOOR"`
	OceanFloorWG           []int64  `nbt:"OCEAN_FLOOR_WG"`
	WorldSurface           []int64  `nbt:"WORLD_SURFACE"`
	WorldSurfaceWG         []int64  `nbt:"WORLD_SURFACE_WG"`
}

type CleanHeightMap struct {
	MotionBlocking [36]uint64 `nbt:"MOTION_BLOCKING"`
}

type PaletteEntry struct {
	Name       string
	Properties map[string]interface{}
}

type ChunkSection struct {
	Y           byte
	Palette     []PaletteEntry
	BlockLight  []byte
	BlockStates []uint64
	SkyLight    []byte
}

type NBTChunkInner struct {
	Heightmaps     HeightMap `nbt:"Heightmaps"`
	Status         string
	ZPos           uint32 `nbt:"zPos"`
	XPos           uint32 `nbt:"xPos"`
	LastUpdate     uint64
	Biomes         []uint32
	InhabitedTime  uint64
	TileEntities   []interface{}
	Entities       []interface{}
	IsLightOn      byte `nbt:"isLightOn"`
	TileTicks      []interface{}
	Sections       []ChunkSection
	PostProcessing []interface{}
	Structures     map[string]interface{}
	LiquidTicks    []interface{}
}

type NBTChunk struct {
	Level       NBTChunkInner `nbt:"Level"`
	DataVersion uint32
}

type CleanSection struct {
	data       []uint64
	palette    map[int]string
	paletteRev map[string]int
}

type ChunkEmissionJob struct {
	x    int
	z    int
	reg  int
	data []byte
}

// TODO: Multithread the shit out of this!!!!

var size int = 0

//
//func parseChunkAndEmitPacket(x int, z int, reg int, chunkData NBTChunk) { // Test code
//	finalChunk := make([]byte, 316155)
//	cache := make([]byte, 64)
//	var packetPointer = 0
//
//	// Write chunk X
//	binary.BigEndian.PutUint32(cache, uint32(x))
//	copy(finalChunk[packetPointer:packetPointer+4], cache)
//	packetPointer += 4
//
//	// Write chunk Z
//	binary.BigEndian.PutUint32(cache, uint32(z))
//	copy(finalChunk[packetPointer:packetPointer+4], cache)
//	packetPointer += 4
//
//	// Full chunk flag
//	finalChunk[packetPointer] = 1
//	packetPointer += 1
//
//	bitMask := 63 // 111111 (6 cubes)
//
//	cache, size = writeVarInt(bitMask)
//	copy(finalChunk[packetPointer:packetPointer+size], cache)
//	packetPointer += size
//
//	// Prepare height map
//	hmPa := mcpackedarray.NewPackedArray(9, 256)
//	for i := int32(0); i < 256; i++ {
//		hmPa.Set(i, 95)
//	}
//
//	hmBytes := hmPa.Serialise()
//
//	hm := new(CleanHeightMap)
//	for i := 0; i < 36; i++ {
//		hm.MotionBlocking[i] = binary.BigEndian.Uint64(hmBytes[i * 8 : i * 8 + 8])
//	}
//
//	fmt.Println(hm.MotionBlocking)
//
//	var hmData bytes.Buffer
//	nbt.Marshal(nbt.Uncompressed, &hmData, hm)
//
//	tmp := hmData.Bytes()
//	copy(finalChunk[packetPointer:], tmp)
//	packetPointer += len(tmp)
//
//	fmt.Println(hex.Dump(tmp))
//}

func parseChunkAndEmitPacket(x int, z int, reg int, chunkData NBTChunk) { // 1.13-1.15 code!!
	finalChunk := make([]byte, 316155)
	cache := make([]byte, 64)
	var packetPointer = 0

	// Write chunk X
	binary.BigEndian.PutUint32(cache, uint32(x))
	copy(finalChunk[packetPointer:packetPointer+4], cache)
	packetPointer += 4

	// Write chunk Z
	binary.BigEndian.PutUint32(cache, uint32(z))
	copy(finalChunk[packetPointer:packetPointer+4], cache)
	packetPointer += 4

	// Full chunk flag
	finalChunk[packetPointer] = 1
	packetPointer += 1

	// Bit-mask
	bitMask := 0

	for _, section := range chunkData.Level.Sections {
		if section.Y < 17 && len(section.Palette) > 0 {
			bitMask += int(math.Pow(2, float64(section.Y)))
		}
	}

	cache, size = writeVarInt(bitMask)
	copy(finalChunk[packetPointer:packetPointer+size], cache)
	packetPointer += size

	// Prepare height map
	// TODO: Optimize the appends.........
	hm := new(CleanHeightMap)
	for i, value := range chunkData.Level.Heightmaps.MotionBlocking {
		hm.MotionBlocking[i] = value
	}

	var hmData bytes.Buffer
	nbt.Marshal(nbt.Uncompressed, &hmData, hm)

	tmp := hmData.Bytes()
	copy(finalChunk[packetPointer:], tmp)
	packetPointer += len(tmp)

	// Allocate biomes
	for _, value := range chunkData.Level.Biomes {
		a := value >> 24
		finalChunk[packetPointer] = byte(a)
		value -= a * 16777216

		a = value >> 16
		finalChunk[packetPointer+1] = byte(a)
		value -= a * 65536

		a = value >> 8
		finalChunk[packetPointer+2] = byte(a)
		value -= a * 256

		finalChunk[packetPointer+3] = byte(value)

		packetPointer += 4
	}

	// TODO: Continue optim under this line.

	// Actually generate chunk information
	// Issue: we must write data size before we write chunk data
	// Solution: Allocate a NEW chunk data object in accordance with our previous findings!

	fullChunkData := make([]byte, 311392)
	fullChunkDataPtr := 0

	for _, section := range chunkData.Level.Sections {
		if section.Y < 17 && len(section.Palette) > 0 {
			// Read palette
			sectionBlocks := make([]string, len(section.Palette))
			palettePtr := 0
			absolutePalette := false
			mappedPaletteEntrySize := 4

			for _, paletteEntry := range section.Palette {
				props := make([]string, len(paletteEntry.Properties))
				propPtr := 0
				for i, v := range paletteEntry.Properties {
					if valu, ok := v.(string); ok {
						props[propPtr] = i + "=" + valu
					}

					propPtr++
				}

				sectionBlocks[palettePtr] = makeBlockStateIdentifier(paletteEntry.Name, props)

				palettePtr++
			}

			if len(sectionBlocks) <= 16 {
				absolutePalette = false
				mappedPaletteEntrySize = 4
			} else if len(sectionBlocks) > 16 && len(sectionBlocks) <= 256 {
				absolutePalette = false
				mappedPaletteEntrySize = int(math.Ceil(math.Log2(float64(len(sectionBlocks)))))
			} else {
				absolutePalette = true
				mappedPaletteEntrySize = 14
			}

			// Load block states into a virtual data object
			byteData := make([]byte, len(section.BlockStates)*8)
			byteBit := make([]byte, 8)

			for v := 0; v < len(section.BlockStates); v++ {
				binary.BigEndian.PutUint64(byteBit, section.BlockStates[v])
				byteData[v*8+0] = byteBit[0]
				byteData[v*8+1] = byteBit[1]
				byteData[v*8+2] = byteBit[2]
				byteData[v*8+3] = byteBit[3]
				byteData[v*8+4] = byteBit[4]
				byteData[v*8+5] = byteBit[5]
				byteData[v*8+6] = byteBit[6]
				byteData[v*8+7] = byteBit[7]
			}

			bitsPerEntrySource := math.Ceil(math.Log2(float64(len(section.Palette))))
			if bitsPerEntrySource < 4 {
				bitsPerEntrySource = 4
			}

			blockStates := mcpackedarray.PackedArrayFromData(byteData, byte(bitsPerEntrySource))
			// Calculate non-air blocks
			var nonAirBlocks int16 = 0

			for _, blockState := range blockStates.Entries {
				if blockState != 0 {
					nonAirBlocks++
				}
			}

			topByte := byte(nonAirBlocks >> 8)
			lowerByte := byte((nonAirBlocks) - (int16(topByte) * 256))

			fullChunkData[fullChunkDataPtr] = topByte
			fullChunkData[fullChunkDataPtr+1] = lowerByte
			fullChunkData[fullChunkDataPtr+2] = byte(mappedPaletteEntrySize)
			fullChunkDataPtr += 3

			if absolutePalette == false {
				// Write palette
				cache, size = writeVarInt(len(sectionBlocks))
				copy(fullChunkData[fullChunkDataPtr:fullChunkDataPtr+size], cache)
				fullChunkDataPtr += size

				for _, paletteEntry := range sectionBlocks {
					registryId := registry[reg].blockStates[paletteEntry]
					cache, size = writeVarInt(registryId)

					copy(fullChunkData[fullChunkDataPtr:fullChunkDataPtr+size], cache)
					fullChunkDataPtr += size
				}
			}

			// Calculate size of data array
			sizeOfData := (mappedPaletteEntrySize * 512) / 8
			cache, size := writeVarInt(int(sizeOfData))
			copy(fullChunkData[fullChunkDataPtr:fullChunkDataPtr+size], cache)
			fullChunkDataPtr += size

			// Create the actual compacted array for the chunk data
			chunkBlockStates := mcpackedarray.NewPackedArray(byte(mappedPaletteEntrySize), 4096)

			for i := int32(0); i < 4096; i++ {
				if absolutePalette == true {
					chunkBlockStates.Set(i, uint32(registry[reg].blockStates[sectionBlocks[blockStates.Get(i)]]))
				} else {
					chunkBlockStates.Set(i, blockStates.Get(i))
				}
			}

			tmp = chunkBlockStates.Serialise()

			copy(fullChunkData[fullChunkDataPtr:fullChunkDataPtr+len(tmp)], tmp)
			fullChunkDataPtr += len(tmp)
		}
	}

	cache, size = writeVarInt(fullChunkDataPtr)
	copy(finalChunk[packetPointer:packetPointer+size], cache)
	packetPointer += size

	copy(finalChunk[packetPointer:packetPointer+fullChunkDataPtr], fullChunkData)
	packetPointer += fullChunkDataPtr
	finalChunk[packetPointer] = 0
	packetPointer += 1

	finalChunk = finalChunk[:packetPointer]

	packet := make([]byte, 0)
	packet = append(packet, 0)

	var compressed bytes.Buffer
	compressedWriter := zlib.NewWriter(&compressed)
	compressedWriter.Write(finalChunk)
	compressedWriter.Close()

	chunkLock.Lock()
	// All the thread-unsafe IO operations with the global chunk map

	if _, ok := chunkCache[reg]; ok != true {
		chunkCache[reg] = make(map[int]map[int][]byte)
	}

	if _, ok := chunkCache[reg][x]; ok != true {
		chunkCache[reg][x] = make(map[int][]byte)
	}

	chunkCache[reg][x][z] = finalChunk

	size += len(chunkCache[reg][x][z])

	chunkLock.Unlock()
}

func runTaskSet(wg *sync.WaitGroup, jobs []ChunkEmissionJob) {
	for _, entry := range jobs {
		r, _ := zlib.NewReader(bytes.NewReader(entry.data))
		chunkNbtData, _ := ioutil.ReadAll(r)

		var aa NBTChunk
		_ = nbt.Unmarshal(nbt.Uncompressed, bytes.NewReader(chunkNbtData), &aa)
		// Generate chunk packet itself

		parseChunkAndEmitPacket(entry.x, entry.z, entry.reg, aa)

		r.Close()
		aa = NBTChunk{}
	}

	wg.Done()
}

func chunkGeneration(compressedChunks map[int]map[int][]byte) {
	for a, _ := range registry {
		var jobList = make([]ChunkEmissionJob, len(compressedChunks)*len(compressedChunks[0]))
		var ptr = 0
		for i := 0; i < len(compressedChunks); i++ {
			for j := 0; j < len(compressedChunks[i]); j++ {
				index := i*len(compressedChunks[i]) + j
				index += 1

				entry := compressedChunks[i][j]
				compressionScheme := entry[4]

				if compressionScheme == 2 {
					nbtData := entry[5:]

					job := ChunkEmissionJob{
						x:    i,
						z:    j,
						reg:  a,
						data: nbtData,
					}
					jobList[ptr] = job
					ptr++
				} else {
					panic("!!!! UNSUPPORTED COMPRESSION SCHEME !!!! THIS MAP IS NOT STANDARD !!!!")
				}
			}
		}

		log("> Running jobs")
		threadsToStart := runtime.NumCPU()

		var wg sync.WaitGroup

		for i := threadsToStart; i >= 1; i-- {
			part := jobList[0:int(math.Floor(float64(len(jobList)/i)))]
			jobList = jobList[int(math.Floor(float64(len(jobList)/i))):]

			wg.Add(1)

			go runTaskSet(&wg, part)
		}
		log("Tasks started")

		wg.Wait()

		log("[CHUNK GENERATION] Done for registry", a)
	}
}
