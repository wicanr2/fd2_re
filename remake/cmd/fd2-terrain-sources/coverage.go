package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sort"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

type coverageHD struct {
	Asset string `json:"asset_id"`
	Map   int    `json:"map"`
	Frame int    `json:"frame"`
	File  string `json:"file"`
	SHA   string `json:"sha256"`
}

type coverageSelector struct {
	Tile  uint16 `json:"tile_word"`
	Cell  int    `json:"cell"`
	Flip  int    `json:"flip"`
	Cycle int    `json:"cycle"`
	Pass  int    `json:"event_update_pass"`
}

type coverageFrame struct {
	Frame     int              `json:"frame"`
	Role      string           `json:"role"`
	Content   string           `json:"native_content_sha256"`
	Writes    int              `json:"source_pixels"`
	Remaps    int              `json:"remap_pixels"`
	Status    string           `json:"status"`
	Selector  coverageSelector `json:"selector"`
	HD        *coverageHD      `json:"hd,omitempty"`
	Reuse     *coverageHD      `json:"reuse,omitempty"`
	Conflicts []string         `json:"conflicting_hd_sha256,omitempty"`
}

type coverageMap struct {
	Map               int             `json:"map"`
	Bank              fileBinding     `json:"bank"`
	Data              fileBinding     `json:"map_data"`
	InitialTerrain    []int           `json:"initial_terrain_frames"`
	InitialForeground []int           `json:"initial_foreground_frames"`
	EventPasses       int             `json:"event_update_passes"`
	SelectorChecks    int             `json:"selector_checks"`
	Frames            []coverageFrame `json:"required"`
}

type coverageReport struct {
	Schema     int            `json:"schema_version"`
	Kind       string         `json:"kind"`
	Scope      string         `json:"scope"`
	CatalogSHA string         `json:"catalog_sha256"`
	Palette    fileBinding    `json:"palette"`
	Maps       []coverageMap  `json:"maps"`
	Summary    map[string]int `json:"summary"`
}

// Use the existing event writer on a private copy. Setting every event active
// gives a source upper bound, not evidence that those events occur in play.
func coverageStates(data mapData) ([][]uint16, error) {
	n := data.Width * data.Height
	if data.Width <= 0 || data.Width > 255 || data.Height <= 0 || data.Height > 255 ||
		len(data.Tiles) != n || len(data.Events) != n || len(data.Modes) != n {
		return nil, errors.New("coverage: complete native composition required")
	}
	st := &battle.State{W: data.Width, H: data.Height, HasNativeMapEventGrid: true,
		NativeTerrainControl: append([]byte(nil), data.Controls...), NativeMapEventGrid: make([]byte, 4+4*n)}
	st.NativeMapEventGrid[0], st.NativeMapEventGrid[2] = byte(data.Width), byte(data.Height)
	for i, word := range data.Tiles {
		off := 4 + 4*i
		binary.LittleEndian.PutUint16(st.NativeMapEventGrid[off:off+2], word)
		st.NativeMapEventGrid[off+2], st.NativeMapEventGrid[off+3] = data.Events[i], data.Modes[i]
	}
	for i := range st.NativeEventState {
		st.NativeEventState[i] = 1
	}
	states := [][]uint16{append([]uint16(nil), data.Tiles...)}
	seen := map[string]bool{string(st.NativeMapEventGrid): true}
	for pass := 0; pass < 1024; pass++ {
		before := append([]byte(nil), st.NativeMapEventGrid...)
		if !st.CompleteTreasureMapUpdate() {
			return nil, errors.New("coverage: formal event writer rejected input")
		}
		if bytes.Equal(before, st.NativeMapEventGrid) {
			return states, nil
		}
		key := string(st.NativeMapEventGrid)
		if seen[key] {
			return nil, errors.New("coverage: event writer did not converge")
		}
		seen[key] = true
		words := make([]uint16, n)
		for i := range words {
			words[i] = binary.LittleEndian.Uint16(st.NativeMapEventGrid[4+4*i : 6+4*i])
		}
		states = append(states, words)
	}
	return nil, errors.New("coverage: event writer exceeded bounded passes")
}

func coverageRequired(data mapData, count int) (map[[2]int]coverageSelector, int, int, error) {
	states, err := coverageStates(data)
	if err != nil {
		return nil, 0, 0, err
	}
	selected := map[[2]int]coverageSelector{}
	checks := 0
	for pass, words := range states {
		for cell, word := range words {
			tile := int(word & 1023)
			if tile*4 >= len(data.Controls) {
				return nil, 0, 0, errors.New("coverage: missing control")
			}
			for flip := 0; flip < 2; flip++ {
				for cycle := 0; cycle < 4; cycle++ {
					frame, err := fdicon.NativeTerrainFrameIndex(tile, data.Controls[tile*4], flip, cycle)
					if err != nil || frame < 0 || frame >= count {
						return nil, 0, 0, errors.New("coverage: terrain selector exceeds actual bank")
					}
					witness := coverageSelector{word, cell, flip, cycle, pass}
					key := [2]int{frame, 0}
					if _, ok := selected[key]; !ok {
						selected[key] = witness
					}
					frame, present, err := fdicon.NativeForegroundFrameIndex(tile, data.Controls[tile*4], flip)
					if err != nil || present && (frame < 0 || frame >= count) {
						return nil, 0, 0, errors.New("coverage: foreground selector exceeds actual bank")
					}
					if present {
						key = [2]int{frame, 1}
						if _, ok := selected[key]; !ok {
							selected[key] = witness
						}
					}
					checks++
				}
			}
		}
	}
	return selected, len(states) - 1, checks, nil
}

func coverageCatalog(path, root string, manifest artManifest, loaded []loadedMap) (map[[3]int]coverageHD, map[string]map[string]coverageHD, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, "", err
	}
	var catalog struct {
		Schema int    `json:"schema_version"`
		Theme  string `json:"theme_id"`
		Assets []struct {
			ID       string   `json:"asset_id"`
			Role     string   `json:"role"`
			Status   string   `json:"status"`
			Map      int      `json:"map_id"`
			Count    int      `json:"frame_count"`
			Width    int      `json:"width"`
			Height   int      `json:"height"`
			Contract string   `json:"consumer_contract"`
			Alpha    string   `json:"alpha_contract"`
			Cycle    string   `json:"cycle_policy"`
			IDs      []int    `json:"frame_ids"`
			Roles    []string `json:"semantics"`
			Files    []string `json:"files"`
			Hashes   []string `json:"frame_sha256"`
		} `json:"assets"`
	}
	if err = json.Unmarshal(raw, &catalog); err != nil {
		return nil, nil, "", err
	}
	if catalog.Schema != 1 || catalog.Theme != "modern-handpainted-a" {
		return nil, nil, "", errors.New("coverage: unsupported catalog")
	}
	registered := map[[3]int]coverageHD{}
	byContent := map[string]map[string]coverageHD{}
	pictures := map[string]image.Image{}
	for _, a := range catalog.Assets {
		if a.Role != "native_terrain_hd_set" {
			continue
		}
		if a.Map < 0 || a.Map >= len(loaded) || a.Count <= 0 || len(a.IDs) != a.Count || len(a.Roles) != a.Count || len(a.Files) != a.Count || len(a.Hashes) != a.Count ||
			a.Contract != "fdshap_native_hd_sparse_v1" || a.Alpha != "native_mask_nearest_6" || a.Cycle != "native_selector" || a.Width != 144 || a.Height != 144 ||
			(a.Status != "runtime_candidate" && a.Status != "runtime_ready") {
			return nil, nil, "", errors.New("coverage: invalid HD set contract")
		}
		for i, frame := range a.IDs {
			if frame < 0 || frame >= len(loaded[a.Map].Bank.Sprites) || filepath.Base(a.Files[i]) != a.Files[i] || filepath.Ext(a.Files[i]) != ".png" {
				return nil, nil, "", errors.New("coverage: invalid HD frame or path")
			}
			roles := []int{0}
			if a.Roles[i] == "foreground" {
				roles = []int{1}
			} else if a.Roles[i] == "both" {
				roles = []int{0, 1}
			} else if a.Roles[i] != "terrain" {
				return nil, nil, "", errors.New("coverage: unknown HD semantics")
			}
			proof := coverageHD{a.ID, a.Map, frame, a.Files[i], a.Hashes[i]}
			key := a.Files[i] + "/" + a.Hashes[i]
			pic := pictures[key]
			if pic == nil {
				png, err := os.ReadFile(filepath.Join(root, a.Files[i]))
				if err != nil {
					return nil, nil, "", err
				}
				if digest(png) != a.Hashes[i] {
					return nil, nil, "", errors.New("coverage: HD SHA mismatch")
				}
				var format string
				pic, format, err = image.Decode(bytes.NewReader(png))
				if err != nil || format != "png" || pic.Bounds() != image.Rect(0, 0, 144, 144) {
					return nil, nil, "", errors.New("coverage: invalid HD PNG")
				}
				pictures[key] = pic
			}
			sprite := loaded[a.Map].Bank.Sprites[frame]
			for _, role := range roles {
				identity := [3]int{a.Map, frame, role}
				if _, ok := registered[identity]; ok {
					return nil, nil, "", errors.New("coverage: duplicate HD slot")
				}
				for y := 0; y < 144; y++ {
					for x := 0; x < 144; x++ {
						n := (y/6)*24 + x/6
						if role == 0 && (sprite.Mask[n] == 0 || sprite.RemapMask[n] != 0) {
							return nil, nil, "", errors.New("coverage: HD base is not opaque")
						}
						_, _, _, alpha := pic.At(x, y).RGBA()
						want := uint32(0)
						if sprite.Mask[n] != 0 {
							want = 65535
						}
						if alpha != want {
							return nil, nil, "", errors.New("coverage: HD alpha differs from native mask")
						}
					}
				}
				registered[identity] = proof
				content := fmt.Sprintf("%s/%d", manifest.Maps[a.Map].Frames[frame].ContentHash, role)
				if byContent[content] == nil {
					byContent[content] = map[string]coverageHD{}
				}
				if old, ok := byContent[content][proof.SHA]; !ok || proof.Map < old.Map || proof.Map == old.Map && proof.Frame < old.Frame {
					byContent[content][proof.SHA] = proof
				}
			}
		}
	}
	return registered, byContent, digest(raw), nil
}

func buildCoverage(manifest artManifest, loaded []loadedMap, catalog, modern string) (coverageReport, error) {
	hd, byContent, hash, err := coverageCatalog(catalog, modern, manifest, loaded)
	report := coverageReport{Schema: 1, Kind: "fd2_hd1_terrain_coverage", CatalogSHA: hash, Palette: manifest.Palette,
		Scope: "33 native maps: initial selectors and active-event writer upper bound; catalog PNG/SHA/native-mask verified. Not normal reachability, original parity, or complete HD.", Summary: map[string]int{}}
	if err != nil {
		return report, err
	}
	for mapID, lm := range loaded {
		src := manifest.Maps[mapID]
		selected, passes, checks, err := coverageRequired(lm.Data, len(lm.Bank.Sprites))
		if err != nil {
			return report, fmt.Errorf("map%d: %w", mapID, err)
		}
		row := coverageMap{Map: mapID, Bank: src.Bank, Data: src.MapData, InitialTerrain: src.TerrainFrames, InitialForeground: src.ForegroundFrames, EventPasses: passes, SelectorChecks: checks}
		keys := make([][2]int, 0, len(selected))
		for key := range selected {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			if keys[i][1] != keys[j][1] {
				return keys[i][1] < keys[j][1]
			}
			return keys[i][0] < keys[j][0]
		})
		for _, key := range keys {
			frame, role := key[0], key[1]
			sprite := lm.Bank.Sprites[frame]
			f := coverageFrame{Frame: frame, Role: []string{"terrain", "foreground"}[role], Content: src.Frames[frame].ContentHash, Status: "missing-art", Selector: selected[key]}
			for _, v := range sprite.Mask {
				if v != 0 {
					f.Writes++
				}
			}
			for _, v := range sprite.RemapMask {
				if v != 0 {
					f.Remaps++
				}
			}
			if f.Writes == 0 {
				f.Status = "no-source-write"
				if role == 0 && f.Remaps > 0 {
					f.Status = "native-remap-only"
				}
			}
			if role == 0 && f.Writes > 0 && (f.Writes < 576 || f.Remaps > 0) {
				f.Status = "missing-composite"
			}
			if proof, ok := hd[[3]int{mapID, frame, role}]; ok {
				f.Status = "registered-hd"
				f.HD = &proof
			}
			if f.Status == "missing-art" {
				matches := byContent[fmt.Sprintf("%s/%d", f.Content, role)]
				if len(matches) == 1 {
					for _, proof := range matches {
						f.Reuse = &proof
					}
					report.Summary["reusable_slots"]++
				}
				if len(matches) > 1 {
					for sha := range matches {
						f.Conflicts = append(f.Conflicts, sha)
					}
					sort.Strings(f.Conflicts)
				}
			}
			report.Summary[f.Status]++
			report.Summary["required_slots"]++
			row.Frames = append(row.Frames, f)
		}
		report.Summary["selector_checks"] += checks
		report.Maps = append(report.Maps, row)
	}
	report.Summary["maps"] = len(report.Maps)
	missing := map[string]bool{}
	for _, row := range report.Maps {
		for _, frame := range row.Frames {
			if frame.Status == "missing-art" || frame.Status == "missing-composite" {
				missing[frame.Content] = true
			}
		}
	}
	report.Summary["missing_unique_native_content"] = len(missing)
	return report, nil
}

func runCoverage(pack, maps, catalog, modern, out string) error {
	if pack == "" || maps == "" || catalog == "" || modern == "" || out == "" {
		return errors.New("coverage requires -pack, -maps, -catalog, -modern-root and -coverage-out")
	}
	if _, err := os.Lstat(out); !os.IsNotExist(err) {
		return errors.New("coverage output must be new")
	}
	manifest, loaded, _, err := loadSources(pack, maps)
	if err != nil {
		return err
	}
	report, err := buildCoverage(manifest, loaded, catalog, modern)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(append(raw, '\n'))
	closed := f.Close()
	if err != nil {
		os.Remove(out)
		return err
	}
	if closed != nil {
		os.Remove(out)
		return closed
	}
	fmt.Printf("HD1 terrain coverage: %v\n", report.Summary)
	return nil
}
