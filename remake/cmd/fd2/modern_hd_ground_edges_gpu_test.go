package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func TestModernHDGroundEdgesRealBanksAndGPU(t *testing.T) {
	if os.Getenv("FD2_HD_GROUND_EDGES_GPU") != "1" {
		t.Skip("dedicated real-bank ground-edge process")
	}
	t.Setenv("FD2_MUTE", "1")
	out := os.Getenv("FD2_HD_GROUND_EDGES_OUT")
	if out == "" {
		t.Fatal("fresh output required")
	}
	if _, e := os.Stat(out); !os.IsNotExist(e) {
		t.Fatal("fresh output required")
	}
	if e := os.Mkdir(out, 0755); e != nil {
		t.Fatal(e)
	}
	catalogPath := os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG")
	catalogInput, e := os.ReadFile(catalogPath)
	if e != nil {
		t.Fatal(e)
	}
	recipePath := "../../assets/themes/modern/ground-grass-earth-hd.recipe.json"
	recipeInput, e := os.ReadFile(recipePath)
	if e != nil {
		t.Fatal(e)
	}
	var recipe struct {
		Groups []struct {
			Anchor  [2]int   `json:"anchor"`
			Targets [][2]int `json:"targets"`
			File    string   `json:"file"`
		} `json:"groups"`
	}
	if e = json.Unmarshal(recipeInput, &recipe); e != nil {
		t.Fatal(e)
	}
	if len(recipe.Groups) != 14 {
		t.Fatal("complete edge groups required")
	}
	set := modifierHDTestTheme(t)
	g, _, _ := ordinaryCommand0HDTestGame(t, func() time.Time { return time.Unix(1000, 0) })
	palette := g.nativeMapAssets.Palette
	byMap := map[int][]int{}
	painted := map[string]string{}
	targets := 0
	for _, group := range recipe.Groups {
		asset := set.terrainHD[0][group.Anchor[1]]
		if asset == nil {
			t.Fatal("missing source anchor", group.Anchor)
		}
		painted[group.File] = fmt.Sprintf("%x", sha256.Sum256(asset.Image.Pix))
		for _, target := range group.Targets {
			byMap[target[0]] = append(byMap[target[0]], target[1])
			targets++
		}
	}
	if targets != 141 || len(byMap) != 11 {
		t.Fatal("incomplete real source scope", targets, len(byMap))
	}
	mapIDs := []int{}
	for id := range byMap {
		mapIDs = append(mapIDs, id)
	}
	sort.Ints(mapIDs)
	p := &sharedGroundGPUProbe{set: set, out: out}
	sourceGame := &Game{modernStoryPortraits: set}
	for _, mapID := range mapIDs {
		frames := byMap[mapID]
		sort.Ints(frames)
		bank, controls, e := fdicon.LoadSeparatedFDSHAPBank(separatedAssetPath("tilesets/fdshap"), mapID)
		if e != nil {
			t.Fatal(e)
		}
		if e = set.validateTerrainHD(mapID, bank); e != nil {
			t.Fatal(mapID, e)
		}
		bad := *bank
		bad.Sprites = append(bank.Sprites[:0:0], bank.Sprites...)
		bad.Sprites[frames[0]].Mask = append([]byte(nil), bank.Sprites[frames[0]].Mask...)
		bad.Sprites[frames[0]].Mask[0] = 0
		if e = set.validateTerrainHD(mapID, &bad); e == nil {
			t.Fatal("wrong native mask admitted", mapID)
		}
		var data struct {
			Tiles []uint16 `json:"tiles"`
			Modes []byte   `json:"native_tile_blit_modes"`
		}
		raw, e := os.ReadFile(fmt.Sprintf("../../assets/maps/map%d/map.json", mapID))
		if e != nil {
			t.Fatal(e)
		}
		if e = json.Unmarshal(raw, &data); e != nil {
			t.Fatal(e)
		}
		if len(data.Modes) != len(data.Tiles) {
			t.Fatal("native blit modes missing", mapID)
		}
		cells := []fdicon.NativeTerrainCell{}
		for _, frame := range frames {
			if set.terrainHD[mapID][frame] == nil {
				t.Fatal("missing real frame", mapID, frame)
			}
			found := false
			for i, word := range data.Tiles {
				idx, e := fdicon.NativeTerrainFrameIndex(int(word&0x3ff), controls[int(word&0x3ff)*4], 0, 0)
				if e == nil && idx == frame {
					cells = append(cells, fdicon.NativeTerrainCell{Tile: word, BlitMode: data.Modes[i]})
					found = true
					break
				}
			}
			if !found {
				t.Fatal("no actual steady selector for edge", mapID, frame)
			}
		}
		in := indexedmap.FrameInput{MapWidth: 13, Cells: make([]fdicon.NativeTerrainCell, 13*12), Controls: controls}
		for i := range in.Cells {
			in.Cells[i] = cells[i%len(cells)]
		}
		for _, mode := range []string{"steady", "up1", "up6", "lut", "black", "white"} {
			lut := make([]byte, 256)
			current := append(color.Palette(nil), palette...)
			for i := range lut {
				lut[i] = byte(i)
				if mode == "lut" {
					lut[i] = byte((i + 1) % 256)
				}
				if mode == "black" {
					current[i] = color.RGBA{0, 0, 0, 255}
				}
				if mode == "white" {
					current[i] = color.RGBA{255, 255, 255, 255}
				}
			}
			ctx, e := terrainColourContext(mapID, palette, current, lut)
			if e != nil {
				t.Fatal(e)
			}
			base, camY, rows, view := 0x8088, 0, 8, 0x8088
			if mode == "up1" || mode == "up6" {
				step := 1
				if mode == "up6" {
					step = 6
				}
				base, camY, rows, view, e = indexedmap.NativeUpStepLayout(in, 2, step)
				if e != nil {
					t.Fatal(e)
				}
			}
			plan, e := buildModernHDTerrainPlan(in, base, camY, rows, view, set.terrainHD[mapID], ctx)
			if e != nil {
				t.Fatal(e)
			}
			seen := map[int]bool{}
			for _, tile := range plan {
				seen[tile.Frame] = true
			}
			for _, frame := range frames {
				if !seen[frame] {
					t.Fatal("actual atlas plan omitted frame", mapID, mode, frame)
				}
			}
			f := &modernHDMapFrame{Background: image.NewRGBA(image.Rect(0, 0, 320, 200)), UI: image.NewRGBA(image.Rect(0, 0, 320, 200)), Terrain: plan, Viewport: modernHDMapViewport(), TerrainColourKey: ctx.Key, TransientTerrain: true}
			p.cases = append(p.cases, sharedGroundGPUCase{g: sourceGame, frame: f, name: fmt.Sprintf("map%02d-%s", mapID, mode), mapID: mapID, source: frames[0], keepPNG: mapID == 0 && (mode == "steady" || mode == "up1" || mode == "lut")})
		}
	}
	v := g.st.NativeMapViewState
	v.CameraX, v.CameraY = 6, 15
	v.CursorX, v.VisibleCursorX = 6, 6
	v.CursorY, v.VisibleCursorY = 22, 22
	if e = g.st.MaterializeNativeMapViewState(v); e != nil {
		t.Fatal(e)
	}
	g.curX, g.curY = v.CursorX, v.CursorY
	if e = g.applyTheme(set); e != nil {
		t.Fatal(e)
	}
	g.modernHDMapComposing = true
	if e = g.composeNativeMapFrame(); e != nil {
		t.Fatal(e)
	}
	visible := map[int]bool{}
	for _, tile := range g.modernHDMapFrame.Terrain {
		for _, frame := range byMap[0] {
			if tile.Frame == frame {
				visible[frame] = true
			}
		}
	}
	if len(visible) < 8 {
		t.Fatal("real Game.Draw has insufficient new edges", len(visible))
	}
	p.cases = append(p.cases, sharedGroundGPUCase{g: g, name: "game-map00", mapID: 0, source: 145, keepPNG: true})
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if e = ebiten.RunGame(p); e != nil {
		t.Fatal(e)
	}
	if p.err != nil || p.index != 67 {
		t.Fatalf("edge GPU %d/67: %v", p.index, p.err)
	}
	for i, c := range p.cases {
		frame := c.frame
		if frame == nil {
			frame = c.g.modernHDMapPresented
		}
		seen := map[int]bool{}
		for _, tile := range frame.Terrain {
			for _, source := range byMap[c.mapID] {
				if tile.Frame == source {
					seen[source] = true
				}
			}
		}
		observed := []int{}
		for source := range seen {
			observed = append(observed, source)
		}
		sort.Ints(observed)
		p.rows[i]["source_frames"] = observed
	}
	current, e := os.ReadFile(catalogPath)
	if e != nil || !bytes.Equal(current, catalogInput) {
		t.Fatal("catalog changed during edge GPU", e)
	}
	report := map[string]any{"status": "PASS-GROUND-EDGES-GPU", "catalog_sha256": fmt.Sprintf("%x", sha256.Sum256(catalogInput)), "recipe_sha256": fmt.Sprintf("%x", sha256.Sum256(recipeInput)), "painted_frames_rgba": painted, "maps": 11, "source_groups": 14, "target_slots": targets, "source_worlds": 66, "game_draws": 1, "f2": 1, "wrong_bank_masks_rejected": 11, "native_rng_seed": 7, "fixed_clock_unix": 1000, "cases": p.rows, "scope": "141個實際bank／selector來源；十一圖完整交界圖源測具與一個原樣map0 Game.Draw／F2。非十一章正常玩家或原版RGB對拍。"}
	raw, e := json.MarshalIndent(report, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(out, "gpu.json"), append(raw, '\n'), 0644); e != nil {
		t.Fatal(e)
	}
}
