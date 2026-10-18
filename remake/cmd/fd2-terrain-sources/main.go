// fd2-terrain-sources exports private art references through the game's native
// loaders. It does not decode DAT files or provide a second game renderer.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type fileBinding struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}

type frameSource struct {
	Frame       int           `json:"frame"`
	ContentHash string        `json:"native_content_sha256"`
	Layers      []fileBinding `json:"input_layers"`
}

type mapSource struct {
	Map              int           `json:"map"`
	FrameCount       int           `json:"frame_count"`
	Bank             fileBinding   `json:"bank"`
	MapData          fileBinding   `json:"map_data"`
	Frames           []frameSource `json:"frames"`
	Outputs          []fileBinding `json:"outputs"`
	TerrainFrames    []int         `json:"selected_terrain_frames"`
	ForegroundFrames []int         `json:"selected_foreground_frames"`
	SelectorChecks   int           `json:"selector_checks"`
}

type artManifest struct {
	Schema        int           `json:"schema_version"`
	Kind          string        `json:"kind"`
	PrivateOnly   bool          `json:"private_local_only"`
	HDArt         bool          `json:"hd_art_complete"`
	Palette       fileBinding   `json:"palette"`
	FrameSize     int           `json:"frame_size"`
	Columns       int           `json:"columns"`
	ContentFormat string        `json:"native_content_format"`
	Maps          []mapSource   `json:"maps"`
	UniqueFrames  int           `json:"unique_native_frames"`
	Previews      []fileBinding `json:"previews"`
	Objects       []artObject   `json:"objects"`
}

type objectCell struct {
	X          int    `json:"x"`
	Y          int    `json:"y"`
	TileWord   uint16 `json:"tile_word"`
	Frame      int    `json:"frame"`
	Foreground *int   `json:"foreground_frame,omitempty"`
}

type artObject struct {
	ID        string       `json:"id"`
	Map       int          `json:"map"`
	X         int          `json:"x"`
	Y         int          `json:"y"`
	Width     int          `json:"width"`
	Height    int          `json:"height"`
	Reference fileBinding  `json:"reference"`
	Cells     []objectCell `json:"cells"`
}

type mapData struct {
	Width    int      `json:"w"`
	Height   int      `json:"h"`
	Tiles    []uint16 `json:"tiles"`
	Controls []byte   `json:"native_terrain_control"`
}

type loadedMap struct {
	Bank *fdicon.Bank
	Data mapData
}

func digest(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func binding(root, relative string) (fileBinding, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	return fileBinding{File: relative, SHA256: digest(data)}, err
}

func ordered(values map[int]bool) []int {
	result := make([]int, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Ints(result)
	return result
}

func validateSelectors(data mapData, controls []byte, count int) ([]int, []int, int, error) {
	if data.Width <= 0 || data.Height <= 0 || len(data.Tiles) != data.Width*data.Height ||
		len(data.Controls) != len(controls) || len(controls)%4 != 0 {
		return nil, nil, 0, errors.New("map geometry/control shape mismatch")
	}
	for i := range controls {
		if data.Controls[i] != controls[i] {
			return nil, nil, 0, errors.New("map controls differ from native bank")
		}
	}
	terrain, foreground := make(map[int]bool), make(map[int]bool)
	checks := 0
	for _, raw := range data.Tiles {
		tile := int(raw & 0x3ff)
		if tile*4 >= len(controls) {
			return nil, nil, checks, fmt.Errorf("tile %d outside control table", tile)
		}
		for flip := 0; flip <= 1; flip++ {
			for cycle := 0; cycle <= 3; cycle++ {
				frame, err := fdicon.NativeTerrainFrameIndex(tile, controls[tile*4], flip, cycle)
				if err != nil || frame < 0 || frame >= count {
					return nil, nil, checks, fmt.Errorf("terrain tile %d selects invalid frame %d", tile, frame)
				}
				terrain[frame] = true
				frame, present, err := fdicon.NativeForegroundFrameIndex(tile, controls[tile*4], flip)
				if err != nil || (present && (frame < 0 || frame >= count)) {
					return nil, nil, checks, fmt.Errorf("foreground tile %d selects invalid frame %d", tile, frame)
				}
				if present {
					foreground[frame] = true
				}
				checks++
			}
		}
	}
	return ordered(terrain), ordered(foreground), checks, nil
}

func loadSources(pack, maps string) (artManifest, []loadedMap, color.Palette, error) {
	manifest := artManifest{Schema: 1, Kind: "fd2_native_terrain_art_sources", PrivateOnly: true,
		FrameSize: 24, Columns: 16, ContentFormat: "576 uint8 indices + 576 binary(0/1) opaque + 576 binary(0/1) destination-remap"}
	_, palette, err := fdother.LoadSeparatedFDOTHERPalette(filepath.Join(pack, "palette"), 0)
	if err != nil {
		return manifest, nil, nil, err
	}
	manifest.Palette, err = binding(pack, "palette/fdother_000.json")
	if err != nil {
		return manifest, nil, nil, err
	}
	loaded := make([]loadedMap, 0, fdicon.SeparatedFDSHAPMapCount)
	unique := make(map[string]bool)
	for mapID := 0; mapID < fdicon.SeparatedFDSHAPMapCount; mapID++ {
		bank, controls, err := fdicon.LoadSeparatedFDSHAPBank(filepath.Join(pack, "tilesets/fdshap"), mapID)
		if err != nil {
			return manifest, nil, nil, err
		}
		source := mapSource{Map: mapID, FrameCount: len(bank.Sprites)}
		prefix := fmt.Sprintf("tilesets/fdshap/map_%02d/", mapID)
		source.Bank, err = binding(pack, prefix+"bank.json")
		if err != nil {
			return manifest, nil, nil, err
		}
		mapPath := fmt.Sprintf("map%d/map.json", mapID)
		source.MapData, err = binding(maps, mapPath)
		if err != nil {
			return manifest, nil, nil, err
		}
		raw, err := os.ReadFile(filepath.Join(maps, mapPath))
		if err != nil {
			return manifest, nil, nil, err
		}
		var data mapData
		if err = json.Unmarshal(raw, &data); err != nil {
			return manifest, nil, nil, err
		}
		source.TerrainFrames, source.ForegroundFrames, source.SelectorChecks, err = validateSelectors(data, controls, len(bank.Sprites))
		if err != nil {
			return manifest, nil, nil, fmt.Errorf("map%d: %w", mapID, err)
		}
		for frame, sprite := range bank.Sprites {
			if len(sprite.Pixels) != 576 || len(sprite.Mask) != 576 || len(sprite.RemapMask) != 576 {
				return manifest, nil, nil, errors.New("native frame shape mismatch")
			}
			for i, mask := range sprite.Mask {
				if mask > 1 || sprite.RemapMask[i] > 1 || (mask != 0 && sprite.RemapMask[i] != 0) {
					return manifest, nil, nil, errors.New("native masks overlap or are not binary")
				}
			}
			content := append(append(append([]byte{}, sprite.Pixels...), sprite.Mask...), sprite.RemapMask...)
			entry := frameSource{Frame: frame, ContentHash: digest(content)}
			unique[entry.ContentHash] = true
			for _, name := range []string{"frame.png", "mask.png", "remap_mask.png"} {
				layer, err := binding(pack, fmt.Sprintf("%stile_%04d/%s", prefix, frame, name))
				if err != nil {
					return manifest, nil, nil, err
				}
				entry.Layers = append(entry.Layers, layer)
			}
			source.Frames = append(source.Frames, entry)
		}
		manifest.Maps = append(manifest.Maps, source)
		loaded = append(loaded, loadedMap{Bank: bank, Data: data})
	}
	manifest.UniqueFrames = len(unique)
	if loaded[0].Data.Width < 15 || loaded[0].Data.Height < 12 {
		return manifest, nil, nil, errors.New("map0 does not contain the declared house reference region")
	}
	return manifest, loaded, palette, nil
}

func savePNG(root, name string, value image.Image) (fileBinding, error) {
	file, err := os.OpenFile(filepath.Join(root, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return fileBinding{}, err
	}
	if err = png.Encode(file, value); err != nil {
		file.Close()
		return fileBinding{}, err
	}
	if err = file.Close(); err != nil {
		return fileBinding{}, err
	}
	return binding(root, name)
}

func artCell(dst *image.NRGBA, sprite fdicon.Sprite, x, y int, palette color.Palette) {
	for row := 0; row < 24; row++ {
		for column := 0; column < 24; column++ {
			i := row*24 + column
			if sprite.Mask[i] != 0 {
				dst.Set(x+column, y+row, palette[sprite.Pixels[i]])
			}
		}
	}
}

func writeSources(root string, manifest *artManifest, loaded []loadedMap, palette color.Palette) error {
	for mapID, source := range loaded {
		bounds := image.Rect(0, 0, 384, ((len(source.Bank.Sprites)+15)/16)*24)
		rgba := image.NewNRGBA(bounds)
		indices, mask, remap := image.NewGray(bounds), image.NewGray(bounds), image.NewGray(bounds)
		for frame, sprite := range source.Bank.Sprites {
			x, y := (frame%16)*24, (frame/16)*24
			artCell(rgba, sprite, x, y, palette)
			for row := 0; row < 24; row++ {
				for column := 0; column < 24; column++ {
					i := row*24 + column
					indices.SetGray(x+column, y+row, color.Gray{Y: sprite.Pixels[i]})
					mask.SetGray(x+column, y+row, color.Gray{Y: sprite.Mask[i] * 255})
					remap.SetGray(x+column, y+row, color.Gray{Y: sprite.RemapMask[i] * 255})
				}
			}
		}
		for _, layer := range []struct {
			Name string
			Data image.Image
		}{{"rgba", rgba}, {"indices", indices}, {"mask", mask}, {"remap", remap}} {
			output, err := savePNG(root, fmt.Sprintf("map%02d-%s.png", mapID, layer.Name), layer.Data)
			if err != nil {
				return err
			}
			manifest.Maps[mapID].Outputs = append(manifest.Maps[mapID].Outputs, output)
		}
	}
	// This is an unremapped art reference, not a captured game state. It uses
	// connected cells so a building can be repainted whole before slicing.
	source := loaded[0]
	for _, object := range []artObject{
		{ID: "map00-house", Map: 0, X: 9, Y: 8, Width: 6, Height: 4},
		{ID: "map00-roof-object", Map: 0, X: 11, Y: 4, Width: 2, Height: 4},
	} {
		picture := image.NewNRGBA(image.Rect(0, 0, object.Width*24, object.Height*24))
		for row := 0; row < object.Height; row++ {
			for column := 0; column < object.Width; column++ {
				word := source.Data.Tiles[(row+object.Y)*source.Data.Width+column+object.X]
				tile := int(word & 0x3ff)
				frame, err := fdicon.NativeTerrainFrameIndex(tile, source.Data.Controls[tile*4], 0, 0)
				if err != nil {
					return err
				}
				artCell(picture, source.Bank.Sprites[frame], column*24, row*24, palette)
				cell := objectCell{X: column, Y: row, TileWord: word, Frame: frame}
				foreground, present, err := fdicon.NativeForegroundFrameIndex(tile, source.Data.Controls[tile*4], 0)
				if err != nil {
					return err
				}
				if present {
					cell.Foreground = &foreground
				}
				object.Cells = append(object.Cells, cell)
			}
		}
		preview, err := savePNG(root, object.ID+"-art-reference.png", picture)
		if err != nil {
			return err
		}
		manifest.Previews = append(manifest.Previews, preview)
		object.Reference = preview
		manifest.Objects = append(manifest.Objects, object)
	}
	file, err := os.OpenFile(filepath.Join(root, "manifest.json"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(manifest)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func run(pack, maps, out string) error {
	if pack == "" || maps == "" || out == "" {
		return errors.New("-pack, -maps and -out are required")
	}
	if _, err := os.Lstat(out); !os.IsNotExist(err) {
		return errors.New("output already exists or cannot be checked")
	}
	manifest, loaded, palette, err := loadSources(pack, maps)
	if err != nil {
		return err
	}
	stage, err := os.MkdirTemp(filepath.Dir(out), ".terrain-art-stage-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	if err = writeSources(stage, &manifest, loaded, palette); err != nil {
		return err
	}
	if _, err = os.Lstat(out); !os.IsNotExist(err) {
		return errors.New("output appeared during export; preserving it")
	}
	if err = os.Rename(stage, out); err != nil {
		return err
	}
	fmt.Printf("native art sources: %d maps, %d unique frames; HD art not complete\n", len(manifest.Maps), manifest.UniqueFrames)
	return nil
}

func main() {
	pack := flag.String("pack", "", "legal separated original pack")
	maps := flag.String("maps", "", "remake/assets/maps directory")
	out := flag.String("out", "", "new private local output directory")
	flag.Parse()
	if err := run(*pack, *maps, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
