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

func TestModernHDExactTerrainReuseSourceTransactions(t *testing.T) {
	if os.Getenv("FD2_HD_TERRAIN_REUSE_CPU") != "1" {
		t.Skip("explicit real-pack CPU process")
	}
	raw, err := os.ReadFile(os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog modernThemeCatalog
	if err = json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	set := &modernStoryPortraitSet{}
	for _, a := range catalog.Assets {
		if a.Role == "native_terrain_hd_set" {
			if err = loadModernTerrainHDAsset(set, a, os.Getenv("FD2_TEST_MODERN_EFFECT_PACK")); err != nil {
				t.Fatal(err)
			}
		}
	}
	a, err := loadNativeMapAssets("assets/maps/map0")
	if err != nil {
		t.Fatal(err)
	}
	bank, controls, err := fdicon.LoadSeparatedFDSHAPBank(separatedAssetPath("tilesets/fdshap"), 1)
	if err != nil {
		t.Fatal(err)
	}
	in := indexedmap.NativeFrameInput{Frame: indexedmap.FrameInput{MapWidth: 13, Cells: make([]fdicon.NativeTerrainCell, 13*9), Controls: controls,
		TerrainBank: bank, ForegroundBank: bank, UnitBank: a.Units, RangeBank: a.Range, SelectorCache: &fdicon.NativeSelectorCache{}, LUT: modernHDIdentityLUT(),
		ForegroundUnits: []fdicon.NativeForegroundLayerEntry{{X: 6, Y: 4}}}, Frames: a.Frames, HUDTerrain: bank, HUDUnits: a.Units, HUDCache: &fdicon.NativeSelectorCache{}}
	for i := range in.Frame.Cells {
		in.Frame.Cells[i].Tile = 176
	}
	g := &Game{modernStoryPortraits: set, nativeMapAssets: &nativeMapAssets{MapIndex: 1, Palette: a.Palette}}
	seed := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	before, _ := json.Marshal(in.Frame.Cells)
	frame, err := g.prepareModernHDMapFrame(in, seed, a.Palette)
	if err != nil {
		t.Fatal(err)
	}
	if len(frame.Foreground) != 2 || frame.Foreground[0].SourceFrame != 177 || frame.Foreground[0].HDImage == nil {
		t.Fatal("new foreground not consumed")
	}
	after, _ := json.Marshal(in.Frame.Cells)
	if !bytes.Equal(before, after) || !bytes.Equal(seed, make([]byte, len(seed))) {
		t.Fatal("source transaction mutated input")
	}
}

func TestModernHDExactTerrainReuseRealConsumersGPU(t *testing.T) {
	if os.Getenv("FD2_HD_TERRAIN_REUSE_GPU") != "1" {
		t.Skip("dedicated exact terrain consumer process")
	}
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	oldCache := userDataDirCached
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = oldCache })
	manifestPath, out := os.Getenv("FD2_HD_TERRAIN_REUSE_MANIFEST"), os.Getenv("FD2_HD_TERRAIN_REUSE_OUT")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		CatalogSHA string `json:"catalog_sha256"`
		Bindings   []struct {
			Map    int    `json:"map"`
			Frame  int    `json:"frame"`
			Role   string `json:"role"`
			Source struct {
				Map   int `json:"map"`
				Frame int `json:"frame"`
			} `json:"reuse_from"`
			Selector struct {
				Word  uint16 `json:"tile_word"`
				Cell  int    `json:"cell"`
				Flip  int    `json:"flip"`
				Cycle int    `json:"cycle"`
				Pass  int    `json:"event_update_pass"`
			} `json:"selector"`
		} `json:"bindings"`
	}
	if err = json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	if len(receipt.Bindings) != 46 {
		t.Fatal("complete fixed reuse scope required")
	}
	if _, err = os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("fresh output required")
	}
	if err = os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	catalog, err := os.ReadFile(os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"))
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(catalog)) != receipt.CatalogSHA {
		t.Fatal("candidate source receipt changed")
	}
	set := modifierHDTestTheme(t)
	g, _, _ := ordinaryCommand0HDTestGame(t, func() time.Time { return time.Unix(1000, 0) })
	palette := g.nativeMapAssets.Palette
	template, err := g.nativeCh28PostFrameInput(g.st)
	if err != nil {
		t.Fatal(err)
	}
	probe := &sharedGroundGPUProbe{set: set, out: out}
	banks, controls := map[int]*fdicon.Bank{}, map[int][]byte{}
	sourceGames := map[int]*Game{}
	for _, binding := range receipt.Bindings {
		for _, id := range []int{binding.Map, binding.Source.Map} {
			if banks[id] == nil {
				banks[id], controls[id], err = fdicon.LoadSeparatedFDSHAPBank(separatedAssetPath("tilesets/fdshap"), id)
				if err != nil {
					t.Fatal(err)
				}
				if err = set.validateTerrainHD(id, banks[id]); err != nil {
					t.Fatal(err)
				}
			}
		}
		native, source := banks[binding.Map].Sprites[binding.Frame], banks[binding.Source.Map].Sprites[binding.Source.Frame]
		if !bytes.Equal(native.Pixels, source.Pixels) || !bytes.Equal(native.Mask, source.Mask) || !bytes.Equal(native.RemapMask, source.RemapMask) {
			t.Fatal("native source differs", binding)
		}
		asset, parent := set.terrainHD[binding.Map][binding.Frame], set.terrainHD[binding.Source.Map][binding.Source.Frame]
		if !asset.supports(binding.Role) || !bytes.Equal(asset.Image.Pix, parent.Image.Pix) {
			t.Fatal("HD source differs", binding)
		}
		for _, mode := range []string{"steady", "lut", "black", "white"} {
			lut, current := modernHDIdentityLUT(), append(color.Palette(nil), palette...)
			for i := range lut {
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
			in := template
			in.Frame = indexedmap.FrameInput{MapWidth: 13, Cells: make([]fdicon.NativeTerrainCell, 13*9), Controls: controls[binding.Map],
				TerrainBank: banks[binding.Map], ForegroundBank: banks[binding.Map], LUT: lut, Flip: binding.Selector.Flip, TerrainCycle: binding.Selector.Cycle,
				UnitBank: template.Frame.UnitBank, RangeBank: template.Frame.RangeBank, SelectorCache: template.Frame.SelectorCache}
			for i := range in.Frame.Cells {
				in.Frame.Cells[i] = fdicon.NativeTerrainCell{Tile: binding.Selector.Word}
			}
			if binding.Role == "foreground" {
				in.Frame.ForegroundUnits = []fdicon.NativeForegroundLayerEntry{{X: 6, Y: 4}}
			}
			sourceGame := sourceGames[binding.Map]
			if sourceGame == nil {
				sourceGame = &Game{modernStoryPortraits: set, nativeMapAssets: &nativeMapAssets{MapIndex: binding.Map, Palette: palette}}
				sourceGames[binding.Map] = sourceGame
			}
			frame, err := sourceGame.prepareModernHDMapFrame(in, make([]byte, indexedmap.NativeUnitPresentWorkSize), current)
			if err != nil {
				t.Fatal("formal map consumer", binding.Map, binding.Frame, mode, err)
			}
			seen := false
			if binding.Role == "terrain" {
				for _, tile := range frame.Terrain {
					seen = seen || tile.Frame == binding.Frame && tile.Image != nil
				}
			} else {
				for _, fore := range frame.Foreground {
					seen = seen || fore.SourceFrame == binding.Frame && fore.HDImage != nil
				}
			}
			if !seen {
				t.Fatal("formal consumer omitted HD", binding.Map, binding.Frame, binding.Role)
			}
			probe.cases = append(probe.cases, sharedGroundGPUCase{g: sourceGame, frame: frame, name: fmt.Sprintf("map%02d-%04d-%s-%s", binding.Map, binding.Frame, binding.Role, mode), mapID: binding.Map, source: binding.Frame,
				keepPNG: mode == "steady" && binding.Role == "foreground"})
		}
	}
	// The complete renderer uses map1 data and chapter2 units. Its camera is a
	// declared fixture at a new roof source, rather than a normal input claim.
	if err = g.loadMap("assets/maps/map1"); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map1/map1_units.json", "assets/scenarios/ch02.json")
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	var data struct {
		Width  int      `json:"w"`
		Height int      `json:"h"`
		Tiles  []uint16 `json:"tiles"`
	}
	mapRaw, err := os.ReadFile("../../assets/maps/map1/map.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(mapRaw, &data); err != nil {
		t.Fatal(err)
	}
	cell := 480
	x, y := cell%data.Width, cell/data.Width
	v := g.st.NativeMapViewState
	v.CameraX, v.CameraY = max(0, min(x-5, data.Width-13)), max(0, min(y-4, data.Height-8))
	v.CursorX, v.CursorY, v.VisibleCursorX, v.VisibleCursorY = x, y, x-v.CameraX, y-v.CameraY
	if err = g.st.MaterializeNativeMapViewState(v); err != nil {
		t.Fatal(err)
	}
	g.curX, g.curY = x, y
	if !g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
		t.Fatal("missing HUD fixture")
	}
	actor := g.st.Units[0]
	actor.SetMapPlacement(x, y, 0)
	if err = actor.MaterializeNativeMapPresentation(); err != nil {
		t.Fatal(err)
	}
	g.interfaceMode = playerInterfaceHD
	g.camp = campaign.NewRunner(&campaign.Campaign{Start: "terrain_fixture", Nodes: map[string]*campaign.Node{"terrain_fixture": {Type: "battle"}}})
	if err = g.applyTheme(set); err != nil {
		t.Fatal(err)
	}
	g.modernHDMapComposing = true
	if err = g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range g.modernHDMapFrame.Foreground {
		found = found || f.SourceFrame == 177 && f.HDImage != nil
	}
	if !found {
		t.Fatal("complete map1 missed new roof foreground")
	}
	if err = g.saveGameToSlot(0); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(saveSlotPath(0))
	if err != nil {
		t.Fatal(err)
	}
	g.msg = ""
	units, _ := json.Marshal(g.st.Units)
	rng := g.nativeRNGState
	probe.cases = append(probe.cases, sharedGroundGPUCase{g: g, name: "game-map01-hd-ui", mapID: 1, source: 177, keepPNG: true})
	originalUI := *g
	originalUI.interfaceMode = playerInterfaceOriginal
	originalUI.nativeMapWork = append([]byte(nil), g.nativeMapWork...)
	originalUI.nativeMapVGA = append([]byte(nil), g.nativeMapVGA...)
	originalUI.modernHDMapGPU = nil
	probe.cases = append(probe.cases, sharedGroundGPUCase{g: &originalUI, name: "game-map01-original-ui", mapID: 1, source: 177, keepPNG: true})
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err = ebiten.RunGame(probe); err != nil {
		t.Fatal(err)
	}
	if probe.err != nil || probe.index != len(probe.cases) {
		t.Fatal(probe.index, probe.err)
	}
	after, _ := json.Marshal(g.st.Units)
	if !bytes.Equal(units, after) || rng != g.nativeRNGState {
		t.Fatal("theme changed units or RNG")
	}
	if err = g.saveGameToSlot(0); err != nil {
		t.Fatal(err)
	}
	resaved, err := os.ReadFile(saveSlotPath(0))
	if err != nil || !bytes.Equal(saved, resaved) {
		t.Fatal("terrain changed boundary save", err)
	}
	report := map[string]any{"status": "PASS-EXACT-TERRAIN-REUSE-GPU", "catalog_sha256": receipt.CatalogSHA, "manifest_sha256": fmt.Sprintf("%x", sha256.Sum256(raw)),
		"bindings": len(receipt.Bindings), "source_cases": len(receipt.Bindings) * 4, "complete_game_cases": 2, "cases": probe.rows, "save_bytes_unchanged": len(saved),
		"scope": "controlled native selectors, actual terrain/foreground producer and map1 Game.Draw; theme/UI controls and saved boundary bytes; not normal chapter reachability or platform acceptance"}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(out, "receipt.json"), append(encoded, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
