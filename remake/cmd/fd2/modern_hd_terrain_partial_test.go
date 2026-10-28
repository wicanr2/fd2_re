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
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type partialTerrainSource struct {
	Frame    int    `json:"frame"`
	Role     string `json:"role"`
	Writes   int    `json:"source_pixels"`
	Remaps   int    `json:"remap_pixels"`
	Selector struct {
		Word  uint16 `json:"tile_word"`
		Flip  int    `json:"flip"`
		Cycle int    `json:"cycle"`
	} `json:"selector"`
}

func partialTerrainSources(t *testing.T) map[int][]partialTerrainSource {
	return terrainSources(t, true)
}

func terrainSources(t *testing.T, partialOnly bool) map[int][]partialTerrainSource {
	t.Helper()
	raw, err := os.ReadFile("../../../docs/data/fd2-hd1-terrain-coverage.json")
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Maps []struct {
			Map      int                    `json:"map"`
			Required []partialTerrainSource `json:"required"`
		} `json:"maps"`
	}
	if err = json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	out := map[int][]partialTerrainSource{}
	for _, m := range report.Maps {
		for _, f := range m.Required {
			if f.Role == "terrain" && (!partialOnly || f.Writes < 576) {
				out[m.Map] = append(out[m.Map], f)
			}
		}
	}
	return out
}

func partialTerrainInput(a *nativeMapAssets, bank *fdicon.Bank, controls []byte, s partialTerrainSource, raw bool) indexedmap.NativeFrameInput {
	in := indexedmap.NativeFrameInput{Frame: indexedmap.FrameInput{MapWidth: 13, Cells: make([]fdicon.NativeTerrainCell, 13*9), Controls: controls,
		TerrainBank: bank, ForegroundBank: bank, UnitBank: a.Units, RangeBank: a.Range, SelectorCache: &fdicon.NativeSelectorCache{}, LUT: modernHDIdentityLUT(), Flip: s.Selector.Flip, TerrainCycle: s.Selector.Cycle},
		Frames: a.Frames, HUDTerrain: bank, HUDUnits: a.Units, HUDCache: &fdicon.NativeSelectorCache{}}
	for i := range in.Frame.LUT {
		in.Frame.LUT[i] = byte((i + 17) % 256)
	}
	mode := byte(0)
	if raw {
		mode = 0xff
	}
	for i := range in.Frame.Cells {
		in.Frame.Cells[i] = fdicon.NativeTerrainCell{Tile: s.Selector.Word, BlitMode: mode}
	}
	return in
}

type partialBackdropCheck struct {
	sprite  fdicon.Sprite
	palette color.Palette
	seed    byte
	lut     []byte
	raw     bool
}
type partialTerrainProbe struct {
	sharedGroundGPUProbe
	backgrounds []partialBackdropCheck
	checked     int
}

func (p *partialTerrainProbe) Draw(screen *ebiten.Image) {
	i := p.index
	p.sharedGroundGPUProbe.Draw(screen)
	if p.err != nil || p.index != i+1 {
		return
	}
	c := p.backgrounds[i]
	part := p.cases[i].g.modernHDMapGPU.World.SubImage(image.Rect(24, 24, 168, 168)).(*ebiten.Image)
	pixels := make([]byte, 144*144*4)
	part.ReadPixels(pixels)
	for y := 0; y < 144; y++ {
		for x := 0; x < 144; x++ {
			n := (y/6)*24 + x/6
			if c.sprite.Mask[n] != 0 {
				continue
			}
			index := c.seed
			if !c.raw && c.sprite.RemapMask[n] != 0 {
				index = c.lut[index]
			}
			want := color.RGBAModel.Convert(c.palette[index]).(color.RGBA)
			at := (y*144 + x) * 4
			if pixels[at] != want.R || pixels[at+1] != want.G || pixels[at+2] != want.B || pixels[at+3] != 255 {
				p.err = fmt.Errorf("%s backdrop remapped twice or covered at %d,%d", p.cases[i].name, x, y)
				return
			}
			p.checked++
		}
	}
}

func TestModernHDTerrainPartialNativeMasksAndGPU(t *testing.T) {
	if os.Getenv("FD2_HD_TERRAIN_PARTIAL_GPU") != "1" {
		t.Skip("dedicated real mask GPU")
	}
	a, err := loadNativeMapAssets("assets/maps/map0")
	if err != nil {
		t.Fatal(err)
	}
	sources := partialTerrainSources(t)
	set := &modernStoryPortraitSet{terrainHD: map[int]map[int]*modernTerrainHDAsset{}}
	banks, controls := map[int]*fdicon.Bank{}, map[int][]byte{}
	count := 0
	for mapID, rows := range sources {
		banks[mapID], controls[mapID], err = fdicon.LoadSeparatedFDSHAPBank(separatedAssetPath("tilesets/fdshap"), mapID)
		if err != nil {
			t.Fatal(err)
		}
		set.terrainHD[mapID] = map[int]*modernTerrainHDAsset{}
		for _, s := range rows {
			native := banks[mapID].Sprites[s.Frame]
			pic := image.NewRGBA(image.Rect(0, 0, 144, 144))
			for y := 0; y < 144; y++ {
				for x := 0; x < 144; x++ {
					if native.Mask[(y/6)*24+x/6] != 0 {
						pic.SetRGBA(x, y, color.RGBA{37, 181, 98, 255})
					}
				}
			}
			set.terrainHD[mapID][s.Frame] = &modernTerrainHDAsset{Image: pic, Role: "terrain"}
			count++
		}
		if err = set.validateTerrainHD(mapID, banks[mapID]); err != nil {
			t.Fatal(err)
		}
	}
	if count != 306 {
		t.Fatal("all 296 partial and 10 destination-only sources required", count)
	}
	out := os.Getenv("FD2_HD_TERRAIN_PARTIAL_OUT")
	if _, err = os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("fresh output required")
	}
	if err = os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &partialTerrainProbe{sharedGroundGPUProbe: sharedGroundGPUProbe{set: set, out: out}}
	dest := byte(0)
	for i := 0; i < 256; i++ {
		if a.Palette[i] != a.Palette[(i+17)%256] && a.Palette[(i+17)%256] != a.Palette[(i+34)%256] && a.Palette[i] != a.Palette[(i+34)%256] {
			dest = byte(i)
			break
		}
	}
	for mapID := 0; mapID < 33; mapID++ {
		g := &Game{modernStoryPortraits: set, nativeMapAssets: &nativeMapAssets{MapIndex: mapID, Palette: a.Palette}}
		for _, s := range sources[mapID] {
			for _, raw := range []bool{false, true} {
				in := partialTerrainInput(a, banks[mapID], controls[mapID], s, raw)
				seed := bytes.Repeat([]byte{dest}, indexedmap.NativeUnitPresentWorkSize)
				before, _ := json.Marshal(in.Frame.Cells)
				frame, err := g.prepareModernHDMapFrame(in, seed, a.Palette)
				if err != nil {
					t.Fatal(mapID, s.Frame, err)
				}
				after, _ := json.Marshal(in.Frame.Cells)
				if !bytes.Equal(before, after) || !bytes.Equal(seed, bytes.Repeat([]byte{dest}, len(seed))) {
					t.Fatal("source transaction mutated inputs")
				}
				native := banks[mapID].Sprites[s.Frame]
				for i, mask := range native.Mask {
					want := dest
					if mask != 0 {
						want = native.Pixels[i]
						if !raw {
							want = in.Frame.LUT[want]
						}
					} else if !raw && native.RemapMask[i] != 0 {
						want = in.Frame.LUT[want]
					}
					if frame.Background.RGBAAt(4+i%24, 4+i/24) != color.RGBAModel.Convert(a.Palette[want]).(color.RGBA) {
						t.Fatal("native destination contract differs", mapID, s.Frame, i)
					}
				}
				p.cases = append(p.cases, sharedGroundGPUCase{g: g, frame: frame, name: fmt.Sprintf("partial-map%02d-%04d-raw%v", mapID, s.Frame, raw), mapID: mapID, source: s.Frame})
				p.backgrounds = append(p.backgrounds, partialBackdropCheck{native, a.Palette, dest, in.Frame.LUT, raw})
			}
		}
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err = ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || p.index != 612 {
		t.Fatal(p.index, p.err)
	}
	report := map[string]any{"status": "PASS-PARTIAL-TERRAIN-MASK-GPU", "source_frames": 306, "partial_source_frames": 296, "destination_only_frames": 10, "gpu_cases": p.index, "backdrop_hd_pixels_checked": p.checked, "seed_index": dest, "cases": p.rows, "scope": "real native banks with synthetic RGB markers; no new art/normal reachability/original parity"}
	encoded, _ := json.MarshalIndent(report, "", "  ")
	if err = os.WriteFile(filepath.Join(out, "receipt.json"), append(encoded, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestModernHDCaveRockActualGameGPU(t *testing.T) {
	if os.Getenv("FD2_HD_CAVE_ROCK_GPU") != "1" {
		t.Skip("dedicated real cave art GPU")
	}
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	old := userDataDirCached
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = old })
	manifestPath, out := os.Getenv("FD2_HD_CAVE_ROCK_MANIFEST"), os.Getenv("FD2_HD_CAVE_ROCK_OUT")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Kind       string `json:"kind"`
		CatalogSHA string `json:"catalog_sha256"`
		Frames     []struct {
			Frame int `json:"frame"`
		} `json:"frames"`
	}
	if err = json.Unmarshal(raw, &manifest); err != nil || len(manifest.Frames) != 16 {
		t.Fatal(err)
	}
	firstFrame, status := 32, "PASS-CAVE-ROCK-HD-GPU"
	if manifest.Kind == "fd2_cave_floor_hd" {
		firstFrame, status = 80, "PASS-CAVE-FLOOR-HD-GPU"
	} else if manifest.Kind != "fd2_cave_rock_partial_hd" {
		t.Fatal("unknown cave batch", manifest.Kind)
	}
	for i, frame := range manifest.Frames {
		if frame.Frame != firstFrame+i {
			t.Fatal("incomplete cave source range")
		}
	}
	catalog, err := os.ReadFile(os.Getenv("FD2_TEST_MODERN_EFFECT_CATALOG"))
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(catalog)) != manifest.CatalogSHA {
		t.Fatal("candidate catalog changed", err)
	}
	if _, err = os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("fresh output required")
	}
	if err = os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	set := modifierHDTestTheme(t)
	g, _, _ := ordinaryCommand0HDTestGame(t, func() time.Time { return time.Unix(1000, 0) })
	a := g.nativeMapAssets
	bank, controls, err := fdicon.LoadSeparatedFDSHAPBank(separatedAssetPath("tilesets/fdshap"), 9)
	if err != nil {
		t.Fatal(err)
	}
	p := &sharedGroundGPUProbe{set: set, out: out}
	sourceGame := &Game{modernStoryPortraits: set, nativeMapAssets: &nativeMapAssets{MapIndex: 9, Palette: a.Palette}}
	var first partialTerrainSource
	for _, s := range terrainSources(t, false)[9] {
		if s.Frame < firstFrame || s.Frame >= firstFrame+16 {
			continue
		}
		if s.Frame == firstFrame {
			first = s
		}
		if !set.terrainHD[9][s.Frame].supports("terrain") {
			t.Fatal("cave source missing", s.Frame)
		}
		for _, raw := range []bool{false, true} {
			in := partialTerrainInput(a, bank, controls, s, raw)
			frame, err := sourceGame.prepareModernHDMapFrame(in, make([]byte, indexedmap.NativeUnitPresentWorkSize), a.Palette)
			if err != nil {
				t.Fatal(err)
			}
			p.cases = append(p.cases, sharedGroundGPUCase{g: sourceGame, frame: frame, name: fmt.Sprintf("cave-%04d-raw%v", s.Frame, raw), mapID: 9, source: s.Frame})
		}
	}
	if len(p.cases) != 32 {
		t.Fatal("all actual cave sources required")
	}
	if err = g.loadMap("assets/maps/map9"); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map9/map9_units.json", "assets/scenarios/ch10.json")
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	var data struct {
		Width  int      `json:"w"`
		Height int      `json:"h"`
		Tiles  []uint16 `json:"tiles"`
	}
	mapRaw, err := os.ReadFile("../../assets/maps/map9/map.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(mapRaw, &data); err != nil {
		t.Fatal(err)
	}
	at := -1
	for i, word := range data.Tiles {
		if word == first.Selector.Word {
			at = i
			break
		}
	}
	if at < 0 {
		t.Fatal("actual camera source absent")
	}
	x, y := at%data.Width, at/data.Width
	v := g.st.NativeMapViewState
	v.CameraX, v.CameraY = max(0, min(x-5, data.Width-13)), max(0, min(y-4, data.Height-8))
	v.CursorX, v.CursorY = x, y
	v.VisibleCursorX, v.VisibleCursorY = x-v.CameraX, y-v.CameraY
	if err = g.st.MaterializeNativeMapViewState(v); err != nil {
		t.Fatal(err)
	}
	g.curX, g.curY = x, y
	if !g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
		t.Fatal("cave fixture lacks HUD/range state")
	}
	g.interfaceMode = playerInterfaceHD
	g.camp = campaign.NewRunner(&campaign.Campaign{Start: "cave_fixture", Nodes: map[string]*campaign.Node{"cave_fixture": {Type: "battle"}}})
	if err = g.applyTheme(set); err != nil {
		t.Fatal(err)
	}
	g.modernHDMapComposing = true
	if err = g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tile := range g.modernHDMapFrame.Terrain {
		found = found || tile.Frame >= firstFrame && tile.Frame < firstFrame+16
	}
	if !found {
		t.Fatal("real Game.Draw has no new cave")
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
	p.cases = append(p.cases, sharedGroundGPUCase{g: g, name: "game-map09-hd-ui", mapID: 9, source: firstFrame, keepPNG: true})
	other := *g
	other.interfaceMode = playerInterfaceOriginal
	other.nativeMapWork = append([]byte(nil), g.nativeMapWork...)
	other.nativeMapVGA = append([]byte(nil), g.nativeMapVGA...)
	other.modernHDMapGPU = nil
	p.cases = append(p.cases, sharedGroundGPUCase{g: &other, name: "game-map09-original-ui", mapID: 9, source: firstFrame, keepPNG: true})
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err = ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil || p.index != 34 {
		t.Fatal(p.index, p.err)
	}
	after, _ := json.Marshal(g.st.Units)
	if !bytes.Equal(units, after) || rng != g.nativeRNGState {
		t.Fatal("art changed rules")
	}
	if err = g.saveGameToSlot(0); err != nil {
		t.Fatal(err)
	}
	resaved, err := os.ReadFile(saveSlotPath(0))
	if err != nil || !bytes.Equal(saved, resaved) {
		t.Fatal("art changed save", err)
	}
	frames := []int{}
	for f := firstFrame; f < firstFrame+16; f++ {
		frames = append(frames, f)
	}
	report := map[string]any{"status": status, "catalog_sha256": manifest.CatalogSHA, "manifest_sha256": fmt.Sprintf("%x", sha256.Sum256(raw)), "source_frames": frames, "source_cases": 32, "complete_game_cases": 2, "save_bytes_unchanged": len(saved), "cases": p.rows, "scope": "actual map9 art, controlled camera and chapter10 units; programmatic theme/UI controls, not normal player or original parity"}
	encoded, _ := json.MarshalIndent(report, "", "  ")
	if err = os.WriteFile(filepath.Join(out, "receipt.json"), append(encoded, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
