package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func TestRustGoldenR6DialogueMasks(t *testing.T) {
	out := os.Getenv("FD2_R6_OPENING_OUT")
	if out == "" {
		t.Skip("set FD2_R6_OPENING_OUT to export the Rust golden")
	}
	layouts := storyMaskBindingLayouts(t)
	g := storyMaskTestGame(t, "zh-Hant")
	g.interfaceMode = playerInterfaceOriginal
	base := make([]byte, 320*200)
	for i := range base {
		base[i] = byte(i*7 + i/320)
	}
	selected := make([]*campaign.NativeDialogueLayout, 0, 4)
	var report strings.Builder
	for _, control := range []string{"FFEC", "FFED", "FFEE", "FFEF"} {
		layout := layouts[control]
		selected = append(selected, layout)
		for _, motion := range []int{0, 37} {
			prepareStoryMaskUtterance(t, g, layout, base, motion, true)
			masks := g.nativeDialogueHDMasks
			if masks == nil {
				t.Fatalf("%s/%d: missing masks", control, motion)
			}
			for _, phase := range []struct {
				name   string
				frames [][]byte
				masks  [][]bool
			}{
				{"opening", g.nativeDialogueOpening, masks.opening},
				{"page", g.nativeDialogueFrames, masks.pages},
				{"closing", g.nativeDialogueClosing, masks.closing},
			} {
				for i, frame := range phase.frames {
					picture, err := modernHDTimedDialoguePixels(frame, phase.masks[i], g.nativeClassUI.palette)
					if err != nil {
						t.Fatal(err)
					}
					fmt.Fprintf(&report, "%s/%d %s %d %x\n", control, motion, phase.name, i, sha256.Sum256(picture.Pix))
				}
			}
		}
	}
	raw, err := json.Marshal(selected)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "dialogue-mask-layouts.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "go-dialogue-masks.txt"), []byte(report.String()), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestRustGoldenR6MapArt(t *testing.T) {
	out := os.Getenv("FD2_R6_OPENING_OUT")
	if out == "" {
		t.Skip("set FD2_R6_OPENING_OUT to export the Rust golden")
	}
	raw, err := os.ReadFile(os.Getenv("FD2_TEST_OPENING_CATALOG"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog modernThemeCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	chosen := catalog.Assets[:0]
	for _, asset := range catalog.Assets {
		switch asset.Role {
		case "map_sprite_set", "map_sprite_hd_set", "story_portrait_frame", "story_portrait_hd_frame", "facility_scene_hd":
			chosen = append(chosen, asset)
		}
	}
	catalog.Assets = chosen
	raw, err = json.Marshal(catalog)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "map-catalog.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	set, err := loadModernStoryPortraitSet(path, os.Getenv("FD2_TEST_OPENING_PACK"))
	if err != nil {
		t.Fatal(err)
	}
	var report strings.Builder
	pixels := func(source image.Image) [32]byte {
		pic := image.NewRGBA(source.Bounds())
		draw.Draw(pic, pic.Bounds(), source, source.Bounds().Min, draw.Src)
		return sha256.Sum256(pic.Pix)
	}
	total := 0
	for group := 0; group < fdicon.NativeSourceGroupCount; group++ {
		for frame, parent := range set.mapSprites[group] {
			area := set.mapSpriteVisualRect(group, frame, image.Rect(-15, 7, 129, 151))
			fmt.Fprintf(&report, "%d/%d %x %x %d %d %d %d\n", group, frame, pixels(parent),
				pixels(set.mapSpritesHD[group][frame]), area.Min.X, area.Min.Y, area.Max.X, area.Max.Y)
			total++
		}
	}
	if len(set.mapSpritesHD) != 125 || total != 1455 {
		t.Fatalf("image count %d / %d", len(set.mapSpritesHD), total)
	}
	if err := os.WriteFile(filepath.Join(out, "go-map-art.txt"), []byte(report.String()), 0600); err != nil {
		t.Fatal(err)
	}
	var speakers []int
	for id := range set.portraits {
		speakers = append(speakers, id)
	}
	sort.Ints(speakers)
	report.Reset()
	for _, id := range speakers {
		portrait := set.portraits[id]
		if portrait.hdImage == nil {
			t.Fatalf("missing formal HD portrait for speaker %d", id)
		}
		fmt.Fprintf(&report, "%d %x %x\n", id, pixels(portrait.image), pixels(portrait.hdImage))
	}
	if err := os.WriteFile(filepath.Join(out, "go-story-portraits.txt"), []byte(report.String()), 0600); err != nil {
		t.Fatal(err)
	}
}

type rustR6LayersProbe struct {
	game   *Game
	frame  *modernHDMapFrame
	report strings.Builder
	phase  int
	failed bool
}

func (p *rustR6LayersProbe) Update() error {
	if p.failed || p.phase == 2 {
		return ebiten.Termination
	}
	return nil
}

func (p *rustR6LayersProbe) Layout(int, int) (int, int) { return 640, 400 }

func (p *rustR6LayersProbe) Draw(screen *ebiten.Image) {
	if p.failed || p.phase == 2 {
		return
	}
	p.frame.OverlaysAboveUI = p.phase == 1
	if !p.game.drawPreparedModernHDMapFrame(screen, p.frame, false) {
		p.failed = true
		return
	}
	for _, picture := range []*ebiten.Image{p.game.modernHDMapGPU.World, screen} {
		pixels := make([]byte, picture.Bounds().Dx()*picture.Bounds().Dy()*4)
		picture.ReadPixels(pixels)
		fmt.Fprintf(&p.report, "%x\n", sha256.Sum256(pixels))
	}
	p.phase++
}

func TestRustGoldenR6MapLayers(t *testing.T) {
	out := os.Getenv("FD2_R6_OPENING_OUT")
	if out == "" {
		t.Skip("set FD2_R6_OPENING_OUT to export the Rust golden")
	}
	paint := func(w, h int, area image.Rectangle, c color.RGBA) *image.RGBA {
		p := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := area.Min.Y; y < area.Max.Y; y++ {
			for x := area.Min.X; x < area.Max.X; x++ {
				p.SetRGBA(x, y, c)
			}
		}
		return p
	}
	rect := image.Rect
	unit := paint(144, 144, rect(5, 8, 111, 129), color.RGBA{120, 100, 60, 255})
	hud := modernHDMapUnitDraw{Group: 0, Rect: rect(66, 56, 210, 200)}
	p := &rustR6LayersProbe{
		game: &Game{modernStoryPortraits: &modernStoryPortraitSet{mapSpritesHD: map[int][]image.Image{0: {unit}}}},
		frame: &modernHDMapFrame{
			Background: paint(320, 200, rect(0, 0, 320, 200), color.RGBA{12, 24, 36, 255}),
			UI:         paint(320, 200, rect(13, 12, 20, 20), color.RGBA{30, 20, 150, 255}),
			Viewport:   rect(31, 33, 148, 150),
			Terrain: []modernHDTerrainDraw{{Rect: rect(15, 20, 159, 164),
				Image: paint(144, 144, rect(0, 0, 135, 120), color.RGBA{21, 92, 52, 255})}},
			Range: paint(320, 200, rect(6, 9, 14, 21), color.RGBA{100, 10, 49, 255}),
			Units: []modernHDMapUnitDraw{{Group: 0, Rect: rect(48, 44, 192, 188), PaletteBand: true}},
			Foreground: []modernHDMapForegroundDraw{
				{Rect: rect(25, 70, 169, 214), Image: paint(24, 24, rect(3, 0, 9, 21), color.RGBA{99, 40, 21, 255})},
				{Rect: rect(60, 30, 204, 174), HDImage: paint(144, 144, rect(6, 10, 33, 120), color.RGBA{10, 24, 52, 128})},
			},
			Overlays: []modernHDMapOverlayDraw{{Rect: rect(79, 67, 105, 119), ScaleToRect: true,
				Image: paint(13, 26, rect(0, 0, 13, 26), color.RGBA{75, 91, 0, 255})}},
			HUDUnit: &modernHDMapHUDUnit{Unit: hud,
				Base:  paint(24, 24, rect(2, 0, 8, 21), color.RGBA{35, 22, 86, 255}),
				Above: paint(24, 24, rect(9, 3, 12, 15), color.RGBA{121, 18, 90, 255})},
		},
	}
	ebiten.SetWindowSize(640, 400)
	ebiten.SetVsyncEnabled(false)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.failed || p.phase != 2 {
		t.Fatalf("layer composition stopped at %d", p.phase)
	}
	if err := os.WriteFile(filepath.Join(out, "go-map-layers.txt"), []byte(p.report.String()), 0600); err != nil {
		t.Fatal(err)
	}
}

// TestRustGoldenR6Terrain 輸出 Go 正式地形取用與原版來源核對的結果。
func TestRustGoldenR6Terrain(t *testing.T) {
	out := os.Getenv("FD2_R6_OPENING_OUT")
	if out == "" {
		t.Skip("set FD2_R6_OPENING_OUT to export the Rust golden")
	}
	raw, err := os.ReadFile(os.Getenv("FD2_TEST_OPENING_CATALOG"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog modernThemeCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	set := &modernStoryPortraitSet{}
	for _, asset := range catalog.Assets {
		if asset.Role == "native_terrain_hd_set" {
			if err := loadModernTerrainHDAsset(set, asset, os.Getenv("FD2_TEST_OPENING_PACK")); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(set.terrainHD) != 33 {
		t.Fatalf("terrain map count %d, want 33", len(set.terrainHD))
	}
	var report, projected strings.Builder
	total := 0
	for id := 0; id < 33; id++ {
		bank, _, err := fdicon.LoadSeparatedFDSHAPBank(separatedAssetPath("tilesets/fdshap"), id)
		if err != nil {
			t.Fatal(err)
		}
		if err := set.validateTerrainHD(id, bank); err != nil {
			t.Fatalf("map %d: %v", id, err)
		}
		frames, err := set.terrainHDMap(id)
		if err != nil || len(frames) == 0 {
			t.Fatalf("map %d has no checked frames: %v", id, err)
		}
		var indexes []int
		for index := range frames {
			indexes = append(indexes, index)
		}
		sort.Ints(indexes)
		for _, index := range indexes {
			frame := frames[index]
			fmt.Fprintf(&report, "%d/%d %s %x %x\n", id, index, frame.Role,
				sha256.Sum256(frame.Image.Pix), sha256.Sum256(frame.Colours))
			total++
		}
		if id == 0 || id == 9 || id == 20 {
			exportRustR6TerrainPlan(t, &projected, id, frames, indexes)
		}
	}
	if total != 3983 {
		t.Fatalf("terrain frame count %d, want 3983", total)
	}
	if err := os.WriteFile(filepath.Join(out, "go-terrain.txt"), []byte(report.String()), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "go-terrain-plan.txt"), []byte(projected.String()), 0600); err != nil {
		t.Fatal(err)
	}
}

func exportRustR6TerrainPlan(t *testing.T, out *strings.Builder, id int, frames map[int]*modernTerrainHDAsset, indexes []int) {
	t.Helper()
	var tiles []int
	for _, index := range indexes {
		if frames[index].supports("terrain") {
			tiles = append(tiles, index)
		}
		if len(tiles) == 4 {
			break
		}
	}
	if len(tiles) != 4 {
		t.Fatalf("map %d: insufficient terrain tiles", id)
	}
	base, current, lut := make(color.Palette, 256), make(color.Palette, 256), make([]byte, 256)
	for i := range base {
		r, g, b := byte(i), byte(i*3), byte(i*7)
		base[i], current[i], lut[i] = color.RGBA{r, g, b, 255}, color.RGBA{g, b, r, 255}, byte(255-i)
	}
	colours, err := terrainColourContext(id, base, current, lut)
	if err != nil {
		t.Fatal(err)
	}
	input := indexedmap.FrameInput{MapWidth: 15, CameraX: 1, CameraY: 2, Controls: make([]byte, 4096)}
	for i := 0; i < 15*12; i++ {
		mode := byte(0)
		if (i/4)%2 == 1 {
			mode = 0xff
		}
		input.Cells = append(input.Cells, fdicon.NativeTerrainCell{Tile: uint16(tiles[i%4]) | 0x8000, BlitMode: mode})
	}
	emit := func(label string, draws []modernHDTerrainDraw, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		if len(draws) == 0 {
			t.Fatalf("map %d: empty plan %s", id, label)
		}
		for _, draw := range draws {
			fmt.Fprintf(out, "%d/%s %d %d %d %d %d %d %x\n", id, label, draw.Frame, draw.WorkOffset,
				draw.Rect.Min.X, draw.Rect.Min.Y, draw.Rect.Max.X, draw.Rect.Max.Y, sha256.Sum256(draw.Image.Pix))
		}
	}
	const steadyBase = 0x8088
	draws, err := buildModernHDTerrainPlan(input, steadyBase, input.CameraY, 8, steadyBase, frames, colours)
	emit("steady", draws, err)
	for motion := 1; motion <= 6; motion++ {
		terrainBase, cameraY, rows, viewportBase, err := indexedmap.NativeUpStepLayout(input, input.CameraY, motion)
		if err != nil {
			t.Fatal(err)
		}
		draws, err := buildModernHDTerrainPlan(input, terrainBase, cameraY, rows, viewportBase, frames, colours)
		emit(fmt.Sprintf("up%d", motion), draws, err)
	}
	draws, err = buildModernHDTerrainRegionPlan(input, steadyBase, input.CameraY, 3, 2, steadyBase,
		image.Rect(100, 150, 168, 200), frames, colours)
	emit("clip", draws, err)
}

func TestRustGoldenR6MapGeometry(t *testing.T) {
	out := os.Getenv("FD2_R6_OPENING_OUT")
	if out == "" {
		t.Skip("set FD2_R6_OPENING_OUT to export the Rust golden")
	}
	var report strings.Builder
	rect := func(r image.Rectangle) string {
		return fmt.Sprintf("%d %d %d %d", r.Min.X, r.Min.Y, r.Max.X, r.Max.Y)
	}
	const base = 0x8088
	in, counts := hdUnitPlanFixture(t)
	positions := [][2]int{{-1, -1}, {0, 0}, {12, 8}, {13, 0}, {0, 9}, {-2, 0}, {4, 3}}
	for _, viewport := range []int{base, base - 12*fdicon.NativeMapStride} {
		for pose := 0; pose < 4; pose++ {
			for motion := 0; motion <= 6; motion++ {
				for force := 0; force < 2; force++ {
					for shift := 0; shift < 2; shift++ {
						in.PixelShift = shift
						for _, pos := range positions {
							in.Units = []fdicon.NativeUnitLayerEntry{{
								X: in.CameraX + pos[0], Y: in.CameraY + pos[1], Slot: 1, Pose: pose,
								MotionOffset: motion, ForceBase: force != 0, Flags: 0x80,
							}}
							plan, err := buildModernHDMapUnitPlan(in, viewport, counts)
							if err != nil {
								t.Fatal(err)
							}
							fmt.Fprintf(&report, "unit %d %d %d %d %d %d %d %d\n",
								viewport, pose, motion, force, shift, pos[0], pos[1], len(plan))
							for _, draw := range plan {
								fmt.Fprintf(&report, "%d %d %d %t %s\n", draw.Group, draw.Frame, draw.WorkOffset, draw.PaletteBand, rect(draw.Rect))
							}
						}
					}
				}
			}
		}
	}

	in, palette := hdForegroundPlanFixture()
	for pose := byte(0); pose < 4; pose++ {
		for _, motion := range []int{0, 3, 6} {
			for flip := 0; flip < 2; flip++ {
				in.Flip = flip
				for _, mode := range []byte{0, 0xff} {
					for i := range in.Cells {
						in.Cells[i].BlitMode = mode
					}
					for suppressed := 0; suppressed < 2; suppressed++ {
						in.ForegroundUnits = []fdicon.NativeForegroundLayerEntry{
							{X: 4, Y: 4, Pose: pose, MotionOffset: motion, Race: byte(4 * suppressed)},
							{X: 1, Y: 1}, {X: 14, Y: 10}, {X: 16, Y: 12}, {X: -1, Y: -1}, {X: 3, Y: 3, Inactive: true},
						}
						plan, err := buildModernHDMapForegroundPlan(in, base, palette)
						if err != nil {
							t.Fatal(err)
						}
						fmt.Fprintf(&report, "foreground %d %d %d %d %d %d\n", pose, motion, flip, mode, suppressed, len(plan))
						for _, draw := range plan {
							fmt.Fprintf(&report, "%d %d %d %s %x\n", draw.SourceFrame, draw.WorkOffset, draw.BlitMode,
								rect(draw.Rect), sha256.Sum256(draw.Image.Pix))
						}
					}
				}
			}
		}
	}
	if err := os.WriteFile(filepath.Join(out, "go-map-geometry.txt"), []byte(report.String()), 0600); err != nil {
		t.Fatal(err)
	}
}
