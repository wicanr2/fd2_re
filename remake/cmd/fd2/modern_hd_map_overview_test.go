package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func TestModernHDMapOverviewGPU(t *testing.T) {
	if os.Getenv("FD2_HD_MAP_OVERVIEW_GPU") != "1" {
		t.Skip("explicit current map overview capture")
	}
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	old := userDataDirCached
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = old })
	out := os.Getenv("FD2_HD_MAP_OVERVIEW_OUT")
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("fresh overview output required")
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	catalog, err := os.ReadFile(os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"))
	if err != nil {
		t.Fatal(err)
	}
	coveragePath := os.Getenv("FD2_HD_MAP_OVERVIEW_COVERAGE")
	if coveragePath == "" {
		coveragePath = "../../../docs/data/fd2-hd1-terrain-coverage.json"
	}
	coverage, err := os.ReadFile(coveragePath)
	if err != nil {
		t.Fatal(err)
	}
	set := modifierHDTestTheme(t)
	probe := &sharedGroundGPUProbe{set: set, out: out}
	detailHash := ""
	if path := os.Getenv("FD2_HD_MAP_DETAIL_MANIFEST"); path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var detail struct {
			Kind       string `json:"kind"`
			CatalogSHA string `json:"catalog_sha256"`
			Bindings   []struct {
				Map int `json:"map"`
				partialTerrainSource
			} `json:"bindings"`
		}
		if err = json.Unmarshal(raw, &detail); err != nil || detail.Kind != "fd2_cave_floor_details_hd" || len(detail.Bindings) != 12 || detail.CatalogSHA != fmt.Sprintf("%x", sha256.Sum256(catalog)) {
			t.Fatal("invalid explicit detail batch", err)
		}
		detailHash = fmt.Sprintf("%x", sha256.Sum256(raw))
		a, err := loadNativeMapAssets("assets/maps/map0")
		if err != nil {
			t.Fatal(err)
		}
		for _, binding := range detail.Bindings {
			bank, controls, err := fdicon.LoadSeparatedFDSHAPBank(separatedAssetPath("tilesets/fdshap"), binding.Map)
			if err != nil {
				t.Fatal(err)
			}
			if err = set.validateTerrainHD(binding.Map, bank); err != nil {
				t.Fatal(err)
			}
			g := &Game{modernStoryPortraits: set, nativeMapAssets: &nativeMapAssets{MapIndex: binding.Map, Palette: a.Palette}}
			for _, mode := range []string{"steady", "lut", "black", "white"} {
				palette := append(color.Palette(nil), a.Palette...)
				if mode == "black" || mode == "white" {
					value := byte(0)
					if mode == "white" {
						value = 255
					}
					for i := range palette {
						palette[i] = color.RGBA{value, value, value, 255}
					}
				}
				in := partialTerrainInput(a, bank, controls, binding.partialTerrainSource, mode != "lut")
				frame, err := g.prepareModernHDMapFrame(in, make([]byte, indexedmap.NativeUnitPresentWorkSize), palette)
				if err != nil {
					t.Fatal(err)
				}
				seen := false
				for _, tile := range frame.Terrain {
					seen = seen || tile.Frame == binding.Frame && tile.Image != nil
				}
				if !seen {
					t.Fatal("detail source absent", binding.Map, binding.Frame)
				}
				probe.cases = append(probe.cases, sharedGroundGPUCase{g: g, frame: frame, name: fmt.Sprintf("details-%02d-%04d-%s", binding.Map, binding.Frame, mode), mapID: binding.Map, source: binding.Frame})
			}
		}
	}
	detailCases := len(probe.cases)
	var games []*Game
	var saves, units [][]byte
	var rngs []uint16
	var states []map[string]any
	for _, mapID := range []int{0, 1, 9, 24, 25, 26} {
		g, _, _ := ordinaryCommand0HDTestGame(t, func() time.Time { return time.Unix(1000, 0) })
		mapPath := fmt.Sprintf("assets/maps/map%d", mapID)
		scenario := fmt.Sprintf("assets/scenarios/ch%02d.json", mapID+1)
		scnRaw, err := os.ReadFile(filepath.Join("../..", scenario))
		if err != nil {
			t.Fatal(err)
		}
		var scn struct {
			Map     int    `json:"map"`
			Chapter int    `json:"chapter"`
			Name    string `json:"name"`
		}
		if err = json.Unmarshal(scnRaw, &scn); err != nil || scn.Map != mapID || scn.Chapter != mapID+1 {
			t.Fatal("scenario/map mismatch", mapID, err)
		}
		if err = g.loadMap(mapPath); err != nil {
			t.Fatal(err)
		}
		g.resetBattle(fmt.Sprintf("%s/map%d_units.json", mapPath, mapID), scenario)
		if g.loadErr != "" {
			t.Fatal(g.loadErr)
		}
		mapRaw, err := os.ReadFile(filepath.Join("../..", mapPath, "map.json"))
		if err != nil {
			t.Fatal(err)
		}
		var data struct {
			W        int      `json:"w"`
			H        int      `json:"h"`
			Tiles    []uint16 `json:"tiles"`
			Controls []byte   `json:"native_terrain_control"`
		}
		if err = json.Unmarshal(mapRaw, &data); err != nil || data.W < 13 || data.H < 8 || len(data.Tiles) != data.W*data.H {
			t.Fatal("invalid map geometry", mapID, err)
		}
		counts := make([]int, len(data.Tiles))
		for i, word := range data.Tiles {
			tile := int(word & 1023)
			if tile*4 >= len(data.Controls) {
				t.Fatal("control missing", mapID, tile)
			}
			frame, err := fdicon.NativeTerrainFrameIndex(tile, data.Controls[tile*4], 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			if set.terrainHD[mapID][frame].supports("terrain") {
				counts[i] = 1
			}
		}
		best, camX, camY := -1, 0, 0
		for cy := 0; cy <= data.H-8; cy++ {
			for cx := 0; cx <= data.W-13; cx++ {
				score := 0
				for y := cy; y < cy+8; y++ {
					for x := cx; x < cx+13; x++ {
						score += counts[y*data.W+x] * 1000
					}
				}
				for _, u := range g.st.Units {
					if u != nil && u.HP > 0 && u.X >= cx && u.X < cx+13 && u.Y >= cy && u.Y < cy+8 {
						score++
					}
				}
				if score > best {
					best, camX, camY = score, cx, cy
				}
			}
		}
		v := g.st.NativeMapViewState
		v.CameraX, v.CameraY = camX, camY
		v.CursorX, v.CursorY, v.VisibleCursorX, v.VisibleCursorY = camX+6, camY+4, 6, 4
		if err = g.st.MaterializeNativeMapViewState(v); err != nil {
			t.Fatal(err)
		}
		g.curX, g.curY = v.CursorX, v.CursorY
		if !g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
			t.Fatal("missing overview HUD state")
		}
		g.interfaceMode = playerInterfaceHD
		g.camp = campaign.NewRunner(&campaign.Campaign{Start: "overview_fixture", Nodes: map[string]*campaign.Node{"overview_fixture": {Type: "battle"}}})
		if err = g.applyTheme(set); err != nil {
			t.Fatal(err)
		}
		g.modernHDMapComposing = true
		if err = g.composeNativeMapFrame(); err != nil {
			t.Fatal(err)
		}
		anchor, painted := -1, 0
		for _, tile := range g.modernHDMapFrame.Terrain {
			if tile.Image != nil {
				anchor, painted = tile.Frame, painted+1
			}
		}
		if anchor < 0 {
			t.Fatal("overview has no HD terrain", mapID)
		}
		if err = g.saveGameToSlot(0); err != nil {
			t.Fatal(err)
		}
		saved, err := os.ReadFile(saveSlotPath(0))
		if err != nil {
			t.Fatal(err)
		}
		g.msg = ""
		state, _ := json.Marshal(g.st.Units)
		games, saves, units = append(games, g), append(saves, saved), append(units, state)
		rngs = append(rngs, g.nativeRNGState)
		states = append(states, map[string]any{"map": mapID, "chapter": scn.Chapter, "name": scn.Name,
			"camera": []int{camX, camY}, "cursor": []int{v.CursorX, v.CursorY}, "painted_terrain_draws": painted,
			"map_sha256": fmt.Sprintf("%x", sha256.Sum256(mapRaw)), "scenario_sha256": fmt.Sprintf("%x", sha256.Sum256(scnRaw)), "save_bytes": len(saved)})
		probe.cases = append(probe.cases, sharedGroundGPUCase{g: g, name: fmt.Sprintf("overview-map%02d", mapID), mapID: mapID, source: anchor, keepPNG: true})
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err = ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	if probe.err != nil || probe.index != detailCases+6 {
		t.Fatal(probe.index, probe.err)
	}
	for i, g := range games {
		after, _ := json.Marshal(g.st.Units)
		if !bytes.Equal(after, units[i]) || rngs[i] != g.nativeRNGState {
			t.Fatal("overview altered units or RNG", i)
		}
		if err = g.saveGameToSlot(0); err != nil {
			t.Fatal(err)
		}
		after, err := os.ReadFile(saveSlotPath(0))
		if err != nil || !bytes.Equal(after, saves[i]) {
			t.Fatal("overview altered save", i, err)
		}
	}
	code, count := hdConsumerSourceTree(t)
	report := map[string]any{"status": "PASS-HD-MAP-OVERVIEW-GPU", "catalog_sha256": fmt.Sprintf("%x", sha256.Sum256(catalog)),
		"coverage_sha256": fmt.Sprintf("%x", sha256.Sum256(coverage)), "source_tree_sha256": code, "source_file_count": count,
		"detail_manifest_sha256": detailHash, "detail_source_cases": detailCases,
		"maps": states, "cases": probe.rows, "scope": "six current Game.Draw captures; controlled cameras, authored units unchanged; programmatic F2 and boundary saves; not normal player or original parity"}
	encoded, _ := json.MarshalIndent(report, "", "  ")
	if err = os.WriteFile(filepath.Join(out, "receipt.json"), append(encoded, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
